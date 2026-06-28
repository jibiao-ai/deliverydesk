import React, { useState, useEffect, useCallback } from 'react';
import {
  Clock, Users, TrendingUp, Calendar, Plus, Trash2, UserPlus, Search,
  ChevronDown, ChevronRight, BarChart3, Briefcase, RefreshCw, Download,
  Activity, Timer, Target, AlertCircle
} from 'lucide-react';
import { getWorktimeStats, getWorktimeUsers, addWorktimeUser, removeWorktimeUser } from '../services/api';
import toast from 'react-hot-toast';

/* ═══════════════════════════════════════════════════════════════════
   Elastic animation keyframes injected via style tag
═══════════════════════════════════════════════════════════════════ */
const styleTag = document.createElement('style');
styleTag.textContent = `
  @keyframes elasticIn {
    0% { opacity: 0; transform: scale(0.3) translateY(20px); }
    50% { opacity: 1; transform: scale(1.05) translateY(-5px); }
    70% { transform: scale(0.95) translateY(2px); }
    100% { opacity: 1; transform: scale(1) translateY(0); }
  }
  @keyframes elasticPulse {
    0%, 100% { transform: scale(1); }
    50% { transform: scale(1.02); }
  }
  @keyframes fadeSlideUp {
    0% { opacity: 0; transform: translateY(15px); }
    100% { opacity: 1; transform: translateY(0); }
  }
  @keyframes shimmer {
    0% { background-position: -200% 0; }
    100% { background-position: 200% 0; }
  }
  .elastic-in { animation: elasticIn 0.6s cubic-bezier(0.68, -0.55, 0.27, 1.55) forwards; }
  .elastic-pulse:hover { animation: elasticPulse 0.3s ease; }
  .fade-slide-up { animation: fadeSlideUp 0.4s ease forwards; }
  .shimmer-loading {
    background: linear-gradient(90deg, #f0f0f0 25%, #e0e0e0 50%, #f0f0f0 75%);
    background-size: 200% 100%;
    animation: shimmer 1.5s infinite;
  }
`;
if (!document.getElementById('worktime-animations')) {
  styleTag.id = 'worktime-animations';
  document.head.appendChild(styleTag);
}

/* ═══════════════════════════════════════════════════════════════════
   Frosted Glass Card Component
═══════════════════════════════════════════════════════════════════ */
function GlassCard({ children, className = '', delay = 0, hover = true }) {
  return (
    <div
      className={`
        rounded-2xl border border-white/40 shadow-lg
        bg-white/70 backdrop-blur-xl
        ${hover ? 'elastic-pulse hover:shadow-xl hover:border-primary-200/60 transition-all duration-300' : ''}
        elastic-in ${className}
      `}
      style={{ animationDelay: `${delay}ms` }}
    >
      {children}
    </div>
  );
}

/* ═══════════════════════════════════════════════════════════════════
   Period Selector Component
═══════════════════════════════════════════════════════════════════ */
function PeriodSelector({ period, onChange }) {
  const periods = [
    { value: 'month', label: '本月', icon: Calendar },
    { value: 'quarter', label: '本季度', icon: BarChart3 },
    { value: 'year', label: '本年度', icon: TrendingUp },
  ];

  return (
    <div className="flex gap-2 p-1 bg-white/50 backdrop-blur-sm rounded-xl border border-gray-200/50">
      {periods.map((p) => {
        const Icon = p.icon;
        const isActive = period === p.value;
        return (
          <button
            key={p.value}
            onClick={() => onChange(p.value)}
            className={`flex items-center gap-1.5 px-4 py-2 rounded-lg text-sm font-medium transition-all duration-300 ${
              isActive
                ? 'bg-primary-600 text-white shadow-md shadow-primary-200 scale-105'
                : 'text-gray-600 hover:bg-gray-100/80 hover:text-gray-800'
            }`}
          >
            <Icon className="w-4 h-4" />
            {p.label}
          </button>
        );
      })}
    </div>
  );
}

