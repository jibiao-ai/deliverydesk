import React, { useState, useEffect } from 'react';
import { Shield, Plus, CheckCircle, XCircle, Clock, RefreshCw, FileText, Eye, EyeOff, Search } from 'lucide-react';
import useStore from '../store/useStore';
import toast from 'react-hot-toast';
import {
  createTotpApplication,
  getMyTotpApplications,
  getPendingTotpReviews,
  getAllTotpApplications,
  auditTotpApplications,
  checkTotpIssue,
} from '../services/api';

const STATUS_MAP = {
  pending: { label: '待审核', color: 'bg-yellow-100 text-yellow-700', icon: Clock },
  approved: { label: '已通过', color: 'bg-green-100 text-green-700', icon: CheckCircle },
  rejected: { label: '已拒绝', color: 'bg-red-100 text-red-700', icon: XCircle },
};

function StatusBadge({ status }) {
  const cfg = STATUS_MAP[status] || STATUS_MAP.pending;
  const Icon = cfg.icon;
  return (
    <span className={`inline-flex items-center gap-1 px-2 py-0.5 rounded-full text-xs font-medium ${cfg.color}`}>
      <Icon className="w-3 h-3" />
      {cfg.label}
    </span>
  );
}

function PasswordCell({ pass, status }) {
  const [visible, setVisible] = useState(false);
  if (status !== 'approved' || !pass) return <span className="text-gray-400">-</span>;
  return (
    <div className="flex items-center gap-1">
      <code className="bg-gray-100 px-2 py-0.5 rounded text-sm font-mono">
        {visible ? pass : '••••••'}
      </code>
      <button onClick={() => setVisible(!visible)} className="text-gray-400 hover:text-gray-600">
        {visible ? <EyeOff className="w-3.5 h-3.5" /> : <Eye className="w-3.5 h-3.5" />}
      </button>
    </div>
  );
}

