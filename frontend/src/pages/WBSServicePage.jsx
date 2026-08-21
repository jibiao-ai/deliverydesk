import React, { useState, useEffect, useCallback } from 'react';
import { FileSpreadsheet, Plus, Download, Trash2, ChevronRight, ChevronLeft, Search, Package, Wrench, Building2, CheckCircle2, X, Clock, Hash, User, MapPin, ChevronDown, Check, AlertCircle, Server, Settings } from 'lucide-react';
import { getWBSCatalog, saveWBSOrder, listWBSOrders, deleteWBSOrder } from '../services/api';

// === Step definitions for the wizard ===
const STEPS = [
  { id: 'opportunity', label: '商机信息', icon: Building2 },
  { id: 'environments', label: '环境信息', icon: Server },
  { id: 'products', label: '产品选择', icon: Package },
  { id: 'services', label: '服务选择', icon: Wrench },
  { id: 'summary', label: '汇总确认', icon: CheckCircle2 },
];

const EMPTY_OPP = {
  opportunity_name: '', opportunity_no: '', sales_order: '', contract_no: '',
  customer_name: '', agent: '', deploy_location: '',
  sales_director: '', sales_vp: '', sales: '', pre_sales: '',
  delivery_leader_email: '', project_manager_email: '',
};

const EMPTY_ENV = {
  env_name: '',
  env_type: '新建',
  product_version: 'ECF V612',
  license_type: '正式（软件永久许可）',
  arch_type: 'X86',
  sla: '7x24',
  maintenance_yr: 1,
  change_logo: false,
};

const ENV_TYPES = ['新建', '扩容', '纯服务', '升级'];
const PRODUCT_VERSIONS = ['ECF V612', 'ECNF V612', 'ECF V611', 'ECNF V611'];
const LICENSE_TYPES = ['正式（软件永久许可）', '正式（软件订阅）', '预交付', 'POC'];
const ARCH_TYPES = ['X86', 'Arm'];
const SLA_OPTIONS = ['7x24', '5x9'];

// Map product version to catalog version for filtering
function getVersionFilter(productVersion) {
  if (productVersion.includes('V612')) return 'V621';
  if (productVersion.includes('V611')) return 'V611';
  return '';
}

// === Glassmorphism + rounded card modal ===
function GlassModal({ open, onClose, title, children, maxWidth = 'max-w-lg' }) {
  if (!open) return null;
  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center p-4">
      <div className="absolute inset-0 bg-black/30 backdrop-blur-sm" onClick={onClose} />
      <div className={`relative ${maxWidth} w-full bg-white/90 backdrop-blur-xl rounded-2xl shadow-2xl border border-white/40 p-6 animate-in fade-in zoom-in-95 duration-200`}>
        <div className="flex items-center justify-between mb-4">
          <h3 className="text-lg font-semibold text-gray-800">{title}</h3>
          <button onClick={onClose} className="p-1.5 rounded-xl hover:bg-gray-100/80 transition-colors">
            <X className="w-5 h-5 text-gray-500" />
          </button>
        </div>
        {children}
      </div>
    </div>
  );
}

// === Glassmorphism select dropdown ===
function GlassSelect({ value, onChange, options, placeholder, className = '' }) {
  const [open, setOpen] = useState(false);
  const selected = options.find(o => o.value === value);

  return (
    <div className={`relative ${className}`}>
      <button
        type="button"
        onClick={() => setOpen(!open)}
        className="w-full flex items-center justify-between px-3.5 py-2.5 bg-white/70 backdrop-blur-md border border-gray-200/60 rounded-xl text-sm hover:bg-white/90 transition-all shadow-sm"
      >
        <span className={selected ? 'text-gray-800' : 'text-gray-400'}>{selected?.label || placeholder}</span>
        <ChevronDown className={`w-4 h-4 text-gray-400 transition-transform ${open ? 'rotate-180' : ''}`} />
      </button>
      {open && (
        <div className="absolute z-50 mt-1.5 w-full bg-white/95 backdrop-blur-xl rounded-xl shadow-xl border border-white/60 overflow-hidden py-1 max-h-60 overflow-y-auto">
          {options.map(opt => (
            <button
              key={opt.value}
              onClick={() => { onChange(opt.value); setOpen(false); }}
              className={`w-full text-left px-3.5 py-2 text-sm hover:bg-primary-50/70 transition-colors flex items-center gap-2 ${value === opt.value ? 'text-primary-700 bg-primary-50/50 font-medium' : 'text-gray-700'}`}
            >
              {value === opt.value && <Check className="w-3.5 h-3.5 text-primary-600" />}
              <span className={value === opt.value ? '' : 'pl-5'}>{opt.label}</span>
            </button>
          ))}
        </div>
      )}
    </div>
  );
}

// === Service category metadata for step-by-step selection ===
const SERVICE_CATEGORIES_META = [
  { id: '安装部署服务', label: '安装部署服务', desc: '标准安装、扩容服务包' },
  { id: '标准维保服务', label: '标准维保服务', desc: 'A/B类产品维保、EOS升级' },
  { id: 'EOS升级订阅', label: 'EOS升级订阅', desc: 'EOS版本大版本升级' },
  { id: '增值运维服务', label: '增值运维服务', desc: 'S1/S2/S3增值运维' },
  { id: '产品高级服务', label: '产品高级服务', desc: '巡检、安全加固、镜像制作等' },
  { id: '服务人天', label: '服务人天', desc: '远程/现场/高级工程师人天' },
  { id: '培训服务', label: '培训服务', desc: 'CKA/COA认证培训' },
];

