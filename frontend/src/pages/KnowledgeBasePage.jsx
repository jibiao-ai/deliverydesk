import React, { useState } from 'react';
import { BookOpen, Send, Eye, ExternalLink, Loader2, AlertTriangle, CheckCircle2, Copy, RefreshCw } from 'lucide-react';
import toast from 'react-hot-toast';
import { previewKB, publishKB } from '../services/api';

export default function KnowledgeBasePage() {
  const [issueKey, setIssueKey] = useState('');
  const [jiraServer, setJiraServer] = useState('');
  const [jiraUser, setJiraUser] = useState('');
  const [jiraToken, setJiraToken] = useState('');
  const [confluenceURL, setConfluenceURL] = useState('');
  const [loading, setLoading] = useState(false);
  const [publishing, setPublishing] = useState(false);
  const [preview, setPreview] = useState(null); // { title, preview, content }
  const [publishResult, setPublishResult] = useState(null);
  const [showCredentials, setShowCredentials] = useState(false);
  const [step, setStep] = useState(1); // 1: input, 2: preview, 3: published

  // Parse issue key from URL or direct input
  const parseIssueInput = (input) => {
    const trimmed = input.trim();
    // If it's a URL like https://easystack.atlassian.net/browse/ECSL2-50377
    const urlMatch = trimmed.match(/\/browse\/([A-Z0-9]+-\d+)/i);
    if (urlMatch) {
      setIssueKey(urlMatch[1]);
      // Also extract server if not set
      const serverMatch = trimmed.match(/^(https?:\/\/[^/]+)/);
      if (serverMatch && !jiraServer) {
        setJiraServer(serverMatch[1]);
      }
    } else {
      setIssueKey(trimmed);
    }
  };

  const handlePreview = async () => {
    if (!issueKey) {
      toast.error('请输入工单号');
      return;
    }
    setLoading(true);
    setPreview(null);
    setPublishResult(null);
    try {
      const res = await previewKB({
        issue_key: issueKey,
        jira_server: jiraServer || undefined,
        jira_user: jiraUser || undefined,
        jira_token: jiraToken || undefined,
      });
      if (res?.code === 0 && res.data) {
        setPreview(res.data);
        setStep(2);
        toast.success('知识库内容生成成功！');
      } else {
        toast.error(res?.message || '生成失败');
      }
    } catch (e) {
      toast.error('请求失败: ' + (e.message || '网络错误'));
    }
    setLoading(false);
  };

  const handlePublish = async () => {
    if (!confluenceURL) {
      toast.error('请输入 Confluence 目标页面 URL');
      return;
    }
    if (!preview?.content) {
      toast.error('请先生成预览内容');
      return;
    }
    setPublishing(true);
    try {
      const res = await publishKB({
        content: preview.content,
        confluence_url: confluenceURL,
        jira_server: jiraServer || undefined,
        jira_user: jiraUser || undefined,
        jira_token: jiraToken || undefined,
        issue_key: issueKey,
        desc_html: preview.preview, // pass preview HTML as desc for attachment context
      });
      if (res?.code === 0 && res.data) {
        setPublishResult(res.data);
        setStep(3);
        toast.success('知识库页面已发布到 Confluence！');
      } else {
        toast.error(res?.message || '发布失败');
      }
    } catch (e) {
      toast.error('发布失败: ' + (e.message || '网络错误'));
    }
    setPublishing(false);
  };

  const handleReset = () => {
    setStep(1);
    setPreview(null);
    setPublishResult(null);
  };

  return (
    <div className="h-full overflow-y-auto p-6">
      <div className="max-w-5xl mx-auto space-y-6">
        {/* Header */}
        <div className="flex items-center gap-3 mb-2">
          <div className="w-10 h-10 rounded-xl bg-gradient-to-br from-indigo-500 to-purple-600 flex items-center justify-center">
            <BookOpen className="w-5 h-5 text-white" />
          </div>
          <div>
            <h2 className="text-xl font-bold text-gray-800">Jira → Confluence 知识库</h2>
            <p className="text-sm text-gray-500">将 Jira 工单处理过程整理为结构化知识库并发布到 Confluence</p>
          </div>
        </div>

        {/* Steps indicator */}
        <div className="flex items-center gap-2 bg-white rounded-xl border border-gray-200 p-4">
          {[
            { num: 1, label: '输入信息', desc: '填写工单号和认证' },
            { num: 2, label: 'AI 润色预览', desc: '查看生成内容' },
            { num: 3, label: '发布完成', desc: '知识库已上线' },
          ].map((s, i) => (
            <React.Fragment key={s.num}>
              {i > 0 && <div className={`flex-1 h-0.5 ${step >= s.num ? 'bg-primary-500' : 'bg-gray-200'}`} />}
              <div className="flex items-center gap-2">
                <div className={`w-7 h-7 rounded-full flex items-center justify-center text-xs font-bold ${
                  step >= s.num ? 'bg-primary-600 text-white' : 'bg-gray-200 text-gray-500'
                }`}>
                  {step > s.num ? <CheckCircle2 className="w-4 h-4" /> : s.num}
                </div>
                <div className="hidden sm:block">
                  <p className={`text-xs font-medium ${step >= s.num ? 'text-primary-600' : 'text-gray-500'}`}>{s.label}</p>
                  <p className="text-[10px] text-gray-400">{s.desc}</p>
                </div>
              </div>
            </React.Fragment>
          ))}
        </div>

        {/* Step 1: Input */}
        {step === 1 && (
          <div className="bg-white rounded-xl border border-gray-200 p-6 space-y-5">
            <div>
              <label className="block text-sm font-medium text-gray-700 mb-1.5">
                工单号 / 工单链接 <span className="text-red-500">*</span>
              </label>
              <input
                type="text"
                placeholder="例: ECSL2-50377 或 https://easystack.atlassian.net/browse/ECSL2-50377"
                value={issueKey}
                onChange={(e) => parseIssueInput(e.target.value)}
                className="w-full px-4 py-2.5 border border-gray-300 rounded-lg focus:ring-2 focus:ring-primary-500 focus:border-transparent text-sm"
              />
              <p className="mt-1 text-xs text-gray-400">支持直接粘贴 Jira 工单 URL，会自动解析工单号和服务器地址</p>
            </div>

            <div>
              <label className="block text-sm font-medium text-gray-700 mb-1.5">
                Confluence 目标页面 URL <span className="text-red-500">*</span>
              </label>
              <input
                type="text"
                placeholder="例: https://easystack.atlassian.net/wiki/spaces/PSBC/pages/3495297025"
                value={confluenceURL}
                onChange={(e) => setConfluenceURL(e.target.value)}
                className="w-full px-4 py-2.5 border border-gray-300 rounded-lg focus:ring-2 focus:ring-primary-500 focus:border-transparent text-sm"
              />
              <p className="mt-1 text-xs text-gray-400">知识库将作为该页面的子页面创建</p>
            </div>

            {/* Credentials toggle */}
            <div>
              <button
                onClick={() => setShowCredentials(!showCredentials)}
                className="text-sm text-primary-600 hover:text-primary-700 font-medium flex items-center gap-1"
              >
                {showCredentials ? '收起' : '展开'} Jira/Confluence 认证配置
                <span className="text-xs text-gray-400">（留空则使用系统设置中的配置）</span>
              </button>
              {showCredentials && (
                <div className="mt-3 grid grid-cols-1 sm:grid-cols-2 gap-4 p-4 bg-gray-50 rounded-lg border border-gray-200">
                  <div>
                    <label className="block text-xs font-medium text-gray-600 mb-1">Jira 服务器</label>
                    <input
                      type="text"
                      placeholder="https://easystack.atlassian.net"
                      value={jiraServer}
                      onChange={(e) => setJiraServer(e.target.value)}
                      className="w-full px-3 py-2 border border-gray-300 rounded-lg text-sm focus:ring-2 focus:ring-primary-500 focus:border-transparent"
                    />
                  </div>
                  <div>
                    <label className="block text-xs font-medium text-gray-600 mb-1">用户名（邮箱）</label>
                    <input
                      type="text"
                      placeholder="esoncall@easystack.cn"
                      value={jiraUser}
                      onChange={(e) => setJiraUser(e.target.value)}
                      className="w-full px-3 py-2 border border-gray-300 rounded-lg text-sm focus:ring-2 focus:ring-primary-500 focus:border-transparent"
                    />
                  </div>
                  <div className="sm:col-span-2">
                    <label className="block text-xs font-medium text-gray-600 mb-1">API Token</label>
                    <input
                      type="password"
                      placeholder="ATATT3xFfGF0..."
                      value={jiraToken}
                      onChange={(e) => setJiraToken(e.target.value)}
                      className="w-full px-3 py-2 border border-gray-300 rounded-lg text-sm focus:ring-2 focus:ring-primary-500 focus:border-transparent"
                    />
                  </div>
                </div>
              )}
            </div>

            {/* Action */}
            <div className="flex items-center gap-3 pt-2">
              <button
                onClick={handlePreview}
                disabled={loading || !issueKey}
                className="flex items-center gap-2 px-5 py-2.5 bg-primary-600 text-white rounded-lg hover:bg-primary-700 disabled:opacity-50 disabled:cursor-not-allowed transition-colors text-sm font-medium"
              >
                {loading ? <Loader2 className="w-4 h-4 animate-spin" /> : <Eye className="w-4 h-4" />}
                {loading ? '正在获取工单并生成知识库...' : '生成预览'}
              </button>
              {loading && (
                <span className="text-xs text-gray-400">此过程需要调用大模型润色，可能需要 30-60 秒</span>
              )}
            </div>

            {/* Info card */}
            <div className="bg-indigo-50 border border-indigo-100 rounded-lg p-4">
              <h4 className="text-sm font-semibold text-indigo-800 mb-2 flex items-center gap-1.5">
                <BookOpen className="w-4 h-4" /> 知识库生成说明
              </h4>
              <ul className="text-xs text-indigo-700 space-y-1.5">
                <li>• 自动从 Jira 拉取工单字段、评论、状态变更历史和附件</li>
                <li>• 调用系统配置的大模型（AI 模型页）对内容进行智能润色</li>
                <li>• 生成七段式知识库：问题背景 / 工单信息 / 客户名称 / 问题现象 / 解决过程 / 问题总结 / 改进建议</li>
                <li>• 发布为 Confluence 子页面，附件自动上传并嵌入</li>
                <li>• 认证优先使用本页填写的信息，留空时自动使用系统设置中的 Jira 配置</li>
              </ul>
            </div>
          </div>
        )}

        {/* Step 2: Preview */}
        {step === 2 && preview && (
          <div className="space-y-4">
            {/* Title & actions */}
            <div className="bg-white rounded-xl border border-gray-200 p-5">
              <div className="flex items-center justify-between mb-4">
                <div>
                  <h3 className="text-lg font-bold text-gray-800">{preview.title || preview.content?.title}</h3>
                  <p className="text-sm text-gray-500 mt-0.5">预览生成的知识库内容（AI 润色后）</p>
                </div>
                <div className="flex items-center gap-2">
                  <button
                    onClick={handleReset}
                    className="flex items-center gap-1.5 px-3 py-2 text-sm border border-gray-300 text-gray-600 rounded-lg hover:bg-gray-50 transition-colors"
                  >
                    <RefreshCw className="w-3.5 h-3.5" /> 重新生成
                  </button>
                  <button
                    onClick={handlePublish}
                    disabled={publishing || !confluenceURL}
                    className="flex items-center gap-1.5 px-4 py-2 bg-green-600 text-white text-sm rounded-lg hover:bg-green-700 disabled:opacity-50 disabled:cursor-not-allowed transition-colors font-medium"
                  >
                    {publishing ? <Loader2 className="w-4 h-4 animate-spin" /> : <Send className="w-4 h-4" />}
                    {publishing ? '发布中...' : '发布到 Confluence'}
                  </button>
                </div>
              </div>

              {!confluenceURL && (
                <div className="flex items-center gap-2 p-3 bg-amber-50 border border-amber-200 rounded-lg mb-4">
                  <AlertTriangle className="w-4 h-4 text-amber-600 flex-shrink-0" />
                  <span className="text-sm text-amber-700">请先填写 Confluence 目标页面 URL 后再发布</span>
                  <input
                    type="text"
                    placeholder="https://easystack.atlassian.net/wiki/spaces/PSBC/pages/..."
                    value={confluenceURL}
                    onChange={(e) => setConfluenceURL(e.target.value)}
                    className="flex-1 px-3 py-1.5 border border-amber-300 rounded text-sm focus:ring-2 focus:ring-amber-400 focus:border-transparent"
                  />
                </div>
              )}
            </div>

            {/* Structured content display */}
            {preview.content && (
              <div className="bg-white rounded-xl border border-gray-200 p-6 space-y-5">
                {/* Background */}
                <section>
                  <h4 className="text-base font-bold text-gray-800 border-l-4 border-primary-500 pl-3 mb-2">一、问题背景</h4>
                  <p className="text-sm text-gray-700 leading-relaxed">{preview.content.background}</p>
                </section>

                {/* Ticket Table */}
                <section>
                  <h4 className="text-base font-bold text-gray-800 border-l-4 border-primary-500 pl-3 mb-2">二、工单信息</h4>
                  <div className="overflow-x-auto text-sm" dangerouslySetInnerHTML={{ __html: preview.content.ticket_table }} />
                </section>

                {/* Customer */}
                <section>
                  <h4 className="text-base font-bold text-gray-800 border-l-4 border-primary-500 pl-3 mb-2">三、客户名称</h4>
                  <p className="text-sm text-gray-700">{preview.content.customer}</p>
                </section>

                {/* Timeline */}
                <section>
                  <h4 className="text-base font-bold text-gray-800 border-l-4 border-primary-500 pl-3 mb-2">五、解决过程</h4>
                  <h5 className="text-sm font-semibold text-gray-700 mb-2">5.1 时间线</h5>
                  {preview.content.timeline && preview.content.timeline.length > 0 && (
                    <div className="overflow-x-auto">
                      <table className="min-w-full text-sm border border-gray-200 rounded-lg">
                        <thead className="bg-gray-50">
                          <tr>
                            <th className="px-3 py-2 text-left font-medium text-gray-600 border-b">时间</th>
                            <th className="px-3 py-2 text-left font-medium text-gray-600 border-b">操作人</th>
                            <th className="px-3 py-2 text-left font-medium text-gray-600 border-b">内容</th>
                          </tr>
                        </thead>
                        <tbody>
                          {preview.content.timeline.map((ev, i) => (
                            <tr key={i} className="border-b border-gray-100 hover:bg-gray-50">
                              <td className="px-3 py-2 text-gray-600 whitespace-nowrap text-xs">{ev.time}</td>
                              <td className="px-3 py-2 text-gray-700 whitespace-nowrap">{ev.author}</td>
                              <td className="px-3 py-2 text-gray-700">{ev.content}</td>
                            </tr>
                          ))}
                        </tbody>
                      </table>
                    </div>
                  )}
                  <h5 className="text-sm font-semibold text-gray-700 mt-4 mb-2">5.2 处理内容汇总</h5>
                  <p className="text-sm text-gray-700 leading-relaxed bg-gray-50 p-3 rounded-lg">{preview.content.process_summary}</p>
                </section>

                {/* Summary */}
                <section>
                  <h4 className="text-base font-bold text-gray-800 border-l-4 border-primary-500 pl-3 mb-2">六、问题总结</h4>
                  <div className="space-y-3">
                    <div>
                      <h5 className="text-sm font-semibold text-gray-700 mb-1">技术结论</h5>
                      <p className="text-sm text-gray-700 leading-relaxed bg-blue-50 p-3 rounded-lg">{preview.content.tech_conclusion}</p>
                    </div>
                    <div>
                      <h5 className="text-sm font-semibold text-gray-700 mb-1">结果</h5>
                      <p className="text-sm text-gray-700 leading-relaxed">{preview.content.result}</p>
                    </div>
                  </div>
                </section>

                {/* Suggestions */}
                <section>
                  <h4 className="text-base font-bold text-gray-800 border-l-4 border-primary-500 pl-3 mb-2">七、改进建议</h4>
                  <ol className="list-decimal list-inside space-y-1.5 text-sm text-gray-700">
                    {(preview.content.suggestions || []).map((s, i) => (
                      <li key={i} className="leading-relaxed">{s}</li>
                    ))}
                  </ol>
                </section>
              </div>
            )}
          </div>
        )}

        {/* Step 3: Published */}
        {step === 3 && publishResult && (
          <div className="bg-white rounded-xl border border-gray-200 p-8 text-center space-y-4">
            <div className="w-16 h-16 bg-green-100 rounded-full flex items-center justify-center mx-auto">
              <CheckCircle2 className="w-8 h-8 text-green-600" />
            </div>
            <h3 className="text-xl font-bold text-gray-800">知识库页面已发布！</h3>
            <p className="text-sm text-gray-500">{publishResult.title}</p>
            {publishResult.page_url && (
              <div className="flex items-center justify-center gap-2 mt-4">
                <a
                  href={publishResult.page_url}
                  target="_blank"
                  rel="noopener noreferrer"
                  className="flex items-center gap-2 px-5 py-2.5 bg-primary-600 text-white rounded-lg hover:bg-primary-700 transition-colors text-sm font-medium"
                >
                  <ExternalLink className="w-4 h-4" /> 在 Confluence 中查看
                </a>
                <button
                  onClick={() => { navigator.clipboard.writeText(publishResult.page_url); toast.success('链接已复制'); }}
                  className="flex items-center gap-2 px-4 py-2.5 border border-gray-300 text-gray-600 rounded-lg hover:bg-gray-50 transition-colors text-sm"
                >
                  <Copy className="w-4 h-4" /> 复制链接
                </button>
              </div>
            )}
            <div className="pt-4">
              <button
                onClick={handleReset}
                className="text-sm text-primary-600 hover:text-primary-700 font-medium"
              >
                ← 继续生成下一个知识库
              </button>
            </div>
          </div>
        )}
      </div>
    </div>
  );
}
