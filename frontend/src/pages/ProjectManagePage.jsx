import React, { useState, useEffect, useCallback, useRef, useMemo } from 'react';
import ReactDOM from 'react-dom';
import {
  LineChart, Line, XAxis, YAxis, CartesianGrid, Tooltip, ResponsiveContainer,
  PieChart, Pie, Cell,
  BarChart, Bar,
} from 'recharts';
import { ComposableMap, Geographies, Geography, ZoomableGroup } from 'react-simple-maps';
import {
  FolderKanban, TrendingUp, Calendar, RefreshCw, ChevronDown, ChevronLeft, ChevronRight,
  MapPin, Users, Building2, Layers, Activity, Award, CalendarDays, CalendarRange, BarChart3,
} from 'lucide-react';
import useStore from '../store/useStore';
import toast from 'react-hot-toast';

const API_BASE = import.meta.env.VITE_API_BASE || '/api';
const CHINA_GEO_URL = '/china.json';

// Color palette for charts
const COLORS = ['#6366f1', '#8b5cf6', '#a78bfa', '#c4b5fd', '#e0e7ff', '#818cf8', '#7c3aed', '#5b21b6', '#4f46e5', '#4338ca'];

// Province name mapping: short name -> GeoJSON full name
const PROVINCE_MAP = {
  '北京': '北京市', '上海': '上海市', '天津': '天津市', '重庆': '重庆市',
  '广东': '广东省', '江苏': '江苏省', '浙江': '浙江省', '山东': '山东省',
  '河南': '河南省', '四川': '四川省', '湖北': '湖北省', '湖南': '湖南省',
  '河北': '河北省', '福建': '福建省', '安徽': '安徽省', '辽宁': '辽宁省',
  '陕西': '陕西省', '陕西省': '陕西省', '江西': '江西省', '广西': '广西壮族自治区',
  '云南': '云南省', '贵州': '贵州省', '山西': '山西省', '吉林': '吉林省',
  '黑龙江': '黑龙江省', '甘肃': '甘肃省', '内蒙古': '内蒙古自治区',
  '新疆': '新疆维吾尔自治区', '宁夏': '宁夏回族自治区', '海南': '海南省',
  '西藏': '西藏自治区', '青海': '青海省', '台湾': '台湾省',
  '香港': '香港特别行政区', '澳门': '澳门特别行政区',
};

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
   Date Picker Calendar (from WorktimePage pattern)
═══════════════════════════════════════════════════════════ */
function DatePickerCalendar({ value, onChange, label }) {
  const [viewDate, setViewDate] = useState(() => value ? new Date(value) : new Date());
  const year = viewDate.getFullYear();
  const month = viewDate.getMonth();
  const daysInMonth = new Date(year, month + 1, 0).getDate();
  const firstDayOfWeek = new Date(year, month, 1).getDay();
  const weekDays = ['日', '一', '二', '三', '四', '五', '六'];

  const selectedDay = value && value.startsWith(`${year}-${String(month + 1).padStart(2, '0')}`)
    ? parseInt(value.split('-')[2], 10) : null;

  const handleSelect = (day) => {
    const dateStr = `${year}-${String(month + 1).padStart(2, '0')}-${String(day).padStart(2, '0')}`;
    onChange(dateStr);
  };

  return (
    <div className="rounded-2xl border border-white/50 shadow-xl bg-white/80 backdrop-blur-xl p-4 w-72">
      <p className="text-xs text-gray-400 mb-2 font-medium">{label}</p>
      <div className="flex items-center justify-between mb-3">
        <button onClick={() => setViewDate(new Date(year, month - 1, 1))} className="p-1.5 rounded-lg hover:bg-gray-100 transition-colors">
          <ChevronLeft className="w-4 h-4 text-gray-500" />
        </button>
        <span className="text-sm font-semibold text-gray-700">{year}年{month + 1}月</span>
        <button onClick={() => setViewDate(new Date(year, month + 1, 1))} className="p-1.5 rounded-lg hover:bg-gray-100 transition-colors">
          <ChevronRight className="w-4 h-4 text-gray-500" />
        </button>
      </div>
      <div className="grid grid-cols-7 gap-1 mb-1">
        {weekDays.map((d) => (
          <div key={d} className="text-center text-xs text-gray-400 font-medium py-1">{d}</div>
        ))}
      </div>
      <div className="grid grid-cols-7 gap-1">
        {Array.from({ length: firstDayOfWeek }).map((_, i) => <div key={`e-${i}`} />)}
        {Array.from({ length: daysInMonth }).map((_, i) => {
          const day = i + 1;
          const isSelected = day === selectedDay;
          const isToday = (() => { const t = new Date(); return t.getFullYear() === year && t.getMonth() === month && t.getDate() === day; })();
          return (
            <button key={day} onClick={() => handleSelect(day)}
              className={`w-8 h-8 rounded-lg text-sm font-medium transition-all duration-150 ${
                isSelected ? 'bg-primary-600 text-white shadow-md' : isToday ? 'bg-primary-50 text-primary-700 border border-primary-200' : 'text-gray-600 hover:bg-gray-100'
              }`}>{day}</button>
          );
        })}
      </div>
      {value && <p className="mt-2 text-xs text-center text-primary-600 font-medium">已选: {value}</p>}
    </div>
  );
}

