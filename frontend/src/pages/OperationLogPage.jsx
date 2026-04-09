import React, { useEffect, useState, useCallback } from 'react';
import {
  FileText, Search, Filter, ChevronLeft, ChevronRight,
  Clock, User, Monitor, Shield, Bot, MessageSquare, Cpu,
  LogIn, LogOut, RefreshCw, AlertCircle, Server,
} from 'lucide-react';
import { getOperationLogs } from '../services/api';

// ── Module display config ───────────────────────────────────────────────────
const MODULE_CONFIG = {
  auth:         { label: '用户认证', icon: LogIn,          color: 'bg-blue-50 text-blue-600',    badge: 'bg-blue-100 text-blue-700' },
  user:         { label: '用户管理', icon: User,           color: 'bg-indigo-50 text-indigo-600', badge: 'bg-indigo-100 text-indigo-700' },
  agent:        { label: '智能体',   icon: Bot,            color: 'bg-emerald-50 text-emerald-600', badge: 'bg-emerald-100 text-emerald-700' },
  conversation: { label: '对话交互', icon: MessageSquare,  color: 'bg-amber-50 text-amber-600',   badge: 'bg-amber-100 text-amber-700' },
  ai_provider:  { label: 'AI模型',  icon: Cpu,            color: 'bg-purple-50 text-purple-600',  badge: 'bg-purple-100 text-purple-700' },
  ldap:         { label: 'LDAP',    icon: Server,          color: 'bg-rose-50 text-rose-600',      badge: 'bg-rose-100 text-rose-700' },
};

// ── Action display config ───────────────────────────────────────────────────
const ACTION_CONFIG = {
  login:          { label: '登录成功', color: 'text-green-600' },
  login_failed:   { label: '登录失败', color: 'text-red-500' },
  create:         { label: '创建',     color: 'text-blue-600' },
  update:         { label: '更新',     color: 'text-amber-600' },
  delete:         { label: '删除',     color: 'text-red-600' },
  send_message:   { label: '发送消息', color: 'text-emerald-600' },
  test:           { label: '测试',     color: 'text-purple-600' },
};

// ── Module filter options ───────────────────────────────────────────────────
const MODULE_OPTIONS = [
  { value: '',             label: '全部模块' },
  { value: 'auth',         label: '用户认证' },
  { value: 'user',         label: '用户管理' },
  { value: 'agent',        label: '智能体' },
  { value: 'conversation', label: '对话交互' },
  { value: 'ai_provider',  label: 'AI模型' },
  { value: 'ldap',         label: 'LDAP' },
];

// ── Format date ─────────────────────────────────────────────────────────────
function formatTime(dateStr) {
  if (!dateStr) return '--';
  const d = new Date(dateStr);
  const pad = (n) => String(n).padStart(2, '0');
  return `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())} ${pad(d.getHours())}:${pad(d.getMinutes())}:${pad(d.getSeconds())}`;
}

