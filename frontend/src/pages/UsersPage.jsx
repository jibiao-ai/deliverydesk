import React, { useEffect, useState, useCallback } from 'react';
import {
  Users, Plus, Search, Edit2, Trash2, Shield, ShieldCheck,
  User, X, Check, RefreshCw, AlertCircle, Server, Lock,
  Mail, UserCircle, KeyRound, Eye, EyeOff,
} from 'lucide-react';
import { getUsers, createUser, updateUser, deleteUser } from '../services/api';

// ── Role badge display ──────────────────────────────────────────────────────
const ROLE_CONFIG = {
  admin: {
    label: '管理员',
    icon: ShieldCheck,
    badgeCls: 'bg-primary-100 text-primary-700 border-primary-200',
    desc: '拥有所有权限',
  },
  user: {
    label: '普通用户',
    icon: User,
    badgeCls: 'bg-gray-100 text-gray-600 border-gray-200',
    desc: '仅交付工作台权限',
  },
};

// ── Auth type badge display ─────────────────────────────────────────────────
const AUTH_TYPE_CONFIG = {
  local: {
    label: '本地用户',
    icon: Lock,
    badgeCls: 'bg-blue-50 text-blue-600 border-blue-200',
  },
  ldap: {
    label: 'LDAP用户',
    icon: Server,
    badgeCls: 'bg-amber-50 text-amber-600 border-amber-200',
  },
};

// ── Format date ─────────────────────────────────────────────────────────────
function formatTime(dateStr) {
  if (!dateStr) return '--';
  const d = new Date(dateStr);
  const pad = (n) => String(n).padStart(2, '0');
  return `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())} ${pad(d.getHours())}:${pad(d.getMinutes())}`;
}

