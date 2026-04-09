import React, { useState, useEffect, useCallback } from 'react';
import { Bot, Plus, Trash2, Edit3, Loader2, Zap, Eye, EyeOff, Shield, X, Check, Copy, ExternalLink, AlertTriangle } from 'lucide-react';
import { getAgents, createAgent, updateAgent, deleteAgent, getSkills } from '../services/api';
import toast from 'react-hot-toast';

// ── Elegant Delete Confirmation Modal ─────────────────────────────────────────
function DeleteAgentConfirm({ agent, onCancel, onConfirm, deleting }) {
  return (
    <div className="fixed inset-0 bg-black/40 backdrop-blur-sm z-50 flex items-center justify-center p-4" onClick={onCancel}>
      <div className="bg-white rounded-2xl shadow-2xl w-full max-w-sm overflow-hidden" onClick={e => e.stopPropagation()}>
        {/* Top accent bar */}
        <div className="h-1 bg-gradient-to-r from-red-400 via-red-500 to-red-600" />
        <div className="p-6 text-center">
          <div className="w-14 h-14 rounded-full bg-red-50 flex items-center justify-center mx-auto mb-4">
            <AlertTriangle className="w-7 h-7 text-red-500" />
          </div>
          <h3 className="text-lg font-semibold text-gray-800 mb-2">确认删除智能体</h3>
          <p className="text-sm text-gray-500 mb-1">
            确定要删除智能体
          </p>
          <p className="text-sm font-semibold text-gray-800 mb-1">
            <span className="inline-flex items-center gap-1.5 px-2.5 py-1 bg-primary-50 text-primary-700 rounded-lg">
              <Bot className="w-3.5 h-3.5" />{agent.name}
            </span>
          </p>
          <p className="text-xs text-gray-400 mt-3">
            此操作不可撤销，相关的对话记录也将被清除
          </p>
        </div>
        <div className="px-6 pb-6 flex items-center gap-3 justify-center">
          <button
            onClick={onCancel}
            className="px-5 py-2.5 text-sm font-medium text-gray-600 bg-gray-100 rounded-xl hover:bg-gray-200 transition-colors"
          >
            取消
          </button>
          <button
            onClick={onConfirm}
            disabled={deleting}
            className="px-5 py-2.5 text-sm font-medium text-white bg-red-500 rounded-xl hover:bg-red-600 transition-colors disabled:opacity-60 flex items-center gap-2 shadow-sm shadow-red-200"
          >
            {deleting ? <Loader2 className="w-4 h-4 animate-spin" /> : <Trash2 className="w-4 h-4" />}
            确认删除
          </button>
        </div>
      </div>
    </div>
  );
}

