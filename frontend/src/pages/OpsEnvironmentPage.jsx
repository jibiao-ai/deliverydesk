import React, { useState, useEffect, useCallback, useMemo } from 'react';
import {
  BarChart, Bar, XAxis, YAxis, CartesianGrid, Tooltip, ResponsiveContainer,
  PieChart, Pie, Cell,
} from 'recharts';
import {
  Server, RefreshCw, Search, ChevronLeft, ChevronRight, Monitor,
  Shield, AlertTriangle, XCircle, Activity, Layers, MapPin, Cpu,
  Calendar, CalendarDays, BarChart3, Users, HardDrive, TrendingUp, Truck,
} from 'lucide-react';
import useStore from '../store/useStore';
import toast from 'react-hot-toast';

const API_BASE = import.meta.env.VITE_API_BASE || '/api';

const COLORS = ['#6366f1', '#8b5cf6', '#a78bfa', '#c4b5fd', '#e0e7ff', '#818cf8', '#7c3aed', '#5b21b6', '#4f46e5', '#4338ca'];

/* ═══════════════════════════════════════════════════════════
   Glassmorphism Card
═══════════════════════════════════════════════════════════ */
function GlassCard({ children, className = '', icon: Icon, title, subtitle }) {
  return (
    <div className={`rounded-2xl border border-white/40 bg-white/70 backdrop-blur-xl shadow-lg p-4 ${className}`}>
      {(title || Icon) && (
        <div className="flex items-center gap-2 mb-3">
          {Icon && (
            <div className="w-8 h-8 rounded-xl bg-gradient-to-br from-indigo-500 to-purple-600 flex items-center justify-center shadow-md">
              <Icon className="text-white" size={15} />
            </div>
          )}
          <div>
            {title && <h3 className="text-xs font-semibold text-gray-800">{title}</h3>}
            {subtitle && <p className="text-[10px] text-gray-400">{subtitle}</p>}
          </div>
        </div>
      )}
      {children}
    </div>
  );
}

/* ═══════════════════════════════════════════════════════════
   Status Badge
═══════════════════════════════════════════════════════════ */
function StatusBadge({ status }) {
  const cfg_map = {
    'In Progress': { label: '维保中', color: 'bg-green-100 text-green-700 border-green-200', icon: Shield },
    '正在进行': { label: '维保中', color: 'bg-green-100 text-green-700 border-green-200', icon: Shield },
    'Done': { label: '过保', color: 'bg-amber-100 text-amber-700 border-amber-200', icon: AlertTriangle },
    '已完成': { label: '过保', color: 'bg-amber-100 text-amber-700 border-amber-200', icon: AlertTriangle },
    'Discarded': { label: '已弃用', color: 'bg-red-100 text-red-700 border-red-200', icon: XCircle },
    '已弃用': { label: '已弃用', color: 'bg-red-100 text-red-700 border-red-200', icon: XCircle },
    '待办': { label: '代建转运', color: 'bg-blue-100 text-blue-700 border-blue-200', icon: Truck },
  };
  const cfg = cfg_map[status] || { label: status || '未知', color: 'bg-gray-100 text-gray-600 border-gray-200', icon: Activity };
  const Icon = cfg.icon;
  return (
    <span className={`inline-flex items-center gap-1 px-2 py-0.5 text-xs font-medium rounded-full border ${cfg.color}`}>
      <Icon size={11} />
      {cfg.label}
    </span>
  );
}

/* ═══════════════════════════════════════════════════════════
   Calendar View Selector
═══════════════════════════════════════════════════════════ */
function CalendarViewSelector({ view, onChange }) {
  const views = [
    { value: 'month', label: '月历', icon: Calendar },
    { value: 'year', label: '年历', icon: BarChart3 },
  ];
  return (
    <div className="flex gap-1 p-0.5 bg-white/50 backdrop-blur-sm rounded-lg border border-gray-200/50">
      {views.map((v) => {
        const Icon = v.icon;
        const isActive = view === v.value;
        return (
          <button key={v.value} onClick={() => onChange(v.value)}
            className={`flex items-center gap-1 px-2 py-1 rounded-md text-[10px] font-medium transition-all duration-200 ${
              isActive ? 'bg-primary-600 text-white shadow-sm' : 'text-gray-600 hover:bg-gray-100/80'
            }`}>
            <Icon className="w-3 h-3" /> {v.label}
          </button>
        );
      })}
    </div>
  );
}

