import React, { useState, useEffect, useCallback } from 'react';
import {
  LineChart, Line, XAxis, YAxis, CartesianGrid, Tooltip, Legend, ResponsiveContainer,
  PieChart, Pie, Cell,
  BarChart, Bar,
} from 'recharts';
import {
  FolderKanban, TrendingUp, Calendar, RefreshCw, ChevronDown,
  MapPin, Users, Building2, Layers, Activity, Award,
} from 'lucide-react';
import useStore from '../store/useStore';

const API_BASE = import.meta.env.VITE_API_BASE || '/api';

// Color palette for charts
const COLORS = ['#6366f1', '#8b5cf6', '#a78bfa', '#c4b5fd', '#e0e7ff', '#818cf8', '#7c3aed', '#5b21b6', '#4f46e5', '#4338ca'];
const GRADIENT_COLORS = ['#6366f1', '#8b5cf6', '#a78bfa', '#c4b5fd', '#ddd6fe'];

// Period options
const PERIOD_OPTIONS = [
  { value: 'month', label: '本月' },
  { value: 'quarter', label: '本季度' },
  { value: 'half_year', label: '近半年' },
  { value: 'year', label: '本年度' },
  { value: 'custom', label: '自定义' },
];

function getPeriodDates(period) {
  const now = new Date();
  let start, end;
  end = now.toISOString().split('T')[0];

  switch (period) {
    case 'month':
      start = new Date(now.getFullYear(), now.getMonth(), 1).toISOString().split('T')[0];
      break;
    case 'quarter': {
      const qMonth = Math.floor(now.getMonth() / 3) * 3;
      start = new Date(now.getFullYear(), qMonth, 1).toISOString().split('T')[0];
      break;
    }
    case 'half_year':
      start = new Date(now.getFullYear(), now.getMonth() - 6, now.getDate()).toISOString().split('T')[0];
      break;
    case 'year':
      start = new Date(now.getFullYear(), 0, 1).toISOString().split('T')[0];
      break;
    default:
      start = new Date(now.getFullYear(), 0, 1).toISOString().split('T')[0];
  }
  return { start, end };
}

// Glassmorphism card component
function GlassCard({ children, className = '', icon: Icon, title, subtitle }) {
  return (
    <div className={`rounded-2xl border border-white/40 bg-white/70 backdrop-blur-xl shadow-lg p-5 ${className}`}>
      {(title || Icon) && (
        <div className="flex items-center gap-2.5 mb-4">
          {Icon && (
            <div className="w-9 h-9 rounded-xl bg-gradient-to-br from-indigo-500 to-purple-600 flex items-center justify-center shadow-md">
              <Icon className="w-4.5 h-4.5 text-white" size={18} />
            </div>
          )}
          <div>
            {title && <h3 className="text-sm font-semibold text-gray-800">{title}</h3>}
            {subtitle && <p className="text-xs text-gray-400">{subtitle}</p>}
          </div>
        </div>
      )}
      {children}
    </div>
  );
}

// Stat card component
function StatCard({ icon: Icon, label, value, trend, color = 'indigo' }) {
  const colorMap = {
    indigo: 'from-indigo-500 to-purple-600',
    blue: 'from-blue-500 to-cyan-600',
    green: 'from-emerald-500 to-teal-600',
    orange: 'from-orange-500 to-amber-600',
  };
  return (
    <div className="rounded-2xl border border-white/40 bg-white/70 backdrop-blur-xl shadow-lg p-5 flex items-center gap-4">
      <div className={`w-12 h-12 rounded-xl bg-gradient-to-br ${colorMap[color]} flex items-center justify-center shadow-md`}>
        <Icon className="w-6 h-6 text-white" />
      </div>
      <div>
        <p className="text-xs text-gray-400 font-medium">{label}</p>
        <p className="text-2xl font-bold text-gray-800">{value ?? '-'}</p>
        {trend !== undefined && (
          <p className={`text-xs mt-0.5 ${trend >= 0 ? 'text-emerald-600' : 'text-red-500'}`}>
            {trend >= 0 ? '↑' : '↓'} {Math.abs(trend)}%
          </p>
        )}
      </div>
    </div>
  );
}