export default function OperationLogPage() {
  const [logs, setLogs] = useState([]);
  const [loading, setLoading] = useState(true);
  const [total, setTotal] = useState(0);
  const [page, setPage] = useState(1);
  const [pageSize] = useState(20);
  const [moduleFilter, setModuleFilter] = useState('');
  const [usernameSearch, setUsernameSearch] = useState('');

  const loadLogs = useCallback(async () => {
    setLoading(true);
    try {
      const params = { page, page_size: pageSize };
      if (moduleFilter) params.module = moduleFilter;
      if (usernameSearch.trim()) params.username = usernameSearch.trim();
      const res = await getOperationLogs(params);
      if (res.code === 0) {
        setLogs(res.data?.items || []);
        setTotal(res.data?.total || 0);
      }
    } catch (err) {
      console.error('Failed to load logs:', err);
    } finally {
      setLoading(false);
    }
  }, [page, pageSize, moduleFilter, usernameSearch]);

  useEffect(() => {
    loadLogs();
  }, [loadLogs]);

  // Reset to page 1 when filters change
  const handleModuleChange = (val) => {
    setModuleFilter(val);
    setPage(1);
  };

  const handleUsernameSearch = (val) => {
    setUsernameSearch(val);
    setPage(1);
  };

  const totalPages = Math.ceil(total / pageSize);

  return (
    <div className="h-full overflow-y-auto">
      <div className="p-4 sm:p-6 space-y-4 w-full max-w-[1600px] mx-auto">
        {/* ── Filters ── */}
        <div className="bg-white rounded-2xl border border-gray-200 shadow-sm px-6 py-4">
          <div className="flex flex-col sm:flex-row items-start sm:items-center gap-3">
            {/* Module filter */}
            <div className="flex items-center gap-2">
              <Filter className="w-4 h-4 text-gray-400" />
              <select
                value={moduleFilter}
                onChange={(e) => handleModuleChange(e.target.value)}
                className="border border-gray-200 rounded-lg px-3 py-2 text-sm focus:ring-2 focus:ring-primary-500/20 focus:border-primary-500 outline-none bg-gray-50"
              >
                {MODULE_OPTIONS.map((opt) => (
                  <option key={opt.value} value={opt.value}>{opt.label}</option>
                ))}
              </select>
            </div>

            {/* Username search */}
            <div className="flex-1 relative w-full sm:w-auto">
              <Search className="w-4 h-4 text-gray-400 absolute left-3 top-1/2 -translate-y-1/2 pointer-events-none" />
              <input
                type="text"
                value={usernameSearch}
                onChange={(e) => handleUsernameSearch(e.target.value)}
                placeholder="搜索操作用户..."
                className="w-full pl-10 pr-4 py-2 border border-gray-200 rounded-lg text-sm focus:ring-2 focus:ring-primary-500/20 focus:border-primary-500 outline-none bg-gray-50 focus:bg-white"
              />
            </div>

            {/* Stats & Refresh */}
            <div className="flex items-center gap-3 flex-shrink-0">
              <span className="text-sm text-gray-500">
                共 <strong className="text-gray-800">{total}</strong> 条记录
              </span>
              <button
                onClick={loadLogs}
                className="p-2 rounded-lg text-gray-400 hover:text-primary-600 hover:bg-primary-50 transition-colors"
                title="刷新"
              >
                <RefreshCw className={`w-4 h-4 ${loading ? 'animate-spin' : ''}`} />
              </button>
            </div>
          </div>
        </div>

        {/* ── Log table ── */}
        <div className="bg-white rounded-2xl border border-gray-200 shadow-sm overflow-hidden">
          <div className="overflow-x-auto">
            <table className="w-full">
              <thead>
                <tr className="bg-gray-50 border-b border-gray-200">
                  <th className="px-6 py-3.5 text-left text-xs font-semibold text-gray-500 uppercase tracking-wider">时间</th>
                  <th className="px-6 py-3.5 text-left text-xs font-semibold text-gray-500 uppercase tracking-wider">用户</th>
                  <th className="px-6 py-3.5 text-left text-xs font-semibold text-gray-500 uppercase tracking-wider">模块</th>
                  <th className="px-6 py-3.5 text-left text-xs font-semibold text-gray-500 uppercase tracking-wider">操作</th>
                  <th className="px-6 py-3.5 text-left text-xs font-semibold text-gray-500 uppercase tracking-wider">目标</th>
                  <th className="px-6 py-3.5 text-left text-xs font-semibold text-gray-500 uppercase tracking-wider">详情</th>
                  <th className="px-6 py-3.5 text-left text-xs font-semibold text-gray-500 uppercase tracking-wider">IP</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-gray-100">
                {loading ? (
                  <tr>
                    <td colSpan={7} className="px-6 py-16 text-center">
                      <RefreshCw className="w-6 h-6 animate-spin text-primary-600 mx-auto mb-2" />
                      <p className="text-sm text-gray-400">加载中...</p>
                    </td>
                  </tr>
                ) : logs.length === 0 ? (
                  <tr>
                    <td colSpan={7} className="px-6 py-16 text-center">
                      <FileText className="w-10 h-10 text-gray-200 mx-auto mb-3" />
                      <p className="text-sm text-gray-400">暂无操作日志</p>
                    </td>
                  </tr>
                ) : (
                  logs.map((log) => {
                    const modCfg = MODULE_CONFIG[log.module] || { label: log.module, icon: Monitor, color: 'bg-gray-50 text-gray-600', badge: 'bg-gray-100 text-gray-700' };
                    const actCfg = ACTION_CONFIG[log.action] || { label: log.action, color: 'text-gray-600' };
                    const ModIcon = modCfg.icon;
                    return (
                      <tr key={log.id} className="hover:bg-gray-50/50 transition-colors">
                        {/* Time */}
                        <td className="px-6 py-3.5 whitespace-nowrap">
                          <div className="flex items-center gap-2 text-sm text-gray-600">
                            <Clock className="w-3.5 h-3.5 text-gray-400 flex-shrink-0" />
                            {formatTime(log.created_at)}
                          </div>
                        </td>
                        {/* User */}
                        <td className="px-6 py-3.5 whitespace-nowrap">
                          <div className="flex items-center gap-2">
                            <div className="w-7 h-7 rounded-full bg-primary-100 flex items-center justify-center text-xs font-bold text-primary-600 flex-shrink-0">
                              {(log.username || '?').slice(0, 1).toUpperCase()}
                            </div>
                            <span className="text-sm font-medium text-gray-700">{log.username || '--'}</span>
                          </div>
                        </td>
                        {/* Module */}
                        <td className="px-6 py-3.5 whitespace-nowrap">
                          <span className={`inline-flex items-center gap-1.5 text-xs font-medium px-2.5 py-1 rounded-full ${modCfg.badge}`}>
                            <ModIcon className="w-3 h-3" />
                            {modCfg.label}
                          </span>
                        </td>
                        {/* Action */}
                        <td className="px-6 py-3.5 whitespace-nowrap">
                          <span className={`text-sm font-medium ${actCfg.color}`}>{actCfg.label}</span>
                        </td>
                        {/* Target */}
                        <td className="px-6 py-3.5">
                          <span className="text-sm text-gray-600 truncate block max-w-[200px]" title={log.target_name}>
                            {log.target_name || '--'}
                          </span>
                        </td>
                        {/* Detail */}
                        <td className="px-6 py-3.5">
                          <span className="text-sm text-gray-500 truncate block max-w-[300px]" title={log.detail}>
                            {log.detail || '--'}
                          </span>
                        </td>
                        {/* IP */}
                        <td className="px-6 py-3.5 whitespace-nowrap">
                          <span className="text-sm text-gray-400 font-mono">{log.ip || '--'}</span>
                        </td>
                      </tr>
                    );
                  })
                )}
              </tbody>
            </table>
          </div>

          {/* ── Pagination ── */}
          {totalPages > 1 && (
            <div className="px-6 py-4 border-t border-gray-100 flex items-center justify-between">
              <p className="text-sm text-gray-500">
                第 {page} / {totalPages} 页，共 {total} 条
              </p>
              <div className="flex items-center gap-2">
                <button
                  onClick={() => setPage(Math.max(1, page - 1))}
                  disabled={page <= 1}
                  className="px-3 py-1.5 text-sm rounded-lg border border-gray-200 hover:bg-gray-50 disabled:opacity-50 disabled:cursor-not-allowed transition-colors flex items-center gap-1"
                >
                  <ChevronLeft className="w-4 h-4" /> 上一页
                </button>
                {/* Page numbers */}
                <div className="flex items-center gap-1">
                  {Array.from({ length: Math.min(5, totalPages) }, (_, i) => {
                    let p;
                    if (totalPages <= 5) {
                      p = i + 1;
                    } else if (page <= 3) {
                      p = i + 1;
                    } else if (page >= totalPages - 2) {
                      p = totalPages - 4 + i;
                    } else {
                      p = page - 2 + i;
                    }
                    return (
                      <button
                        key={p}
                        onClick={() => setPage(p)}
                        className={`w-8 h-8 text-sm rounded-lg transition-colors ${
                          p === page
                            ? 'bg-primary-600 text-white font-medium'
                            : 'text-gray-600 hover:bg-gray-100'
                        }`}
                      >
                        {p}
                      </button>
                    );
                  })}
                </div>
                <button
                  onClick={() => setPage(Math.min(totalPages, page + 1))}
                  disabled={page >= totalPages}
                  className="px-3 py-1.5 text-sm rounded-lg border border-gray-200 hover:bg-gray-50 disabled:opacity-50 disabled:cursor-not-allowed transition-colors flex items-center gap-1"
                >
                  下一页 <ChevronRight className="w-4 h-4" />
                </button>
              </div>
            </div>
          )}
        </div>
      </div>
    </div>
  );
}