/* ═══════════════════════════════════════════════════════════
   Period Selector (matching WorktimePage style)
═══════════════════════════════════════════════════════════ */
function PeriodSelector({ period, onChange, customRange, onCustomRangeChange }) {
  const [showCustom, setShowCustom] = useState(false);
  const [tempStart, setTempStart] = useState(customRange?.start || '');
  const [tempEnd, setTempEnd] = useState(customRange?.end || '');
  const popoverRef = useRef(null);
  const btnRef = useRef(null);
  const [popoverPos, setPopoverPos] = useState({ top: 0, left: 0 });

  const periods = [
    { value: 'month', label: '本月', icon: Calendar },
    { value: 'quarter', label: '本季度', icon: BarChart3 },
    { value: 'year', label: '本年度', icon: TrendingUp },
  ];

  useEffect(() => {
    if (showCustom && btnRef.current) {
      const rect = btnRef.current.getBoundingClientRect();
      const popoverWidth = 640;
      let left = rect.left + rect.width / 2 - popoverWidth / 2;
      if (left < 12) left = 12;
      if (left + popoverWidth > window.innerWidth - 12) left = window.innerWidth - popoverWidth - 12;
      setPopoverPos({ top: rect.bottom + 8, left });
    }
  }, [showCustom]);

  useEffect(() => {
    const handler = (e) => {
      if (popoverRef.current && !popoverRef.current.contains(e.target) && btnRef.current && !btnRef.current.contains(e.target)) {
        setShowCustom(false);
      }
    };
    if (showCustom) document.addEventListener('mousedown', handler);
    return () => document.removeEventListener('mousedown', handler);
  }, [showCustom]);

  useEffect(() => {
    const handler = (e) => { if (e.key === 'Escape') setShowCustom(false); };
    if (showCustom) document.addEventListener('keydown', handler);
    return () => document.removeEventListener('keydown', handler);
  }, [showCustom]);

  const handleApplyCustom = () => {
    if (tempStart && tempEnd) {
      onCustomRangeChange(tempStart, tempEnd);
      setShowCustom(false);
    } else {
      toast.error('请选择起止日期');
    }
  };

  const popoverContent = showCustom ? ReactDOM.createPortal(
    <div ref={popoverRef} className="rounded-2xl border border-gray-200 shadow-2xl bg-white p-5"
      style={{ position: 'fixed', top: `${popoverPos.top}px`, left: `${popoverPos.left}px`, zIndex: 99999, minWidth: '620px' }}>
      <p className="text-sm font-semibold text-gray-700 mb-3 flex items-center gap-2">
        <CalendarRange className="w-4 h-4 text-primary-500" /> 自定义统计周期
      </p>
      <div className="flex gap-4">
        <DatePickerCalendar value={tempStart} onChange={setTempStart} label="开始日期" />
        <DatePickerCalendar value={tempEnd} onChange={setTempEnd} label="结束日期" />
      </div>
      <div className="flex items-center justify-between mt-4 pt-3 border-t border-gray-100">
        <div className="text-xs text-gray-400">{tempStart && tempEnd ? `${tempStart} 至 ${tempEnd}` : '请选择日期范围'}</div>
        <div className="flex gap-2">
          <button onClick={() => setShowCustom(false)} className="px-3 py-1.5 rounded-lg text-sm text-gray-500 hover:bg-gray-100">取消</button>
          <button onClick={handleApplyCustom} disabled={!tempStart || !tempEnd}
            className="px-4 py-1.5 rounded-lg text-sm font-medium bg-primary-600 text-white hover:bg-primary-700 disabled:opacity-50 shadow-sm">确定</button>
        </div>
      </div>
    </div>, document.body
  ) : null;

  return (
    <div className="flex gap-2 p-1 bg-white/50 backdrop-blur-sm rounded-xl border border-gray-200/50">
      {periods.map((p) => {
        const Icon = p.icon;
        const isActive = period === p.value;
        return (
          <button key={p.value} onClick={() => { onChange(p.value); setShowCustom(false); }}
            className={`flex items-center gap-1.5 px-3 py-1.5 rounded-lg text-sm font-medium transition-all duration-200 ${
              isActive ? 'bg-primary-600 text-white shadow-md shadow-primary-200' : 'text-gray-600 hover:bg-gray-100/80'
            }`}>
            <Icon className="w-3.5 h-3.5" /> {p.label}
          </button>
        );
      })}
      <button ref={btnRef} onClick={() => setShowCustom(!showCustom)}
        className={`flex items-center gap-1.5 px-3 py-1.5 rounded-lg text-sm font-medium transition-all duration-200 ${
          period === 'custom' ? 'bg-primary-600 text-white shadow-md shadow-primary-200' : 'text-gray-600 hover:bg-gray-100/80'
        }`}>
        <CalendarRange className="w-3.5 h-3.5" /> 自定义
      </button>
      {popoverContent}
    </div>
  );
}