export default function AgentsPage() {
  const [agents, setAgents] = useState([]);
  const [skills, setSkills] = useState([]);
  const [loading, setLoading] = useState(true);
  const [showModal, setShowModal] = useState(false);
  const [editingAgent, setEditingAgent] = useState(null);
  const [showApiModal, setShowApiModal] = useState(null);
  const [deleteTarget, setDeleteTarget] = useState(null);
  const [deleting, setDeleting] = useState(false);
  const [form, setForm] = useState({
    name: '', description: '', system_prompt: '', model: '',
    temperature: 0.7, max_tokens: 4096, is_active: true,
    is_published: false, iron_rules: false, skill_ids: [],
  });

  const loadData = useCallback(async () => {
    setLoading(true);
    try {
      const [agentRes, skillRes] = await Promise.all([getAgents(), getSkills()]);
      if (agentRes.code === 0) setAgents(agentRes.data || []);
      if (skillRes.code === 0) setSkills(skillRes.data || []);
    } catch (e) { toast.error('加载失败'); }
    finally { setLoading(false); }
  }, []);

  useEffect(() => { loadData(); }, [loadData]);

  const handleCreate = () => {
    setEditingAgent(null);
    setForm({ name: '', description: '', system_prompt: '', model: '', temperature: 0.7, max_tokens: 4096, is_active: true, is_published: false, iron_rules: false, skill_ids: [] });
    setShowModal(true);
  };

  const handleEdit = (agent) => {
    setEditingAgent(agent);
    const skillIds = (agent.agent_skills || []).map(as => as.skill_id);
    setForm({
      name: agent.name, description: agent.description, system_prompt: agent.system_prompt || '',
      model: agent.model || '', temperature: agent.temperature, max_tokens: agent.max_tokens,
      is_active: agent.is_active, is_published: agent.is_published || false,
      iron_rules: agent.iron_rules || false, skill_ids: skillIds,
    });
    setShowModal(true);
  };

  const handleSave = async () => {
    if (!form.name.trim()) { toast.error('请输入智能体名称'); return; }
    try {
      const data = { ...form };
      if (editingAgent) {
        const res = await updateAgent(editingAgent.id, data);
        if (res.code === 0) { toast.success('更新成功'); loadData(); setShowModal(false); }
        else toast.error(res.message || '更新失败');
      } else {
        const res = await createAgent(data);
        if (res.code === 0) { toast.success('创建成功'); loadData(); setShowModal(false); }
        else toast.error(res.message || '创建失败');
      }
    } catch (e) { toast.error('操作失败'); }
  };

  const handleDelete = async () => {
    if (!deleteTarget) return;
    setDeleting(true);
    try {
      const res = await deleteAgent(deleteTarget.id);
      if (res.code === 0) { toast.success('删除成功'); setDeleteTarget(null); loadData(); }
      else toast.error(res.message || '删除失败');
    } catch (e) { toast.error('删除失败'); }
    finally { setDeleting(false); }
  };

  const togglePublish = async (agent) => {
    const newVal = !agent.is_published;
    try {
      const res = await updateAgent(agent.id, { is_published: newVal });
      if (res.code === 0) {
        toast.success(newVal ? '已发布为外部接口' : '已取消发布');
        loadData();
      }
    } catch (e) { toast.error('操作失败'); }
  };

  const toggleSkill = (skillId) => {
    setForm(prev => {
      const ids = prev.skill_ids.includes(skillId)
        ? prev.skill_ids.filter(id => id !== skillId)
        : [...prev.skill_ids, skillId];
      return { ...prev, skill_ids: ids };
    });
  };

  const copyApiUrl = (agentId) => {
    const url = `${window.location.origin}/api/published-agents/${agentId}/chat`;
    navigator.clipboard.writeText(url);
    toast.success('API地址已复制');
  };

  return (
    <div className="h-full overflow-y-auto">
      <div className="p-4 sm:p-6 space-y-4">
        {/* Stats */}
        <div className="grid grid-cols-2 sm:grid-cols-4 gap-3">
          <div className="bg-white rounded-xl border border-gray-200 p-4">
            <p className="text-xs text-gray-500">智能体总数</p>
            <p className="text-2xl font-bold text-gray-800">{agents.length}</p>
          </div>
          <div className="bg-white rounded-xl border border-gray-200 p-4">
            <p className="text-xs text-gray-500">已启用</p>
            <p className="text-2xl font-bold text-green-600">{agents.filter(a => a.is_active).length}</p>
          </div>
          <div className="bg-white rounded-xl border border-gray-200 p-4">
            <p className="text-xs text-gray-500">已发布</p>
            <p className="text-2xl font-bold text-blue-600">{agents.filter(a => a.is_published).length}</p>
          </div>
          <div className="bg-white rounded-xl border border-gray-200 p-4">
            <p className="text-xs text-gray-500">铁律模式</p>
            <p className="text-2xl font-bold text-orange-600">{agents.filter(a => a.iron_rules).length}</p>
          </div>
        </div>

        {/* Header */}
        <div className="flex items-center justify-between">
          <h2 className="text-lg font-semibold text-gray-800">智能体列表</h2>
          <button onClick={handleCreate}
            className="flex items-center gap-2 px-4 py-2 bg-primary-600 text-white rounded-lg text-sm hover:bg-primary-700">
            <Plus className="w-4 h-4" /> 新建智能体
          </button>
        </div>

        {/* Agent List */}
        {loading ? (
          <div className="flex justify-center py-12"><Loader2 className="w-6 h-6 animate-spin text-primary-600" /></div>
        ) : agents.length === 0 ? (
          <div className="bg-white rounded-xl border border-gray-200 p-8 text-center text-gray-400">
            暂无智能体
          </div>
        ) : (
          <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
            {agents.map(agent => (
              <div key={agent.id} className="bg-white rounded-xl border border-gray-200 shadow-sm p-5">
                <div className="flex items-start gap-3">
                  <div className={`w-10 h-10 rounded-lg flex items-center justify-center flex-shrink-0 ${
                    agent.is_published ? 'bg-primary-50' : 'bg-gray-50'
                  }`}>
                    <Bot className={`w-5 h-5 ${agent.is_published ? 'text-primary-600' : 'text-gray-400'}`} />
                  </div>
                  <div className="flex-1 min-w-0">
                    <div className="flex items-center gap-2 mb-1 flex-wrap">
                      <h3 className="text-sm font-semibold text-gray-800">{agent.name}</h3>
                      {agent.is_active ? (
                        <span className="px-1.5 py-0.5 rounded text-[10px] bg-green-100 text-green-700">启用</span>
                      ) : (
                        <span className="px-1.5 py-0.5 rounded text-[10px] bg-gray-100 text-gray-500">禁用</span>
                      )}
                      {agent.is_published && (
                        <span className="px-1.5 py-0.5 rounded text-[10px] bg-blue-100 text-blue-700">已发布</span>
                      )}
                      {agent.iron_rules && (
                        <span className="px-1.5 py-0.5 rounded text-[10px] bg-orange-100 text-orange-700">铁律</span>
                      )}
                    </div>
                    <p className="text-xs text-gray-500 line-clamp-2 mb-2">{agent.description}</p>
                    {/* Skills */}
                    {agent.agent_skills && agent.agent_skills.length > 0 && (
                      <div className="flex flex-wrap gap-1 mb-2">
                        {agent.agent_skills.map(as => (
                          <span key={as.id} className="inline-flex items-center gap-1 px-2 py-0.5 rounded-full text-[10px] bg-purple-50 text-purple-600">
                            <Zap className="w-2.5 h-2.5" />{as.skill?.name || `Skill #${as.skill_id}`}
                          </span>
                        ))}
                      </div>
                    )}
                    <div className="text-xs text-gray-400">
                      <span>模型: {agent.model || '默认'}</span>
                      <span className="mx-2">|</span>
                      <span>温度: {agent.temperature}</span>
                      <span className="mx-2">|</span>
                      <span>MaxTokens: {agent.max_tokens}</span>
                    </div>
                  </div>
                </div>
                <div className="flex items-center gap-1.5 mt-4 pt-3 border-t border-gray-100">
                  <button onClick={() => togglePublish(agent)}
                    className={`flex items-center gap-1 px-2.5 py-1.5 rounded-lg text-xs ${
                      agent.is_published ? 'bg-blue-50 text-blue-600 hover:bg-blue-100' : 'bg-gray-50 text-gray-500 hover:bg-gray-100'
                    }`}>
                    {agent.is_published ? <Eye className="w-3 h-3" /> : <EyeOff className="w-3 h-3" />}
                    {agent.is_published ? '取消发布' : '发布'}
                  </button>
                  {agent.is_published && (
                    <button onClick={() => setShowApiModal(agent)}
                      className="flex items-center gap-1 px-2.5 py-1.5 rounded-lg text-xs bg-green-50 text-green-600 hover:bg-green-100">
                      <ExternalLink className="w-3 h-3" /> API
                    </button>
                  )}
                  <div className="flex-1" />
                  <button onClick={() => handleEdit(agent)}
                    className="p-1.5 rounded-lg text-gray-400 hover:bg-gray-100 hover:text-gray-600" title="编辑">
                    <Edit3 className="w-3.5 h-3.5" />
                  </button>
                  <button onClick={() => setDeleteTarget(agent)}
                    className="p-1.5 rounded-lg text-gray-400 hover:bg-red-50 hover:text-red-500" title="删除">
                    <Trash2 className="w-3.5 h-3.5" />
                  </button>
                </div>
              </div>
            ))}
          </div>
        )}
      </div>

      {/* Create/Edit Modal */}
      {showModal && (
        <div className="fixed inset-0 bg-black/50 flex items-center justify-center z-50">
          <div className="bg-white rounded-2xl shadow-2xl w-full max-w-2xl mx-4 max-h-[90vh] overflow-y-auto">
            <div className="flex items-center justify-between px-6 py-4 border-b border-gray-100 sticky top-0 bg-white z-10">
              <h3 className="text-base font-semibold text-gray-800">{editingAgent ? '编辑智能体' : '新建智能体'}</h3>
              <button onClick={() => setShowModal(false)} className="p-1 rounded-lg hover:bg-gray-100"><X className="w-4 h-4" /></button>
            </div>
            <div className="p-6 space-y-4">
              <div>
                <label className="block text-xs font-medium text-gray-600 mb-1">名称</label>
                <input value={form.name} onChange={(e) => setForm({ ...form, name: e.target.value })}
                  placeholder="例如：交付专家" className="w-full px-3 py-2 border border-gray-200 rounded-lg text-sm focus:ring-2 focus:ring-primary-500 outline-none" />
              </div>
              <div>
                <label className="block text-xs font-medium text-gray-600 mb-1">描述</label>
                <textarea value={form.description} onChange={(e) => setForm({ ...form, description: e.target.value })}
                  rows={2} placeholder="描述智能体的功能"
                  className="w-full px-3 py-2 border border-gray-200 rounded-lg text-sm focus:ring-2 focus:ring-primary-500 outline-none resize-none" />
              </div>
              <div>
                <label className="block text-xs font-medium text-gray-600 mb-1">系统提示词</label>
                <textarea value={form.system_prompt} onChange={(e) => setForm({ ...form, system_prompt: e.target.value })}
                  rows={4} placeholder="定义智能体的角色和行为规则"
                  className="w-full px-3 py-2 border border-gray-200 rounded-lg text-sm focus:ring-2 focus:ring-primary-500 outline-none resize-none font-mono" />
              </div>
              <div className="grid grid-cols-3 gap-4">
                <div>
                  <label className="block text-xs font-medium text-gray-600 mb-1">模型 (留空用默认)</label>
                  <input value={form.model} onChange={(e) => setForm({ ...form, model: e.target.value })}
                    placeholder="gpt-4o" className="w-full px-3 py-2 border border-gray-200 rounded-lg text-sm" />
                </div>
                <div>
                  <label className="block text-xs font-medium text-gray-600 mb-1">温度</label>
                  <input type="number" step="0.1" min="0" max="2" value={form.temperature}
                    onChange={(e) => setForm({ ...form, temperature: parseFloat(e.target.value) || 0 })}
                    className="w-full px-3 py-2 border border-gray-200 rounded-lg text-sm" />
                </div>
                <div>
                  <label className="block text-xs font-medium text-gray-600 mb-1">Max Tokens</label>
                  <input type="number" step="512" min="256" value={form.max_tokens}
                    onChange={(e) => setForm({ ...form, max_tokens: parseInt(e.target.value) || 4096 })}
                    className="w-full px-3 py-2 border border-gray-200 rounded-lg text-sm" />
                </div>
              </div>
              {/* Toggles */}
              <div className="flex items-center gap-6">
                <label className="flex items-center gap-2 text-sm cursor-pointer">
                  <input type="checkbox" checked={form.is_active} onChange={(e) => setForm({ ...form, is_active: e.target.checked })}
                    className="rounded border-gray-300 text-primary-600 focus:ring-primary-500" />
                  <span className="text-gray-700">启用</span>
                </label>
                <label className="flex items-center gap-2 text-sm cursor-pointer">
                  <input type="checkbox" checked={form.is_published} onChange={(e) => setForm({ ...form, is_published: e.target.checked })}
                    className="rounded border-gray-300 text-blue-600 focus:ring-blue-500" />
                  <span className="text-gray-700">发布（提供外部对话接口）</span>
                </label>
                <label className="flex items-center gap-2 text-sm cursor-pointer">
                  <input type="checkbox" checked={form.iron_rules} onChange={(e) => setForm({ ...form, iron_rules: e.target.checked })}
                    className="rounded border-gray-300 text-orange-600 focus:ring-orange-500" />
                  <span className="text-gray-700">铁律模式</span>
                </label>
              </div>
              {/* Skill Selection */}
              <div>
                <label className="block text-xs font-medium text-gray-600 mb-2">关联技能</label>
                <div className="grid grid-cols-2 gap-2">
                  {skills.map(sk => (
                    <button key={sk.id} onClick={() => toggleSkill(sk.id)}
                      className={`flex items-center gap-2 px-3 py-2 rounded-lg border text-sm text-left ${
                        form.skill_ids.includes(sk.id)
                          ? 'border-primary-300 bg-primary-50 text-primary-700'
                          : 'border-gray-200 bg-white text-gray-600 hover:bg-gray-50'
                      }`}>
                      {form.skill_ids.includes(sk.id) ? <Check className="w-4 h-4 text-primary-600" /> : <Zap className="w-4 h-4 text-gray-300" />}
                      <div className="flex-1 min-w-0">
                        <div className="font-medium truncate">{sk.name}</div>
                        <div className="text-[10px] text-gray-400 truncate">{sk.description}</div>
                      </div>
                    </button>
                  ))}
                </div>
              </div>
            </div>
            <div className="flex justify-end gap-2 px-6 py-4 border-t border-gray-100 sticky bottom-0 bg-white">
              <button onClick={() => setShowModal(false)} className="px-4 py-2 text-sm text-gray-600 hover:bg-gray-100 rounded-lg">取消</button>
              <button onClick={handleSave} className="px-4 py-2 bg-primary-600 text-white rounded-lg text-sm hover:bg-primary-700">保存</button>
            </div>
          </div>
        </div>
      )}

      {/* Delete Confirmation Modal */}
      {deleteTarget && (
        <DeleteAgentConfirm
          agent={deleteTarget}
          onCancel={() => setDeleteTarget(null)}
          onConfirm={handleDelete}
          deleting={deleting}
        />
      )}

      {/* API Modal */}
      {showApiModal && (
        <div className="fixed inset-0 bg-black/50 flex items-center justify-center z-50">
          <div className="bg-white rounded-2xl shadow-2xl w-full max-w-lg mx-4">
            <div className="flex items-center justify-between px-6 py-4 border-b border-gray-100">
              <h3 className="text-base font-semibold text-gray-800">外部对话接口 - {showApiModal.name}</h3>
              <button onClick={() => setShowApiModal(null)} className="p-1 rounded-lg hover:bg-gray-100"><X className="w-4 h-4" /></button>
            </div>
            <div className="p-6 space-y-4">
              <div>
                <label className="block text-xs font-medium text-gray-600 mb-1">API 地址</label>
                <div className="flex gap-2">
                  <code className="flex-1 px-3 py-2 bg-gray-50 border border-gray-200 rounded-lg text-xs break-all font-mono">
                    POST {window.location.origin}/api/published-agents/{showApiModal.id}/chat
                  </code>
                  <button onClick={() => copyApiUrl(showApiModal.id)}
                    className="px-3 py-2 bg-primary-600 text-white rounded-lg text-xs hover:bg-primary-700 flex-shrink-0">
                    <Copy className="w-3.5 h-3.5" />
                  </button>
                </div>
              </div>
              <div>
                <label className="block text-xs font-medium text-gray-600 mb-1">请求示例</label>
                <pre className="px-3 py-2 bg-gray-900 text-green-400 rounded-lg text-xs overflow-x-auto font-mono">
{`curl -X POST \\
  ${window.location.origin}/api/published-agents/${showApiModal.id}/chat \\
  -H "Content-Type: application/json" \\
  -d '{"message": "请帮我查询ECF V6.2.1的兼容性列表"}'`}
                </pre>
              </div>
              <div>
                <label className="block text-xs font-medium text-gray-600 mb-1">响应格式</label>
                <pre className="px-3 py-2 bg-gray-50 border border-gray-200 rounded-lg text-xs overflow-x-auto font-mono">
{`{
  "code": 0,
  "data": {
    "agent": "${showApiModal.name}",
    "message": "根据文档..."
  }
}`}
                </pre>
              </div>
            </div>
            <div className="flex justify-end px-6 py-4 border-t border-gray-100">
              <button onClick={() => setShowApiModal(null)} className="px-4 py-2 text-sm text-gray-600 hover:bg-gray-100 rounded-lg">关闭</button>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}