/* ═══════════════════════════════════════════════════════════════════
   Stats Card Component
═══════════════════════════════════════════════════════════════════ */
function StatCard({ icon: Icon, label, value, unit, color, delay }) {
  const colorClasses = {
    blue: 'from-blue-500 to-blue-600 shadow-blue-200',
    green: 'from-emerald-500 to-emerald-600 shadow-emerald-200',
    purple: 'from-purple-500 to-purple-600 shadow-purple-200',
    orange: 'from-orange-500 to-orange-600 shadow-orange-200',
  };

  return (
    <GlassCard delay={delay} className="p-5">
      <div className="flex items-center justify-between">
        <div>
          <p className="text-sm text-gray-500 font-medium">{label}</p>
          <div className="flex items-baseline gap-1 mt-1">
            <span className="text-2xl font-bold text-gray-800">
              {typeof value === 'number' ? value.toLocaleString('zh-CN', { maximumFractionDigits: 1 }) : value}
            </span>
            {unit && <span className="text-sm text-gray-400">{unit}</span>}
          </div>
        </div>
        <div className={`w-12 h-12 rounded-xl bg-gradient-to-br ${colorClasses[color]} shadow-lg flex items-center justify-center`}>
          <Icon className="w-6 h-6 text-white" />
        </div>
      </div>
    </GlassCard>
  );
}

/* ═══════════════════════════════════════════════════════════════════
   User Management Panel
═══════════════════════════════════════════════════════════════════ */
function UserManagePanel({ users, onAdd, onRemove, loading }) {
  const [newName, setNewName] = useState('');
  const [showPanel, setShowPanel] = useState(false);

  const handleAdd = () => {
    if (!newName.trim()) return;
    onAdd(newName.trim());
    setNewName('');
  };

  return (
    <GlassCard delay={200} className="p-5">
      <div className="flex items-center justify-between mb-4">
        <div className="flex items-center gap-2">
          <div className="w-8 h-8 rounded-lg bg-gradient-to-br from-indigo-500 to-indigo-600 flex items-center justify-center shadow-md shadow-indigo-200">
            <Users className="w-4 h-4 text-white" />
          </div>
          <h3 className="font-semibold text-gray-800">人员管理</h3>
          <span className="text-xs bg-gray-100 text-gray-500 px-2 py-0.5 rounded-full">{users.length} 人</span>
        </div>
        <button
          onClick={() => setShowPanel(!showPanel)}
          className="flex items-center gap-1 px-3 py-1.5 rounded-lg text-sm font-medium text-primary-600 hover:bg-primary-50 transition-all"
        >
          {showPanel ? <ChevronDown className="w-4 h-4" /> : <ChevronRight className="w-4 h-4" />}
          {showPanel ? '收起' : '管理'}
        </button>
      </div>

      {showPanel && (
        <div className="fade-slide-up">
          {/* Add user input */}
          <div className="flex gap-2 mb-3">
            <div className="flex-1 relative">
              <UserPlus className="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-gray-400" />
              <input
                type="text"
                value={newName}
                onChange={(e) => setNewName(e.target.value)}
                onKeyDown={(e) => e.key === 'Enter' && handleAdd()}
                placeholder="输入姓名添加人员..."
                className="w-full pl-9 pr-4 py-2 rounded-xl border border-gray-200/80 bg-white/60 backdrop-blur-sm text-sm focus:outline-none focus:ring-2 focus:ring-primary-300 focus:border-primary-300 transition-all"
              />
            </div>
            <button
              onClick={handleAdd}
              disabled={!newName.trim()}
              className="px-4 py-2 rounded-xl bg-primary-600 text-white text-sm font-medium hover:bg-primary-700 disabled:opacity-50 disabled:cursor-not-allowed transition-all shadow-md shadow-primary-200 hover:shadow-lg hover:shadow-primary-300"
            >
              <Plus className="w-4 h-4" />
            </button>
          </div>

          {/* User list */}
          <div className="max-h-48 overflow-y-auto space-y-1.5 pr-1" style={{ scrollbarWidth: 'thin' }}>
            {users.map((u, idx) => (
              <div
                key={u.id}
                className="flex items-center justify-between px-3 py-2 rounded-xl bg-gray-50/80 hover:bg-gray-100/80 group transition-all"
                style={{ animationDelay: `${idx * 30}ms` }}
              >
                <div className="flex items-center gap-2">
                  <div className="w-7 h-7 rounded-full bg-gradient-to-br from-primary-400 to-primary-600 flex items-center justify-center text-xs text-white font-medium shadow-sm">
                    {u.name.slice(0, 1)}
                  </div>
                  <span className="text-sm text-gray-700">{u.name}</span>
                </div>
                <button
                  onClick={() => onRemove(u.id)}
                  className="opacity-0 group-hover:opacity-100 p-1.5 rounded-lg text-gray-400 hover:text-red-500 hover:bg-red-50 transition-all"
                >
                  <Trash2 className="w-3.5 h-3.5" />
                </button>
              </div>
            ))}
            {users.length === 0 && (
              <div className="text-center py-6 text-sm text-gray-400">
                <AlertCircle className="w-8 h-8 mx-auto mb-2 text-gray-300" />
                暂无人员，请添加需要统计的人员
              </div>
            )}
          </div>
        </div>
      )}

      {/* Collapsed view: show user avatars */}
      {!showPanel && users.length > 0 && (
        <div className="flex items-center gap-1 flex-wrap">
          {users.slice(0, 15).map((u) => (
            <div
              key={u.id}
              title={u.name}
              className="w-7 h-7 rounded-full bg-gradient-to-br from-primary-400 to-primary-600 flex items-center justify-center text-xs text-white font-medium shadow-sm -ml-1 first:ml-0 border-2 border-white"
            >
              {u.name.slice(0, 1)}
            </div>
          ))}
          {users.length > 15 && (
            <span className="text-xs text-gray-400 ml-2">+{users.length - 15} 人</span>
          )}
        </div>
      )}
    </GlassCard>
  );
}