/* ═══════════════════════════════════════════════════════════
   Custom Tooltip
═══════════════════════════════════════════════════════════ */
function CustomTooltip({ active, payload, label }) {
  if (!active || !payload?.length) return null;
  return (
    <div className="rounded-xl border border-white/50 bg-white/90 backdrop-blur-md shadow-lg px-3 py-2">
      <p className="text-xs font-medium text-gray-600 mb-1">{label}</p>
      {payload.map((entry, idx) => (
        <p key={idx} className="text-sm font-semibold" style={{ color: entry.color }}>{entry.name}: {entry.value}</p>
      ))}
    </div>
  );
}

/* ═══════════════════════════════════════════════════════════
   China Map Component
═══════════════════════════════════════════════════════════ */
function ChinaMap({ provinceData }) {
  const [tooltipContent, setTooltipContent] = useState('');
  const [tooltipPos, setTooltipPos] = useState({ x: 0, y: 0 });

  // Build lookup: geoJSON name -> count
  const provinceCountMap = useMemo(() => {
    const map = {};
    (provinceData || []).forEach(item => {
      const fullName = PROVINCE_MAP[item.name] || item.name;
      map[fullName] = (map[fullName] || 0) + item.count;
    });
    return map;
  }, [provinceData]);

  const maxCount = useMemo(() => Math.max(...Object.values(provinceCountMap), 1), [provinceCountMap]);

  return (
    <div className="relative w-full h-full">
      <ComposableMap
        projection="geoMercator"
        projectionConfig={{ center: [104, 35], scale: 580 }}
        width={500}
        height={420}
        style={{ width: '100%', height: '100%' }}
      >
        <Geographies geography={CHINA_GEO_URL}>
          {({ geographies }) =>
            geographies.map((geo) => {
              const name = geo.properties.name;
              const count = provinceCountMap[name] || 0;
              const intensity = count > 0 ? 0.3 + (count / maxCount) * 0.7 : 0;
              const fillColor = count > 0 ? `rgba(99, 102, 241, ${intensity})` : '#f1f5f9';
              return (
                <Geography
                  key={geo.rpisvgGeographyId || geo.properties.adcode}
                  geography={geo}
                  fill={fillColor}
                  stroke="#e2e8f0"
                  strokeWidth={0.5}
                  style={{
                    default: { outline: 'none' },
                    hover: { fill: count > 0 ? '#6366f1' : '#e2e8f0', outline: 'none', cursor: count > 0 ? 'pointer' : 'default' },
                    pressed: { outline: 'none' },
                  }}
                  onMouseEnter={(e) => {
                    const shortName = Object.entries(PROVINCE_MAP).find(([, v]) => v === name)?.[0] || name;
                    setTooltipContent(count > 0 ? `${shortName}: ${count} 个项目` : `${shortName}: 暂无项目`);
                    setTooltipPos({ x: e.clientX, y: e.clientY });
                  }}
                  onMouseMove={(e) => setTooltipPos({ x: e.clientX, y: e.clientY })}
                  onMouseLeave={() => setTooltipContent('')}
                />
              );
            })
          }
        </Geographies>
      </ComposableMap>
      {tooltipContent && ReactDOM.createPortal(
        <div className="fixed rounded-lg bg-gray-900/90 text-white text-xs px-3 py-1.5 pointer-events-none shadow-lg z-[99999]"
          style={{ top: tooltipPos.y - 36, left: tooltipPos.x + 10 }}>
          {tooltipContent}
        </div>, document.body
      )}
    </div>
  );
}

