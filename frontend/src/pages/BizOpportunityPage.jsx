import React, { useState, useEffect, useCallback } from 'react';
import {
  Upload, FileSpreadsheet, TrendingUp, Users, Building2, MapPin, BarChart3,
  RefreshCw, Download, Trash2, Search, Filter, ChevronDown, ChevronUp,
  AlertCircle, CheckCircle2, Clock, Target, DollarSign, Layers, PieChart as PieIcon,
  History, X, ArrowUpDown,
} from 'lucide-react';
import {
  BarChart, Bar, LineChart, Line, PieChart, Pie, Cell,
  RadarChart, Radar, PolarGrid, PolarAngleAxis, PolarRadiusAxis,
  XAxis, YAxis, CartesianGrid, Tooltip, Legend, ResponsiveContainer,
} from 'recharts';
import {
  uploadBizExcel, getBizList, getBizStats, getBizHistory,
  deleteBizUpload, getBizMonths, getBizFilters,
} from '../services/api';
import useStore from '../store/useStore';

const COLORS = ['#6C5CE7', '#00B894', '#FDCB6E', '#E17055', '#0984E3', '#D63031',
  '#E84393', '#00CEC9', '#FD79A8', '#636E72', '#55E6C1', '#F8C291'];

const CHART_COLORS = {
  primary: '#6C5CE7',
  success: '#00B894',
  warning: '#FDCB6E',
  danger: '#E17055',
  info: '#0984E3',
};

function formatAmount(val) {
  if (val >= 10000) return (val / 10000).toFixed(2) + ' 万';
  return val?.toFixed(2) || '0';
}

function formatAmountWan(val) {
  if (!val) return '0';
  return (val / 10000).toFixed(2);
}