// ── Modal component ─────────────────────────────────────────────────────────
function UserModal({ user, onClose, onSave }) {
  const isEdit = !!user?.id;
  const [form, setForm] = useState({
    username: user?.username || '',
    password: '',
    email: user?.email || '',
    display_name: user?.display_name || '',
    role: user?.role || 'user',
    auth_type: user?.auth_type || 'local',
  });
  const [showPassword, setShowPassword] = useState(false);
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState('');

  const handleSubmit = async (e) => {
    e.preventDefault();
    if (!form.username.trim()) {
      setError('用户名不能为空');
      return;
    }
    if (!isEdit && !form.password && form.auth_type === 'local') {
      setError('本地用户必须设置密码');
      return;
    }
    setSaving(true);
    setError('');
    try {
      let res;
      if (isEdit) {
        const updateData = { ...form };
        // Don't send empty password (means no change)
        if (!updateData.password) delete updateData.password;
        // Don't allow changing auth_type during edit
        delete updateData.auth_type;
        res = await updateUser(user.id, updateData);
      } else {
        res = await createUser(form);
      }
      if (res.code === 0) {
        onSave();
      } else {
        setError(res.message || '操作失败');
      }
    } catch (err) {
      setError(err.message || '网络错误');
    } finally {
      setSaving(false);
    }
  };

  return (
    <div className="fixed inset-0 bg-black/40 backdrop-blur-sm z-50 flex items-center justify-center p-4">
      <div className="bg-white rounded-2xl shadow-2xl w-full max-w-lg overflow-hidden">
        {/* Header */}
        <div className="px-6 py-4 border-b border-gray-100 flex items-center justify-between">
          <h3 className="text-lg font-semibold text-gray-800">
            {isEdit ? '编辑用户' : '新建用户'}
          </h3>
          <button onClick={onClose} className="p-1.5 rounded-lg text-gray-400 hover:text-gray-600 hover:bg-gray-100 transition-colors">
            <X className="w-5 h-5" />
          </button>
        </div>

        {/* Form */}
        <form onSubmit={handleSubmit} className="p-6 space-y-4">
          {error && (
            <div className="flex items-center gap-2 px-4 py-3 bg-red-50 text-red-600 rounded-xl text-sm">
              <AlertCircle className="w-4 h-4 flex-shrink-0" />
              {error}
            </div>
          )}

          {/* Username */}
          <div>
            <label className="block text-sm font-medium text-gray-700 mb-1.5">用户名 *</label>
            <div className="relative">
              <UserCircle className="w-4 h-4 text-gray-400 absolute left-3 top-1/2 -translate-y-1/2" />
              <input
                type="text"
                value={form.username}
                onChange={(e) => setForm({ ...form, username: e.target.value })}
                disabled={isEdit}
                placeholder="请输入用户名"
                className="w-full pl-10 pr-4 py-2.5 border border-gray-200 rounded-xl text-sm focus:ring-2 focus:ring-primary-500/20 focus:border-primary-500 outline-none disabled:bg-gray-50 disabled:text-gray-500"
              />
            </div>
          </div>

          {/* Password */}
          <div>
            <label className="block text-sm font-medium text-gray-700 mb-1.5">
              {isEdit ? '密码（留空不修改）' : '密码 *'}
            </label>
            <div className="relative">
              <KeyRound className="w-4 h-4 text-gray-400 absolute left-3 top-1/2 -translate-y-1/2" />
              <input
                type={showPassword ? 'text' : 'password'}
                value={form.password}
                onChange={(e) => setForm({ ...form, password: e.target.value })}
                placeholder={isEdit ? '留空保持原密码' : '请输入密码'}
                className="w-full pl-10 pr-10 py-2.5 border border-gray-200 rounded-xl text-sm focus:ring-2 focus:ring-primary-500/20 focus:border-primary-500 outline-none"
              />
              <button
                type="button"
                onClick={() => setShowPassword(!showPassword)}
                className="absolute right-3 top-1/2 -translate-y-1/2 text-gray-400 hover:text-gray-600"
              >
                {showPassword ? <EyeOff className="w-4 h-4" /> : <Eye className="w-4 h-4" />}
              </button>
            </div>
          </div>

          {/* Email */}
          <div>
            <label className="block text-sm font-medium text-gray-700 mb-1.5">邮箱</label>
            <div className="relative">
              <Mail className="w-4 h-4 text-gray-400 absolute left-3 top-1/2 -translate-y-1/2" />
              <input
                type="email"
                value={form.email}
                onChange={(e) => setForm({ ...form, email: e.target.value })}
                placeholder="请输入邮箱"
                className="w-full pl-10 pr-4 py-2.5 border border-gray-200 rounded-xl text-sm focus:ring-2 focus:ring-primary-500/20 focus:border-primary-500 outline-none"
              />
            </div>
          </div>

          {/* Display Name */}
          <div>
            <label className="block text-sm font-medium text-gray-700 mb-1.5">显示名称</label>
            <input
              type="text"
              value={form.display_name}
              onChange={(e) => setForm({ ...form, display_name: e.target.value })}
              placeholder="请输入显示名称"
              className="w-full px-4 py-2.5 border border-gray-200 rounded-xl text-sm focus:ring-2 focus:ring-primary-500/20 focus:border-primary-500 outline-none"
            />
          </div>

          {/* Role */}
          <div>
            <label className="block text-sm font-medium text-gray-700 mb-1.5">角色 *</label>
            <div className="grid grid-cols-2 gap-3">
              {Object.entries(ROLE_CONFIG).map(([key, cfg]) => {
                const Icon = cfg.icon;
                const isSelected = form.role === key;
                return (
                  <button
                    key={key}
                    type="button"
                    onClick={() => setForm({ ...form, role: key })}
                    className={`flex items-center gap-2.5 px-4 py-3 rounded-xl border-2 transition-all text-left ${
                      isSelected
                        ? 'border-primary-500 bg-primary-50'
                        : 'border-gray-200 hover:border-gray-300'
                    }`}
                  >
                    <Icon className={`w-5 h-5 flex-shrink-0 ${isSelected ? 'text-primary-600' : 'text-gray-400'}`} />
                    <div>
                      <p className={`text-sm font-medium ${isSelected ? 'text-primary-700' : 'text-gray-700'}`}>{cfg.label}</p>
                      <p className="text-xs text-gray-400">{cfg.desc}</p>
                    </div>
                  </button>
                );
              })}
            </div>
          </div>

          {/* Auth Type (only for creation) */}
          {!isEdit && (
            <div>
              <label className="block text-sm font-medium text-gray-700 mb-1.5">认证方式</label>
              <div className="grid grid-cols-2 gap-3">
                {Object.entries(AUTH_TYPE_CONFIG).map(([key, cfg]) => {
                  const Icon = cfg.icon;
                  const isSelected = form.auth_type === key;
                  return (
                    <button
                      key={key}
                      type="button"
                      onClick={() => setForm({ ...form, auth_type: key })}
                      className={`flex items-center gap-2.5 px-4 py-3 rounded-xl border-2 transition-all text-left ${
                        isSelected
                          ? 'border-primary-500 bg-primary-50'
                          : 'border-gray-200 hover:border-gray-300'
                      }`}
                    >
                      <Icon className={`w-4 h-4 flex-shrink-0 ${isSelected ? 'text-primary-600' : 'text-gray-400'}`} />
                      <span className={`text-sm font-medium ${isSelected ? 'text-primary-700' : 'text-gray-700'}`}>{cfg.label}</span>
                    </button>
                  );
                })}
              </div>
            </div>
          )}

          {/* Actions */}
          <div className="flex items-center justify-end gap-3 pt-2">
            <button
              type="button"
              onClick={onClose}
              className="px-4 py-2.5 text-sm font-medium text-gray-600 bg-gray-100 rounded-xl hover:bg-gray-200 transition-colors"
            >
              取消
            </button>
            <button
              type="submit"
              disabled={saving}
              className="px-6 py-2.5 text-sm font-medium text-white bg-primary-600 rounded-xl hover:bg-primary-700 transition-colors disabled:opacity-60 flex items-center gap-2"
            >
              {saving ? <RefreshCw className="w-4 h-4 animate-spin" /> : <Check className="w-4 h-4" />}
              {isEdit ? '保存更改' : '创建用户'}
            </button>
          </div>
        </form>
      </div>
    </div>
  );
}