/* ═══════════════════════════════════════════════════════════
   Discard Calendar Heat Map
═══════════════════════════════════════════════════════════ */
function DiscardCalendarView({ data, view, year, month }) {
  const dateCountMap = useMemo(() => {
    const m = {};
    (data || []).forEach(item => {
      if (item.date) m[item.date] = item.count;
    });
    return m;
  }, [data]);

  const maxCount = useMemo(() => {
    const vals = Object.values(dateCountMap);
    return vals.length > 0 ? Math.max(...vals, 1) : 1;
  }, [dateCountMap]);

  // Year view: build monthly data
  const monthlyData = useMemo(() => {
    const months = [];
    for (let i = 1; i <= 12; i++) {
      const key = `${year}-${String(i).padStart(2, '0')}`;
      months.push({ month: `${i}月`, key, count: dateCountMap[key] || 0 });
    }
    return months;
  }, [dateCountMap, year]);

  const getColorClass = (count) => {
    if (!count || count === 0) return 'bg-gray-50 border-gray-100';
    const ratio = count / maxCount;
    if (ratio > 0.75) return 'bg-red-500 border-red-400 text-white';
    if (ratio > 0.5) return 'bg-red-400 border-red-300 text-white';
    if (ratio > 0.25) return 'bg-red-200 border-red-200 text-red-800';
    return 'bg-red-100 border-red-100 text-red-700';
  };

  // Year View: 12-month bar chart
  if (view === 'year') {
    return (
      <div className="h-36">
        <ResponsiveContainer width="100%" height="100%">
          <BarChart data={monthlyData} margin={{ top: 5, right: 5, left: -10, bottom: 5 }}>
            <CartesianGrid strokeDasharray="3 3" stroke="#f0f0f0" />
            <XAxis dataKey="month" tick={{ fontSize: 9 }} />
            <YAxis tick={{ fontSize: 9 }} allowDecimals={false} />
            <Tooltip
              contentStyle={{ borderRadius: '12px', border: '1px solid rgba(255,255,255,0.5)', background: 'rgba(255,255,255,0.9)', backdropFilter: 'blur(8px)', fontSize: 11 }}
              formatter={(val) => [`${val} 个`, '弃用数']}
            />
            <Bar dataKey="count" fill="#ef4444" radius={[3, 3, 0, 0]} />
          </BarChart>
        </ResponsiveContainer>
      </div>
    );
  }

  // Month View: Calendar Grid
  const daysInMonth = new Date(year, month, 0).getDate();
  const firstDayOfWeek = new Date(year, month - 1, 1).getDay();
  const weekDays = ['日', '一', '二', '三', '四', '五', '六'];
  const now = new Date();

  return (
    <div className="space-y-1">
      <div className="grid grid-cols-7 gap-0.5">
        {weekDays.map((d) => (
          <div key={d} className="text-center text-[8px] text-gray-400 font-medium py-0.5">{d}</div>
        ))}
      </div>
      <div className="grid grid-cols-7 gap-0.5">
        {Array.from({ length: firstDayOfWeek }).map((_, i) => <div key={`e-${i}`} />)}
        {Array.from({ length: daysInMonth }).map((_, i) => {
          const day = i + 1;
          const dateStr = `${year}-${String(month).padStart(2, '0')}-${String(day).padStart(2, '0')}`;
          const count = dateCountMap[dateStr] || 0;
          const isToday = now.getFullYear() === year && now.getMonth() + 1 === month && now.getDate() === day;

          return (
            <div key={day} title={`${dateStr}: ${count} 个弃用`}
              className={`relative aspect-square rounded border flex flex-col items-center justify-center transition-all duration-150 ${getColorClass(count)} ${isToday ? 'ring-1 ring-red-400 ring-offset-1' : ''}`}>
              <span className="text-[8px] font-medium leading-none">{day}</span>
              {count > 0 && <span className="text-[7px] font-bold leading-none mt-0.5">{count}</span>}
            </div>
          );
        })}
      </div>
      <div className="flex items-center justify-end gap-0.5 mt-0.5">
        <span className="text-[7px] text-gray-400">少</span>
        <div className="w-2.5 h-2.5 rounded-sm bg-gray-50 border border-gray-100" />
        <div className="w-2.5 h-2.5 rounded-sm bg-red-100 border border-red-100" />
        <div className="w-2.5 h-2.5 rounded-sm bg-red-200 border border-red-200" />
        <div className="w-2.5 h-2.5 rounded-sm bg-red-400 border border-red-300" />
        <div className="w-2.5 h-2.5 rounded-sm bg-red-500 border border-red-400" />
        <span className="text-[7px] text-gray-400">多</span>
      </div>
    </div>
  );
}