export default function WBSServicePage() {
  const [mode, setMode] = useState('list'); // list | create
  const [step, setStep] = useState(0);
  const [catalog, setCatalog] = useState({ products: [], services: [] });
  const [opportunity, setOpportunity] = useState({ ...EMPTY_OPP });
  const [environments, setEnvironments] = useState([{ ...EMPTY_ENV, env_name: '第1套环境' }]);
  const [activeEnvIndex, setActiveEnvIndex] = useState(0);
  const [selectedProducts, setSelectedProducts] = useState({}); // { envIndex: { itemId: qty } }
  const [selectedServices, setSelectedServices] = useState({});
  const [orders, setOrders] = useState([]);
  const [totalOrders, setTotalOrders] = useState(0);
  const [page, setPage] = useState(1);
  const [loading, setLoading] = useState(false);
  const [saving, setSaving] = useState(false);
  const [deleteConfirm, setDeleteConfirm] = useState(null);
  const [jumpPage, setJumpPage] = useState('');

  // Sub-step navigation for products & services
  const [productMajorCategory, setProductMajorCategory] = useState('自有产品');
  const [productSubCategory, setProductSubCategory] = useState(null);
  const [serviceCategory, setServiceCategory] = useState(null);
  const [filterArch, setFilterArch] = useState('all');
  const [searchTerm, setSearchTerm] = useState('');
  const [hoverTooltip, setHoverTooltip] = useState({ visible: false, content: '', title: '', x: 0, y: 0 });

  const PAGE_SIZE = 10;

  // Load catalog on mount (load all, filter client-side by version)
  useEffect(() => {
    getWBSCatalog().then(res => {
      if (res?.code === 0) setCatalog(res.data);
    });
  }, []);

  // Get products filtered by current environment's version
  const getFilteredProducts = useCallback(() => {
    const env = environments[activeEnvIndex];
    if (!env) return [];
    const versionFilter = getVersionFilter(env.product_version);
    return catalog.products.filter(p => !versionFilter || p.version === versionFilter);
  }, [catalog.products, environments, activeEnvIndex]);

  // Get product sub-categories dynamically
  const productSubCategoriesMeta = React.useMemo(() => {
    const products = getFilteredProducts();
    const filtered = products.filter(p => p.major_category === productMajorCategory);
    const cats = [];
    const seen = new Set();
    for (const p of filtered) {
      const cat = p.sub_category;
      if (cat && !seen.has(cat)) {
        seen.add(cat);
        cats.push({ id: cat, label: cat });
      }
    }
    return cats;
  }, [getFilteredProducts, productMajorCategory]);

  // Derive service categories dynamically
  const serviceCategoriesMeta = React.useMemo(() => {
    const cats = [];
    const seen = new Set();
    for (const s of catalog.services) {
      const cat = s.sub_category || s.major_category;
      if (cat && !seen.has(cat)) {
        seen.add(cat);
        cats.push({ id: cat, label: cat });
      }
    }
    // Fallback to the predefined list if no data
    return cats.length > 0 ? cats : SERVICE_CATEGORIES_META;
  }, [catalog.services]);

  // Load orders list
  const loadOrders = useCallback(() => {
    setLoading(true);
    listWBSOrders({ page, page_size: PAGE_SIZE }).then(res => {
      if (res?.code === 0) {
        setOrders(res.data.orders || []);
        setTotalOrders(res.data.total || 0);
      }
    }).finally(() => setLoading(false));
  }, [page]);

  useEffect(() => { if (mode === 'list') loadOrders(); }, [mode, loadOrders]);

  const handleDeleteOrder = async (id) => {
    const res = await deleteWBSOrder(id);
    if (res?.code === 0) { setDeleteConfirm(null); loadOrders(); }
  };

  const handleExport = (id) => {
    const token = localStorage.getItem('token');
    window.open(`/api/wbs/orders/${id}/export?token=${token}`, '_blank');
  };

  const handleSaveOrder = async () => {
    setSaving(true);
    // Build products list with env_index
    const products = [];
    Object.entries(selectedProducts).forEach(([envIdx, items]) => {
      Object.entries(items).filter(([, qty]) => qty > 0).forEach(([id, qty]) => {
        const p = catalog.products.find(x => x.id === id);
        if (p) {
          products.push({
            item_id: id, name: p.name, code: p.code, quantity: qty,
            unit: p.unit, category: p.major_category, sub_category: p.sub_category,
            series: p.series, arch: p.arch, module: p.module,
            buy_product: p.buy_product, license_type: p.license_type,
            description: p.description, env_index: parseInt(envIdx) + 1
          });
        }
      });
    });

    // Build services list
    const services = Object.entries(selectedServices)
      .filter(([, qty]) => qty > 0)
      .map(([id, qty]) => {
        const s = catalog.services.find(x => x.id === id);
        return {
          item_id: id, name: s?.name, code: s?.code, quantity: qty,
          unit: s?.unit, category: s?.major_category || s?.sub_category,
          sub_category: s?.sub_category, series: s?.series,
          description: s?.description, env_index: 0
        };
      });

    const res = await saveWBSOrder({ opportunity, environments, products, services, remarks: '' });
    setSaving(false);
    if (res?.code === 0) {
      resetForm();
      setMode('list');
    }
  };

  const resetForm = () => {
    setStep(0);
    setOpportunity({ ...EMPTY_OPP });
    setEnvironments([{ ...EMPTY_ENV, env_name: '第1套环境' }]);
    setActiveEnvIndex(0);
    setSelectedProducts({});
    setSelectedServices({});
    setProductMajorCategory('自有产品');
    setProductSubCategory(null);
    setServiceCategory(null);
  };

  // Add new environment
  const addEnvironment = () => {
    const newIdx = environments.length + 1;
    setEnvironments(prev => [...prev, { ...EMPTY_ENV, env_name: `第${newIdx}套环境` }]);
  };

  // Remove environment
  const removeEnvironment = (idx) => {
    if (environments.length <= 1) return;
    setEnvironments(prev => prev.filter((_, i) => i !== idx));
    setSelectedProducts(prev => {
      const next = { ...prev };
      delete next[idx];
      return next;
    });
    if (activeEnvIndex >= environments.length - 1) {
      setActiveEnvIndex(Math.max(0, environments.length - 2));
    }
  };

  // Update environment field
  const updateEnv = (idx, field, value) => {
    setEnvironments(prev => prev.map((env, i) => i === idx ? { ...env, [field]: value } : env));
  };

  // Count selected products per environment
  const getEnvProductCount = (envIdx) => {
    const items = selectedProducts[envIdx] || {};
    return Object.values(items).filter(v => v > 0).length;
  };

  // Count total selected
  const selectedProductCount = Object.values(selectedProducts).reduce((sum, items) => 
    sum + Object.values(items).filter(v => v > 0).length, 0);
  const selectedServiceCount = Object.values(selectedServices).filter(v => v > 0).length;
  const totalPages = Math.ceil(totalOrders / PAGE_SIZE);

  // === LIST MODE ===
  if (mode === 'list') {
    return (
      <div className="h-full overflow-auto p-6 bg-gradient-to-br from-slate-50 to-blue-50/30">
        <div className="max-w-7xl mx-auto">
          {/* Header */}
          <div className="flex items-center justify-between mb-6">
            <div>
              <h2 className="text-xl font-bold text-gray-800 flex items-center gap-2">
                <FileSpreadsheet className="w-6 h-6 text-primary-600" />
                项目WBS服务
              </h2>
              <p className="text-sm text-gray-500 mt-1">Work Breakdown Structure - 工作任务分解与报价汇总</p>
            </div>
            <button
              onClick={() => { resetForm(); setMode('create'); }}
              className="flex items-center gap-2 px-5 py-2.5 bg-gradient-to-r from-primary-600 to-primary-700 text-white rounded-xl hover:from-primary-700 hover:to-primary-800 transition-all shadow-lg shadow-primary-200/50 font-medium"
            >
              <Plus className="w-4 h-4" /> 新建WBS
            </button>
          </div>

          {/* Orders Table */}
          <div className="bg-white/80 backdrop-blur-xl rounded-2xl border border-white/60 shadow-xl overflow-hidden">
            <div className="px-5 py-4 border-b border-gray-100/80 flex items-center justify-between">
              <h3 className="font-semibold text-gray-700 flex items-center gap-2">
                <Clock className="w-4 h-4 text-gray-400" />
                历史记录
              </h3>
              <span className="text-xs text-gray-400 bg-gray-100/80 px-2.5 py-1 rounded-lg">共 {totalOrders} 条记录</span>
            </div>
            <table className="w-full text-sm">
              <thead className="bg-gray-50/80">
                <tr>
                  <th className="px-5 py-3 text-left font-medium text-gray-500 text-xs uppercase tracking-wider">商机号</th>
                  <th className="px-5 py-3 text-left font-medium text-gray-500 text-xs uppercase tracking-wider">客户名称</th>
                  <th className="px-5 py-3 text-left font-medium text-gray-500 text-xs uppercase tracking-wider">商机名称</th>
                  <th className="px-5 py-3 text-center font-medium text-gray-500 text-xs uppercase tracking-wider">产品</th>
                  <th className="px-5 py-3 text-center font-medium text-gray-500 text-xs uppercase tracking-wider">服务</th>
                  <th className="px-5 py-3 text-left font-medium text-gray-500 text-xs uppercase tracking-wider">创建人</th>
                  <th className="px-5 py-3 text-left font-medium text-gray-500 text-xs uppercase tracking-wider">创建时间</th>
                  <th className="px-5 py-3 text-center font-medium text-gray-500 text-xs uppercase tracking-wider">操作</th>
                </tr>
              </thead>
              <tbody>
                {loading ? (
                  <tr><td colSpan={8} className="px-5 py-16 text-center text-gray-400">
                    <div className="animate-pulse flex flex-col items-center gap-2">
                      <div className="w-8 h-8 rounded-full border-2 border-primary-200 border-t-primary-600 animate-spin" />
                      <span className="text-sm">加载中...</span>
                    </div>
                  </td></tr>
                ) : orders.length === 0 ? (
                  <tr><td colSpan={8} className="px-5 py-16 text-center text-gray-400">
                    <FileSpreadsheet className="w-14 h-14 mx-auto mb-3 text-gray-200" />
                    <p className="text-base font-medium text-gray-400 mb-1">暂无WBS订单</p>
                    <p className="text-xs text-gray-300">点击"新建WBS"开始创建第一个项目订单</p>
                  </td></tr>
                ) : orders.map((o, idx) => (
                  <tr key={o.id} className={`border-b border-gray-50 hover:bg-primary-50/30 transition-colors ${idx % 2 === 0 ? 'bg-white/50' : 'bg-gray-50/30'}`}>
                    <td className="px-5 py-3.5">
                      <span className="font-mono text-xs bg-blue-50 text-blue-700 px-2 py-0.5 rounded-lg">{o.opportunity_no || '-'}</span>
                    </td>
                    <td className="px-5 py-3.5 font-medium text-gray-800">{o.customer_name || '-'}</td>
                    <td className="px-5 py-3.5 max-w-[200px] truncate text-gray-600" title={o.opportunity_name}>{o.opportunity_name || '-'}</td>
                    <td className="px-5 py-3.5 text-center">
                      <span className="inline-flex items-center justify-center w-7 h-7 rounded-lg bg-green-50 text-green-700 text-xs font-bold">{o.product_count || 0}</span>
                    </td>
                    <td className="px-5 py-3.5 text-center">
                      <span className="inline-flex items-center justify-center w-7 h-7 rounded-lg bg-purple-50 text-purple-700 text-xs font-bold">{o.service_count || 0}</span>
                    </td>
                    <td className="px-5 py-3.5 text-gray-600 text-xs">{o.username}</td>
                    <td className="px-5 py-3.5 text-xs text-gray-400">{new Date(o.created_at).toLocaleString('zh-CN', { month: '2-digit', day: '2-digit', hour: '2-digit', minute: '2-digit' })}</td>
                    <td className="px-5 py-3.5 text-center">
                      <div className="flex items-center justify-center gap-1">
                        <button onClick={() => handleExport(o.id)} title="导出Excel"
                          className="p-2 rounded-xl hover:bg-green-50 text-green-600 transition-colors">
                          <Download className="w-4 h-4" />
                        </button>
                        <button onClick={() => setDeleteConfirm(o.id)} title="删除"
                          className="p-2 rounded-xl hover:bg-red-50 text-red-400 transition-colors">
                          <Trash2 className="w-4 h-4" />
                        </button>
                      </div>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>

            {/* Pagination */}
            {totalOrders > 0 && (
              <div className="flex items-center justify-between px-5 py-3.5 border-t border-gray-100/80 bg-gray-50/50">
                <span className="text-xs text-gray-500">
                  第 <span className="font-medium text-gray-700">{(page - 1) * PAGE_SIZE + 1}</span> - <span className="font-medium text-gray-700">{Math.min(page * PAGE_SIZE, totalOrders)}</span> 条，共 {totalOrders} 条
                </span>
                <div className="flex items-center gap-2">
                  <button disabled={page <= 1} onClick={() => setPage(1)} className="px-2.5 py-1.5 text-xs rounded-lg border border-gray-200/80 bg-white/80 disabled:opacity-40 hover:bg-gray-50 transition-all">首页</button>
                  <button disabled={page <= 1} onClick={() => setPage(p => p - 1)} className="px-3 py-1.5 text-xs rounded-lg border border-gray-200/80 bg-white/80 disabled:opacity-40 hover:bg-gray-50 transition-all flex items-center gap-1"><ChevronLeft className="w-3 h-3" />上一页</button>
                  <div className="flex items-center gap-1">
                    {Array.from({ length: Math.min(totalPages, 7) }, (_, i) => {
                      let pageNum;
                      if (totalPages <= 7) { pageNum = i + 1; }
                      else if (page <= 4) { pageNum = i + 1; }
                      else if (page >= totalPages - 3) { pageNum = totalPages - 6 + i; }
                      else { pageNum = page - 3 + i; }
                      return (
                        <button key={pageNum} onClick={() => setPage(pageNum)}
                          className={`w-8 h-8 text-xs rounded-lg transition-all ${page === pageNum ? 'bg-primary-600 text-white shadow-md shadow-primary-200' : 'border border-gray-200/80 bg-white/80 hover:bg-gray-50 text-gray-600'}`}
                        >{pageNum}</button>
                      );
                    })}
                  </div>
                  <button disabled={page >= totalPages} onClick={() => setPage(p => p + 1)} className="px-3 py-1.5 text-xs rounded-lg border border-gray-200/80 bg-white/80 disabled:opacity-40 hover:bg-gray-50 transition-all flex items-center gap-1">下一页<ChevronRight className="w-3 h-3" /></button>
                  <button disabled={page >= totalPages} onClick={() => setPage(totalPages)} className="px-2.5 py-1.5 text-xs rounded-lg border border-gray-200/80 bg-white/80 disabled:opacity-40 hover:bg-gray-50 transition-all">末页</button>
                  <div className="flex items-center gap-1 ml-2 pl-2 border-l border-gray-200">
                    <span className="text-xs text-gray-400">跳至</span>
                    <input type="number" min={1} max={totalPages} value={jumpPage} onChange={e => setJumpPage(e.target.value)}
                      onKeyDown={e => { if (e.key === 'Enter') { const p = parseInt(jumpPage); if (p >= 1 && p <= totalPages) { setPage(p); setJumpPage(''); } } }}
                      className="w-12 px-2 py-1.5 text-xs border border-gray-200/80 rounded-lg text-center bg-white/80 focus:ring-2 focus:ring-primary-200 focus:border-primary-400" placeholder="#" />
                    <span className="text-xs text-gray-400">页</span>
                    <button onClick={() => { const p = parseInt(jumpPage); if (p >= 1 && p <= totalPages) { setPage(p); setJumpPage(''); } }}
                      className="px-2 py-1.5 text-xs rounded-lg bg-primary-50 text-primary-700 hover:bg-primary-100 transition-colors font-medium">GO</button>
                  </div>
                </div>
              </div>
            )}
          </div>
        </div>

        {/* Delete confirmation modal */}
        <GlassModal open={!!deleteConfirm} onClose={() => setDeleteConfirm(null)} title="确认删除">
          <div className="flex items-center gap-3 mb-5 p-3 bg-red-50/80 rounded-xl">
            <AlertCircle className="w-5 h-5 text-red-500 shrink-0" />
            <p className="text-sm text-red-700">此操作不可恢复，确定要删除此WBS订单吗？</p>
          </div>
          <div className="flex justify-end gap-3">
            <button onClick={() => setDeleteConfirm(null)} className="px-4 py-2 text-sm rounded-xl border border-gray-200/80 bg-white/80 hover:bg-gray-50 transition-all">取消</button>
            <button onClick={() => handleDeleteOrder(deleteConfirm)} className="px-4 py-2 text-sm rounded-xl bg-red-500 text-white hover:bg-red-600 transition-all shadow-md shadow-red-200">确认删除</button>
          </div>
        </GlassModal>
      </div>
    );
  }

  // === CREATE MODE ===
  return (
    <div className="h-full overflow-auto p-6 bg-gradient-to-br from-slate-50 to-blue-50/30">
      <div className="max-w-6xl mx-auto">
        {/* Header */}
        <div className="flex items-center justify-between mb-5">
          <div className="flex items-center gap-3">
            <button onClick={() => setMode('list')} className="flex items-center gap-1 text-sm text-gray-500 hover:text-primary-600 transition-colors px-3 py-1.5 rounded-xl hover:bg-white/80">
              <ChevronLeft className="w-4 h-4" /> 返回列表
            </button>
            <h2 className="text-lg font-bold text-gray-800">新建 WBS 订单</h2>
          </div>
          <div className="flex items-center gap-3 text-xs text-gray-400">
            {environments.length > 0 && <span className="bg-blue-50 text-blue-700 px-2.5 py-1 rounded-lg font-medium">{environments.length} 套环境</span>}
            {selectedProductCount > 0 && <span className="bg-green-50 text-green-700 px-2.5 py-1 rounded-lg font-medium">已选 {selectedProductCount} 产品</span>}
            {selectedServiceCount > 0 && <span className="bg-purple-50 text-purple-700 px-2.5 py-1 rounded-lg font-medium">已选 {selectedServiceCount} 服务</span>}
          </div>
        </div>

        {/* Steps indicator */}
        <div className="flex items-center gap-1 mb-5 bg-white/70 backdrop-blur-xl rounded-2xl border border-white/60 shadow-lg px-5 py-3">
          {STEPS.map((s, i) => {
            const Icon = s.icon;
            const isActive = i === step;
            const isDone = i < step;
            return (
              <React.Fragment key={s.id}>
                {i > 0 && <ChevronRight className="w-4 h-4 text-gray-200 mx-1" />}
                <button
                  onClick={() => setStep(i)}
                  className={`flex items-center gap-2 px-4 py-2 rounded-xl text-sm transition-all ${
                    isActive ? 'bg-primary-100/80 text-primary-700 font-semibold shadow-sm' :
                    isDone ? 'text-green-600 hover:bg-green-50/50' : 'text-gray-400 hover:bg-gray-50/50'
                  }`}
                >
                  <div className={`w-7 h-7 rounded-lg flex items-center justify-center ${
                    isActive ? 'bg-primary-600 text-white' :
                    isDone ? 'bg-green-100 text-green-600' : 'bg-gray-100 text-gray-400'
                  }`}>
                    {isDone ? <Check className="w-4 h-4" /> : <Icon className="w-4 h-4" />}
                  </div>
                  {s.label}
                </button>
              </React.Fragment>
            );
          })}
        </div>

        {/* Step Content */}
        <div className="bg-white/80 backdrop-blur-xl rounded-2xl border border-white/60 shadow-xl p-6 min-h-[520px]">
          {/* Step 0: Opportunity Info */}
          {step === 0 && (
            <div>
              <h3 className="text-base font-semibold mb-5 flex items-center gap-2">
                <Building2 className="w-5 h-5 text-primary-600" />
                商机基础信息
              </h3>
              <div className="grid grid-cols-2 gap-4 max-w-3xl">
                {[
                  ['opportunity_name', '商机名称', '请输入CRM中的商机名称', 'col-span-2'],
                  ['opportunity_no', '商机号', 'SJ-XXXXXXXXXXX（必填）', ''],
                  ['customer_name', '客户名称', '请输入客户名称（必填）', ''],
                  ['agent', '代理商', '请与CRM信息保持一致', ''],
                  ['deploy_location', '部署地点', '如：北京', ''],
                  ['sales_director', '销售总监', '销售总监姓名', ''],
                  ['sales_vp', '销售VP', '销售VP姓名', ''],
                  ['sales', '销售', '负责销售姓名', ''],
                  ['pre_sales', '售前', '负责售前姓名', ''],
                  ['project_manager_email', '项目经理邮箱', 'xxx@easystack.cn（必填）', ''],
                  ['delivery_leader_email', '区域交付Leader邮箱', 'xxx@easystack.cn', ''],
                  ['sales_order', '销售订单', '如无则留空', ''],
                  ['contract_no', '合同号', '如无则留空', ''],
                ].map(([key, label, placeholder, extra]) => (
                  <div key={key} className={extra}>
                    <label className="block text-xs font-medium text-gray-600 mb-1.5">{label}</label>
                    <input
                      className="w-full px-3.5 py-2.5 bg-white/70 backdrop-blur-md border border-gray-200/60 rounded-xl text-sm focus:ring-2 focus:ring-primary-200 focus:border-primary-400 transition-all shadow-sm placeholder:text-gray-300"
                      placeholder={placeholder}
                      value={opportunity[key] || ''}
                      onChange={e => setOpportunity(prev => ({ ...prev, [key]: e.target.value }))}
                    />
                  </div>
                ))}
              </div>
            </div>
          )}

          {/* Step 1: Environment Info */}
          {step === 1 && (
            <div>
              <div className="flex items-center justify-between mb-5">
                <h3 className="text-base font-semibold flex items-center gap-2">
                  <Server className="w-5 h-5 text-primary-600" />
                  环境信息配置
                </h3>
                <button onClick={addEnvironment}
                  className="flex items-center gap-1.5 px-4 py-2 text-sm bg-primary-50 text-primary-700 rounded-xl hover:bg-primary-100 transition-colors font-medium">
                  <Plus className="w-4 h-4" /> 新增环境
                </button>
              </div>

              {/* Environment tabs */}
              <div className="flex items-center gap-2 mb-4 border-b border-gray-100 pb-2 overflow-x-auto">
                {environments.map((env, idx) => (
                  <button key={idx} onClick={() => setActiveEnvIndex(idx)}
                    className={`relative flex items-center gap-2 px-4 py-2 text-sm rounded-xl transition-all whitespace-nowrap ${
                      activeEnvIndex === idx ? 'bg-primary-100/80 text-primary-700 font-semibold shadow-sm' : 'text-gray-500 hover:bg-gray-50'
                    }`}>
                    <Server className="w-3.5 h-3.5" />
                    {env.env_name || `第${idx + 1}套环境`}
                    {environments.length > 1 && (
                      <span onClick={(e) => { e.stopPropagation(); removeEnvironment(idx); }}
                        className="ml-1 p-0.5 rounded hover:bg-red-100 text-gray-400 hover:text-red-500">
                        <X className="w-3 h-3" />
                      </span>
                    )}
                  </button>
                ))}
              </div>

              {/* Active environment form */}
              {environments[activeEnvIndex] && (
                <div className="grid grid-cols-2 gap-4 max-w-3xl">
                  <div>
                    <label className="block text-xs font-medium text-gray-600 mb-1.5">环境名称</label>
                    <input className="w-full px-3.5 py-2.5 bg-white/70 border border-gray-200/60 rounded-xl text-sm focus:ring-2 focus:ring-primary-200 shadow-sm"
                      value={environments[activeEnvIndex].env_name}
                      onChange={e => updateEnv(activeEnvIndex, 'env_name', e.target.value)} />
                  </div>
                  <div>
                    <label className="block text-xs font-medium text-gray-600 mb-1.5">状态</label>
                    <GlassSelect value={environments[activeEnvIndex].env_type}
                      onChange={v => updateEnv(activeEnvIndex, 'env_type', v)}
                      options={ENV_TYPES.map(t => ({ value: t, label: t }))} placeholder="选择状态" />
                  </div>
                  <div>
                    <label className="block text-xs font-medium text-gray-600 mb-1.5">购买产品（版本）</label>
                    <GlassSelect value={environments[activeEnvIndex].product_version}
                      onChange={v => updateEnv(activeEnvIndex, 'product_version', v)}
                      options={PRODUCT_VERSIONS.map(t => ({ value: t, label: t }))} placeholder="选择产品版本" />
                  </div>
                  <div>
                    <label className="block text-xs font-medium text-gray-600 mb-1.5">License授权类型</label>
                    <GlassSelect value={environments[activeEnvIndex].license_type}
                      onChange={v => updateEnv(activeEnvIndex, 'license_type', v)}
                      options={LICENSE_TYPES.map(t => ({ value: t, label: t }))} placeholder="选择授权类型" />
                  </div>
                  <div>
                    <label className="block text-xs font-medium text-gray-600 mb-1.5">架构类型</label>
                    <GlassSelect value={environments[activeEnvIndex].arch_type}
                      onChange={v => updateEnv(activeEnvIndex, 'arch_type', v)}
                      options={ARCH_TYPES.map(t => ({ value: t, label: t }))} placeholder="选择架构" />
                  </div>
                  <div>
                    <label className="block text-xs font-medium text-gray-600 mb-1.5">SLA</label>
                    <GlassSelect value={environments[activeEnvIndex].sla}
                      onChange={v => updateEnv(activeEnvIndex, 'sla', v)}
                      options={SLA_OPTIONS.map(t => ({ value: t, label: t }))} placeholder="选择SLA" />
                  </div>
                  <div>
                    <label className="block text-xs font-medium text-gray-600 mb-1.5">维保年限</label>
                    <input type="number" min="1" max="10" className="w-full px-3.5 py-2.5 bg-white/70 border border-gray-200/60 rounded-xl text-sm focus:ring-2 focus:ring-primary-200 shadow-sm"
                      value={environments[activeEnvIndex].maintenance_yr}
                      onChange={e => updateEnv(activeEnvIndex, 'maintenance_yr', parseInt(e.target.value) || 1)} />
                  </div>
                  <div className="flex items-center gap-3 pt-6">
                    <label className="flex items-center gap-2 text-sm text-gray-600 cursor-pointer">
                      <input type="checkbox" className="w-4 h-4 rounded border-gray-300 text-primary-600 focus:ring-primary-500"
                        checked={environments[activeEnvIndex].change_logo}
                        onChange={e => updateEnv(activeEnvIndex, 'change_logo', e.target.checked)} />
                      是否更换Logo
                    </label>
                  </div>
                </div>
              )}

              {/* Environment summary */}
              {environments.length > 0 && (
                <div className="mt-6 p-4 bg-gradient-to-r from-blue-50/60 to-indigo-50/40 rounded-xl border border-blue-100/50">
                  <h4 className="text-xs font-semibold text-gray-500 uppercase tracking-wide mb-2">环境总览</h4>
                  <div className="grid grid-cols-1 gap-2">
                    {environments.map((env, idx) => (
                      <div key={idx} className="flex items-center gap-4 text-xs">
                        <span className="font-medium text-gray-700 w-24">{env.env_name}</span>
                        <span className="px-2 py-0.5 rounded bg-blue-100 text-blue-700">{env.env_type}</span>
                        <span className="text-gray-500">{env.product_version}</span>
                        <span className="text-gray-500">{env.arch_type}</span>
                        <span className="text-gray-500">{env.license_type}</span>
                        <span className="text-gray-500">{env.sla}</span>
                        <span className="text-gray-500">{env.maintenance_yr}年</span>
                      </div>
                    ))}
                  </div>
                </div>
              )}
            </div>
          )}

          {/* Step 2: Products - per environment selection */}
          {step === 2 && (
            <div>
              <div className="flex items-center justify-between mb-4">
                <h3 className="text-base font-semibold flex items-center gap-2">
                  <Package className="w-5 h-5 text-primary-600" />
                  选择产品
                </h3>
                <span className="text-sm text-primary-600 font-medium bg-primary-50/80 px-3 py-1 rounded-xl">已选 {selectedProductCount} 项</span>
              </div>

              {/* Environment selector for products */}
              <div className="flex items-center gap-2 mb-3 pb-2 border-b border-gray-100">
                <span className="text-xs text-gray-500 font-medium">选择环境:</span>
                {environments.map((env, idx) => (
                  <button key={idx} onClick={() => setActiveEnvIndex(idx)}
                    className={`flex items-center gap-1.5 px-3 py-1.5 text-xs rounded-lg transition-all ${
                      activeEnvIndex === idx ? 'bg-primary-600 text-white shadow-sm' : 'bg-gray-100 text-gray-600 hover:bg-gray-200'
                    }`}>
                    <Server className="w-3 h-3" />
                    {env.env_name}
                    {getEnvProductCount(idx) > 0 && <span className="bg-white/30 px-1.5 rounded text-[10px]">{getEnvProductCount(idx)}</span>}
                  </button>
                ))}
              </div>

              {/* Current environment info bar */}
              {environments[activeEnvIndex] && (
                <div className="flex items-center gap-3 mb-3 px-3 py-2 bg-blue-50/50 rounded-xl text-xs text-blue-700">
                  <Settings className="w-3.5 h-3.5" />
                  <span className="font-medium">{environments[activeEnvIndex].env_name}</span>
                  <span>|</span>
                  <span>{environments[activeEnvIndex].product_version}</span>
                  <span>{environments[activeEnvIndex].arch_type}</span>
                  <span>{environments[activeEnvIndex].license_type}</span>
                </div>
              )}

              {/* Major category tabs: 自有产品 vs 云平台增值软件及服务 */}
              <div className="flex items-center gap-1 mb-3">
                {['自有产品', '云平台增值软件及服务'].map(cat => (
                  <button key={cat} onClick={() => { setProductMajorCategory(cat); setProductSubCategory(null); setSearchTerm(''); }}
                    className={`px-4 py-2 text-sm font-medium rounded-xl transition-all ${
                      productMajorCategory === cat ? 'bg-primary-600 text-white shadow-sm' : 'bg-gray-100 text-gray-600 hover:bg-gray-200'
                    }`}>
                    {cat}
                  </button>
                ))}
              </div>

              {/* Sub-category tabs */}
              <div className="flex items-center gap-0.5 mb-3 overflow-x-auto border-b border-gray-100 pb-1">
                {productSubCategoriesMeta.map(cat => {
                  const activeCat = productSubCategory || productSubCategoriesMeta[0]?.id;
                  return (
                    <button key={cat.id} onClick={() => { setProductSubCategory(cat.id); setSearchTerm(''); }}
                      className={`relative whitespace-nowrap px-3 py-2 text-xs font-medium transition-all ${
                        activeCat === cat.id ? 'text-primary-700' : 'text-gray-500 hover:text-gray-700'
                      }`}>
                      {cat.label}
                      {activeCat === cat.id && <div className="absolute bottom-0 left-1 right-1 h-0.5 bg-primary-600 rounded-full" />}
                    </button>
                  );
                })}
              </div>

              {/* Filters */}
              <div className="flex items-center gap-3 mb-3">
                <div className="relative flex-1 max-w-sm">
                  <Search className="absolute left-3.5 top-2.5 w-4 h-4 text-gray-300" />
                  <input className="w-full pl-10 pr-3.5 py-2.5 bg-white/70 border border-gray-200/60 rounded-xl text-sm shadow-sm focus:ring-2 focus:ring-primary-200 placeholder:text-gray-300"
                    placeholder="搜索产品名称或编码..." value={searchTerm} onChange={e => setSearchTerm(e.target.value)} />
                </div>
                <GlassSelect value={filterArch} onChange={setFilterArch}
                  options={[{ value: 'all', label: '全部架构' }, { value: 'X86', label: 'X86' }, { value: 'Arm', label: 'Arm' }]}
                  placeholder="架构" className="w-36" />
              </div>

              {/* Product list */}
              <div className="space-y-2 max-h-[320px] overflow-y-auto pr-1">
                {getFilteredProducts()
                  .filter(p => p.major_category === productMajorCategory)
                  .filter(p => p.sub_category === (productSubCategory || productSubCategoriesMeta[0]?.id))
                  .filter(p => !searchTerm || p.name.includes(searchTerm) || p.code.includes(searchTerm))
                  .filter(p => filterArch === 'all' || p.arch === filterArch || !p.arch)
                  .map(p => {
                    const envProducts = selectedProducts[activeEnvIndex] || {};
                    const qty = envProducts[p.id] || 0;
                    return (
                      <div key={p.id}
                        className={`relative flex items-center gap-4 p-3.5 rounded-xl border transition-all ${
                          qty > 0 ? 'border-primary-200 bg-primary-50/50 shadow-sm' : 'border-gray-100/80 bg-white/50 hover:bg-white/80 hover:border-gray-200'
                        }`}
                        onMouseEnter={e => {
                          if (p.description) {
                            const rect = e.currentTarget.getBoundingClientRect();
                            setHoverTooltip({ visible: true, content: p.description, title: p.name, x: rect.left + 16, y: rect.bottom + 4 });
                          }
                        }}
                        onMouseLeave={() => setHoverTooltip(prev => ({ ...prev, visible: false }))}
                      >
                        <div className="flex-1 min-w-0">
                          <div className="flex items-center gap-2 mb-0.5">
                            <span className="font-medium text-sm text-gray-800 truncate">{p.name}</span>
                            {p.arch && <span className={`text-[10px] px-1.5 py-0.5 rounded-md font-medium ${p.arch === 'X86' ? 'bg-blue-100/80 text-blue-700' : 'bg-purple-100/80 text-purple-700'}`}>{p.arch}</span>}
                            {p.module && <span className="text-[10px] px-1.5 py-0.5 rounded-md bg-amber-50 text-amber-700">{p.module}</span>}
                          </div>
                          <div className="flex items-center gap-3 text-xs text-gray-400">
                            <span className="font-mono">{p.code}</span>
                            <span>|</span>
                            <span>{p.unit}</span>
                            {p.description && <span className="text-primary-400 cursor-help">i</span>}
                          </div>
                        </div>
                        <div className="flex items-center gap-2">
                          <button onClick={() => setSelectedProducts(prev => ({
                            ...prev, [activeEnvIndex]: { ...(prev[activeEnvIndex] || {}), [p.id]: Math.max(0, qty - 1) }
                          }))} className="w-7 h-7 rounded-lg border border-gray-200/80 bg-white/80 flex items-center justify-center text-gray-500 hover:bg-gray-50 text-lg font-light">-</button>
                          <input type="number" min="0"
                            className="w-14 px-2 py-1.5 border border-gray-200/60 rounded-xl text-center text-sm bg-white/70 focus:ring-2 focus:ring-primary-200"
                            value={qty}
                            onChange={e => setSelectedProducts(prev => ({
                              ...prev, [activeEnvIndex]: { ...(prev[activeEnvIndex] || {}), [p.id]: Math.max(0, parseInt(e.target.value) || 0) }
                            }))} />
                          <button onClick={() => setSelectedProducts(prev => ({
                            ...prev, [activeEnvIndex]: { ...(prev[activeEnvIndex] || {}), [p.id]: qty + 1 }
                          }))} className="w-7 h-7 rounded-lg border border-primary-200/80 bg-primary-50/80 flex items-center justify-center text-primary-600 hover:bg-primary-100 text-lg font-light">+</button>
                        </div>
                      </div>
                    );
                  })}
              </div>
            </div>
          )}

          {/* Step 3: Services */}
          {step === 3 && (
            <div>
              <div className="flex items-center justify-between mb-4">
                <h3 className="text-base font-semibold flex items-center gap-2">
                  <Wrench className="w-5 h-5 text-primary-600" />
                  选择服务
                </h3>
                <span className="text-sm text-purple-600 font-medium bg-purple-50/80 px-3 py-1 rounded-xl">已选 {selectedServiceCount} 项</span>
              </div>

              {/* Service category tabs */}
              <div className="flex items-center gap-0.5 mb-4 border-b border-gray-100 overflow-x-auto">
                {SERVICE_CATEGORIES_META.map(cat => {
                  const count = catalog.services.filter(s => (s.sub_category || s.major_category) === cat.id).length;
                  if (count === 0) return null;
                  const activeCat = serviceCategory || SERVICE_CATEGORIES_META[0].id;
                  const selected = Object.entries(selectedServices).filter(([id, qty]) => qty > 0 && (catalog.services.find(s => s.id === id)?.sub_category || catalog.services.find(s => s.id === id)?.major_category) === cat.id).length;
                  return (
                    <button key={cat.id} onClick={() => { setServiceCategory(cat.id); setSearchTerm(''); }}
                      className={`relative whitespace-nowrap px-3.5 py-2.5 text-xs font-medium transition-all ${
                        activeCat === cat.id ? 'text-purple-700' : 'text-gray-500 hover:text-gray-700'
                      }`}>
                      <span>{cat.label}</span>
                      {selected > 0 && <span className="ml-1 text-[10px] bg-green-100 text-green-700 px-1.5 py-0.5 rounded-full font-medium">{selected}</span>}
                      {activeCat === cat.id && <div className="absolute bottom-0 left-2 right-2 h-0.5 bg-purple-600 rounded-full" />}
                    </button>
                  );
                })}
              </div>

              {/* Search */}
              <div className="relative max-w-sm mb-3">
                <Search className="absolute left-3.5 top-2.5 w-4 h-4 text-gray-300" />
                <input className="w-full pl-10 pr-3.5 py-2.5 bg-white/70 border border-gray-200/60 rounded-xl text-sm shadow-sm focus:ring-2 focus:ring-primary-200 placeholder:text-gray-300"
                  placeholder="搜索服务名称或编码..." value={searchTerm} onChange={e => setSearchTerm(e.target.value)} />
              </div>

              {/* Service list */}
              <div className="space-y-2 max-h-[360px] overflow-y-auto pr-1">
                {catalog.services
                  .filter(s => (s.sub_category || s.major_category) === (serviceCategory || SERVICE_CATEGORIES_META[0].id))
                  .filter(s => !searchTerm || s.name.includes(searchTerm) || s.code.includes(searchTerm))
                  .map(s => (
                    <div key={s.id}
                      className={`relative flex items-center gap-4 p-3.5 rounded-xl border transition-all ${
                        selectedServices[s.id] > 0 ? 'border-purple-200 bg-purple-50/50 shadow-sm' : 'border-gray-100/80 bg-white/50 hover:bg-white/80 hover:border-gray-200'
                      }`}
                      onMouseEnter={e => {
                        if (s.description) {
                          const rect = e.currentTarget.getBoundingClientRect();
                          setHoverTooltip({ visible: true, content: s.description, title: s.name, x: rect.left + 16, y: rect.bottom + 4 });
                        }
                      }}
                      onMouseLeave={() => setHoverTooltip(prev => ({ ...prev, visible: false }))}
                    >
                      <div className="flex-1 min-w-0">
                        <div className="font-medium text-sm text-gray-800 mb-0.5 truncate">{s.name}</div>
                        <div className="flex items-center gap-3 text-xs text-gray-400">
                          <span className="font-mono">{s.code}</span>
                          <span>|</span>
                          <span>{s.unit}</span>
                          {s.description && <span className="text-purple-400 cursor-help">i</span>}
                        </div>
                      </div>
                      <div className="flex items-center gap-2">
                        <button onClick={() => setSelectedServices(prev => ({ ...prev, [s.id]: Math.max(0, (prev[s.id] || 0) - 1) }))}
                          className="w-7 h-7 rounded-lg border border-gray-200/80 bg-white/80 flex items-center justify-center text-gray-500 hover:bg-gray-50 text-lg font-light">-</button>
                        <input type="number" min="0"
                          className="w-14 px-2 py-1.5 border border-gray-200/60 rounded-xl text-center text-sm bg-white/70 focus:ring-2 focus:ring-purple-200"
                          value={selectedServices[s.id] || 0}
                          onChange={e => setSelectedServices(prev => ({ ...prev, [s.id]: Math.max(0, parseInt(e.target.value) || 0) }))} />
                        <button onClick={() => setSelectedServices(prev => ({ ...prev, [s.id]: (prev[s.id] || 0) + 1 }))}
                          className="w-7 h-7 rounded-lg border border-purple-200/80 bg-purple-50/80 flex items-center justify-center text-purple-600 hover:bg-purple-100 text-lg font-light">+</button>
                      </div>
                    </div>
                  ))}
              </div>
            </div>
          )}

          {/* Step 4: Summary */}
          {step === 4 && (
            <div>
              <h3 className="text-base font-semibold mb-5 flex items-center gap-2">
                <CheckCircle2 className="w-5 h-5 text-green-600" />
                Order 汇总确认
              </h3>

              {/* Opportunity Summary */}
              <div className="mb-4 p-4 bg-gradient-to-r from-blue-50/60 to-indigo-50/40 rounded-xl border border-blue-100/50">
                <h4 className="text-xs font-semibold text-gray-500 uppercase tracking-wide mb-3 flex items-center gap-1.5">
                  <Building2 className="w-3.5 h-3.5" /> 商机信息
                </h4>
                <div className="grid grid-cols-3 gap-x-6 gap-y-2 text-sm">
                  <div><span className="text-gray-400 text-xs">商机号: </span><span className="text-gray-700 font-mono">{opportunity.opportunity_no || '-'}</span></div>
                  <div><span className="text-gray-400 text-xs">客户: </span><span className="text-gray-700 font-medium">{opportunity.customer_name || '-'}</span></div>
                  <div><span className="text-gray-400 text-xs">部署地点: </span><span className="text-gray-700">{opportunity.deploy_location || '-'}</span></div>
                  <div className="col-span-3"><span className="text-gray-400 text-xs">商机名称: </span><span className="text-gray-700">{opportunity.opportunity_name || '-'}</span></div>
                </div>
              </div>

              {/* Environments Summary */}
              <div className="mb-4 p-4 bg-gradient-to-r from-green-50/60 to-emerald-50/40 rounded-xl border border-green-100/50">
                <h4 className="text-xs font-semibold text-gray-500 uppercase tracking-wide mb-3 flex items-center gap-1.5">
                  <Server className="w-3.5 h-3.5" /> 环境信息 ({environments.length} 套)
                </h4>
                <div className="space-y-2">
                  {environments.map((env, idx) => (
                    <div key={idx} className="flex items-center gap-3 text-xs bg-white/60 rounded-lg p-2">
                      <span className="font-medium text-gray-700">{env.env_name}</span>
                      <span className="px-1.5 py-0.5 rounded bg-blue-100 text-blue-700">{env.env_type}</span>
                      <span className="text-gray-600">{env.product_version}</span>
                      <span className="text-gray-600">{env.arch_type}</span>
                      <span className="text-gray-600">{env.sla}</span>
                      <span className="text-gray-600">{env.maintenance_yr}年维保</span>
                      {getEnvProductCount(idx) > 0 && <span className="ml-auto bg-green-100 text-green-700 px-1.5 py-0.5 rounded">{getEnvProductCount(idx)} 个产品</span>}
                    </div>
                  ))}
                </div>
              </div>

              {/* Products Summary per environment */}
              {selectedProductCount > 0 && (
                <div className="mb-4">
                  <h4 className="text-xs font-semibold text-gray-500 uppercase tracking-wide mb-2 flex items-center gap-1.5">
                    <Package className="w-3.5 h-3.5" /> 产品明细 ({selectedProductCount} 项)
                  </h4>
                  {environments.map((env, envIdx) => {
                    const envItems = selectedProducts[envIdx] || {};
                    const items = Object.entries(envItems).filter(([, q]) => q > 0);
                    if (items.length === 0) return null;
                    return (
                      <div key={envIdx} className="mb-3">
                        <div className="text-xs font-medium text-gray-600 mb-1 flex items-center gap-2">
                          <Server className="w-3 h-3" /> {env.env_name} <span className="text-gray-400">({env.product_version})</span>
                        </div>
                        <div className="border border-gray-100/80 rounded-xl overflow-hidden bg-white/60">
                          <table className="w-full text-xs">
                            <thead className="bg-gray-50/80">
                              <tr>
                                <th className="px-3 py-2 text-left font-medium text-gray-500">产品名称</th>
                                <th className="px-3 py-2 text-left font-medium text-gray-500">编码</th>
                                <th className="px-3 py-2 text-center font-medium text-gray-500">架构</th>
                                <th className="px-3 py-2 text-center font-medium text-gray-500">数量</th>
                                <th className="px-3 py-2 text-center font-medium text-gray-500">单位</th>
                              </tr>
                            </thead>
                            <tbody>
                              {items.map(([id, qty]) => {
                                const p = catalog.products.find(x => x.id === id);
                                if (!p) return null;
                                return (
                                  <tr key={id} className="border-t border-gray-50">
                                    <td className="px-3 py-1.5 text-gray-700">{p.name}</td>
                                    <td className="px-3 py-1.5 font-mono text-gray-500">{p.code}</td>
                                    <td className="px-3 py-1.5 text-center"><span className={`px-1.5 py-0.5 rounded text-[10px] ${p.arch === 'X86' ? 'bg-blue-50 text-blue-700' : 'bg-purple-50 text-purple-700'}`}>{p.arch || '-'}</span></td>
                                    <td className="px-3 py-1.5 text-center font-bold text-primary-600">{qty}</td>
                                    <td className="px-3 py-1.5 text-center text-gray-500">{p.unit}</td>
                                  </tr>
                                );
                              })}
                            </tbody>
                          </table>
                        </div>
                      </div>
                    );
                  })}
                </div>
              )}

              {/* Services Summary */}
              {selectedServiceCount > 0 && (
                <div className="mb-4">
                  <h4 className="text-xs font-semibold text-gray-500 uppercase tracking-wide mb-2 flex items-center gap-1.5">
                    <Wrench className="w-3.5 h-3.5" /> 服务明细 ({selectedServiceCount} 项)
                  </h4>
                  <div className="border border-gray-100/80 rounded-xl overflow-hidden bg-white/60">
                    <table className="w-full text-xs">
                      <thead className="bg-gray-50/80">
                        <tr>
                          <th className="px-4 py-2.5 text-left font-medium text-gray-500">服务名称</th>
                          <th className="px-4 py-2.5 text-left font-medium text-gray-500">服务编码</th>
                          <th className="px-4 py-2.5 text-center font-medium text-gray-500">数量</th>
                          <th className="px-4 py-2.5 text-center font-medium text-gray-500">单位</th>
                        </tr>
                      </thead>
                      <tbody>
                        {Object.entries(selectedServices).filter(([,q]) => q > 0).map(([id, qty]) => {
                          const s = catalog.services.find(x => x.id === id);
                          if (!s) return null;
                          return (
                            <tr key={id} className="border-t border-gray-50">
                              <td className="px-4 py-2 text-gray-700">{s.name}</td>
                              <td className="px-4 py-2 font-mono text-gray-500">{s.code}</td>
                              <td className="px-4 py-2 text-center font-bold text-purple-600">{qty}</td>
                              <td className="px-4 py-2 text-center text-gray-500">{s.unit}</td>
                            </tr>
                          );
                        })}
                      </tbody>
                    </table>
                  </div>
                </div>
              )}

              {selectedProductCount === 0 && selectedServiceCount === 0 && (
                <div className="text-center py-12 text-gray-400">
                  <Package className="w-12 h-12 mx-auto mb-3 text-gray-200" />
                  <p className="text-sm">尚未选择任何产品或服务</p>
                  <p className="text-xs mt-1 text-gray-300">请返回上一步进行选择</p>
                </div>
              )}
            </div>
          )}
        </div>

        {/* Footer Navigation */}
        <div className="flex items-center justify-between mt-5">
          <button
            onClick={() => { setStep(s => Math.max(0, s - 1)); setSearchTerm(''); }}
            disabled={step === 0}
            className="flex items-center gap-1.5 px-5 py-2.5 text-sm border border-gray-200/80 bg-white/80 backdrop-blur-sm rounded-xl disabled:opacity-40 hover:bg-white hover:shadow-sm transition-all"
          >
            <ChevronLeft className="w-4 h-4" /> 上一步
          </button>
          <div className="flex gap-3">
            {step < 4 ? (
              <button
                onClick={() => { setStep(s => Math.min(4, s + 1)); setSearchTerm(''); }}
                className="flex items-center gap-1.5 px-5 py-2.5 text-sm bg-gradient-to-r from-primary-600 to-primary-700 text-white rounded-xl hover:from-primary-700 hover:to-primary-800 shadow-lg shadow-primary-200/50 transition-all font-medium"
              >
                下一步 <ChevronRight className="w-4 h-4" />
              </button>
            ) : (
              <button
                onClick={handleSaveOrder}
                disabled={saving || (!opportunity.customer_name && !opportunity.opportunity_no)}
                className="flex items-center gap-2 px-6 py-2.5 text-sm bg-gradient-to-r from-green-600 to-emerald-600 text-white rounded-xl hover:from-green-700 hover:to-emerald-700 disabled:opacity-50 shadow-lg shadow-green-200/50 transition-all font-medium"
              >
                {saving ? (
                  <>
                    <div className="w-4 h-4 rounded-full border-2 border-white/40 border-t-white animate-spin" />
                    保存中...
                  </>
                ) : (
                  <>保存并生成Order <FileSpreadsheet className="w-4 h-4" /></>
                )}
              </button>
            )}
          </div>
        </div>

        {/* Floating tooltip portal */}
        {hoverTooltip.visible && (
          <div className="fixed z-[9999] pointer-events-none"
            style={{ left: hoverTooltip.x, top: hoverTooltip.y, maxWidth: '420px' }}>
            <div className="p-3 bg-gray-900/95 backdrop-blur-md text-white text-xs rounded-xl shadow-2xl leading-relaxed border border-white/10">
              <div className="font-medium text-white/90 mb-1 text-sm">{hoverTooltip.title}</div>
              <div className="text-white/80">{hoverTooltip.content}</div>
            </div>
          </div>
        )}
      </div>
    </div>
  );
}