export default function TotpPage() {
  const user = useStore((s) => s.user);
  const isAdmin = user?.role === 'admin';

  const [tab, setTab] = useState('my'); // my, review, all
  const [loading, setLoading] = useState(false);
  const [data, setData] = useState([]);
  const [total, setTotal] = useState(0);
  const [page, setPage] = useState(1);
  const [showApplyModal, setShowApplyModal] = useState(false);
  const [showAuditModal, setShowAuditModal] = useState(false);
  const [selectedRows, setSelectedRows] = useState([]);
  const [statusFilter, setStatusFilter] = useState('all');

  const fetchData = async () => {
    setLoading(true);
    try {
      let res;
      if (tab === 'my') {
        res = await getMyTotpApplications({ page, page_size: 10 });
      } else if (tab === 'review') {
        res = await getPendingTotpReviews({ page, page_size: 10 });
      } else {
        res = await getAllTotpApplications({ page, page_size: 10, status: statusFilter });
      }
      if (res?.code === 0) {
        setData(res.data.items || []);
        setTotal(res.data.total || 0);
      }
    } catch (e) {
      toast.error('加载数据失败');
    }
    setLoading(false);
  };

  useEffect(() => { fetchData(); }, [tab, page, statusFilter]);

  const handleAudit = async (approved) => {
    if (selectedRows.length === 0) {
      toast.error('请选择要审核的申请');
      return;
    }
    try {
      const res = await auditTotpApplications({
        ids: selectedRows.map((r) => r.id),
        approved,
        remark: '',
      });
      if (res?.code === 0) {
        toast.success(`审核完成: ${res.data.count} 条`);
        setSelectedRows([]);
        setShowAuditModal(false);
        fetchData();
      } else {
        toast.error(res?.message || '审核失败');
      }
    } catch (e) {
      toast.error('审核失败');
    }
  };

  const tabs = [
    { id: 'my', label: '我的申请', icon: FileText },
    ...(isAdmin ? [
      { id: 'review', label: '待审核', icon: Clock },
      { id: 'all', label: '全部记录', icon: Shield },
    ] : []),
  ];

  return (
    <div className="h-full flex flex-col overflow-hidden">
      {/* Tabs + Actions */}
      <div className="flex items-center justify-between px-6 py-3 border-b border-gray-200 bg-white flex-shrink-0">
        <div className="flex items-center gap-1 bg-gray-100 rounded-lg p-0.5">
          {tabs.map((t) => {
            const Icon = t.icon;
            return (
              <button
                key={t.id}
                onClick={() => { setTab(t.id); setPage(1); setSelectedRows([]); }}
                className={`flex items-center gap-1.5 px-3 py-1.5 rounded-md text-sm font-medium transition-all ${
                  tab === t.id
                    ? 'bg-white text-primary-700 shadow-sm'
                    : 'text-gray-500 hover:text-gray-700'
                }`}
              >
                <Icon className="w-3.5 h-3.5" />
                {t.label}
              </button>
            );
          })}
        </div>
        <div className="flex items-center gap-2">
          {tab === 'all' && (
            <select
              value={statusFilter}
              onChange={(e) => { setStatusFilter(e.target.value); setPage(1); }}
              className="text-sm border border-gray-300 rounded-lg px-2 py-1.5 bg-white"
            >
              <option value="all">全部状态</option>
              <option value="pending">待审核</option>
              <option value="approved">已通过</option>
              <option value="rejected">已拒绝</option>
            </select>
          )}
          <button
            onClick={fetchData}
            className="p-2 text-gray-400 hover:text-gray-600 hover:bg-gray-100 rounded-lg transition-colors"
          >
            <RefreshCw className={`w-4 h-4 ${loading ? 'animate-spin' : ''}`} />
          </button>
          {tab === 'my' && (
            <button
              onClick={() => setShowApplyModal(true)}
              className="flex items-center gap-1.5 px-3 py-1.5 bg-primary-600 text-white rounded-lg text-sm font-medium hover:bg-primary-700 transition-colors"
            >
              <Plus className="w-4 h-4" />
              申请双因子
            </button>
          )}
          {tab === 'review' && selectedRows.length > 0 && (
            <div className="flex items-center gap-2">
              <span className="text-sm text-gray-500">已选 {selectedRows.length} 条</span>
              <button
                onClick={() => handleAudit(true)}
                className="flex items-center gap-1 px-3 py-1.5 bg-green-600 text-white rounded-lg text-sm font-medium hover:bg-green-700"
              >
                <CheckCircle className="w-3.5 h-3.5" />
                批准
              </button>
              <button
                onClick={() => handleAudit(false)}
                className="flex items-center gap-1 px-3 py-1.5 bg-red-600 text-white rounded-lg text-sm font-medium hover:bg-red-700"
              >
                <XCircle className="w-3.5 h-3.5" />
                拒绝
              </button>
            </div>
          )}
        </div>
      </div>

      {/* Table */}
      <div className="flex-1 overflow-auto px-6 py-4">
        <div className="bg-white rounded-xl border border-gray-200 overflow-hidden">
          <table className="w-full text-sm">
            <thead>
              <tr className="bg-gray-50 text-gray-500 text-xs uppercase tracking-wide">
                {tab === 'review' && (
                  <th className="px-4 py-3 text-left w-10">
                    <input
                      type="checkbox"
                      checked={selectedRows.length === data.length && data.length > 0}
                      onChange={(e) => setSelectedRows(e.target.checked ? [...data] : [])}
                      className="rounded border-gray-300"
                    />
                  </th>
                )}
                <th className="px-4 py-3 text-left">申请时间</th>
                <th className="px-4 py-3 text-left">申请人</th>
                <th className="px-4 py-3 text-left">工单</th>
                <th className="px-4 py-3 text-left">客户</th>
                <th className="px-4 py-3 text-left">项目</th>
                <th className="px-4 py-3 text-left">版本</th>
                <th className="px-4 py-3 text-left">类型</th>
                <th className="px-4 py-3 text-left">状态</th>
                <th className="px-4 py-3 text-left">密码</th>
                <th className="px-4 py-3 text-left">原因</th>
              </tr>
            </thead>
            <tbody className="divide-y divide-gray-100">
              {loading ? (
                <tr><td colSpan={tab === 'review' ? 11 : 10} className="text-center py-12 text-gray-400">加载中...</td></tr>
              ) : data.length === 0 ? (
                <tr><td colSpan={tab === 'review' ? 11 : 10} className="text-center py-12 text-gray-400">暂无数据</td></tr>
              ) : (
                data.map((row) => (
                  <tr key={row.id} className="hover:bg-gray-50 transition-colors">
                    {tab === 'review' && (
                      <td className="px-4 py-3">
                        <input
                          type="checkbox"
                          checked={selectedRows.some((r) => r.id === row.id)}
                          onChange={(e) => {
                            if (e.target.checked) {
                              setSelectedRows([...selectedRows, row]);
                            } else {
                              setSelectedRows(selectedRows.filter((r) => r.id !== row.id));
                            }
                          }}
                          className="rounded border-gray-300"
                        />
                      </td>
                    )}
                    <td className="px-4 py-3 text-gray-500 whitespace-nowrap">
                      {new Date(row.created_at).toLocaleString('zh-CN', { month: '2-digit', day: '2-digit', hour: '2-digit', minute: '2-digit' })}
                    </td>
                    <td className="px-4 py-3 font-medium text-gray-700">{row.username}</td>
                    <td className="px-4 py-3">
                      {row.issue ? (
                        <span className="text-primary-600 cursor-pointer hover:underline">{row.issue}</span>
                      ) : '-'}
                    </td>
                    <td className="px-4 py-3 text-gray-600">{row.customer}</td>
                    <td className="px-4 py-3 text-gray-600">{row.project}</td>
                    <td className="px-4 py-3">
                      <span className="px-1.5 py-0.5 bg-blue-50 text-blue-600 rounded text-xs">{row.version}</span>
                    </td>
                    <td className="px-4 py-3">
                      <span className={`px-1.5 py-0.5 rounded text-xs ${row.totp_type === 'roller' ? 'bg-purple-50 text-purple-600' : 'bg-orange-50 text-orange-600'}`}>
                        {row.totp_type === 'roller' ? 'Roller' : 'TOTP'}
                      </span>
                    </td>
                    <td className="px-4 py-3"><StatusBadge status={row.audit_status} /></td>
                    <td className="px-4 py-3"><PasswordCell pass={row.totp_pass} status={row.audit_status} /></td>
                    <td className="px-4 py-3 text-gray-500 max-w-[150px] truncate" title={row.reason}>{row.reason || '-'}</td>
                  </tr>
                ))
              )}
            </tbody>
          </table>
        </div>

        {/* Pagination */}
        {total > 10 && (
          <div className="flex items-center justify-between mt-4">
            <span className="text-sm text-gray-500">共 {total} 条记录</span>
            <div className="flex items-center gap-2">
              <button
                onClick={() => setPage(Math.max(1, page - 1))}
                disabled={page <= 1}
                className="px-3 py-1 text-sm border rounded-lg disabled:opacity-40 hover:bg-gray-50"
              >
                上一页
              </button>
              <span className="text-sm text-gray-600">第 {page} 页</span>
              <button
                onClick={() => setPage(page + 1)}
                disabled={page * 10 >= total}
                className="px-3 py-1 text-sm border rounded-lg disabled:opacity-40 hover:bg-gray-50"
              >
                下一页
              </button>
            </div>
          </div>
        )}
      </div>

      {/* Apply Modal */}
      {showApplyModal && <ApplyModal onClose={() => setShowApplyModal(false)} onSuccess={() => { setShowApplyModal(false); fetchData(); }} />}
    </div>
  );
}

