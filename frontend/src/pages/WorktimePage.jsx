import React, { useState, useEffect, useCallback, useRef } from 'react';
import {
  Clock, Users, TrendingUp, Calendar, Plus, Trash2, UserPlus, Search,
  ChevronDown, ChevronRight, ChevronLeft, BarChart3, Briefcase, RefreshCw, Download,
  Activity, Timer, Target, AlertCircle, FileSpreadsheet, CalendarRange
} from 'lucide-react';
import { getWorktimeStats, getWorktimeUsers, addWorktimeUser, removeWorktimeUser } from '../services/api';
import toast from 'react-hot-toast';

/* ═══════════════════════════════════════════════════════════════════
   Minimal animation styles (no elastic/bounce)
═══════════════════════════════════════════════════════════════════ */
const styleTag = document.createElement('style');
styleTag.textContent = `
  @keyframes shimmer {
    0% { background-position: -200% 0; }
    100% { background-position: 200% 0; }
  }
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
   Frosted Glass Card Component (no elastic animation)
═══════════════════════════════════════════════════════════════════ */
function GlassCard({ children, className = '', hover = true }) {
  return (
    <div
      className={`
        rounded-2xl border border-white/40 shadow-lg
        bg-white/70 backdrop-blur-xl
        ${hover ? 'hover:shadow-xl hover:border-primary-200/60 transition-all duration-300' : ''}
        ${className}
      `}
    >
      {children}
    </div>
  );
}

/* ═══════════════════════════════════════════════════════════════════
   Date Picker Calendar (frosted glass + rounded)
═══════════════════════════════════════════════════════════════════ */
function DatePickerCalendar({ value, onChange, onClose, label }) {
  const [viewDate, setViewDate] = useState(() => value ? new Date(value) : new Date());

  const year = viewDate.getFullYear();
  const month = viewDate.getMonth();
  const daysInMonth = new Date(year, month + 1, 0).getDate();
  const firstDayOfWeek = new Date(year, month, 1).getDay(); // 0=Sun
  const weekDays = ['日', '一', '二', '三', '四', '五', '六'];

  const prevMonth = () => setViewDate(new Date(year, month - 1, 1));
  const nextMonth = () => setViewDate(new Date(year, month + 1, 1));

  const handleSelect = (day) => {
    const dateStr = `${year}-${String(month + 1).padStart(2, '0')}-${String(day).padStart(2, '0')}`;
    onChange(dateStr);
  };

  const selectedDay = value && value.startsWith(`${year}-${String(month + 1).padStart(2, '0')}`)
    ? parseInt(value.split('-')[2], 10) : null;

  return (
    <div className="rounded-2xl border border-white/50 shadow-xl bg-white/80 backdrop-blur-xl p-4 w-72">
      <p className="text-xs text-gray-400 mb-2 font-medium">{label}</p>
      {/* Month navigation */}
      <div className="flex items-center justify-between mb-3">
        <button onClick={prevMonth} className="p-1.5 rounded-lg hover:bg-gray-100 transition-colors">
          <ChevronLeft className="w-4 h-4 text-gray-500" />
        </button>
        <span className="text-sm font-semibold text-gray-700">{year}年{month + 1}月</span>
        <button onClick={nextMonth} className="p-1.5 rounded-lg hover:bg-gray-100 transition-colors">
          <ChevronRight className="w-4 h-4 text-gray-500" />
        </button>
      </div>
      {/* Week headers */}
      <div className="grid grid-cols-7 gap-1 mb-1">
        {weekDays.map((d) => (
          <div key={d} className="text-center text-xs text-gray-400 font-medium py-1">{d}</div>
        ))}
      </div>
      {/* Days grid */}
      <div className="grid grid-cols-7 gap-1">
        {Array.from({ length: firstDayOfWeek }).map((_, i) => (
          <div key={`empty-${i}`} />
        ))}
        {Array.from({ length: daysInMonth }).map((_, i) => {
          const day = i + 1;
          const isSelected = day === selectedDay;
          const isToday = (() => {
            const t = new Date();
            return t.getFullYear() === year && t.getMonth() === month && t.getDate() === day;
          })();
          return (
            <button
              key={day}
              onClick={() => handleSelect(day)}
              className={`w-8 h-8 rounded-lg text-sm font-medium transition-all duration-150 ${
                isSelected
                  ? 'bg-primary-600 text-white shadow-md shadow-primary-200'
                  : isToday
                    ? 'bg-primary-50 text-primary-700 border border-primary-200'
                    : 'text-gray-600 hover:bg-gray-100'
              }`}
            >
              {day}
            </button>
          );
        })}
      </div>
      {value && (
        <p className="mt-2 text-xs text-center text-primary-600 font-medium">已选: {value}</p>
      )}
    </div>
  );
}

/* ═══════════════════════════════════════════════════════════════════
   Period Selector Component (with custom date range picker)
═══════════════════════════════════════════════════════════════════ */
function PeriodSelector({ period, onChange, customRange, onCustomRangeChange }) {
  const [showCustom, setShowCustom] = useState(false);
  const [tempStart, setTempStart] = useState(customRange?.start_date || '');
  const [tempEnd, setTempEnd] = useState(customRange?.end_date || '');
  const popoverRef = useRef(null);

  const periods = [
    { value: 'month', label: '本月', icon: Calendar },
    { value: 'quarter', label: '本季度', icon: BarChart3 },
    { value: 'year', label: '本年度', icon: TrendingUp },
  ];

  // Close popover on outside click
  useEffect(() => {
    const handler = (e) => {
      if (popoverRef.current && !popoverRef.current.contains(e.target)) {
        setShowCustom(false);
      }
    };
    if (showCustom) document.addEventListener('mousedown', handler);
    return () => document.removeEventListener('mousedown', handler);
  }, [showCustom]);

  const handleApplyCustom = () => {
    if (tempStart && tempEnd) {
      onCustomRangeChange(tempStart, tempEnd);
      setShowCustom(false);
    } else {
      toast.error('请选择起止日期');
    }
  };

  return (
    <div className="flex gap-2 p-1 bg-white/50 backdrop-blur-sm rounded-xl border border-gray-200/50 relative">
      {periods.map((p) => {
        const Icon = p.icon;
        const isActive = period === p.value;
        return (
          <button
            key={p.value}
            onClick={() => { onChange(p.value); setShowCustom(false); }}
            className={`flex items-center gap-1.5 px-4 py-2 rounded-lg text-sm font-medium transition-all duration-200 ${
              isActive
                ? 'bg-primary-600 text-white shadow-md shadow-primary-200'
                : 'text-gray-600 hover:bg-gray-100/80 hover:text-gray-800'
            }`}
          >
            <Icon className="w-4 h-4" />
            {p.label}
          </button>
        );
      })}
      {/* Custom period button */}
      <button
        onClick={() => setShowCustom(!showCustom)}
        className={`flex items-center gap-1.5 px-4 py-2 rounded-lg text-sm font-medium transition-all duration-200 ${
          period === 'custom'
            ? 'bg-primary-600 text-white shadow-md shadow-primary-200'
            : 'text-gray-600 hover:bg-gray-100/80 hover:text-gray-800'
        }`}
      >
        <CalendarRange className="w-4 h-4" />
        自定义
      </button>

      {/* Custom date range popover */}
      {showCustom && (
        <div
          ref={popoverRef}
          className="absolute top-full right-0 mt-2 z-50 rounded-2xl border border-white/50 shadow-2xl bg-white/85 backdrop-blur-2xl p-5"
        >
          <p className="text-sm font-semibold text-gray-700 mb-3 flex items-center gap-2">
            <CalendarRange className="w-4 h-4 text-primary-500" />
            自定义统计周期
          </p>
          <div className="flex gap-4">
            <DatePickerCalendar
              value={tempStart}
              onChange={setTempStart}
              label="开始日期"
            />
            <DatePickerCalendar
              value={tempEnd}
              onChange={setTempEnd}
              label="结束日期"
            />
          </div>
          <div className="flex items-center justify-between mt-4 pt-3 border-t border-gray-100">
            <div className="text-xs text-gray-400">
              {tempStart && tempEnd ? `${tempStart} 至 ${tempEnd}` : '请选择日期范围'}
            </div>
            <div className="flex gap-2">
              <button
                onClick={() => setShowCustom(false)}
                className="px-3 py-1.5 rounded-lg text-sm text-gray-500 hover:bg-gray-100 transition-colors"
              >
                取消
              </button>
              <button
                onClick={handleApplyCustom}
                disabled={!tempStart || !tempEnd}
                className="px-4 py-1.5 rounded-lg text-sm font-medium bg-primary-600 text-white hover:bg-primary-700 disabled:opacity-50 disabled:cursor-not-allowed transition-all shadow-sm"
              >
                确定
              </button>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}

/* ═══════════════════════════════════════════════════════════════════
   Stats Card Component
═══════════════════════════════════════════════════════════════════ */
function StatCard({ icon: Icon, label, value, unit, color }) {
  const colorClasses = {
    blue: 'from-blue-500 to-blue-600 shadow-blue-200',
    green: 'from-emerald-500 to-emerald-600 shadow-emerald-200',
    purple: 'from-purple-500 to-purple-600 shadow-purple-200',
    orange: 'from-orange-500 to-orange-600 shadow-orange-200',
  };

  return (
    <GlassCard className="p-5">
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
function UserManagePanel({ users, onAdd, onRemove }) {
  const [newName, setNewName] = useState('');
  const [showPanel, setShowPanel] = useState(false);

  const handleAdd = () => {
    if (!newName.trim()) return;
    onAdd(newName.trim());
    setNewName('');
  };

  return (
    <GlassCard className="p-5">
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
        <div>
          {/* Add user input - accepts any Chinese name, no validation */}
          <div className="flex gap-2 mb-3">
            <div className="flex-1 relative">
              <UserPlus className="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-gray-400" />
              <input
                type="text"
                value={newName}
                onChange={(e) => setNewName(e.target.value)}
                onKeyDown={(e) => e.key === 'Enter' && handleAdd()}
                placeholder="输入中文姓名添加人员..."
                className="w-full pl-9 pr-4 py-2 rounded-xl border border-gray-200/80 bg-white/60 backdrop-blur-sm text-sm focus:outline-none focus:ring-2 focus:ring-primary-300 focus:border-primary-300 transition-all"
              />
            </div>
            <button
              onClick={handleAdd}
              disabled={!newName.trim()}
              className="px-4 py-2 rounded-xl bg-primary-600 text-white text-sm font-medium hover:bg-primary-700 disabled:opacity-50 disabled:cursor-not-allowed transition-all shadow-md shadow-primary-200"
            >
              <Plus className="w-4 h-4" />
            </button>
          </div>

          {/* User list */}
          <div className="max-h-48 overflow-y-auto space-y-1.5 pr-1" style={{ scrollbarWidth: 'thin' }}>
            {users.map((u) => (
              <div
                key={u.id}
                className="flex items-center justify-between px-3 py-2 rounded-xl bg-gray-50/80 hover:bg-gray-100/80 group transition-colors"
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
function UserDetailCard({ user }) {
  const [expanded, setExpanded] = useState(false);

  if (user.total_hours === 0 && user.project_details?.length === 0) {
    return (
      <div className="flex items-center gap-3 px-4 py-3 rounded-xl bg-white/50 backdrop-blur-sm border border-gray-100/50">
        <div className="w-9 h-9 rounded-full bg-gray-200 flex items-center justify-center text-sm text-gray-500 font-medium">
          {user.name.slice(0, 1)}
        </div>
        <span className="text-sm text-gray-600">{user.name}</span>
        <span className="ml-auto text-xs text-gray-400">暂无工时记录</span>
      </div>
    );
  }

  return (
    <div className="rounded-xl border border-gray-100/60 bg-white/60 backdrop-blur-sm overflow-hidden transition-all hover:shadow-md">
      {/* Header */}
      <button
        onClick={() => setExpanded(!expanded)}
        className="w-full flex items-center gap-3 px-5 py-4 text-left hover:bg-gray-50/50 transition-colors"
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
        <ChevronRight className={`w-4 h-4 text-gray-400 transition-transform duration-200 ${expanded ? 'rotate-90' : ''}`} />
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
  const [customRange, setCustomRange] = useState({ start_date: '', end_date: '' });
  const [stats, setStats] = useState(null);
  const [users, setUsers] = useState([]);
  const [loading, setLoading] = useState(false);
  const [exporting, setExporting] = useState(false);
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
      const params = { period };
      if (period === 'custom' && customRange.start_date && customRange.end_date) {
        params.start_date = customRange.start_date;
        params.end_date = customRange.end_date;
      }
      const res = await getWorktimeStats(params);
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
  }, [period, customRange]);

  useEffect(() => { fetchUsers(); }, [fetchUsers]);
  useEffect(() => { fetchStats(); }, [fetchStats]);

  // Handle custom range selection
  const handleCustomRange = (start, end) => {
    setCustomRange({ start_date: start, end_date: end });
    setPeriod('custom');
  };

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

  // Export CSV files
  const handleExport = async (type) => {
    setExporting(true);
    try {
      const token = localStorage.getItem('token');
      const params = new URLSearchParams({ period, type });
      if (period === 'custom' && customRange.start_date && customRange.end_date) {
        params.set('start_date', customRange.start_date);
        params.set('end_date', customRange.end_date);
      }
      const response = await fetch(`/api/worktime/export?${params}`, {
        headers: { Authorization: `Bearer ${token}` },
      });
      if (!response.ok) {
        const err = await response.json();
        toast.error(err.message || '导出失败');
        return;
      }
      const blob = await response.blob();
      const url = window.URL.createObjectURL(blob);
      const a = document.createElement('a');
      a.href = url;
      const disposition = response.headers.get('Content-Disposition');
      const filename = disposition?.match(/filename\*?=(?:UTF-8'')?([^;\n]+)/)?.[1] || 
        (type === 'delivery' ? '交付工时.csv' : '人员工时.csv');
      a.download = decodeURIComponent(filename);
      document.body.appendChild(a);
      a.click();
      a.remove();
      window.URL.revokeObjectURL(url);
      toast.success('导出成功');
    } catch (err) {
      toast.error('导出失败: ' + (err.message || '网络错误'));
    } finally {
      setExporting(false);
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

          <div className="flex items-center gap-3 flex-wrap">
            <PeriodSelector
              period={period}
              onChange={setPeriod}
              customRange={customRange}
              onCustomRangeChange={handleCustomRange}
            />
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

        {/* Date range + Export buttons */}
        {stats && (
          <div className="flex items-center justify-between flex-wrap gap-3">
            <div className="flex items-center gap-2 text-sm text-gray-500">
              <Calendar className="w-4 h-4" />
              <span>统计周期：{stats.start_date} 至 {stats.end_date}</span>
            </div>
            <div className="flex items-center gap-2">
              <button
                onClick={() => handleExport('delivery')}
                disabled={exporting}
                className="flex items-center gap-1.5 px-3 py-1.5 rounded-lg bg-emerald-50 text-emerald-700 text-sm font-medium hover:bg-emerald-100 border border-emerald-200/50 transition-all disabled:opacity-50"
              >
                <FileSpreadsheet className="w-4 h-4" />
                导出交付工时
              </button>
              <button
                onClick={() => handleExport('personnel')}
                disabled={exporting}
                className="flex items-center gap-1.5 px-3 py-1.5 rounded-lg bg-blue-50 text-blue-700 text-sm font-medium hover:bg-blue-100 border border-blue-200/50 transition-all disabled:opacity-50"
              >
                <Download className="w-4 h-4" />
                导出人员工时
              </button>
            </div>
          </div>
        )}

        {/* Summary Stats Cards */}
        <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-4">
          <StatCard icon={Timer} label="总工时" value={stats?.total?.total_hours || 0} unit="小时" color="blue" />
          <StatCard icon={Target} label="实际人天" value={stats?.total?.total_man_days || 0} unit="天" color="green" />
          <StatCard icon={Activity} label="成本人天" value={stats?.total?.total_cost_days || 0} unit="天" color="purple" />
          <StatCard icon={Briefcase} label="涉及项目" value={stats?.total?.project_count || 0} unit="个" color="orange" />
        </div>

        {/* User Management */}
        <UserManagePanel
          users={users}
          onAdd={handleAddUser}
          onRemove={handleRemoveUser}
        />

        {/* User Worktime Details */}
        <GlassCard className="p-5" hover={false}>
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
              {filteredUsers.map((user) => (
                <UserDetailCard key={user.name} user={user} />
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