export default function BizOpportunityPage() {
  const theme = useStore((s) => s.theme);
  const isDark = theme === 'dark';

  // Data state
  const [stats, setStats] = useState(null);
  const [records, setRecords] = useState([]);
  const [total, setTotal] = useState(0);
  const [history, setHistory] = useState([]);
  const [months, setMonths] = useState([]);
  const [filterOptions, setFilterOptions] = useState({});

  // UI state
  const [loading, setLoading] = useState(false);
  const [uploading, setUploading] = useState(false);
  const [activeTab, setActiveTab] = useState('overview'); // overview, charts, data, history
  const [selectedMonth, setSelectedMonth] = useState('');
  const [filters, setFilters] = useState({ status: '', region: '', biz_type: '', search: '' });
  const [page, setPage] = useState(1);
  const [showUploadModal, setShowUploadModal] = useState(false);
  const [uploadMonth, setUploadMonth] = useState('');
  const [uploadResult, setUploadResult] = useState(null);
  const [sortField, setSortField] = useState('amount');
  const [sortDir, setSortDir] = useState('desc');

  // Load data
  const loadData = useCallback(async () => {
    setLoading(true);
    try {
      const [statsRes, monthsRes, historyRes] = await Promise.all([
        getBizStats({ month: selectedMonth }),
        getBizMonths(),
        getBizHistory(),
      ]);
      if (statsRes?.code === 0) setStats(statsRes.data);
      if (monthsRes?.code === 0) setMonths(monthsRes.data || []);
      if (historyRes?.code === 0) setHistory(historyRes.data || []);

      const filtersRes = await getBizFilters({ month: selectedMonth });
      if (filtersRes?.code === 0) setFilterOptions(filtersRes.data || {});
    } catch (e) { console.error(e); }
    finally { setLoading(false); }
  }, [selectedMonth]);

  const loadRecords = useCallback(async () => {
    try {
      const res = await getBizList({
        month: selectedMonth,
        page,
        page_size: 50,
        ...filters,
      });
      if (res?.code === 0) {
        setRecords(res.data || []);
        setTotal(res.total || 0);
      }
    } catch (e) { console.error(e); }
  }, [selectedMonth, page, filters]);

  useEffect(() => { loadData(); }, [loadData]);
  useEffect(() => { loadRecords(); }, [loadRecords]);

  // Upload handler
  const handleUpload = async (e) => {
    const file = e.target.files?.[0];
    if (!file) return;
    setUploading(true);
    setUploadResult(null);
    try {
      const res = await uploadBizExcel(file, uploadMonth);
      if (res?.code === 0) {
        setUploadResult(res.data);
        loadData();
        loadRecords();
      } else {
        setUploadResult({ error: res?.message || '上传失败' });
      }
    } catch (e) {
      setUploadResult({ error: e.message || '上传失败' });
    } finally {
      setUploading(false);
      e.target.value = '';
    }
  };

  const handleDeleteUpload = async (id) => {
    if (!window.confirm('确定删除该上传记录及其所有数据？')) return;
    try {
      const res = await deleteBizUpload(id);
      if (res?.code === 0) {
        loadData();
        loadRecords();
      }
    } catch (e) { console.error(e); }
  };

  // Sort locally
  const sortedRecords = [...records].sort((a, b) => {
    const dir = sortDir === 'desc' ? -1 : 1;
    if (sortField === 'amount') return (a.amount - b.amount) * dir;
    if (sortField === 'service_amount') return (a.service_amount - b.service_amount) * dir;
    if (sortField === 'node_count') return (a.node_count - b.node_count) * dir;
    return 0;
  });

  const cardClass = `rounded-xl border ${isDark ? 'bg-slate-800 border-slate-700' : 'bg-white border-gray-200'}`;
  const textMain = isDark ? 'text-slate-200' : 'text-gray-800';
  const textSub = isDark ? 'text-slate-400' : 'text-gray-500';
  const textMuted = isDark ? 'text-slate-500' : 'text-gray-400';

  const tabs = [
    { id: 'overview', label: '数据概览', icon: BarChart3 },
    { id: 'charts', label: '多维图表', icon: PieIcon },
    { id: 'data', label: '数据明细', icon: FileSpreadsheet },
    { id: 'history', label: '上传历史', icon: History },
  ];

  const ov = stats?.overview;

  return (
    <div className="h-full overflow-y-auto p-6 space-y-5">
      {/* Top Bar: Month selector + Upload + Tabs */}
      <div className="flex flex-wrap items-center gap-3">
        <div className="flex items-center gap-2">
          <label className={`text-sm font-medium ${textSub}`}>数据月份</label>
          <select
            value={selectedMonth}
            onChange={(e) => { setSelectedMonth(e.target.value); setPage(1); }}
            className={`text-sm px-3 py-1.5 rounded-lg border ${isDark ? 'bg-slate-700 border-slate-600 text-slate-200' : 'bg-white border-gray-300 text-gray-700'}`}
          >
            <option value="">全部月份</option>
            {months.map((m) => (
              <option key={m} value={m}>{m}</option>
            ))}
          </select>
        </div>

        <button
          onClick={() => setShowUploadModal(true)}
          className="flex items-center gap-2 px-4 py-1.5 bg-primary-600 text-white text-sm rounded-lg hover:bg-primary-700 transition-colors"
        >
          <Upload className="w-4 h-4" /> 上传Excel
        </button>

        <button onClick={() => { loadData(); loadRecords(); }} className={`p-1.5 rounded-lg transition-colors ${isDark ? 'text-slate-400 hover:bg-slate-700' : 'text-gray-400 hover:bg-gray-100'}`}>
          <RefreshCw className={`w-4 h-4 ${loading ? 'animate-spin' : ''}`} />
        </button>

        <div className="flex-1" />

        <div className={`flex rounded-lg border ${isDark ? 'border-slate-700' : 'border-gray-200'}`}>
          {tabs.map((t) => {
            const Icon = t.icon;
            return (
              <button
                key={t.id}
                onClick={() => setActiveTab(t.id)}
                className={`flex items-center gap-1.5 px-3 py-1.5 text-sm transition-colors ${
                  activeTab === t.id
                    ? 'bg-primary-600 text-white'
                    : isDark ? 'text-slate-300 hover:bg-slate-700' : 'text-gray-600 hover:bg-gray-50'
                } ${t.id === tabs[0].id ? 'rounded-l-lg' : ''} ${t.id === tabs[tabs.length - 1].id ? 'rounded-r-lg' : ''}`}
              >
                <Icon className="w-3.5 h-3.5" /> {t.label}
              </button>
            );
          })}
        </div>
      </div>

      {/* Upload Modal */}
      {showUploadModal && (
        <div className="fixed inset-0 bg-black/50 z-50 flex items-center justify-center" onClick={() => !uploading && setShowUploadModal(false)}>
          <div className={`${cardClass} p-6 w-[480px] max-w-[95vw]`} onClick={(e) => e.stopPropagation()}>
            <div className="flex items-center justify-between mb-4">
              <h3 className={`text-lg font-semibold ${textMain}`}>上传商机Excel</h3>
              <button onClick={() => !uploading && setShowUploadModal(false)} className={`p-1 rounded-lg ${isDark ? 'hover:bg-slate-700' : 'hover:bg-gray-100'}`}>
                <X className={`w-5 h-5 ${textSub}`} />
              </button>
            </div>
            <div className="space-y-4">
              <div>
                <label className={`text-sm font-medium ${textSub}`}>数据月份</label>
                <input
                  type="month"
                  value={uploadMonth}
                  onChange={(e) => setUploadMonth(e.target.value)}
                  className={`mt-1 w-full text-sm px-3 py-2 rounded-lg border ${isDark ? 'bg-slate-700 border-slate-600 text-slate-200' : 'bg-white border-gray-300 text-gray-700'}`}
                  placeholder="留空则自动识别"
                />
                <p className={`text-xs mt-1 ${textMuted}`}>留空将自动使用当前年月</p>
              </div>
              <div>
                <label className={`text-sm font-medium ${textSub}`}>选择文件</label>
                <label className={`mt-1 flex flex-col items-center justify-center w-full h-32 border-2 border-dashed rounded-xl cursor-pointer transition-colors ${
                  isDark ? 'border-slate-600 hover:border-primary-400 hover:bg-slate-700/50' : 'border-gray-300 hover:border-primary-400 hover:bg-primary-50/30'
                }`}>
                  {uploading ? (
                    <div className="text-center">
                      <RefreshCw className="w-8 h-8 mx-auto mb-2 animate-spin text-primary-500" />
                      <p className={`text-sm ${textSub}`}>正在解析上传...</p>
                    </div>
                  ) : (
                    <div className="text-center">
                      <Upload className={`w-8 h-8 mx-auto mb-2 ${textMuted}`} />
                      <p className={`text-sm ${textSub}`}>点击选择 .xlsx 文件</p>
                      <p className={`text-xs ${textMuted}`}>将自动筛选含"维保"/"续保"的商机</p>
                    </div>
                  )}
                  <input type="file" accept=".xlsx,.xls" className="hidden" onChange={handleUpload} disabled={uploading} />
                </label>
              </div>
              {uploadResult && (
                <div className={`p-3 rounded-lg text-sm ${
                  uploadResult.error
                    ? isDark ? 'bg-red-900/30 text-red-300' : 'bg-red-50 text-red-700'
                    : isDark ? 'bg-green-900/30 text-green-300' : 'bg-green-50 text-green-700'
                }`}>
                  {uploadResult.error ? (
                    <div className="flex items-center gap-2"><AlertCircle className="w-4 h-4" /> {uploadResult.error}</div>
                  ) : (
                    <div className="space-y-1">
                      <div className="flex items-center gap-2"><CheckCircle2 className="w-4 h-4" /> 上传成功！</div>
                      <p>文件: {uploadResult.file_name}</p>
                      <p>月份: {uploadResult.month}</p>
                      <p>总行数: {uploadResult.total_rows}，筛选维保/续保: <strong>{uploadResult.filtered_rows}</strong> 条</p>
                    </div>
                  )}
                </div>
              )}
            </div>
          </div>
        </div>
      )}

      {/* No Data Placeholder */}
      {!loading && (!stats || stats.total === 0) && activeTab !== 'history' && (
        <div className={`${cardClass} p-16 text-center`}>
          <FileSpreadsheet className={`w-16 h-16 mx-auto mb-4 ${textMuted}`} />
          <h3 className={`text-lg font-medium mb-2 ${textMain}`}>暂无商机数据</h3>
          <p className={`text-sm mb-4 ${textSub}`}>请先上传包含维保/续保商机的Excel表格</p>
          <button
            onClick={() => setShowUploadModal(true)}
            className="px-5 py-2 bg-primary-600 text-white rounded-lg hover:bg-primary-700 transition-colors text-sm"
          >
            <Upload className="w-4 h-4 inline mr-2" /> 上传Excel
          </button>
        </div>
      )}

      {/* ====== TAB: Overview ====== */}
      {activeTab === 'overview' && ov && (
        <div className="space-y-5">
          {/* KPI Cards */}
          <div className="grid grid-cols-2 md:grid-cols-3 lg:grid-cols-6 gap-3">
            {[
              { label: '商机总数', value: ov.total_count, icon: Target, color: 'primary' },
              { label: '总金额(万)', value: formatAmountWan(ov.total_amount), icon: DollarSign, color: 'success' },
              { label: '服务金额(万)', value: formatAmountWan(ov.total_service_amount), icon: TrendingUp, color: 'info' },
              { label: '均单金额(万)', value: formatAmountWan(ov.avg_amount), icon: BarChart3, color: 'warning' },
              { label: '进行中', value: ov.status_in_progress || 0, icon: Clock, color: 'info' },
              { label: '赢单', value: ov.status_won || 0, icon: CheckCircle2, color: 'success' },
            ].map((kpi, i) => {
              const Icon = kpi.icon;
              return (
                <div key={i} className={`${cardClass} p-4`}>
                  <div className="flex items-center gap-2 mb-2">
                    <div className={`w-8 h-8 rounded-lg flex items-center justify-center`}
                      style={{ background: CHART_COLORS[kpi.color] + '20' }}>
                      <Icon className="w-4 h-4" style={{ color: CHART_COLORS[kpi.color] }} />
                    </div>
                    <span className={`text-xs ${textSub}`}>{kpi.label}</span>
                  </div>
                  <p className={`text-xl font-bold ${textMain}`}>{kpi.value}</p>
                </div>
              );
            })}
          </div>

          {/* TOP10 Section */}
          <div className="grid grid-cols-1 lg:grid-cols-3 gap-4">
            {/* TOP10 Owners */}
            <div className={`${cardClass} p-4`}>
              <h4 className={`text-sm font-semibold mb-3 flex items-center gap-2 ${textMain}`}>
                <Users className="w-4 h-4 text-primary-500" /> TOP10 负责人
              </h4>
              <div className="space-y-2">
                {(stats?.top10_owners || []).map((item, i) => (
                  <div key={i} className="flex items-center gap-2">
                    <span className={`w-5 h-5 rounded-full text-xs flex items-center justify-center font-bold flex-shrink-0 ${
                      i < 3 ? 'bg-primary-600 text-white' : isDark ? 'bg-slate-700 text-slate-400' : 'bg-gray-100 text-gray-500'
                    }`}>{i + 1}</span>
                    <span className={`text-sm flex-1 truncate ${textMain}`}>{item.name}</span>
                    <span className={`text-xs ${textSub}`}>{item.count}笔</span>
                    <span className="text-sm font-medium text-primary-500">{formatAmount(item.value)}</span>
                  </div>
                ))}
                {(stats?.top10_owners || []).length === 0 && <p className={`text-sm ${textMuted}`}>暂无数据</p>}
              </div>
            </div>

            {/* TOP10 Customers */}
            <div className={`${cardClass} p-4`}>
              <h4 className={`text-sm font-semibold mb-3 flex items-center gap-2 ${textMain}`}>
                <Building2 className="w-4 h-4 text-green-500" /> TOP10 客户
              </h4>
              <div className="space-y-2">
                {(stats?.top10_customers || []).map((item, i) => (
                  <div key={i} className="flex items-center gap-2">
                    <span className={`w-5 h-5 rounded-full text-xs flex items-center justify-center font-bold flex-shrink-0 ${
                      i < 3 ? 'bg-green-600 text-white' : isDark ? 'bg-slate-700 text-slate-400' : 'bg-gray-100 text-gray-500'
                    }`}>{i + 1}</span>
                    <span className={`text-sm flex-1 truncate ${textMain}`} title={item.name}>{item.name}</span>
                    <span className={`text-xs ${textSub}`}>{item.count}笔</span>
                    <span className="text-sm font-medium text-green-500">{formatAmount(item.value)}</span>
                  </div>
                ))}
                {(stats?.top10_customers || []).length === 0 && <p className={`text-sm ${textMuted}`}>暂无数据</p>}
              </div>
            </div>

            {/* TOP10 Region (负责人所属核心管控单元) */}
            <div className={`${cardClass} p-4`}>
              <h4 className={`text-sm font-semibold mb-3 flex items-center gap-2 ${textMain}`}>
                <MapPin className="w-4 h-4 text-orange-500" /> TOP10 区域
              </h4>
              <div className="space-y-2">
                {(stats?.top10_region || []).map((item, i) => (
                  <div key={i} className="flex items-center gap-2">
                    <span className={`w-5 h-5 rounded-full text-xs flex items-center justify-center font-bold flex-shrink-0 ${
                      i < 3 ? 'bg-orange-600 text-white' : isDark ? 'bg-slate-700 text-slate-400' : 'bg-gray-100 text-gray-500'
                    }`}>{i + 1}</span>
                    <span className={`text-sm flex-1 truncate ${textMain}`}>{item.name}</span>
                    <span className={`text-xs ${textSub}`}>{item.count}笔</span>
                    <span className="text-sm font-medium text-orange-500">{formatAmount(item.value)}</span>
                  </div>
                ))}
                {(stats?.top10_region || []).length === 0 && <p className={`text-sm ${textMuted}`}>暂无数据</p>}
              </div>
            </div>
          </div>

          {/* Region bar chart + Status pie */}
          <div className="grid grid-cols-1 lg:grid-cols-2 gap-4">
            <div className={`${cardClass} p-4`}>
              <h4 className={`text-sm font-semibold mb-3 ${textMain}`}>区域金额分布 (柱状图)</h4>
              <ResponsiveContainer width="100%" height={280}>
                <BarChart data={stats?.region_data || []} margin={{ top: 5, right: 20, bottom: 5, left: 10 }}>
                  <CartesianGrid strokeDasharray="3 3" stroke={isDark ? '#334155' : '#e5e7eb'} />
                  <XAxis dataKey="name" tick={{ fontSize: 11, fill: isDark ? '#94a3b8' : '#6b7280' }} angle={-20} textAnchor="end" height={50} />
                  <YAxis tick={{ fontSize: 11, fill: isDark ? '#94a3b8' : '#6b7280' }} tickFormatter={(v) => formatAmountWan(v) + '万'} />
                  <Tooltip
                    contentStyle={{ background: isDark ? '#1e293b' : '#fff', border: isDark ? '1px solid #334155' : '1px solid #e5e7eb', borderRadius: 8, fontSize: 12 }}
                    formatter={(val) => [formatAmount(val), '金额']}
                  />
                  <Bar dataKey="value" fill={CHART_COLORS.primary} radius={[4, 4, 0, 0]} />
                </BarChart>
              </ResponsiveContainer>
            </div>

            <div className={`${cardClass} p-4`}>
              <h4 className={`text-sm font-semibold mb-3 ${textMain}`}>商机状态分布 (饼图)</h4>
              <ResponsiveContainer width="100%" height={280}>
                <PieChart>
                  <Pie
                    data={stats?.status_data || []}
                    cx="50%" cy="50%"
                    outerRadius={90}
                    dataKey="value"
                    nameKey="name"
                    label={({ name, percent }) => `${name} ${(percent * 100).toFixed(0)}%`}
                  >
                    {(stats?.status_data || []).map((_, i) => (
                      <Cell key={i} fill={COLORS[i % COLORS.length]} />
                    ))}
                  </Pie>
                  <Tooltip formatter={(val) => [val, '数量']} contentStyle={{ background: isDark ? '#1e293b' : '#fff', border: isDark ? '1px solid #334155' : '1px solid #e5e7eb', borderRadius: 8, fontSize: 12 }} />
                  <Legend wrapperStyle={{ fontSize: 12 }} />
                </PieChart>
              </ResponsiveContainer>
            </div>
          </div>
        </div>
      )}

      {/* ====== TAB: Charts ====== */}
      {activeTab === 'charts' && stats && stats.total !== 0 && (
        <div className="space-y-4">
          <div className="grid grid-cols-1 lg:grid-cols-2 gap-4">
            {/* Monthly Trend Line */}
            <div className={`${cardClass} p-4`}>
              <h4 className={`text-sm font-semibold mb-3 ${textMain}`}>月度趋势 (折线图)</h4>
              <ResponsiveContainer width="100%" height={280}>
                <LineChart data={stats?.month_trend || []} margin={{ top: 5, right: 20, bottom: 5, left: 10 }}>
                  <CartesianGrid strokeDasharray="3 3" stroke={isDark ? '#334155' : '#e5e7eb'} />
                  <XAxis dataKey="month" tick={{ fontSize: 11, fill: isDark ? '#94a3b8' : '#6b7280' }} />
                  <YAxis yAxisId="left" tick={{ fontSize: 11, fill: isDark ? '#94a3b8' : '#6b7280' }} tickFormatter={(v) => formatAmountWan(v) + '万'} />
                  <YAxis yAxisId="right" orientation="right" tick={{ fontSize: 11, fill: isDark ? '#94a3b8' : '#6b7280' }} />
                  <Tooltip contentStyle={{ background: isDark ? '#1e293b' : '#fff', border: isDark ? '1px solid #334155' : '1px solid #e5e7eb', borderRadius: 8, fontSize: 12 }} formatter={(val, name) => [name === 'amount' ? formatAmount(val) : val, name === 'amount' ? '金额' : '数量']} />
                  <Legend wrapperStyle={{ fontSize: 12 }} formatter={(val) => val === 'amount' ? '金额' : '数量'} />
                  <Line yAxisId="left" type="monotone" dataKey="amount" stroke={CHART_COLORS.primary} strokeWidth={2} dot={{ r: 4 }} />
                  <Line yAxisId="right" type="monotone" dataKey="count" stroke={CHART_COLORS.success} strokeWidth={2} dot={{ r: 4 }} />
                </LineChart>
              </ResponsiveContainer>
            </div>

            {/* Radar Chart by Region */}
            <div className={`${cardClass} p-4`}>
              <h4 className={`text-sm font-semibold mb-3 ${textMain}`}>区域多维对比 (雷达图)</h4>
              <ResponsiveContainer width="100%" height={280}>
                <RadarChart data={(stats?.radar_data || []).slice(0, 8)} cx="50%" cy="50%" outerRadius="70%">
                  <PolarGrid stroke={isDark ? '#334155' : '#e5e7eb'} />
                  <PolarAngleAxis dataKey="region" tick={{ fontSize: 10, fill: isDark ? '#94a3b8' : '#6b7280' }} />
                  <PolarRadiusAxis tick={{ fontSize: 9, fill: isDark ? '#64748b' : '#9ca3af' }} />
                  <Radar name="金额" dataKey="amount" stroke={CHART_COLORS.primary} fill={CHART_COLORS.primary} fillOpacity={0.2} />
                  <Radar name="数量" dataKey="count" stroke={CHART_COLORS.success} fill={CHART_COLORS.success} fillOpacity={0.2} />
                  <Legend wrapperStyle={{ fontSize: 12 }} />
                  <Tooltip contentStyle={{ background: isDark ? '#1e293b' : '#fff', border: isDark ? '1px solid #334155' : '1px solid #e5e7eb', borderRadius: 8, fontSize: 12 }} />
                </RadarChart>
              </ResponsiveContainer>
            </div>

            {/* BuyType Pie */}
            <div className={`${cardClass} p-4`}>
              <h4 className={`text-sm font-semibold mb-3 ${textMain}`}>购买类型分布 (饼图)</h4>
              <ResponsiveContainer width="100%" height={280}>
                <PieChart>
                  <Pie data={stats?.buy_type_data || []} cx="50%" cy="50%" outerRadius={90} dataKey="value" nameKey="name"
                    label={({ name, percent }) => `${name} ${(percent * 100).toFixed(0)}%`}>
                    {(stats?.buy_type_data || []).map((_, i) => <Cell key={i} fill={COLORS[(i + 2) % COLORS.length]} />)}
                  </Pie>
                  <Tooltip formatter={(val) => [val, '数量']} contentStyle={{ background: isDark ? '#1e293b' : '#fff', border: isDark ? '1px solid #334155' : '1px solid #e5e7eb', borderRadius: 8, fontSize: 12 }} />
                  <Legend wrapperStyle={{ fontSize: 12 }} />
                </PieChart>
              </ResponsiveContainer>
            </div>

            {/* BizType Pie */}
            <div className={`${cardClass} p-4`}>
              <h4 className={`text-sm font-semibold mb-3 ${textMain}`}>维保/续保类型分布</h4>
              <ResponsiveContainer width="100%" height={280}>
                <PieChart>
                  <Pie data={stats?.biz_type_data || []} cx="50%" cy="50%" innerRadius={50} outerRadius={90} dataKey="value" nameKey="name"
                    label={({ name, percent }) => `${name} ${(percent * 100).toFixed(0)}%`}>
                    {(stats?.biz_type_data || []).map((_, i) => <Cell key={i} fill={COLORS[(i + 4) % COLORS.length]} />)}
                  </Pie>
                  <Tooltip formatter={(val) => [val, '数量']} contentStyle={{ background: isDark ? '#1e293b' : '#fff', border: isDark ? '1px solid #334155' : '1px solid #e5e7eb', borderRadius: 8, fontSize: 12 }} />
                  <Legend wrapperStyle={{ fontSize: 12 }} />
                </PieChart>
              </ResponsiveContainer>
            </div>

            {/* WinRate distribution bar */}
            <div className={`${cardClass} p-4`}>
              <h4 className={`text-sm font-semibold mb-3 ${textMain}`}>赢率分布 (柱状图)</h4>
              <ResponsiveContainer width="100%" height={280}>
                <BarChart data={stats?.win_rate_data || []} margin={{ top: 5, right: 20, bottom: 5, left: 10 }}>
                  <CartesianGrid strokeDasharray="3 3" stroke={isDark ? '#334155' : '#e5e7eb'} />
                  <XAxis dataKey="name" tick={{ fontSize: 11, fill: isDark ? '#94a3b8' : '#6b7280' }} />
                  <YAxis tick={{ fontSize: 11, fill: isDark ? '#94a3b8' : '#6b7280' }} />
                  <Tooltip contentStyle={{ background: isDark ? '#1e293b' : '#fff', border: isDark ? '1px solid #334155' : '1px solid #e5e7eb', borderRadius: 8, fontSize: 12 }} formatter={(val) => [val, '数量']} />
                  <Bar dataKey="value" fill={CHART_COLORS.warning} radius={[4, 4, 0, 0]} />
                </BarChart>
              </ResponsiveContainer>
            </div>

            {/* Region stacked bar (amount + service amount) */}
            <div className={`${cardClass} p-4`}>
              <h4 className={`text-sm font-semibold mb-3 ${textMain}`}>区域金额与服务金额对比</h4>
              <ResponsiveContainer width="100%" height={280}>
                <BarChart data={(stats?.radar_data || []).map(d => ({ ...d, service_amount: d.service_amount || 0 }))} margin={{ top: 5, right: 20, bottom: 5, left: 10 }}>
                  <CartesianGrid strokeDasharray="3 3" stroke={isDark ? '#334155' : '#e5e7eb'} />
                  <XAxis dataKey="region" tick={{ fontSize: 10, fill: isDark ? '#94a3b8' : '#6b7280' }} angle={-15} textAnchor="end" height={50} />
                  <YAxis tick={{ fontSize: 11, fill: isDark ? '#94a3b8' : '#6b7280' }} tickFormatter={(v) => formatAmountWan(v) + '万'} />
                  <Tooltip contentStyle={{ background: isDark ? '#1e293b' : '#fff', border: isDark ? '1px solid #334155' : '1px solid #e5e7eb', borderRadius: 8, fontSize: 12 }} formatter={(val, name) => [formatAmount(val), name === 'amount' ? '总金额' : '服务金额']} />
                  <Legend wrapperStyle={{ fontSize: 12 }} formatter={(val) => val === 'amount' ? '总金额' : '服务金额'} />
                  <Bar dataKey="amount" fill={CHART_COLORS.primary} radius={[4, 4, 0, 0]} />
                  <Bar dataKey="service_amount" fill={CHART_COLORS.info} radius={[4, 4, 0, 0]} />
                </BarChart>
              </ResponsiveContainer>
            </div>
          </div>
        </div>
      )}

      {/* ====== TAB: Data ====== */}
      {activeTab === 'data' && (
        <div className="space-y-4">
          {/* Filters */}
          <div className={`${cardClass} p-4 flex flex-wrap items-center gap-3`}>
            <Filter className={`w-4 h-4 ${textSub}`} />
            <select value={filters.status} onChange={(e) => { setFilters(f => ({...f, status: e.target.value})); setPage(1); }}
              className={`text-sm px-3 py-1.5 rounded-lg border ${isDark ? 'bg-slate-700 border-slate-600 text-slate-200' : 'bg-white border-gray-300 text-gray-700'}`}>
              <option value="">全部状态</option>
              {(filterOptions.statuses || []).map(s => <option key={s} value={s}>{s}</option>)}
            </select>
            <select value={filters.region} onChange={(e) => { setFilters(f => ({...f, region: e.target.value})); setPage(1); }}
              className={`text-sm px-3 py-1.5 rounded-lg border ${isDark ? 'bg-slate-700 border-slate-600 text-slate-200' : 'bg-white border-gray-300 text-gray-700'}`}>
              <option value="">全部区域</option>
              {(filterOptions.regions || []).map(r => <option key={r} value={r}>{r}</option>)}
            </select>
            <select value={filters.biz_type} onChange={(e) => { setFilters(f => ({...f, biz_type: e.target.value})); setPage(1); }}
              className={`text-sm px-3 py-1.5 rounded-lg border ${isDark ? 'bg-slate-700 border-slate-600 text-slate-200' : 'bg-white border-gray-300 text-gray-700'}`}>
              <option value="">全部类型</option>
              {(filterOptions.biz_types || []).map(b => <option key={b} value={b}>{b}</option>)}
            </select>
            <div className="relative flex-1 min-w-[200px]">
              <Search className={`absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 ${textMuted}`} />
              <input
                placeholder="搜索商机名/客户/负责人..."
                value={filters.search}
                onChange={(e) => { setFilters(f => ({...f, search: e.target.value})); setPage(1); }}
                className={`w-full pl-9 pr-3 py-1.5 text-sm rounded-lg border ${isDark ? 'bg-slate-700 border-slate-600 text-slate-200 placeholder-slate-500' : 'bg-white border-gray-300 text-gray-700 placeholder-gray-400'}`}
              />
            </div>
            <span className={`text-xs ${textSub}`}>共 {total} 条</span>
          </div>

          {/* Data Table */}
          <div className={`${cardClass} overflow-hidden`}>
            <div className="overflow-x-auto">
              <table className="w-full text-sm">
                <thead>
                  <tr className={isDark ? 'bg-slate-750 border-b border-slate-700' : 'bg-gray-50 border-b border-gray-200'}>
                    <th className={`px-3 py-2.5 text-left font-medium ${textSub}`}>商机名称</th>
                    <th className={`px-3 py-2.5 text-left font-medium ${textSub}`}>客户</th>
                    <th className={`px-3 py-2.5 text-left font-medium ${textSub} cursor-pointer`} onClick={() => { setSortField('amount'); setSortDir(d => d === 'desc' ? 'asc' : 'desc'); }}>
                      <span className="flex items-center gap-1">金额 <ArrowUpDown className="w-3 h-3" /></span>
                    </th>
                    <th className={`px-3 py-2.5 text-left font-medium ${textSub}`}>状态</th>
                    <th className={`px-3 py-2.5 text-left font-medium ${textSub}`}>负责人</th>
                    <th className={`px-3 py-2.5 text-left font-medium ${textSub}`}>区域</th>
                    <th className={`px-3 py-2.5 text-left font-medium ${textSub}`}>类型</th>
                    <th className={`px-3 py-2.5 text-left font-medium ${textSub}`}>购买类型</th>
                    <th className={`px-3 py-2.5 text-left font-medium ${textSub}`}>赢率</th>
                  </tr>
                </thead>
                <tbody>
                  {sortedRecords.map((r, i) => (
                    <tr key={r.id || i} className={`border-b last:border-0 ${isDark ? 'border-slate-700 hover:bg-slate-750' : 'border-gray-100 hover:bg-gray-50'} transition-colors`}>
                      <td className={`px-3 py-2 ${textMain} max-w-[250px] truncate`} title={r.name}>{r.name}</td>
                      <td className={`px-3 py-2 ${textSub} max-w-[180px] truncate`} title={r.customer}>{r.customer}</td>
                      <td className="px-3 py-2 font-medium text-primary-500">{formatAmount(r.amount)}</td>
                      <td className="px-3 py-2">
                        <span className={`text-xs px-2 py-0.5 rounded-full ${
                          r.status === '赢单' ? 'bg-green-100 text-green-700' : 'bg-blue-100 text-blue-700'
                        }`}>{r.status}</span>
                      </td>
                      <td className={`px-3 py-2 ${textSub}`}>{r.owner}</td>
                      <td className={`px-3 py-2 ${textSub}`}>{r.region}</td>
                      <td className="px-3 py-2">
                        <span className={`text-xs px-2 py-0.5 rounded-full ${
                          r.biz_type === '维保' ? 'bg-purple-100 text-purple-700' : r.biz_type === '续保' ? 'bg-orange-100 text-orange-700' : 'bg-pink-100 text-pink-700'
                        }`}>{r.biz_type}</span>
                      </td>
                      <td className={`px-3 py-2 ${textSub}`}>{r.buy_type}</td>
                      <td className={`px-3 py-2 ${textSub}`}>{r.win_rate}</td>
                    </tr>
                  ))}
                  {sortedRecords.length === 0 && (
                    <tr><td colSpan={9} className={`text-center py-8 ${textMuted}`}>暂无数据</td></tr>
                  )}
                </tbody>
              </table>
            </div>
            {/* Pagination */}
            {total > 50 && (
              <div className={`px-4 py-3 border-t ${isDark ? 'border-slate-700' : 'border-gray-200'} flex items-center justify-between`}>
                <span className={`text-xs ${textSub}`}>第 {page} 页 / 共 {Math.ceil(total / 50)} 页</span>
                <div className="flex gap-2">
                  <button disabled={page <= 1} onClick={() => setPage(p => p - 1)}
                    className={`px-3 py-1 text-sm rounded border ${isDark ? 'border-slate-600 text-slate-300 hover:bg-slate-700 disabled:opacity-40' : 'border-gray-300 text-gray-600 hover:bg-gray-50 disabled:opacity-40'}`}>上一页</button>
                  <button disabled={page >= Math.ceil(total / 50)} onClick={() => setPage(p => p + 1)}
                    className={`px-3 py-1 text-sm rounded border ${isDark ? 'border-slate-600 text-slate-300 hover:bg-slate-700 disabled:opacity-40' : 'border-gray-300 text-gray-600 hover:bg-gray-50 disabled:opacity-40'}`}>下一页</button>
                </div>
              </div>
            )}
          </div>
        </div>
      )}

      {/* ====== TAB: History ====== */}
      {activeTab === 'history' && (
        <div className={`${cardClass} overflow-hidden`}>
          <div className="overflow-x-auto">
            <table className="w-full text-sm">
              <thead>
                <tr className={isDark ? 'bg-slate-750 border-b border-slate-700' : 'bg-gray-50 border-b border-gray-200'}>
                  <th className={`px-4 py-3 text-left font-medium ${textSub}`}>文件名</th>
                  <th className={`px-4 py-3 text-left font-medium ${textSub}`}>数据月份</th>
                  <th className={`px-4 py-3 text-left font-medium ${textSub}`}>总行数</th>
                  <th className={`px-4 py-3 text-left font-medium ${textSub}`}>筛选行数</th>
                  <th className={`px-4 py-3 text-left font-medium ${textSub}`}>上传人</th>
                  <th className={`px-4 py-3 text-left font-medium ${textSub}`}>上传时间</th>
                  <th className={`px-4 py-3 text-left font-medium ${textSub}`}>操作</th>
                </tr>
              </thead>
              <tbody>
                {history.map((h) => (
                  <tr key={h.id} className={`border-b last:border-0 ${isDark ? 'border-slate-700 hover:bg-slate-750' : 'border-gray-100 hover:bg-gray-50'}`}>
                    <td className={`px-4 py-3 ${textMain}`}>{h.file_name}</td>
                    <td className={`px-4 py-3 ${textSub}`}>{h.month}</td>
                    <td className={`px-4 py-3 ${textSub}`}>{h.total_rows}</td>
                    <td className="px-4 py-3 font-medium text-primary-500">{h.filtered_rows}</td>
                    <td className={`px-4 py-3 ${textSub}`}>{h.uploaded_name}</td>
                    <td className={`px-4 py-3 ${textSub}`}>{new Date(h.created_at).toLocaleString('zh-CN')}</td>
                    <td className="px-4 py-3">
                      <button onClick={() => handleDeleteUpload(h.id)}
                        className="text-red-500 hover:text-red-700 p-1 rounded hover:bg-red-50 transition-colors"
                        title="删除该月数据"
                      >
                        <Trash2 className="w-4 h-4" />
                      </button>
                    </td>
                  </tr>
                ))}
                {history.length === 0 && (
                  <tr><td colSpan={7} className={`text-center py-12 ${textMuted}`}>
                    <History className={`w-10 h-10 mx-auto mb-2 ${textMuted}`} />
                    暂无上传记录
                  </td></tr>
                )}
              </tbody>
            </table>
          </div>
        </div>
      )}
    </div>
  );
}
