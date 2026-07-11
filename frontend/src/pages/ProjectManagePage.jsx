import React, { useState, useEffect, useCallback, useRef, useMemo } from 'react';
import ReactDOM from 'react-dom';
import {
  LineChart, Line, XAxis, YAxis, CartesianGrid, Tooltip, ResponsiveContainer,
  PieChart, Pie, Cell,
  BarChart, Bar,
} from 'recharts';
import {
  FolderKanban, TrendingUp, Calendar, RefreshCw, ChevronLeft, ChevronRight,
  MapPin, Users, Building2, Layers, Activity, Award, CalendarDays, CalendarRange, BarChart3,
  PackageCheck,
} from 'lucide-react';
import useStore from '../store/useStore';
import FullscreenButton from '../components/FullscreenButton';
import toast from 'react-hot-toast';

const API_BASE = import.meta.env.VITE_API_BASE || '/api';

// Color palette for charts
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
   Calendar Heat Map for Period New Projects
   - month: calendar (monthly view with day cells)
   - quarter: week-based (grouped by week per month)
   - year: 12-month grid (yearly overview)
═══════════════════════════════════════════════════════════ */
function PeriodCalendarView({ data, period, customRange }) {
  // Build date->count lookup
  const dateCountMap = useMemo(() => {
    const map = {};
    (data || []).forEach(item => {
      if (item.date) map[item.date] = item.count;
    });
    return map;
  }, [data]);

  const maxCount = useMemo(() => Math.max(...Object.values(dateCountMap), 1), [dateCountMap]);

  // Determine view type
  const viewType = useMemo(() => {
    if (period === 'month') return 'month';
    if (period === 'quarter') return 'quarter';
    if (period === 'year') return 'year';
    // custom: determine by duration
    if (customRange?.start && customRange?.end) {
      const diffDays = (new Date(customRange.end) - new Date(customRange.start)) / (1000 * 60 * 60 * 24);
      if (diffDays <= 60) return 'month';
      if (diffDays <= 120) return 'quarter';
      return 'year';
    }
    return 'month';
  }, [period, customRange]);

  // Week data for quarter view (always computed to avoid conditional hooks)
  const weekData = useMemo(() => {
    const weeks = [];
    const entries = Object.entries(dateCountMap).sort((a, b) => a[0].localeCompare(b[0]));
    if (entries.length === 0) return [];

    let currentWeek = [];
    let weekStart = null;

    entries.forEach(([date, count]) => {
      const d = new Date(date);
      const dayOfWeek = d.getDay();
      if (dayOfWeek === 1 && currentWeek.length > 0) {
        weeks.push({ start: weekStart, days: [...currentWeek], total: currentWeek.reduce((s, v) => s + v.count, 0) });
        currentWeek = [];
      }
      if (currentWeek.length === 0) weekStart = date;
      currentWeek.push({ date, count, dayOfWeek });
    });
    if (currentWeek.length > 0) {
      weeks.push({ start: weekStart, days: [...currentWeek], total: currentWeek.reduce((s, v) => s + v.count, 0) });
    }
    return weeks;
  }, [dateCountMap]);

  // Monthly data for year view (always computed)
  const monthlyData = useMemo(() => {
    const monthMap = {};
    Object.entries(dateCountMap).forEach(([date, count]) => {
      const m = date.slice(0, 7); // YYYY-MM
      monthMap[m] = (monthMap[m] || 0) + count;
    });
    const now = new Date();
    const year = now.getFullYear();
    const months = [];
    for (let i = 0; i < 12; i++) {
      const key = `${year}-${String(i + 1).padStart(2, '0')}`;
      months.push({ month: `${i + 1}月`, key, count: monthMap[key] || 0 });
    }
    return months;
  }, [dateCountMap]);

  // Get color intensity for a count value
  const getColorClass = (count) => {
    if (!count || count === 0) return 'bg-gray-50 border-gray-100';
    const ratio = count / maxCount;
    if (ratio > 0.75) return 'bg-indigo-500 border-indigo-400 text-white';
    if (ratio > 0.5) return 'bg-indigo-400 border-indigo-300 text-white';
    if (ratio > 0.25) return 'bg-indigo-200 border-indigo-200 text-indigo-800';
    return 'bg-indigo-100 border-indigo-100 text-indigo-700';
  };

  // ─── Month View: Calendar Grid ───
  if (viewType === 'month') {
    const now = new Date();
    let year, month;
    if (period === 'custom' && customRange?.start) {
      const d = new Date(customRange.start);
      year = d.getFullYear();
      month = d.getMonth();
    } else {
      year = now.getFullYear();
      month = now.getMonth();
    }

    const daysInMonth = new Date(year, month + 1, 0).getDate();
    const firstDayOfWeek = new Date(year, month, 1).getDay(); // 0=Sun
    const weekDays = ['日', '一', '二', '三', '四', '五', '六'];

    return (
      <div className="space-y-2">
        <div className="grid grid-cols-7 gap-1">
          {weekDays.map((d) => (
            <div key={d} className="text-center text-[9px] text-gray-400 font-medium py-0.5">{d}</div>
          ))}
        </div>
        <div className="grid grid-cols-7 gap-1">
          {Array.from({ length: firstDayOfWeek }).map((_, i) => <div key={`e-${i}`} />)}
          {Array.from({ length: daysInMonth }).map((_, i) => {
            const day = i + 1;
            const dateStr = `${year}-${String(month + 1).padStart(2, '0')}-${String(day).padStart(2, '0')}`;
            const count = dateCountMap[dateStr] || 0;
            const isToday = now.getFullYear() === year && now.getMonth() === month && now.getDate() === day;

            return (
              <div key={day} title={`${dateStr}: ${count} 个项目`}
                className={`relative w-full aspect-square rounded-lg border flex flex-col items-center justify-center transition-all duration-150 ${getColorClass(count)} ${isToday ? 'ring-2 ring-indigo-400 ring-offset-1' : ''}`}>
                <span className="text-[9px] font-medium leading-none">{day}</span>
                {count > 0 && <span className="text-[8px] font-bold leading-none mt-0.5">{count}</span>}
              </div>
            );
          })}
        </div>
        {/* Legend */}
        <div className="flex items-center justify-end gap-1 mt-1">
          <span className="text-[8px] text-gray-400">少</span>
          <div className="w-3 h-3 rounded bg-gray-50 border border-gray-100" />
          <div className="w-3 h-3 rounded bg-indigo-100 border border-indigo-100" />
          <div className="w-3 h-3 rounded bg-indigo-200 border border-indigo-200" />
          <div className="w-3 h-3 rounded bg-indigo-400 border border-indigo-300" />
          <div className="w-3 h-3 rounded bg-indigo-500 border border-indigo-400" />
          <span className="text-[8px] text-gray-400">多</span>
        </div>
      </div>
    );
  }

  // ─── Quarter View: Week-based grid ───
  if (viewType === 'quarter') {
    const weekMax = Math.max(...weekData.map(w => w.total), 1);

    return (
      <div className="space-y-1.5 max-h-56 overflow-y-auto pr-1" style={{ scrollbarWidth: 'thin' }}>
        {weekData.map((week, idx) => {
          const ratio = week.total / weekMax;
          const barWidth = Math.max(ratio * 100, 2);
          return (
            <div key={idx} className="flex items-center gap-2">
              <span className="text-[8px] text-gray-400 w-14 shrink-0 text-right">{week.start?.slice(5)}</span>
              <div className="flex-1 h-5 bg-gray-50 rounded-md overflow-hidden relative">
                <div className="h-full rounded-md bg-gradient-to-r from-indigo-400 to-purple-500 transition-all duration-300"
                  style={{ width: `${barWidth}%` }} />
                {week.total > 0 && (
                  <span className="absolute right-1.5 top-1/2 -translate-y-1/2 text-[9px] font-bold text-gray-600">{week.total}</span>
                )}
              </div>
            </div>
          );
        })}
        {weekData.length === 0 && (
          <div className="flex flex-col items-center justify-center py-8 text-gray-400">
            <CalendarDays size={24} className="mb-1 opacity-40" />
            <p className="text-[10px]">暂无数据</p>
          </div>
        )}
      </div>
    );
  }

  // ─── Year View: 12-month grid ───
  const yearMax = Math.max(...monthlyData.map(m => m.count), 1);

  return (
    <div className="grid grid-cols-4 gap-2">
      {monthlyData.map((m) => {
        const ratio = m.count / yearMax;
        const bgClass = m.count === 0 ? 'bg-gray-50 border-gray-100' :
          ratio > 0.75 ? 'bg-indigo-500 border-indigo-400' :
          ratio > 0.5 ? 'bg-indigo-400 border-indigo-300' :
          ratio > 0.25 ? 'bg-indigo-200 border-indigo-200' : 'bg-indigo-100 border-indigo-100';
        const textClass = m.count === 0 ? 'text-gray-400' :
          ratio > 0.5 ? 'text-white' : 'text-indigo-700';

        return (
          <div key={m.key} title={`${m.key}: ${m.count} 个项目`}
            className={`rounded-xl border p-2 flex flex-col items-center justify-center transition-all duration-200 ${bgClass}`}>
            <span className={`text-[10px] font-medium ${textClass}`}>{m.month}</span>
            <span className={`text-sm font-bold ${textClass}`}>{m.count}</span>
          </div>
        );
      })}
    </div>
  );
}