// Custom tooltip for charts
function CustomTooltip({ active, payload, label }) {
  if (!active || !payload?.length) return null;
  return (
    <div className="rounded-xl border border-white/50 bg-white/90 backdrop-blur-md shadow-lg px-3 py-2">
      <p className="text-xs font-medium text-gray-600 mb-1">{label}</p>
      {payload.map((entry, idx) => (
        <p key={idx} className="text-sm font-semibold" style={{ color: entry.color }}>
          {entry.name}: {entry.value}
        </p>
      ))}
    </div>
  );
}

export default function ProjectManagePage() {
  const token = useStore((s) => s.token);
  const [stats, setStats] = useState(null);
  const [loading, setLoading] = useState(true);
  const [syncing, setSyncing] = useState(false);
  const [period, setPeriod] = useState('year');
  const [customStart, setCustomStart] = useState('');
  const [customEnd, setCustomEnd] = useState('');
  const [showPeriodMenu, setShowPeriodMenu] = useState(false);

  const fetchStats = useCallback(async () => {
    setLoading(true);
    try {
      let startDate, endDate;
      if (period === 'custom' && customStart && customEnd) {
        startDate = customStart;
        endDate = customEnd;
      } else {
        const dates = getPeriodDates(period);
        startDate = dates.start;
        endDate = dates.end;
      }

      const res = await fetch(
        `${API_BASE}/projects/stats?period_type=${period}&start_date=${startDate}&end_date=${endDate}`,
        { headers: { Authorization: `Bearer ${token}` } }
      );
      const data = await res.json();
      if (data.code === 0) {
        setStats(data.data);
      }
    } catch (e) {
      console.error('Failed to fetch project stats:', e);
    } finally {
      setLoading(false);
    }
  }, [token, period, customStart, customEnd]);

  useEffect(() => {
    fetchStats();
  }, [fetchStats]);

  const handleSync = async () => {
    setSyncing(true);
    try {
      await fetch(`${API_BASE}/projects/sync`, {
        method: 'POST',
        headers: { Authorization: `Bearer ${token}` },
      });
      // Wait a bit then refresh
      setTimeout(() => {
        fetchStats();
        setSyncing(false);
      }, 3000);
    } catch (e) {
      setSyncing(false);
    }
  };

  if (loading && !stats) {
    return (
      <div className="flex items-center justify-center h-full">
        <div className="animate-spin w-8 h-8 border-4 border-indigo-500 border-t-transparent rounded-full" />
      </div>
    );
  }

  return (
    <div className="h-full overflow-y-auto p-6 space-y-6" style={{ scrollbarWidth: 'thin' }}>
      {/* Header with period selector */}
      <div className="flex flex-wrap items-center justify-between gap-4">
        <div className="flex items-center gap-3">
          <div className="w-10 h-10 rounded-xl bg-gradient-to-br from-indigo-500 to-purple-600 flex items-center justify-center shadow-md">
            <FolderKanban className="w-5 h-5 text-white" />
          </div>
          <div>
            <h2 className="text-lg font-bold text-gray-800">项目管理看板</h2>
            <p className="text-xs text-gray-400">
              Redmine 项目立项数据统计分析
              {stats?.last_sync_time && ` · 最后同步: ${stats.last_sync_time}`}
            </p>
          </div>
        </div>

        <div className="flex items-center gap-3">
          {/* Period selector */}
          <div className="relative">
            <button
              onClick={() => setShowPeriodMenu(!showPeriodMenu)}
              className="flex items-center gap-2 px-4 py-2 rounded-xl border border-white/40 bg-white/70 backdrop-blur-xl shadow-sm text-sm font-medium text-gray-700 hover:bg-white/90 transition"
            >
              <Calendar size={14} />
              {PERIOD_OPTIONS.find(p => p.value === period)?.label}
              <ChevronDown size={14} />
            </button>
            {showPeriodMenu && (
              <div className="absolute right-0 top-full mt-1 z-50 rounded-xl border border-white/40 bg-white/90 backdrop-blur-xl shadow-xl py-1 min-w-[140px]">
                {PERIOD_OPTIONS.map(opt => (
                  <button
                    key={opt.value}
                    onClick={() => { setPeriod(opt.value); setShowPeriodMenu(false); }}
                    className={`w-full text-left px-4 py-2 text-sm hover:bg-indigo-50 transition ${period === opt.value ? 'text-indigo-600 font-medium bg-indigo-50/50' : 'text-gray-700'}`}
                  >
                    {opt.label}
                  </button>
                ))}
              </div>
            )}
          </div>

          {/* Custom date range */}
          {period === 'custom' && (
            <div className="flex items-center gap-2">
              <input
                type="date"
                value={customStart}
                onChange={e => setCustomStart(e.target.value)}
                className="px-3 py-1.5 rounded-lg border border-gray-200 text-sm bg-white/80"
              />
              <span className="text-gray-400 text-sm">至</span>
              <input
                type="date"
                value={customEnd}
                onChange={e => setCustomEnd(e.target.value)}
                className="px-3 py-1.5 rounded-lg border border-gray-200 text-sm bg-white/80"
              />
            </div>
          )}

          {/* Sync button */}
          <button
            onClick={handleSync}
            disabled={syncing}
            className="flex items-center gap-2 px-4 py-2 rounded-xl bg-gradient-to-r from-indigo-500 to-purple-600 text-white text-sm font-medium shadow-md hover:shadow-lg transition disabled:opacity-60"
          >
            <RefreshCw size={14} className={syncing ? 'animate-spin' : ''} />
            {syncing ? '同步中...' : '同步数据'}
          </button>
        </div>
      </div>

      {/* Stat overview cards */}
      <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-4">
        <StatCard icon={FolderKanban} label="项目总数" value={stats?.total_projects} color="indigo" />
        <StatCard icon={TrendingUp} label="周期新增" value={stats?.period_new} color="blue" />
        <StatCard icon={Activity} label="近7天新增" value={stats?.week_new} color="green" />
        <StatCard icon={Award} label="年度立项数" value={stats?.yoy_comparison?.length ? stats.yoy_comparison[stats.yoy_comparison.length - 1]?.count : '-'} color="orange" />
      </div>

      {/* Charts Row 1: Line chart + Donut chart */}
      <div className="grid grid-cols-1 lg:grid-cols-3 gap-4">
        {/* Monthly trend line chart */}
        <GlassCard className="lg:col-span-2" icon={TrendingUp} title="月度立项趋势" subtitle="近12个月项目立项数量变化">
          <div className="h-64">
            <ResponsiveContainer width="100%" height="100%">
              <LineChart data={stats?.monthly_trend || []}>
                <CartesianGrid strokeDasharray="3 3" stroke="#e5e7eb" />
                <XAxis dataKey="month" tick={{ fontSize: 11 }} stroke="#9ca3af" />
                <YAxis tick={{ fontSize: 11 }} stroke="#9ca3af" />
                <Tooltip content={<CustomTooltip />} />
                <Line
                  type="monotone"
                  dataKey="count"
                  name="立项数"
                  stroke="#6366f1"
                  strokeWidth={2.5}
                  dot={{ fill: '#6366f1', r: 4 }}
                  activeDot={{ r: 6 }}
                />
              </LineChart>
            </ResponsiveContainer>
          </div>
        </GlassCard>

        {/* Region distribution donut */}
        <GlassCard icon={MapPin} title="区域分布" subtitle="项目区域分布TOP10">
          <div className="h-64">
            <ResponsiveContainer width="100%" height="100%">
              <PieChart>
                <Pie
                  data={stats?.region_distribution || []}
                  cx="50%"
                  cy="50%"
                  innerRadius={50}
                  outerRadius={80}
                  dataKey="count"
                  nameKey="name"
                  label={({ name, percent }) => `${name} ${(percent * 100).toFixed(0)}%`}
                  labelLine={{ strokeWidth: 1 }}
                >
                  {(stats?.region_distribution || []).map((_, idx) => (
                    <Cell key={idx} fill={COLORS[idx % COLORS.length]} />
                  ))}
                </Pie>
                <Tooltip />
              </PieChart>
            </ResponsiveContainer>
          </div>
        </GlassCard>
      </div>

      {/* Charts Row 2: YoY comparison + Project type donut */}
      <div className="grid grid-cols-1 lg:grid-cols-3 gap-4">
        {/* Year-over-year bar chart */}
        <GlassCard className="lg:col-span-2" icon={Layers} title="年度环比对比" subtitle="各年度立项数量对比">
          <div className="h-64">
            <ResponsiveContainer width="100%" height="100%">
              <BarChart data={stats?.yoy_comparison || []}>
                <CartesianGrid strokeDasharray="3 3" stroke="#e5e7eb" />
                <XAxis dataKey="year" tick={{ fontSize: 11 }} stroke="#9ca3af" />
                <YAxis tick={{ fontSize: 11 }} stroke="#9ca3af" />
                <Tooltip content={<CustomTooltip />} />
                <Bar dataKey="count" name="立项数" radius={[6, 6, 0, 0]}>
                  {(stats?.yoy_comparison || []).map((_, idx) => (
                    <Cell key={idx} fill={COLORS[idx % COLORS.length]} />
                  ))}
                </Bar>
              </BarChart>
            </ResponsiveContainer>
          </div>
        </GlassCard>

        {/* Project type donut */}
        <GlassCard icon={FolderKanban} title="项目类型分布" subtitle="按类型统计">
          <div className="h-64">
            <ResponsiveContainer width="100%" height="100%">
              <PieChart>
                <Pie
                  data={stats?.type_distribution || []}
                  cx="50%"
                  cy="50%"
                  innerRadius={50}
                  outerRadius={80}
                  dataKey="count"
                  nameKey="name"
                  label={({ name, percent }) => `${name} ${(percent * 100).toFixed(0)}%`}
                  labelLine={{ strokeWidth: 1 }}
                >
                  {(stats?.type_distribution || []).map((_, idx) => (
                    <Cell key={idx} fill={GRADIENT_COLORS[idx % GRADIENT_COLORS.length]} />
                  ))}
                </Pie>
                <Tooltip />
              </PieChart>
            </ResponsiveContainer>
          </div>
        </GlassCard>
      </div>

      {/* Charts Row 3: Project status bar + Weekly */}
      <div className="grid grid-cols-1 lg:grid-cols-2 gap-4">
        {/* Project status bar chart */}
        <GlassCard icon={Activity} title="项目状态分布" subtitle="按项目当前状态统计">
          <div className="h-56">
            <ResponsiveContainer width="100%" height="100%">
              <BarChart data={stats?.status_distribution || []} layout="vertical">
                <CartesianGrid strokeDasharray="3 3" stroke="#e5e7eb" />
                <XAxis type="number" tick={{ fontSize: 11 }} stroke="#9ca3af" />
                <YAxis type="category" dataKey="name" tick={{ fontSize: 11 }} stroke="#9ca3af" width={80} />
                <Tooltip content={<CustomTooltip />} />
                <Bar dataKey="count" name="项目数" radius={[0, 6, 6, 0]} fill="#8b5cf6" />
              </BarChart>
            </ResponsiveContainer>
          </div>
        </GlassCard>

        {/* Weekly daily chart */}
        <GlassCard icon={Calendar} title="近一周立项情况" subtitle="过去7天每日新增项目">
          <div className="h-56">
            <ResponsiveContainer width="100%" height="100%">
              <BarChart data={stats?.weekly_projects || []}>
                <CartesianGrid strokeDasharray="3 3" stroke="#e5e7eb" />
                <XAxis dataKey="date" tick={{ fontSize: 11 }} stroke="#9ca3af" />
                <YAxis tick={{ fontSize: 11 }} stroke="#9ca3af" allowDecimals={false} />
                <Tooltip content={<CustomTooltip />} />
                <Bar dataKey="count" name="立项数" radius={[6, 6, 0, 0]} fill="#6366f1" />
              </BarChart>
            </ResponsiveContainer>
          </div>
        </GlassCard>
      </div>

      {/* TOP5 Section */}
      <div className="grid grid-cols-1 lg:grid-cols-3 gap-4">
        {/* TOP5 Regions */}
        <GlassCard icon={MapPin} title="TOP5 区域" subtitle="项目数量最多的区域">
          <div className="space-y-3">
            {(stats?.top5_regions || []).map((item, idx) => (
              <div key={idx} className="flex items-center gap-3">
                <span className={`w-6 h-6 rounded-lg flex items-center justify-center text-xs font-bold text-white ${idx < 3 ? 'bg-gradient-to-br from-indigo-500 to-purple-600' : 'bg-gray-300'}`}>
                  {idx + 1}
                </span>
                <span className="flex-1 text-sm text-gray-700 truncate">{item.name || '未知'}</span>
                <span className="text-sm font-semibold text-indigo-600">{item.count}</span>
              </div>
            ))}
            {(!stats?.top5_regions || stats.top5_regions.length === 0) && (
              <p className="text-sm text-gray-400 text-center py-4">暂无数据</p>
            )}
          </div>
        </GlassCard>

        {/* TOP5 Project Managers */}
        <GlassCard icon={Users} title="TOP5 项目经理" subtitle="管理项目数量最多">
          <div className="space-y-3">
            {(stats?.top5_managers || []).map((item, idx) => (
              <div key={idx} className="flex items-center gap-3">
                <span className={`w-6 h-6 rounded-lg flex items-center justify-center text-xs font-bold text-white ${idx < 3 ? 'bg-gradient-to-br from-blue-500 to-cyan-600' : 'bg-gray-300'}`}>
                  {idx + 1}
                </span>
                <span className="flex-1 text-sm text-gray-700 truncate">{item.name || '未知'}</span>
                <span className="text-sm font-semibold text-blue-600">{item.count}</span>
              </div>
            ))}
            {(!stats?.top5_managers || stats.top5_managers.length === 0) && (
              <p className="text-sm text-gray-400 text-center py-4">暂无数据</p>
            )}
          </div>
        </GlassCard>

        {/* TOP5 Customers */}
        <GlassCard icon={Building2} title="TOP5 客户" subtitle="项目数量最多的客户">
          <div className="space-y-3">
            {(stats?.top5_customers || []).map((item, idx) => (
              <div key={idx} className="flex items-center gap-3">
                <span className={`w-6 h-6 rounded-lg flex items-center justify-center text-xs font-bold text-white ${idx < 3 ? 'bg-gradient-to-br from-emerald-500 to-teal-600' : 'bg-gray-300'}`}>
                  {idx + 1}
                </span>
                <span className="flex-1 text-sm text-gray-700 truncate">{item.name || '未知'}</span>
                <span className="text-sm font-semibold text-emerald-600">{item.count}</span>
              </div>
            ))}
            {(!stats?.top5_customers || stats.top5_customers.length === 0) && (
              <p className="text-sm text-gray-400 text-center py-4">暂无数据</p>
            )}
          </div>
        </GlassCard>
      </div>
    </div>
  );
}