/* ═══════════════════════════════════════════════════════════
   Custom Tooltip
═══════════════════════════════════════════════════════════ */
function CustomPieTooltip({ active, payload }) {
  if (!active || !payload?.length) return null;
  return (
    <div className="rounded-xl border border-white/50 bg-white/90 backdrop-blur-md shadow-lg px-3 py-2">
      <p className="text-xs font-semibold" style={{ color: payload[0].payload.fill }}>{payload[0].name}: {payload[0].value}</p>
    </div>
  );
}

/* ═══════════════════════════════════════════════════════════
   Main Page
═══════════════════════════════════════════════════════════ */
export default function OpsEnvironmentPage() {
  const token = useStore((s) => s.token);

  // List state
  const [items, setItems] = useState([]);
  const [total, setTotal] = useState(0);
  const [page, setPage] = useState(1);
  const [pageSize] = useState(15);
  const [search, setSearch] = useState('');
  const [statusFilter, setStatusFilter] = useState('all');
  const [regionFilter, setRegionFilter] = useState('');
  const [loading, setLoading] = useState(false);

  // Stats state
  const [stats, setStats] = useState({ status_counts: [], region_counts: [], total_nodes: 0 });

  // Calendar state
  const [calendarView, setCalendarView] = useState('month');
  const [calendarYear, setCalendarYear] = useState(new Date().getFullYear());
  const [calendarMonth, setCalendarMonth] = useState(new Date().getMonth() + 1);
  const [calendarData, setCalendarData] = useState([]);

  // Regions
  const [regions, setRegions] = useState([]);

  // TOP10 state
  const [topCustomers, setTopCustomers] = useState([]);
  const [topNodes, setTopNodes] = useState([]);

  // Sync state
  const [syncing, setSyncing] = useState(false);

  const headers = useMemo(() => ({ Authorization: `Bearer ${token}` }), [token]);

  // Fetch list
  const fetchList = useCallback(async () => {
    setLoading(true);
    try {
      const params = new URLSearchParams({
        page: String(page),
        page_size: String(pageSize),
        status: statusFilter,
      });
      if (search) params.set('search', search);
      if (regionFilter) params.set('region', regionFilter);

      const res = await fetch(`${API_BASE}/ops-env/list?${params}`, { headers });
      const json = await res.json();
      if (json.code === 0) {
        setItems(json.data.items || []);
        setTotal(json.data.total || 0);
      }
    } catch (err) {
      console.error('fetch ops-env list error:', err);
    } finally {
      setLoading(false);
    }
  }, [page, pageSize, search, statusFilter, regionFilter, headers]);

  // Fetch stats
  const fetchStats = useCallback(async () => {
    try {
      const res = await fetch(`${API_BASE}/ops-env/stats`, { headers });
      const json = await res.json();
      if (json.code === 0) {
        setStats(json.data);
      }
    } catch (err) {
      console.error('fetch ops-env stats error:', err);
    }
  }, [headers]);

  // Fetch calendar
  const fetchCalendar = useCallback(async () => {
    try {
      const params = new URLSearchParams({
        year: String(calendarYear),
        month: String(calendarMonth),
        view: calendarView,
      });
      const res = await fetch(`${API_BASE}/ops-env/calendar?${params}`, { headers });
      const json = await res.json();
      if (json.code === 0) {
        setCalendarData(json.data.counts || []);
      }
    } catch (err) {
      console.error('fetch ops-env calendar error:', err);
    }
  }, [calendarYear, calendarMonth, calendarView, headers]);

  // Fetch regions
  const fetchRegions = useCallback(async () => {
    try {
      const res = await fetch(`${API_BASE}/ops-env/regions`, { headers });
      const json = await res.json();
      if (json.code === 0) {
        setRegions(json.data || []);
      }
    } catch (err) {
      console.error('fetch ops-env regions error:', err);
    }
  }, [headers]);

  // Fetch TOP10 customers
  const fetchTopCustomers = useCallback(async () => {
    try {
      const res = await fetch(`${API_BASE}/ops-env/top-customers`, { headers });
      const json = await res.json();
      if (json.code === 0) {
        setTopCustomers(json.data || []);
      }
    } catch (err) {
      console.error('fetch top customers error:', err);
    }
  }, [headers]);

  // Fetch TOP10 nodes
  const fetchTopNodes = useCallback(async () => {
    try {
      const res = await fetch(`${API_BASE}/ops-env/top-nodes`, { headers });
      const json = await res.json();
      if (json.code === 0) {
        setTopNodes(json.data || []);
      }
    } catch (err) {
      console.error('fetch top nodes error:', err);
    }
  }, [headers]);

  // Sync
  const handleSync = async () => {
    setSyncing(true);
    try {
      const res = await fetch(`${API_BASE}/ops-env/sync`, { method: 'POST', headers });
      const json = await res.json();
      if (json.code === 0) {
        toast.success(json.message || '同步任务已启动');
      } else {
        toast.error(json.message || '同步失败');
      }
    } catch (err) {
      toast.error('同步请求失败');
    } finally {
      setSyncing(false);
    }
  };

  useEffect(() => { fetchList(); }, [fetchList]);
  useEffect(() => { fetchStats(); fetchRegions(); fetchTopCustomers(); fetchTopNodes(); }, [fetchStats, fetchRegions, fetchTopCustomers, fetchTopNodes]);
  useEffect(() => { fetchCalendar(); }, [fetchCalendar]);

  // Search with debounce
  const [searchInput, setSearchInput] = useState('');
  useEffect(() => {
    const timer = setTimeout(() => {
      setSearch(searchInput);
      setPage(1);
    }, 400);
    return () => clearTimeout(timer);
  }, [searchInput]);

  // Status tabs
  const statusTabs = [
    { value: 'all', label: '全部', icon: Layers },
    { value: 'in_progress', label: '维保中', icon: Shield },
    { value: 'done', label: '过保', icon: AlertTriangle },
    { value: 'discarded', label: '已弃用', icon: XCircle },
    { value: 'pending', label: '代建转运', icon: Truck },
  ];

  // Compute status counts from stats
  const statusCountMap = useMemo(() => {
    const m = { all: 0, in_progress: 0, done: 0, discarded: 0, pending: 0 };
    (stats.status_counts || []).forEach(s => {
      if (s.status === 'In Progress' || s.status === '正在进行') m.in_progress = (m.in_progress || 0) + s.count;
      else if (s.status === 'Done' || s.status === '已完成') m.done = (m.done || 0) + s.count;
      else if (s.status === 'Discarded' || s.status === '已弃用') m.discarded = (m.discarded || 0) + s.count;
      else if (s.status === '待办') m.pending = (m.pending || 0) + s.count;
      m.all += s.count;
    });
    return m;
  }, [stats]);

  // Region pie data
  const regionPieData = useMemo(() => {
    return (stats.region_counts || []).map((r, idx) => ({
      name: r.region,
      value: r.count,
      fill: COLORS[idx % COLORS.length],
    }));
  }, [stats]);

  const totalPages = Math.ceil(total / pageSize);

  // Navigate calendar months/years
  const handleCalendarPrev = () => {
    if (calendarView === 'month') {
      if (calendarMonth === 1) {
        setCalendarMonth(12);
        setCalendarYear(calendarYear - 1);
      } else {
        setCalendarMonth(calendarMonth - 1);
      }
    } else {
      setCalendarYear(calendarYear - 1);
    }
  };
  const handleCalendarNext = () => {
    if (calendarView === 'month') {
      if (calendarMonth === 12) {
        setCalendarMonth(1);
        setCalendarYear(calendarYear + 1);
      } else {
        setCalendarMonth(calendarMonth + 1);
      }
    } else {
      setCalendarYear(calendarYear + 1);
    }
  };

  // TOP10 customers bar data
  const topCustomersData = useMemo(() => {
    return (topCustomers || []).map(c => ({
      name: c.customer_name?.length > 8 ? c.customer_name.slice(0, 8) + '...' : c.customer_name,
      fullName: c.customer_name,
      count: c.count,
    }));
  }, [topCustomers]);

  return (
    <div className="h-full overflow-y-auto p-4 space-y-3" style={{ background: 'linear-gradient(135deg, #f5f3ff 0%, #eef2ff 50%, #f0fdf4 100%)' }}>
      {/* ── Top Stats Cards ── */}
      <div className="grid grid-cols-2 md:grid-cols-6 gap-2">
        <GlassCard icon={Monitor} title="环境总数" subtitle="所有运维环境">
          <p className="text-xl font-bold text-gray-800">{statusCountMap.all}</p>
        </GlassCard>
        <GlassCard icon={Shield} title="维保中" subtitle="正在维保">
          <p className="text-xl font-bold text-green-600">{statusCountMap.in_progress}</p>
        </GlassCard>
        <GlassCard icon={AlertTriangle} title="过保" subtitle="维保已过期">
          <p className="text-xl font-bold text-amber-600">{statusCountMap.done}</p>
        </GlassCard>
        <GlassCard icon={XCircle} title="已弃用" subtitle="环境已弃用">
          <p className="text-xl font-bold text-red-600">{statusCountMap.discarded}</p>
        </GlassCard>
        <GlassCard icon={Truck} title="代建转运" subtitle="待办环境">
          <p className="text-xl font-bold text-blue-600">{statusCountMap.pending}</p>
        </GlassCard>
        <GlassCard icon={HardDrive} title="总节点数" subtitle="所有环境节点">
          <p className="text-xl font-bold text-indigo-600">{stats.total_nodes?.toLocaleString() || 0}</p>
        </GlassCard>
      </div>

      {/* ── Charts Row: 4 columns ── */}
      <div className="grid grid-cols-1 lg:grid-cols-4 gap-3">
        {/* Region Distribution - compact */}
        <GlassCard icon={MapPin} title="区域分布" subtitle="按运维区域">
          {regionPieData.length > 0 ? (
            <div>
              <div className="h-32">
                <ResponsiveContainer width="100%" height="100%">
                  <PieChart>
                    <Pie data={regionPieData} dataKey="value" nameKey="name" cx="50%" cy="50%"
                      outerRadius={50} innerRadius={22} paddingAngle={2}>
                      {regionPieData.map((entry, idx) => (
                        <Cell key={idx} fill={entry.fill} />
                      ))}
                    </Pie>
                    <Tooltip content={<CustomPieTooltip />} />
                  </PieChart>
                </ResponsiveContainer>
              </div>
              <div className="flex flex-wrap gap-1 justify-center">
                {regionPieData.slice(0, 5).map((r, idx) => (
                  <div key={idx} className="flex items-center gap-0.5 text-[9px] text-gray-600">
                    <div className="w-1.5 h-1.5 rounded-full" style={{ background: r.fill }} />
                    {r.name}({r.value})
                  </div>
                ))}
              </div>
            </div>
          ) : (
            <div className="h-32 flex items-center justify-center text-xs text-gray-400">暂无数据</div>
          )}
        </GlassCard>

        {/* Calendar View - compact */}
        <GlassCard icon={CalendarDays} title="弃用日历" subtitle="弃用趋势追踪">
          <div className="flex items-center justify-between mb-2">
            <div className="flex items-center gap-1">
              <button onClick={handleCalendarPrev} className="p-1 rounded hover:bg-gray-100 transition-colors">
                <ChevronLeft className="w-3 h-3 text-gray-500" />
              </button>
              <span className="text-[10px] font-semibold text-gray-700">
                {calendarView === 'month' ? `${calendarYear}年${calendarMonth}月` : `${calendarYear}年`}
              </span>
              <button onClick={handleCalendarNext} className="p-1 rounded hover:bg-gray-100 transition-colors">
                <ChevronRight className="w-3 h-3 text-gray-500" />
              </button>
            </div>
            <CalendarViewSelector view={calendarView} onChange={setCalendarView} />
          </div>
          <DiscardCalendarView data={calendarData} view={calendarView} year={calendarYear} month={calendarMonth} />
        </GlassCard>

        {/* TOP10 Customers */}
        <GlassCard icon={Users} title="TOP10 客户" subtitle="环境数最多">
          {topCustomersData.length > 0 ? (
            <div className="h-40">
              <ResponsiveContainer width="100%" height="100%">
                <BarChart data={topCustomersData} layout="vertical" margin={{ top: 0, right: 10, left: 0, bottom: 0 }}>
                  <XAxis type="number" tick={{ fontSize: 9 }} allowDecimals={false} />
                  <YAxis type="category" dataKey="name" tick={{ fontSize: 9 }} width={60} />
                  <Tooltip
                    contentStyle={{ borderRadius: '8px', border: '1px solid rgba(255,255,255,0.5)', background: 'rgba(255,255,255,0.95)', fontSize: 11 }}
                    formatter={(val, name, props) => [`${val} 个环境`, props.payload.fullName]}
                  />
                  <Bar dataKey="count" fill="#6366f1" radius={[0, 3, 3, 0]} />
                </BarChart>
              </ResponsiveContainer>
            </div>
          ) : (
            <div className="h-40 flex items-center justify-center text-xs text-gray-400">暂无数据</div>
          )}
        </GlassCard>

        {/* TOP10 Nodes */}
        <GlassCard icon={Cpu} title="TOP10 节点数" subtitle="节点数最多的环境">
          {topNodes.length > 0 ? (
            <div className="space-y-1.5 max-h-44 overflow-y-auto" style={{ scrollbarWidth: 'none' }}>
              {topNodes.map((item, idx) => (
                <div key={idx} className="flex items-center gap-2">
                  <span className={`flex-shrink-0 w-4 h-4 rounded-full flex items-center justify-center text-[9px] font-bold ${
                    idx < 3 ? 'bg-indigo-500 text-white' : 'bg-gray-200 text-gray-600'
                  }`}>{idx + 1}</span>
                  <div className="flex-1 min-w-0">
                    <p className="text-[10px] font-medium text-gray-800 truncate" title={item.customer_name}>{item.customer_name}</p>
                    <p className="text-[9px] text-gray-400 truncate" title={item.cse_name}>{item.cse_name}</p>
                  </div>
                  <span className="flex-shrink-0 text-[10px] font-bold text-indigo-600">{item.node_count}</span>
                </div>
              ))}
            </div>
          ) : (
            <div className="h-40 flex items-center justify-center text-xs text-gray-400">暂无数据</div>
          )}
        </GlassCard>
      </div>

      {/* ── Toolbar: Status Tabs + Search + Sync ── */}
      <div className="flex flex-col sm:flex-row items-start sm:items-center justify-between gap-2">
        <div className="flex gap-1 p-0.5 bg-white/50 backdrop-blur-sm rounded-xl border border-gray-200/50">
          {statusTabs.map((tab) => {
            const Icon = tab.icon;
            const isActive = statusFilter === tab.value;
            const count = statusCountMap[tab.value];
            return (
              <button key={tab.value} onClick={() => { setStatusFilter(tab.value); setPage(1); }}
                className={`flex items-center gap-1 px-2.5 py-1.5 rounded-lg text-xs font-medium transition-all duration-200 ${
                  isActive ? 'bg-primary-600 text-white shadow-md shadow-primary-200' : 'text-gray-600 hover:bg-gray-100/80'
                }`}>
                <Icon className="w-3.5 h-3.5" /> {tab.label}
                <span className={`ml-0.5 px-1.5 py-0.5 rounded-full text-[10px] ${isActive ? 'bg-white/20 text-white' : 'bg-gray-200/60 text-gray-500'}`}>{count}</span>
              </button>
            );
          })}
        </div>

        <div className="flex items-center gap-2">
          {/* Region filter */}
          {regions.length > 0 && (
            <select value={regionFilter} onChange={(e) => { setRegionFilter(e.target.value); setPage(1); }}
              className="text-xs px-2 py-1.5 rounded-lg border border-gray-200/50 bg-white/70 backdrop-blur-sm text-gray-600 focus:outline-none focus:ring-2 focus:ring-primary-300">
              <option value="">全部区域</option>
              {regions.map((r) => <option key={r} value={r}>{r}</option>)}
            </select>
          )}

          {/* Search */}
          <div className="relative">
            <Search className="absolute left-2.5 top-1/2 -translate-y-1/2 w-3.5 h-3.5 text-gray-400" />
            <input type="text" placeholder="搜索项目 / 客户名称..."
              value={searchInput} onChange={(e) => setSearchInput(e.target.value)}
              className="pl-8 pr-3 py-1.5 w-56 text-xs rounded-lg border border-gray-200/50 bg-white/70 backdrop-blur-sm focus:outline-none focus:ring-2 focus:ring-primary-300 placeholder-gray-400" />
          </div>

          {/* Sync */}
          <button onClick={handleSync} disabled={syncing}
            className="flex items-center gap-1.5 px-3 py-1.5 text-xs font-medium rounded-lg bg-primary-600 text-white hover:bg-primary-700 transition-colors shadow-sm disabled:opacity-50">
            <RefreshCw className={`w-3.5 h-3.5 ${syncing ? 'animate-spin' : ''}`} />
            同步Jira
          </button>
        </div>
      </div>

      {/* ── Data Table ── */}
      <GlassCard>
        <div className="overflow-x-auto">
          <table className="w-full text-xs">
            <thead>
              <tr className="border-b border-gray-200/50">
                <th className="text-left py-2 px-2 font-semibold text-gray-600">CSE编号</th>
                <th className="text-left py-2 px-2 font-semibold text-gray-600">客户名称</th>
                <th className="text-left py-2 px-2 font-semibold text-gray-600">项目名称</th>
                <th className="text-left py-2 px-2 font-semibold text-gray-600">CSE</th>
                <th className="text-left py-2 px-2 font-semibold text-gray-600">状态</th>
                <th className="text-left py-2 px-2 font-semibold text-gray-600">区域</th>
                <th className="text-left py-2 px-2 font-semibold text-gray-600">版本</th>
                <th className="text-left py-2 px-2 font-semibold text-gray-600">节点</th>
                <th className="text-left py-2 px-2 font-semibold text-gray-600">SLA</th>
                <th className="text-left py-2 px-2 font-semibold text-gray-600">维保结束</th>
                <th className="text-left py-2 px-2 font-semibold text-gray-600">等级</th>
              </tr>
            </thead>
            <tbody>
              {loading ? (
                <tr><td colSpan={11} className="text-center py-10 text-gray-400">
                  <RefreshCw className="w-5 h-5 animate-spin mx-auto mb-2" />加载中...
                </td></tr>
              ) : items.length === 0 ? (
                <tr><td colSpan={11} className="text-center py-10 text-gray-400">
                  <Server className="w-6 h-6 mx-auto mb-2 text-gray-300" />暂无数据，请点击「同步Jira」拉取数据
                </td></tr>
              ) : items.map((item) => (
                <tr key={item.id} className="border-b border-gray-100/50 hover:bg-white/50 transition-colors">
                  <td className="py-2 px-2 font-mono text-indigo-600 text-[11px]" title={item.cse_key}>{item.cse_key || '-'}</td>
                  <td className="py-2 px-2 font-medium text-gray-800 max-w-[130px] truncate" title={item.customer_name}>{item.customer_name || '-'}</td>
                  <td className="py-2 px-2 text-gray-700 max-w-[150px] truncate" title={item.project_name}>{item.project_name || '-'}</td>
                  <td className="py-2 px-2 text-gray-600 max-w-[120px] truncate" title={item.cse_name}>{item.cse_name || '-'}</td>
                  <td className="py-2 px-2"><StatusBadge status={item.status} /></td>
                  <td className="py-2 px-2 text-gray-600">{item.ops_region || '-'}</td>
                  <td className="py-2 px-2 text-gray-600">{item.version || '-'}</td>
                  <td className="py-2 px-2 text-gray-600">{item.node_count || '-'}</td>
                  <td className="py-2 px-2 text-gray-600">{item.sla || '-'}</td>
                  <td className="py-2 px-2 text-gray-600">{item.maintain_end || '-'}</td>
                  <td className="py-2 px-2 text-gray-600">{item.customer_level || '-'}</td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>

        {/* Pagination */}
        {total > 0 && (
          <div className="flex items-center justify-between mt-3 pt-2 border-t border-gray-100/50">
            <span className="text-[10px] text-gray-500">共 {total} 条记录</span>
            <div className="flex items-center gap-1">
              <button onClick={() => setPage(Math.max(1, page - 1))} disabled={page <= 1}
                className="p-1 rounded hover:bg-gray-100 disabled:opacity-30 transition-colors">
                <ChevronLeft className="w-3.5 h-3.5 text-gray-500" />
              </button>
              <span className="text-[10px] text-gray-600 px-1.5">{page} / {totalPages}</span>
              <button onClick={() => setPage(Math.min(totalPages, page + 1))} disabled={page >= totalPages}
                className="p-1 rounded hover:bg-gray-100 disabled:opacity-30 transition-colors">
                <ChevronRight className="w-3.5 h-3.5 text-gray-500" />
              </button>
            </div>
          </div>
        )}
      </GlassCard>
    </div>
  );
}