/* ═══════════════════════════════════════════════════════════
   TOP5 Carousel Component (rotates every 10s)
═══════════════════════════════════════════════════════════ */
function Top5Carousel({ stats }) {
  const [activeIndex, setActiveIndex] = useState(0);
  const intervalRef = useRef(null);

  const tabs = [
    { key: 'regions', label: '区域', icon: MapPin, data: stats?.top5_regions, color: 'from-indigo-500 to-purple-600', textColor: 'text-indigo-600' },
    { key: 'managers', label: '项目经理', icon: Users, data: stats?.top5_managers, color: 'from-blue-500 to-cyan-600', textColor: 'text-blue-600' },
    { key: 'customers', label: '客户', icon: Building2, data: stats?.top5_customers, color: 'from-emerald-500 to-teal-600', textColor: 'text-emerald-600' },
  ];

  // Auto-rotate every 10 seconds
  useEffect(() => {
    intervalRef.current = setInterval(() => {
      setActiveIndex(prev => (prev + 1) % tabs.length);
    }, 10000);
    return () => clearInterval(intervalRef.current);
  }, [tabs.length]);

  // Reset timer on manual tab click
  const handleTabClick = (idx) => {
    setActiveIndex(idx);
    clearInterval(intervalRef.current);
    intervalRef.current = setInterval(() => {
      setActiveIndex(prev => (prev + 1) % tabs.length);
    }, 10000);
  };

  const activeTab = tabs[activeIndex];

  return (
    <div className="h-full flex flex-col">
      {/* Tab indicators */}
      <div className="flex items-center gap-1 mb-3">
        {tabs.map((tab, idx) => {
          const Icon = tab.icon;
          const isActive = idx === activeIndex;
          return (
            <button key={tab.key} onClick={() => handleTabClick(idx)}
              className={`flex items-center gap-1 px-2 py-1 rounded-lg text-[10px] font-medium transition-all duration-300 ${
                isActive ? 'bg-indigo-50 text-indigo-700 border border-indigo-200' : 'text-gray-400 hover:text-gray-600'
              }`}>
              <Icon size={10} />
              {tab.label}
            </button>
          );
        })}
        {/* Progress bar */}
        <div className="flex-1 flex justify-end">
          <div className="flex gap-0.5">
            {tabs.map((_, idx) => (
              <div key={idx} className={`w-5 h-1 rounded-full transition-all duration-300 ${idx === activeIndex ? 'bg-indigo-200' : 'bg-gray-200'}`}>
                {idx === activeIndex && (
                  <div className="h-full rounded-full bg-indigo-600" style={{ animation: 'top5progress 10s linear infinite' }} />
                )}
              </div>
            ))}
          </div>
        </div>
      </div>

      {/* Content with slide animation */}
      <div className="flex-1 overflow-hidden relative">
        <div className="transition-all duration-500 ease-in-out" key={activeTab.key}>
          {(activeTab.data || []).slice(0, 5).map((item, idx) => (
            <div key={idx} className="flex items-center gap-2.5 py-1.5 group">
              <span className={`w-5 h-5 rounded-lg text-[10px] font-bold text-white flex items-center justify-center bg-gradient-to-br ${activeTab.color} shadow-sm`}>
                {idx + 1}
              </span>
              <span className="flex-1 text-xs text-gray-700 truncate group-hover:text-gray-900 transition-colors">{item.name || '未知'}</span>
              <div className="flex items-center gap-1.5">
                <div className="w-16 h-1.5 bg-gray-100 rounded-full overflow-hidden">
                  <div className={`h-full rounded-full bg-gradient-to-r ${activeTab.color} transition-all duration-700`}
                    style={{ width: `${Math.min((item.count / (activeTab.data?.[0]?.count || 1)) * 100, 100)}%` }} />
                </div>
                <span className={`text-xs font-semibold ${activeTab.textColor} min-w-[24px] text-right`}>{item.count}</span>
              </div>
            </div>
          ))}
          {(!activeTab.data || activeTab.data.length === 0) && (
            <div className="flex flex-col items-center justify-center py-6 text-gray-400">
              <activeTab.icon size={20} className="mb-1 opacity-40" />
              <p className="text-[10px]">暂无数据</p>
            </div>
          )}
        </div>
      </div>
    </div>
  );
}