/* ═══════════════════════════════════════════════════════════
   Helper: get period dates
═══════════════════════════════════════════════════════════ */
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
    case 'year':
      start = new Date(now.getFullYear(), 0, 1).toISOString().split('T')[0];
      break;
    default:
      start = new Date(now.getFullYear(), now.getMonth(), 1).toISOString().split('T')[0];
  }
  return { start, end };
}

/* ═══════════════════════════════════════════════════════════
   Main Page Component
═══════════════════════════════════════════════════════════ */
export default function ProjectManagePage() {
  const token = useStore((s) => s.token);
  const [stats, setStats] = useState(null);
  const [loading, setLoading] = useState(true);
  const [syncing, setSyncing] = useState(false);
  const [period, setPeriod] = useState('month');
  const [customRange, setCustomRange] = useState({ start: '', end: '' });

  const fetchStats = useCallback(async () => {
    setLoading(true);
    try {
      let startDate, endDate;
      if (period === 'custom' && customRange.start && customRange.end) {
        startDate = customRange.start;
        endDate = customRange.end;
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
      if (data.code === 0) setStats(data.data);
    } catch (e) {
      console.error('Failed to fetch project stats:', e);
    } finally {
      setLoading(false);
    }
  }, [token, period, customRange]);

  useEffect(() => { fetchStats(); }, [fetchStats]);

  const handlePeriodChange = (p) => setPeriod(p);
  const handleCustomRangeChange = (start, end) => {
    setCustomRange({ start, end });
    setPeriod('custom');
  };

  const handleSync = async () => {
    setSyncing(true);
    try {
      await fetch(`${API_BASE}/projects/sync`, { method: 'POST', headers: { Authorization: `Bearer ${token}` } });
      setTimeout(() => { fetchStats(); setSyncing(false); }, 3000);
    } catch (e) { setSyncing(false); }
  };

  // Determine period granularity for the "period new" grid
  const periodLabel = useMemo(() => {
    if (period === 'month') return '每日';
    if (period === 'quarter') return '每月';
    if (period === 'year') return '每月';
    return '每日';
  }, [period]);

  // Adapt month_daily_projects data: for quarter/year -> aggregate by month
  const periodNewData = useMemo(() => {
    const raw = stats?.month_daily_projects || [];
    if (!raw.length) return [];
    if (period === 'month' || period === 'custom') {
      // Check if custom range <= 60 days -> show daily, else monthly
      if (period === 'custom' && customRange.start && customRange.end) {
        const diffDays = (new Date(customRange.end) - new Date(customRange.start)) / (1000 * 60 * 60 * 24);
        if (diffDays > 60) {
          // Aggregate by month
          const monthMap = {};
          raw.forEach(item => {
            const m = item.date?.slice(0, 7);
            if (m) monthMap[m] = (monthMap[m] || 0) + item.count;
          });
          return Object.entries(monthMap).sort((a, b) => a[0].localeCompare(b[0])).map(([m, c]) => ({ date: m, count: c }));
        }
      }
      return raw;
    }
    // quarter/year -> aggregate by month
    const monthMap = {};
    raw.forEach(item => {
      const m = item.date?.slice(0, 7);
      if (m) monthMap[m] = (monthMap[m] || 0) + item.count;
    });
    return Object.entries(monthMap).sort((a, b) => a[0].localeCompare(b[0])).map(([m, c]) => ({ date: m, count: c }));
  }, [stats, period, customRange]);

  if (loading && !stats) {
    return (
      <div className="flex items-center justify-center h-full">
        <div className="animate-spin w-8 h-8 border-4 border-indigo-500 border-t-transparent rounded-full" />
      </div>
    );
  }

  return (
    <div className="h-full overflow-y-auto p-5 space-y-4" style={{ scrollbarWidth: 'thin' }}>
      {/* Header */}
      <div className="flex flex-wrap items-center justify-between gap-3">
        <div className="flex items-center gap-3">
          <div className="w-9 h-9 rounded-xl bg-gradient-to-br from-indigo-500 to-purple-600 flex items-center justify-center shadow-md">
            <FolderKanban className="w-4.5 h-4.5 text-white" size={18} />
          </div>
          <div>
            <h2 className="text-base font-bold text-gray-800">项目管理看板</h2>
            <p className="text-[10px] text-gray-400">
              Redmine 项目立项数据统计
              {stats?.last_sync_time && ` · 同步: ${stats.last_sync_time}`}
            </p>
          </div>
        </div>
        <div className="flex items-center gap-3">
          <PeriodSelector period={period} onChange={handlePeriodChange} customRange={customRange} onCustomRangeChange={handleCustomRangeChange} />
          <button onClick={handleSync} disabled={syncing}
            className="flex items-center gap-1.5 px-3 py-1.5 rounded-xl bg-gradient-to-r from-indigo-500 to-purple-600 text-white text-xs font-medium shadow-md hover:shadow-lg transition disabled:opacity-60">
            <RefreshCw size={12} className={syncing ? 'animate-spin' : ''} />
            {syncing ? '同步中' : '同步'}
          </button>
        </div>
      </div>

      {/* Row 1: 4 stat cards */}
      <div className="grid grid-cols-2 lg:grid-cols-4 gap-3">
        {[
          { icon: FolderKanban, label: '项目总数', value: stats?.total_projects, color: 'from-indigo-500 to-purple-600' },
          { icon: TrendingUp, label: '周期新增', value: stats?.period_new, color: 'from-blue-500 to-cyan-600' },
          { icon: Activity, label: '近7天新增', value: stats?.week_new, color: 'from-emerald-500 to-teal-600' },
          { icon: Award, label: '年度立项', value: stats?.yoy_comparison?.length ? stats.yoy_comparison[stats.yoy_comparison.length - 1]?.count : '-', color: 'from-orange-500 to-amber-600' },
        ].map((s, i) => (
          <div key={i} className="rounded-2xl border border-white/40 bg-white/70 backdrop-blur-xl shadow-lg p-4 flex items-center gap-3">
            <div className={`w-10 h-10 rounded-xl bg-gradient-to-br ${s.color} flex items-center justify-center shadow-md`}>
              <s.icon className="w-5 h-5 text-white" />
            </div>
            <div>
              <p className="text-[10px] text-gray-400 font-medium">{s.label}</p>
              <p className="text-xl font-bold text-gray-800">{s.value ?? '-'}</p>
            </div>
          </div>
        ))}
      </div>

      {/* Row 2: Period new breakdown + China Map + Recent month trend (3 cols) */}
      <div className="grid grid-cols-1 lg:grid-cols-3 gap-3">
        {/* Period new breakdown */}
        <GlassCard icon={CalendarDays} title={`周期内新增立项（${periodLabel}）`} subtitle={period === 'custom' ? `${customRange.start} 至 ${customRange.end}` : `${getPeriodDates(period).start} 至 ${getPeriodDates(period).end}`}>
          {periodNewData.length === 0 ? (
            <div className="flex flex-col items-center justify-center py-8 text-gray-400">
              <CalendarDays size={32} className="mb-2 opacity-40" />
              <p className="text-xs">暂无数据</p>
            </div>
          ) : (
            <div className="h-52">
              <ResponsiveContainer width="100%" height="100%">
                <BarChart data={periodNewData}>
                  <CartesianGrid strokeDasharray="3 3" stroke="#e5e7eb" />
                  <XAxis dataKey="date" tick={{ fontSize: 9 }} stroke="#9ca3af" interval={period === 'month' ? 2 : 0} angle={period === 'year' ? -30 : 0} />
                  <YAxis tick={{ fontSize: 9 }} stroke="#9ca3af" allowDecimals={false} />
                  <Tooltip content={<CustomTooltip />} />
                  <Bar dataKey="count" name="新增数" radius={[4, 4, 0, 0]} fill="#6366f1" />
                </BarChart>
              </ResponsiveContainer>
            </div>
          )}
        </GlassCard>

        {/* China Map */}
        <GlassCard icon={MapPin} title="全国项目分布" subtitle="按省份点亮（紫色=有项目）">
          <div className="h-52">
            <ChinaMap provinceData={stats?.province_distribution} />
          </div>
        </GlassCard>

        {/* Recent month trend (line) */}
        <GlassCard icon={TrendingUp} title="近一个月立项趋势" subtitle="过去30天每日新增">
          <div className="h-52">
            <ResponsiveContainer width="100%" height="100%">
              <LineChart data={stats?.recent_month_trend || []}>
                <CartesianGrid strokeDasharray="3 3" stroke="#e5e7eb" />
                <XAxis dataKey="date" tick={{ fontSize: 9 }} stroke="#9ca3af" interval={4} />
                <YAxis tick={{ fontSize: 9 }} stroke="#9ca3af" allowDecimals={false} />
                <Tooltip content={<CustomTooltip />} />
                <Line type="monotone" dataKey="count" name="立项数" stroke="#6366f1" strokeWidth={2} dot={{ r: 2 }} activeDot={{ r: 4 }} />
              </LineChart>
            </ResponsiveContainer>
          </div>
        </GlassCard>
      </div>

      {/* Row 3: Monthly trend + YoY + Type distribution (3 cols) */}
      <div className="grid grid-cols-1 lg:grid-cols-3 gap-3">
        {/* Monthly trend line chart */}
        <GlassCard icon={Layers} title="月度立项趋势" subtitle="近12个月">
          <div className="h-48">
            <ResponsiveContainer width="100%" height="100%">
              <LineChart data={stats?.monthly_trend || []}>
                <CartesianGrid strokeDasharray="3 3" stroke="#e5e7eb" />
                <XAxis dataKey="month" tick={{ fontSize: 9 }} stroke="#9ca3af" />
                <YAxis tick={{ fontSize: 9 }} stroke="#9ca3af" />
                <Tooltip content={<CustomTooltip />} />
                <Line type="monotone" dataKey="count" name="立项数" stroke="#8b5cf6" strokeWidth={2} dot={{ fill: '#8b5cf6', r: 3 }} />
              </LineChart>
            </ResponsiveContainer>
          </div>
        </GlassCard>

        {/* Year-over-year bar chart */}
        <GlassCard icon={BarChart3} title="年度环比对比" subtitle="历年立项数量">
          <div className="h-48">
            <ResponsiveContainer width="100%" height="100%">
              <BarChart data={stats?.yoy_comparison || []}>
                <CartesianGrid strokeDasharray="3 3" stroke="#e5e7eb" />
                <XAxis dataKey="year" tick={{ fontSize: 9 }} stroke="#9ca3af" />
                <YAxis tick={{ fontSize: 9 }} stroke="#9ca3af" />
                <Tooltip content={<CustomTooltip />} />
                <Bar dataKey="count" name="立项数" radius={[4, 4, 0, 0]}>
                  {(stats?.yoy_comparison || []).map((_, idx) => (<Cell key={idx} fill={COLORS[idx % COLORS.length]} />))}
                </Bar>
              </BarChart>
            </ResponsiveContainer>
          </div>
        </GlassCard>

        {/* Project type donut */}
        <GlassCard icon={FolderKanban} title="项目类型分布" subtitle="按类型统计">
          <div className="h-48">
            <ResponsiveContainer width="100%" height="100%">
              <PieChart>
                <Pie data={stats?.type_distribution || []} cx="50%" cy="50%" innerRadius={40} outerRadius={65} dataKey="count" nameKey="name"
                  label={({ name, percent }) => `${name} ${(percent * 100).toFixed(0)}%`} labelLine={{ strokeWidth: 1 }}>
                  {(stats?.type_distribution || []).map((_, idx) => (<Cell key={idx} fill={COLORS[idx % COLORS.length]} />))}
                </Pie>
                <Tooltip />
              </PieChart>
            </ResponsiveContainer>
          </div>
        </GlassCard>
      </div>

      {/* Row 4: Status bar + Region donut + TOP5 (3 cols) */}
      <div className="grid grid-cols-1 lg:grid-cols-3 gap-3">
        {/* Project status bar chart */}
        <GlassCard icon={Activity} title="项目状态分布" subtitle="按当前状态统计">
          <div className="h-48">
            <ResponsiveContainer width="100%" height="100%">
              <BarChart data={stats?.status_distribution || []} layout="vertical">
                <CartesianGrid strokeDasharray="3 3" stroke="#e5e7eb" />
                <XAxis type="number" tick={{ fontSize: 9 }} stroke="#9ca3af" />
                <YAxis type="category" dataKey="name" tick={{ fontSize: 9 }} stroke="#9ca3af" width={70} />
                <Tooltip content={<CustomTooltip />} />
                <Bar dataKey="count" name="项目数" radius={[0, 4, 4, 0]} fill="#8b5cf6" />
              </BarChart>
            </ResponsiveContainer>
          </div>
        </GlassCard>

        {/* Region distribution donut */}
        <GlassCard icon={MapPin} title="区域分布" subtitle="按区域统计TOP10">
          <div className="h-48">
            <ResponsiveContainer width="100%" height="100%">
              <PieChart>
                <Pie data={(stats?.region_distribution || []).slice(0, 8)} cx="50%" cy="50%" innerRadius={40} outerRadius={65} dataKey="count" nameKey="name"
                  label={({ name, percent }) => `${name} ${(percent * 100).toFixed(0)}%`} labelLine={{ strokeWidth: 1 }}>
                  {(stats?.region_distribution || []).slice(0, 8).map((_, idx) => (<Cell key={idx} fill={COLORS[idx % COLORS.length]} />))}
                </Pie>
                <Tooltip />
              </PieChart>
            </ResponsiveContainer>
          </div>
        </GlassCard>

        {/* TOP5 combined */}
        <GlassCard icon={Award} title="TOP5 排行" subtitle="区域 / 项目经理 / 客户">
          <div className="h-48 overflow-y-auto space-y-3" style={{ scrollbarWidth: 'thin' }}>
            {/* TOP5 Regions */}
            <div>
              <p className="text-[10px] font-semibold text-gray-500 mb-1">区域</p>
              {(stats?.top5_regions || []).slice(0, 3).map((item, idx) => (
                <div key={idx} className="flex items-center gap-2 py-0.5">
                  <span className="w-4 h-4 rounded text-[9px] font-bold text-white flex items-center justify-center bg-gradient-to-br from-indigo-500 to-purple-600">{idx + 1}</span>
                  <span className="flex-1 text-xs text-gray-700 truncate">{item.name || '未知'}</span>
                  <span className="text-xs font-semibold text-indigo-600">{item.count}</span>
                </div>
              ))}
            </div>
            {/* TOP5 Managers */}
            <div>
              <p className="text-[10px] font-semibold text-gray-500 mb-1">项目经理</p>
              {(stats?.top5_managers || []).slice(0, 3).map((item, idx) => (
                <div key={idx} className="flex items-center gap-2 py-0.5">
                  <span className="w-4 h-4 rounded text-[9px] font-bold text-white flex items-center justify-center bg-gradient-to-br from-blue-500 to-cyan-600">{idx + 1}</span>
                  <span className="flex-1 text-xs text-gray-700 truncate">{item.name || '未知'}</span>
                  <span className="text-xs font-semibold text-blue-600">{item.count}</span>
                </div>
              ))}
            </div>
            {/* TOP5 Customers */}
            <div>
              <p className="text-[10px] font-semibold text-gray-500 mb-1">客户</p>
              {(stats?.top5_customers || []).slice(0, 3).map((item, idx) => (
                <div key={idx} className="flex items-center gap-2 py-0.5">
                  <span className="w-4 h-4 rounded text-[9px] font-bold text-white flex items-center justify-center bg-gradient-to-br from-emerald-500 to-teal-600">{idx + 1}</span>
                  <span className="flex-1 text-xs text-gray-700 truncate">{item.name || '未知'}</span>
                  <span className="text-xs font-semibold text-emerald-600">{item.count}</span>
                </div>
              ))}
            </div>
          </div>
        </GlassCard>
      </div>
    </div>
  );
}