// ── Delete confirmation dialog ──────────────────────────────────────────────
function DeleteConfirm({ username, onCancel, onConfirm, deleting }) {
  return (
    <div className="fixed inset-0 bg-black/40 backdrop-blur-sm z-50 flex items-center justify-center p-4">
      <div className="bg-white rounded-2xl shadow-2xl w-full max-w-sm p-6 text-center">
        <div className="w-14 h-14 rounded-full bg-red-50 flex items-center justify-center mx-auto mb-4">
          <AlertCircle className="w-7 h-7 text-red-500" />
        </div>
        <h3 className="text-lg font-semibold text-gray-800 mb-2">确认删除</h3>
        <p className="text-sm text-gray-500 mb-6">
          确定要删除用户 <strong className="text-gray-700">{username}</strong> 吗？此操作不可撤销。
        </p>
        <div className="flex items-center gap-3 justify-center">
          <button
            onClick={onCancel}
            className="px-5 py-2.5 text-sm font-medium text-gray-600 bg-gray-100 rounded-xl hover:bg-gray-200 transition-colors"
          >
            取消
          </button>
          <button
            onClick={onConfirm}
            disabled={deleting}
            className="px-5 py-2.5 text-sm font-medium text-white bg-red-500 rounded-xl hover:bg-red-600 transition-colors disabled:opacity-60 flex items-center gap-2"
          >
            {deleting ? <RefreshCw className="w-4 h-4 animate-spin" /> : <Trash2 className="w-4 h-4" />}
            确认删除
          </button>
        </div>
      </div>
    </div>
  );
}