function ApplyModal({ onClose, onSuccess }) {
  const [form, setForm] = useState({
    issue: '',
    customer: '',
    project: '',
    version: 'V5',
    totp_type: 'roller',
    reason: '',
  });
  const [loading, setLoading] = useState(false);
  const [checking, setChecking] = useState(false);

  const handleCheckIssue = async () => {
    if (!form.issue.trim()) {
      toast.error('请先输入工单号');
      return;
    }
    setChecking(true);
    try {
      const res = await checkTotpIssue(form.issue.trim());
      if (res?.code === 0 && res.data) {
        const updates = {};
        if (res.data.customer) updates.customer = res.data.customer;
        if (res.data.project) updates.project = res.data.project;
        if (res.data.version && (res.data.version === 'V5' || res.data.version === 'V6')) {
          updates.version = res.data.version;
        }
        if (Object.keys(updates).length > 0) {
          setForm({ ...form, ...updates });
          toast.success('工单信息已自动填充');
        } else {
          toast.success('工单存在，但未找到客户/项目信息，请手动填写');
        }
      } else {
        toast.error(res?.message || '工单检查失败');
      }
    } catch (e) {
      toast.error('检查失败，请确认Jira配置正确');
    }
    setChecking(false);
  };

  const handleSubmit = async (e) => {
    e.preventDefault();
    if (!form.customer || !form.project || !form.reason) {
      toast.error('请填写必填项');
      return;
    }
    setLoading(true);
    try {
      const res = await createTotpApplication(form);
      if (res?.code === 0) {
        toast.success('申请已提交');
        onSuccess();
      } else {
        toast.error(res?.message || '申请失败');
      }
    } catch (e) {
      toast.error('提交失败');
    }
    setLoading(false);
  };

  return (
    <div className="fixed inset-0 bg-black/50 flex items-center justify-center z-50" onClick={onClose}>
      <div className="bg-white rounded-2xl shadow-2xl w-full max-w-lg p-6 mx-4" onClick={(e) => e.stopPropagation()}>
        <div className="flex items-center gap-3 mb-6">
          <div className="w-10 h-10 rounded-xl bg-primary-100 flex items-center justify-center">
            <Shield className="w-5 h-5 text-primary-600" />
          </div>
          <div>
            <h3 className="text-lg font-semibold text-gray-800">申请双因子密码</h3>
            <p className="text-sm text-gray-400">填写申请信息，提交后等待管理员审核</p>
          </div>
        </div>

        <form onSubmit={handleSubmit} className="space-y-4">
          <div>
            <label className="block text-sm font-medium text-gray-700 mb-1">关联工单</label>
            <div className="flex gap-2">
              <input
                type="text"
                value={form.issue}
                onChange={(e) => setForm({ ...form, issue: e.target.value })}
                placeholder="例如: ECSDESK-1234"
                className="flex-1 px-3 py-2 border border-gray-300 rounded-lg text-sm focus:ring-2 focus:ring-primary-500 focus:border-primary-500"
              />
              <button
                type="button"
                onClick={handleCheckIssue}
                disabled={checking || !form.issue.trim()}
                className="flex items-center gap-1.5 px-3 py-2 bg-blue-50 text-blue-600 border border-blue-200 rounded-lg text-sm font-medium hover:bg-blue-100 disabled:opacity-50 disabled:cursor-not-allowed transition-colors whitespace-nowrap"
              >
                <Search className={`w-3.5 h-3.5 ${checking ? 'animate-spin' : ''}`} />
                {checking ? '检查中...' : '检查'}
              </button>
            </div>
            <p className="text-xs text-gray-400 mt-1">输入工单号后点击“检查”可自动填充客户名称和项目名称</p>
          </div>
          <div className="grid grid-cols-2 gap-4">
            <div>
              <label className="block text-sm font-medium text-gray-700 mb-1">客户名称 <span className="text-red-500">*</span></label>
              <input
                type="text"
                value={form.customer}
                onChange={(e) => setForm({ ...form, customer: e.target.value })}
                placeholder="输入客户名称"
                className="w-full px-3 py-2 border border-gray-300 rounded-lg text-sm focus:ring-2 focus:ring-primary-500 focus:border-primary-500"
                required
              />
            </div>
            <div>
              <label className="block text-sm font-medium text-gray-700 mb-1">项目名称 <span className="text-red-500">*</span></label>
              <input
                type="text"
                value={form.project}
                onChange={(e) => setForm({ ...form, project: e.target.value })}
                placeholder="输入项目名称"
                className="w-full px-3 py-2 border border-gray-300 rounded-lg text-sm focus:ring-2 focus:ring-primary-500 focus:border-primary-500"
                required
              />
            </div>
          </div>
          <div className="grid grid-cols-2 gap-4">
            <div>
              <label className="block text-sm font-medium text-gray-700 mb-1">平台版本</label>
              <div className="flex items-center gap-4 mt-1">
                {['V5', 'V6'].map((v) => (
                  <label key={v} className="flex items-center gap-1.5 cursor-pointer">
                    <input
                      type="radio"
                      name="version"
                      value={v}
                      checked={form.version === v}
                      onChange={(e) => setForm({ ...form, version: e.target.value })}
                      className="text-primary-600"
                    />
                    <span className="text-sm">{v}</span>
                  </label>
                ))}
              </div>
            </div>
            <div>
              <label className="block text-sm font-medium text-gray-700 mb-1">类型</label>
              <div className="flex items-center gap-4 mt-1">
                {[{ id: 'roller', label: 'Roller密码' }, { id: 'totp', label: '动态密码' }].map((t) => (
                  <label key={t.id} className="flex items-center gap-1.5 cursor-pointer">
                    <input
                      type="radio"
                      name="totp_type"
                      value={t.id}
                      checked={form.totp_type === t.id}
                      onChange={(e) => setForm({ ...form, totp_type: e.target.value })}
                      className="text-primary-600"
                    />
                    <span className="text-sm">{t.label}</span>
                  </label>
                ))}
              </div>
            </div>
          </div>
          <div>
            <label className="block text-sm font-medium text-gray-700 mb-1">申请原因 <span className="text-red-500">*</span></label>
            <textarea
              value={form.reason}
              onChange={(e) => setForm({ ...form, reason: e.target.value })}
              placeholder="请简要说明申请原因..."
              rows={3}
              className="w-full px-3 py-2 border border-gray-300 rounded-lg text-sm focus:ring-2 focus:ring-primary-500 focus:border-primary-500 resize-none"
              required
            />
          </div>
          <div className="flex justify-end gap-3 pt-2">
            <button
              type="button"
              onClick={onClose}
              className="px-4 py-2 text-sm text-gray-600 border border-gray-300 rounded-lg hover:bg-gray-50"
            >
              取消
            </button>
            <button
              type="submit"
              disabled={loading}
              className="px-4 py-2 text-sm text-white bg-primary-600 rounded-lg hover:bg-primary-700 disabled:opacity-50"
            >
              {loading ? '提交中...' : '提交申请'}
            </button>
          </div>
        </form>
      </div>
    </div>
  );
}