/* ═══════════════════════════════════════════════════════════
   Pre-Delivery Scrolling List Component
═══════════════════════════════════════════════════════════ */
function PreDeliveryList({ data }) {
  const scrollRef = useRef(null);
  const [isPaused, setIsPaused] = useState(false);

  // Auto-scroll effect
  useEffect(() => {
    const container = scrollRef.current;
    if (!container || !data || data.length === 0) return;

    let animId;
    let scrollSpeed = 0.5; // px per frame

    const step = () => {
      if (!isPaused && container) {
        container.scrollTop += scrollSpeed;
        // Reset to top when reaching bottom
        if (container.scrollTop >= container.scrollHeight - container.clientHeight) {
          container.scrollTop = 0;
        }
      }
      animId = requestAnimationFrame(step);
    };

    animId = requestAnimationFrame(step);
    return () => cancelAnimationFrame(animId);
  }, [data, isPaused]);

  if (!data || data.length === 0) {
    return (
      <div className="flex flex-col items-center justify-center py-8 text-gray-400">
        <PackageCheck size={32} className="mb-2 opacity-40" />
        <p className="text-xs">暂无预交付项目</p>
      </div>
    );
  }

  return (
    <div className="space-y-2">
      {/* Table header */}
      <div className="grid grid-cols-7 gap-1 px-2 py-1.5 bg-gradient-to-r from-indigo-50 to-purple-50 rounded-lg border border-indigo-100/50">
        <span className="text-[9px] font-semibold text-indigo-700 col-span-2">项目名称</span>
        <span className="text-[9px] font-semibold text-indigo-700">项目编号</span>
        <span className="text-[9px] font-semibold text-indigo-700">销售</span>
        <span className="text-[9px] font-semibold text-indigo-700">售前</span>
        <span className="text-[9px] font-semibold text-indigo-700">商机号</span>
        <span className="text-[9px] font-semibold text-indigo-700">省份</span>
      </div>
      {/* Scrolling content */}
      <div
        ref={scrollRef}
        className="max-h-48 overflow-hidden"
        onMouseEnter={() => setIsPaused(true)}
        onMouseLeave={() => setIsPaused(false)}
        style={{ scrollbarWidth: 'none' }}
      >
        {/* Duplicate data for seamless scroll */}
        {[...data, ...data].map((item, idx) => (
          <div key={idx}
            className={`grid grid-cols-7 gap-1 px-2 py-1.5 border-b border-gray-50 hover:bg-indigo-50/50 transition-colors ${idx % 2 === 0 ? 'bg-white/50' : 'bg-gray-50/30'}`}>
            <span className="text-[9px] text-gray-700 truncate col-span-2" title={item.project_name}>{item.project_name || '-'}</span>
            <span className="text-[9px] text-gray-600 truncate" title={item.project_no}>{item.project_no || '-'}</span>
            <span className="text-[9px] text-gray-600 truncate" title={item.sale_user}>{item.sale_user || '-'}</span>
            <span className="text-[9px] text-gray-600 truncate" title={item.pre_sale_user}>{item.pre_sale_user || '-'}</span>
            <span className="text-[9px] text-gray-600 truncate" title={item.business_no}>{item.business_no || '-'}</span>
            <span className="text-[9px] text-gray-600 truncate" title={item.province}>{item.province || '-'}</span>
          </div>
        ))}
      </div>
      {/* Footer */}
      <div className="flex items-center justify-between px-2">
        <span className="text-[8px] text-gray-400">共 {data.length} 个预交付项目</span>
        <span className="text-[8px] text-gray-400">鼠标悬停暂停滚动</span>
      </div>
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
  const [preDeliveryList, setPreDeliveryList] = useState([]);

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

  // Fetch pre-delivery list (only once on mount)
  useEffect(() => {
    const fetchPreDelivery = async () => {
      try {
        const res = await fetch(`${API_BASE}/projects/pre-delivery`, {
          headers: { Authorization: `Bearer ${token}` },
        });
        const data = await res.json();
        if (data.code === 0) setPreDeliveryList(data.data || []);
      } catch (e) {
        console.error('Failed to fetch pre-delivery list:', e);
      }
    };
    fetchPreDelivery();
  }, [token]);

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

  // Period label for the calendar card
  const periodLabel = useMemo(() => {
    if (period === 'month') return '月历';
    if (period === 'quarter') return '周历';
    if (period === 'year') return '年历';
    // custom: determine by duration
    if (customRange.start && customRange.end) {
      const diffDays = (new Date(customRange.end) - new Date(customRange.start)) / (1000 * 60 * 60 * 24);
      if (diffDays <= 60) return '月历';
      if (diffDays <= 120) return '周历';
      return '年历';
    }
    return '月历';
  }, [period, customRange]);

  if (loading && !stats) {
    return (
      <div className="flex items-center justify-center h-full">
        <div className="animate-spin w-8 h-8 border-4 border-indigo-500 border-t-transparent rounded-full" />
      </div>
    );
  }

  const pageRef = useRef(null);

  return (
    <div ref={pageRef} className="h-full overflow-y-auto p-5 space-y-4 bg-gradient-to-br from-slate-50 to-purple-50/30" style={{ scrollbarWidth: 'thin' }}>
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
          <FullscreenButton containerRef={pageRef} />
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

      {/* Row 2: Period Calendar + Recent month trend + TOP5 Carousel (3 cols) */}
      <div className="grid grid-cols-1 lg:grid-cols-3 gap-3">
        {/* Period new breakdown - Calendar/Week/Year format */}
        <GlassCard icon={CalendarDays} title={`周期内新增立项（${periodLabel}）`} subtitle={period === 'custom' ? `${customRange.start} 至 ${customRange.end}` : `${getPeriodDates(period).start} 至 ${getPeriodDates(period).end}`}>
          <div className="min-h-[220px] flex items-start">
            {(!stats?.month_daily_projects || stats.month_daily_projects.length === 0) ? (
              <div className="flex flex-col items-center justify-center w-full py-8 text-gray-400">
                <CalendarDays size={32} className="mb-2 opacity-40" />
                <p className="text-xs">暂无数据</p>
              </div>
            ) : (
              <div className="w-full">
                <PeriodCalendarView data={stats.month_daily_projects} period={period} customRange={customRange} />
              </div>
            )}
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

        {/* TOP5 Carousel */}
        <GlassCard icon={Award} title="TOP5 排行" subtitle="区域 / 项目经理 / 客户（自动轮播）">
          <div className="h-52">
            <Top5Carousel stats={stats} />
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

      {/* Row 4: Status bar + Region donut (2 cols) */}
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
      </div>

      {/* Row 5: Pre-delivery list (full width) */}
      <div className="grid grid-cols-1 gap-3">
        <GlassCard icon={PackageCheck} title="预交付清单" subtitle={`交付类型为"预交付"的项目（共 ${preDeliveryList.length} 个）`}>
          <PreDeliveryList data={preDeliveryList} />
        </GlassCard>
      </div>
    </div>
  );
}