/* ═══════════════════════════════════════════════════════════════════
   User Worktime Detail Card (expandable)
═══════════════════════════════════════════════════════════════════ */
function UserDetailCard({ user, index }) {
  const [expanded, setExpanded] = useState(false);

  if (user.total_hours === 0 && user.project_details?.length === 0) {
    return (
      <div
        className="fade-slide-up flex items-center gap-3 px-4 py-3 rounded-xl bg-white/50 backdrop-blur-sm border border-gray-100/50"
        style={{ animationDelay: `${index * 60}ms` }}
      >
        <div className="w-9 h-9 rounded-full bg-gray-200 flex items-center justify-center text-sm text-gray-500 font-medium">
          {user.name.slice(0, 1)}
        </div>
        <span className="text-sm text-gray-600">{user.name}</span>
        <span className="ml-auto text-xs text-gray-400">暂无工时记录</span>
      </div>
    );
  }

  return (
    <div
      className="fade-slide-up rounded-xl border border-gray-100/60 bg-white/60 backdrop-blur-sm overflow-hidden transition-all hover:shadow-md"
      style={{ animationDelay: `${index * 60}ms` }}
    >
      {/* Header */}
      <button
        onClick={() => setExpanded(!expanded)}
        className="w-full flex items-center gap-3 px-5 py-4 text-left hover:bg-gray-50/50 transition-all"
      >
        <div className="w-10 h-10 rounded-full bg-gradient-to-br from-primary-400 to-primary-600 flex items-center justify-center text-sm text-white font-bold shadow-md shadow-primary-200">
          {user.name.slice(0, 1)}
        </div>
        <div className="flex-1 min-w-0">
          <p className="font-semibold text-gray-800">{user.name}</p>
          <p className="text-xs text-gray-400">{user.project_details?.length || 0} 个项目</p>
        </div>
        <div className="flex items-center gap-4 mr-2">
          <div className="text-right">
            <p className="text-xs text-gray-400">总工时</p>
            <p className="text-sm font-bold text-blue-600">{user.total_hours.toFixed(1)}h</p>
          </div>
          <div className="text-right">
            <p className="text-xs text-gray-400">人天</p>
            <p className="text-sm font-bold text-emerald-600">{user.total_man_days.toFixed(1)}</p>
          </div>
          <div className="text-right">
            <p className="text-xs text-gray-400">成本人天</p>
            <p className="text-sm font-bold text-purple-600">{user.total_cost_days.toFixed(1)}</p>
          </div>
        </div>
        <ChevronRight className={`w-4 h-4 text-gray-400 transition-transform duration-300 ${expanded ? 'rotate-90' : ''}`} />
      </button>

      {/* Expanded: project details */}
      {expanded && user.project_details && (
        <div className="px-5 pb-4 space-y-3 border-t border-gray-100/60">
          {user.project_details.map((proj, pidx) => (
            <div key={pidx} className="mt-3 rounded-xl bg-gray-50/80 p-4">
              <div className="flex items-center gap-2 mb-2">
                <Briefcase className="w-4 h-4 text-gray-400" />
                <span className="text-sm font-medium text-gray-700">{proj.project_name || proj.project_no}</span>
                <span className="text-xs text-gray-400 ml-auto">
                  {proj.total_hours.toFixed(1)}h / {proj.total_man_days.toFixed(0)}天
                </span>
              </div>
              {proj.contract_party && (
                <p className="text-xs text-gray-400 mb-2 ml-6">甲方: {proj.contract_party}</p>
              )}
              {/* Month details */}
              {proj.month_details?.map((month, midx) => (
                <div key={midx} className="ml-6 mt-2">
                  <p className="text-xs font-medium text-gray-500 flex items-center gap-1">
                    <Calendar className="w-3 h-3" /> {month.month}
                    <span className="text-gray-400">({month.total_hours.toFixed(1)}h)</span>
                  </p>
                  {month.tasks?.map((task, tidx) => (
                    <div key={tidx} className="flex items-center gap-2 mt-1 text-xs text-gray-500 ml-4">
                      <span className="w-1.5 h-1.5 rounded-full bg-primary-300" />
                      <span className="flex-1 truncate">{task.task_name}</span>
                      <span className="text-gray-400">{task.hours.toFixed(1)}h</span>
                    </div>
                  ))}
                </div>
              ))}
            </div>
          ))}
        </div>
      )}
    </div>
  );
}