// ── Main page component ─────────────────────────────────────────────────────
export default function UsersPage() {
  const [users, setUsers] = useState([]);
  const [loading, setLoading] = useState(true);
  const [search, setSearch] = useState('');
  const [editUser, setEditUser] = useState(null);   // null = closed, {} = create, user = edit
  const [deleteTarget, setDeleteTarget] = useState(null);
  const [deleting, setDeleting] = useState(false);

  const loadUsers = useCallback(async () => {
    setLoading(true);
    try {
      const res = await getUsers();
      if (res.code === 0) {
        setUsers(res.data || []);
      }
    } catch (err) {
      console.error('Failed to load users:', err);
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    loadUsers();
  }, [loadUsers]);

  const handleDelete = async () => {
    if (!deleteTarget) return;
    setDeleting(true);
    try {
      const res = await deleteUser(deleteTarget.id);
      if (res.code === 0) {
        setDeleteTarget(null);
        loadUsers();
      }
    } catch (err) {
      console.error('Failed to delete user:', err);
    } finally {
      setDeleting(false);
    }
  };

  // Filter users by search
  const filteredUsers = users.filter((u) => {
    if (!search.trim()) return true;
    const q = search.toLowerCase();
    return (
      u.username?.toLowerCase().includes(q) ||
      u.email?.toLowerCase().includes(q) ||
      u.display_name?.toLowerCase().includes(q)
    );
  });

  const adminCount = users.filter((u) => u.role === 'admin').length;
  const userCount = users.filter((u) => u.role === 'user').length;
  const ldapCount = users.filter((u) => u.auth_type === 'ldap').length;

  return (
    <div className="h-full overflow-y-auto">
      <div className="p-4 sm:p-6 space-y-4 w-full max-w-[1600px] mx-auto">
        {/* ── Stats & Actions ── */}
        <div className="bg-white rounded-2xl border border-gray-200 shadow-sm px-6 py-4">
          <div className="flex flex-col sm:flex-row items-start sm:items-center gap-3">
            {/* Stats */}
            <div className="flex items-center gap-4 text-sm flex-shrink-0">
              <div className="flex items-center gap-1.5">
                <div className="w-6 h-6 rounded-lg bg-primary-50 flex items-center justify-center">
                  <Users className="w-3.5 h-3.5 text-primary-600" />
                </div>
                <span className="text-gray-500">总计 <strong className="text-gray-800">{users.length}</strong></span>
              </div>
              <div className="w-px h-4 bg-gray-200" />
              <span className="text-gray-500">管理员 <strong className="text-primary-600">{adminCount}</strong></span>
              <div className="w-px h-4 bg-gray-200" />
              <span className="text-gray-500">普通用户 <strong className="text-gray-700">{userCount}</strong></span>
              {ldapCount > 0 && (
                <>
                  <div className="w-px h-4 bg-gray-200" />
                  <span className="text-gray-500">LDAP <strong className="text-amber-600">{ldapCount}</strong></span>
                </>
              )}
            </div>

            {/* Search & Add */}
            <div className="flex-1 flex items-center gap-3 w-full sm:w-auto">
              <div className="flex-1 relative">
                <Search className="w-4 h-4 text-gray-400 absolute left-3 top-1/2 -translate-y-1/2 pointer-events-none" />
                <input
                  type="text"
                  value={search}
                  onChange={(e) => setSearch(e.target.value)}
                  placeholder="搜索用户名、邮箱..."
                  className="w-full pl-10 pr-4 py-2 border border-gray-200 rounded-lg text-sm focus:ring-2 focus:ring-primary-500/20 focus:border-primary-500 outline-none bg-gray-50 focus:bg-white"
                />
              </div>
              <button
                onClick={() => setEditUser({})}
                className="px-4 py-2 text-sm font-medium text-white bg-primary-600 rounded-lg hover:bg-primary-700 transition-colors flex items-center gap-1.5 flex-shrink-0"
              >
                <Plus className="w-4 h-4" /> 新建用户
              </button>
            </div>
          </div>
        </div>

        {/* ── User table ── */}
        <div className="bg-white rounded-2xl border border-gray-200 shadow-sm overflow-hidden">
          <div className="overflow-x-auto">
            <table className="w-full">
              <thead>
                <tr className="bg-gray-50 border-b border-gray-200">
                  <th className="px-6 py-3.5 text-left text-xs font-semibold text-gray-500 uppercase tracking-wider">用户</th>
                  <th className="px-6 py-3.5 text-left text-xs font-semibold text-gray-500 uppercase tracking-wider">邮箱</th>
                  <th className="px-6 py-3.5 text-left text-xs font-semibold text-gray-500 uppercase tracking-wider">角色</th>
                  <th className="px-6 py-3.5 text-left text-xs font-semibold text-gray-500 uppercase tracking-wider">认证方式</th>
                  <th className="px-6 py-3.5 text-left text-xs font-semibold text-gray-500 uppercase tracking-wider">权限范围</th>
                  <th className="px-6 py-3.5 text-left text-xs font-semibold text-gray-500 uppercase tracking-wider">创建时间</th>
                  <th className="px-6 py-3.5 text-center text-xs font-semibold text-gray-500 uppercase tracking-wider">操作</th>
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
                ) : filteredUsers.length === 0 ? (
                  <tr>
                    <td colSpan={7} className="px-6 py-16 text-center">
                      <Users className="w-10 h-10 text-gray-200 mx-auto mb-3" />
                      <p className="text-sm text-gray-400">{search ? '未找到匹配的用户' : '暂无用户'}</p>
                    </td>
                  </tr>
                ) : (
                  filteredUsers.map((u) => {
                    const roleCfg = ROLE_CONFIG[u.role] || ROLE_CONFIG.user;
                    const authCfg = AUTH_TYPE_CONFIG[u.auth_type] || AUTH_TYPE_CONFIG.local;
                    const RoleIcon = roleCfg.icon;
                    const AuthIcon = authCfg.icon;
                    return (
                      <tr key={u.id} className="hover:bg-gray-50/50 transition-colors">
                        {/* User */}
                        <td className="px-6 py-3.5">
                          <div className="flex items-center gap-3">
                            <div className={`w-9 h-9 rounded-full flex items-center justify-center text-sm font-bold text-white flex-shrink-0 ${
                              u.role === 'admin' ? 'bg-primary-600' : 'bg-gray-400'
                            }`}>
                              {(u.username || '?').slice(0, 1).toUpperCase()}
                            </div>
                            <div>
                              <p className="text-sm font-semibold text-gray-800">{u.username}</p>
                              {u.display_name && u.display_name !== u.username && (
                                <p className="text-xs text-gray-400">{u.display_name}</p>
                              )}
                            </div>
                          </div>
                        </td>
                        {/* Email */}
                        <td className="px-6 py-3.5">
                          <span className="text-sm text-gray-600">{u.email || '--'}</span>
                        </td>
                        {/* Role */}
                        <td className="px-6 py-3.5">
                          <span className={`inline-flex items-center gap-1.5 text-xs font-medium px-2.5 py-1 rounded-full border ${roleCfg.badgeCls}`}>
                            <RoleIcon className="w-3 h-3" />
                            {roleCfg.label}
                          </span>
                        </td>
                        {/* Auth Type */}
                        <td className="px-6 py-3.5">
                          <span className={`inline-flex items-center gap-1.5 text-xs font-medium px-2.5 py-1 rounded-full border ${authCfg.badgeCls}`}>
                            <AuthIcon className="w-3 h-3" />
                            {authCfg.label}
                          </span>
                        </td>
                        {/* Permissions */}
                        <td className="px-6 py-3.5">
                          {u.role === 'admin' ? (
                            <div className="text-xs text-gray-500">
                              <span className="text-primary-600 font-medium">全部权限</span>
                              <span className="text-gray-400"> — 交付工作台 + 配置管理 + 系统管理</span>
                            </div>
                          ) : (
                            <div className="text-xs text-gray-500">
                              <span className="text-gray-700 font-medium">交付工作台</span>
                              <span className="text-gray-400"> — 仪表盘、公司系统、即时对话、智能体</span>
                            </div>
                          )}
                        </td>
                        {/* Created */}
                        <td className="px-6 py-3.5 whitespace-nowrap">
                          <span className="text-sm text-gray-500">{formatTime(u.created_at)}</span>
                        </td>
                        {/* Actions */}
                        <td className="px-6 py-3.5">
                          <div className="flex items-center justify-center gap-1">
                            <button
                              onClick={() => setEditUser(u)}
                              className="p-2 rounded-lg text-gray-400 hover:text-primary-600 hover:bg-primary-50 transition-colors"
                              title="编辑"
                            >
                              <Edit2 className="w-4 h-4" />
                            </button>
                            <button
                              onClick={() => setDeleteTarget(u)}
                              disabled={u.username === 'admin'}
                              className="p-2 rounded-lg text-gray-400 hover:text-red-500 hover:bg-red-50 transition-colors disabled:opacity-30 disabled:cursor-not-allowed"
                              title={u.username === 'admin' ? '不能删除默认管理员' : '删除'}
                            >
                              <Trash2 className="w-4 h-4" />
                            </button>
                          </div>
                        </td>
                      </tr>
                    );
                  })
                )}
              </tbody>
            </table>
          </div>

          {/* ── Permissions legend ── */}
          <div className="px-6 py-4 border-t border-gray-100 bg-gray-50/50">
            <div className="flex flex-wrap items-center gap-4 text-xs text-gray-500">
              <span className="font-medium text-gray-600">权限说明：</span>
              <div className="flex items-center gap-1.5">
                <ShieldCheck className="w-3.5 h-3.5 text-primary-600" />
                <span><strong className="text-gray-700">管理员</strong> — 拥有交付工作台、配置管理、系统管理全部权限</span>
              </div>
              <div className="w-px h-3 bg-gray-300" />
              <div className="flex items-center gap-1.5">
                <User className="w-3.5 h-3.5 text-gray-500" />
                <span><strong className="text-gray-700">普通用户</strong> — 仅拥有交付工作台（仪表盘、公司系统、即时对话、智能体）</span>
              </div>
            </div>
          </div>
        </div>

        {/* ── Modals ── */}
        {editUser !== null && (
          <UserModal
            user={editUser.id ? editUser : null}
            onClose={() => setEditUser(null)}
            onSave={() => { setEditUser(null); loadUsers(); }}
          />
        )}

        {deleteTarget && (
          <DeleteConfirm
            username={deleteTarget.username}
            onCancel={() => setDeleteTarget(null)}
            onConfirm={handleDelete}
            deleting={deleting}
          />
        )}
      </div>
    </div>
  );
}