/* ═══════════════════════════════════════════════════════════════════
   Main WorktimePage Component
═══════════════════════════════════════════════════════════════════ */
export default function WorktimePage() {
  const [period, setPeriod] = useState('month');
  const [stats, setStats] = useState(null);
  const [users, setUsers] = useState([]);
  const [loading, setLoading] = useState(false);
  const [searchTerm, setSearchTerm] = useState('');

  // Fetch tracked users
  const fetchUsers = useCallback(async () => {
    const res = await getWorktimeUsers();
    if (res?.code === 0) {
      setUsers(res.data || []);
    }
  }, []);

  // Fetch worktime stats
  const fetchStats = useCallback(async () => {
    setLoading(true);
    try {
      const res = await getWorktimeStats({ period });
      if (res?.code === 0) {
        setStats(res.data);
      } else {
        toast.error(res?.message || '获取工时数据失败');
      }
    } catch (err) {
      toast.error('获取工时数据失败');
    } finally {
      setLoading(false);
    }
  }, [period]);

  useEffect(() => { fetchUsers(); }, [fetchUsers]);
  useEffect(() => { fetchStats(); }, [fetchStats]);

  // Add user
  const handleAddUser = async (name) => {
    const res = await addWorktimeUser(name);
    if (res?.code === 0) {
      toast.success(`已添加 ${name}`);
      fetchUsers();
    } else {
      toast.error(res?.message || '添加失败');
    }
  };

  // Remove user
  const handleRemoveUser = async (id) => {
    const res = await removeWorktimeUser(id);
    if (res?.code === 0) {
      toast.success('已删除');
      fetchUsers();
    } else {
      toast.error('删除失败');
    }
  };

  // Filter users by search
  const filteredUsers = stats?.users?.filter((u) =>
    !searchTerm || u.name.includes(searchTerm)
  ) || [];

  return (
    <div className="h-full overflow-y-auto p-6" style={{ scrollbarWidth: 'thin' }}>
      <div className="max-w-7xl mx-auto space-y-6">
        {/* Header */}
        <div className="flex items-center justify-between flex-wrap gap-4">
          <div className="flex items-center gap-3">
            <div className="w-10 h-10 rounded-xl bg-gradient-to-br from-primary-500 to-indigo-600 flex items-center justify-center shadow-lg shadow-primary-200">
              <Clock className="w-5 h-5 text-white" />
            </div>
            <div>
              <h2 className="text-xl font-bold text-gray-800">工时管理</h2>
              <p className="text-sm text-gray-400">Redmine 工时数据统计与分析</p>
            </div>
          </div>

          <div className="flex items-center gap-3">
            <PeriodSelector period={period} onChange={setPeriod} />
            <button
              onClick={fetchStats}
              disabled={loading}
              className="flex items-center gap-1.5 px-4 py-2 rounded-xl bg-white/70 backdrop-blur-sm border border-gray-200/50 text-sm font-medium text-gray-600 hover:bg-gray-50 hover:text-primary-600 transition-all shadow-sm disabled:opacity-50"
            >
              <RefreshCw className={`w-4 h-4 ${loading ? 'animate-spin' : ''}`} />
              刷新
            </button>
          </div>
        </div>

        {/* Date range display */}
        {stats && (
          <div className="flex items-center gap-2 text-sm text-gray-500 fade-slide-up">
            <Calendar className="w-4 h-4" />
            <span>统计周期：{stats.start_date} 至 {stats.end_date}</span>
          </div>
        )}

        {/* Summary Stats Cards */}
        <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-4">
          <StatCard
            icon={Timer}
            label="总工时"
            value={stats?.total?.total_hours || 0}
            unit="小时"
            color="blue"
            delay={0}
          />
          <StatCard
            icon={Target}
            label="实际人天"
            value={stats?.total?.total_man_days || 0}
            unit="天"
            color="green"
            delay={100}
          />
          <StatCard
            icon={Activity}
            label="成本人天"
            value={stats?.total?.total_cost_days || 0}
            unit="天"
            color="purple"
            delay={200}
          />
          <StatCard
            icon={Briefcase}
            label="涉及项目"
            value={stats?.total?.project_count || 0}
            unit="个"
            color="orange"
            delay={300}
          />
        </div>

        {/* User Management */}
        <UserManagePanel
          users={users}
          onAdd={handleAddUser}
          onRemove={handleRemoveUser}
          loading={loading}
        />

        {/* User Worktime Details */}
        <GlassCard delay={300} className="p-5" hover={false}>
          <div className="flex items-center justify-between mb-4">
            <div className="flex items-center gap-2">
              <div className="w-8 h-8 rounded-lg bg-gradient-to-br from-emerald-500 to-emerald-600 flex items-center justify-center shadow-md shadow-emerald-200">
                <BarChart3 className="w-4 h-4 text-white" />
              </div>
              <h3 className="font-semibold text-gray-800">个人工时明细</h3>
              {stats && (
                <span className="text-xs bg-gray-100 text-gray-500 px-2 py-0.5 rounded-full">
                  {filteredUsers.length} 人
                </span>
              )}
            </div>
            {/* Search */}
            <div className="relative">
              <Search className="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-gray-400" />
              <input
                type="text"
                value={searchTerm}
                onChange={(e) => setSearchTerm(e.target.value)}
                placeholder="搜索姓名..."
                className="pl-9 pr-4 py-2 rounded-xl border border-gray-200/80 bg-white/60 backdrop-blur-sm text-sm focus:outline-none focus:ring-2 focus:ring-primary-300 transition-all w-44"
              />
            </div>
          </div>

          {/* Loading state */}
          {loading && (
            <div className="space-y-3">
              {[1, 2, 3, 4].map((i) => (
                <div key={i} className="h-16 rounded-xl shimmer-loading" />
              ))}
            </div>
          )}

          {/* User list */}
          {!loading && (
            <div className="space-y-2">
              {filteredUsers.map((user, idx) => (
                <UserDetailCard key={user.name} user={user} index={idx} />
              ))}
              {filteredUsers.length === 0 && !loading && (
                <div className="text-center py-12 text-gray-400">
                  <Clock className="w-12 h-12 mx-auto mb-3 text-gray-200" />
                  <p className="text-sm">
                    {users.length === 0 ? '请先添加需要统计工时的人员' : '暂无工时数据'}
                  </p>
                </div>
              )}
            </div>
          )}
        </GlassCard>
      </div>
    </div>
  );
}
