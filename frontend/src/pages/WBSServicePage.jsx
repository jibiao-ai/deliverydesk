import React, { useState, useEffect, useCallback } from 'react';
import { FileSpreadsheet, Plus, Download, Trash2, Eye, ChevronRight, ChevronLeft, Search, Package, Wrench, Building2, CheckCircle2 } from 'lucide-react';
import { getWBSCatalog, saveWBSOrder, listWBSOrders, deleteWBSOrder } from '../services/api';

const STEPS = [
  { id: 'opportunity', label: '商机信息', icon: Building2 },
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

export default function WBSServicePage() {
  const [mode, setMode] = useState('list'); // list | create
  const [step, setStep] = useState(0);
  const [catalog, setCatalog] = useState({ products: [], services: [] });
  const [opportunity, setOpportunity] = useState({ ...EMPTY_OPP });
  const [selectedProducts, setSelectedProducts] = useState({});
  const [selectedServices, setSelectedServices] = useState({});
  const [orders, setOrders] = useState([]);
  const [totalOrders, setTotalOrders] = useState(0);
  const [page, setPage] = useState(1);
  const [loading, setLoading] = useState(false);
  const [searchProduct, setSearchProduct] = useState('');
  const [searchService, setSearchService] = useState('');
  const [filterArch, setFilterArch] = useState('all');
  const [filterCategory, setFilterCategory] = useState('all');
  const [saving, setSaving] = useState(false);

  // Load catalog on mount
  useEffect(() => {
    getWBSCatalog().then(res => {
      if (res?.code === 0) setCatalog(res.data);
    });
  }, []);

  // Load orders list
  const loadOrders = useCallback(() => {
    setLoading(true);
    listWBSOrders({ page, page_size: 10 }).then(res => {
      if (res?.code === 0) {
        setOrders(res.data.orders || []);
        setTotalOrders(res.data.total || 0);
      }
    }).finally(() => setLoading(false));
  }, [page]);

  useEffect(() => { if (mode === 'list') loadOrders(); }, [mode, loadOrders]);

  const handleDeleteOrder = async (id) => {
    if (!confirm('确定删除此WBS订单？')) return;
    const res = await deleteWBSOrder(id);
    if (res?.code === 0) loadOrders();
  };

  const handleExport = (id) => {
    const token = localStorage.getItem('token');
    window.open(`/api/wbs/orders/${id}/export?token=${token}`, '_blank');
  };

  const handleSaveOrder = async () => {
    setSaving(true);
    const products = Object.entries(selectedProducts)
      .filter(([, qty]) => qty > 0)
      .map(([id, qty]) => {
        const p = catalog.products.find(x => x.id === id);
        return { item_id: id, name: p?.name, code: p?.code, quantity: qty, unit: p?.unit, category: p?.category, arch: p?.arch };
      });
    const services = Object.entries(selectedServices)
      .filter(([, qty]) => qty > 0)
      .map(([id, qty]) => {
        const s = catalog.services.find(x => x.id === id);
        return { item_id: id, name: s?.name, code: s?.code, quantity: qty, unit: s?.unit, category: s?.category };
      });

    const res = await saveWBSOrder({ opportunity, products, services, remarks: '' });
    setSaving(false);
    if (res?.code === 0) {
      alert('WBS订单保存成功！');
      resetForm();
      setMode('list');
    } else {
      alert('保存失败: ' + (res?.message || '未知错误'));
    }
  };

  const resetForm = () => {
    setStep(0);
    setOpportunity({ ...EMPTY_OPP });
    setSelectedProducts({});
    setSelectedServices({});
  };

  // Filter products
  const filteredProducts = catalog.products.filter(p => {
    if (searchProduct && !p.name.includes(searchProduct) && !p.code.includes(searchProduct)) return false;
    if (filterArch !== 'all' && p.arch !== filterArch) return false;
    if (filterCategory !== 'all' && p.category !== filterCategory) return false;
    return true;
  });

  // Filter services
  const filteredServices = catalog.services.filter(s => {
    if (searchService && !s.name.includes(searchService) && !s.code.includes(searchService)) return false;
    return true;
  });

  // Get unique categories
  const productCategories = [...new Set(catalog.products.map(p => p.category))];
  const serviceCategories = [...new Set(catalog.services.map(s => s.category))];

  // Count selected
  const selectedProductCount = Object.values(selectedProducts).filter(v => v > 0).length;
  const selectedServiceCount = Object.values(selectedServices).filter(v => v > 0).length;

  if (mode === 'list') {
    return (
      <div className="h-full overflow-auto p-6">
        <div className="max-w-7xl mx-auto">
          <div className="flex items-center justify-between mb-6">
            <div>
              <h2 className="text-xl font-bold text-gray-800">项目WBS服务</h2>
              <p className="text-sm text-gray-500 mt-1">Work Breakdown Structure - 工作任务分解与报价汇总</p>
            </div>
            <button
              onClick={() => { resetForm(); setMode('create'); }}
              className="flex items-center gap-2 px-4 py-2 bg-primary-600 text-white rounded-lg hover:bg-primary-700 transition-colors"
            >
              <Plus className="w-4 h-4" /> 新建WBS
            </button>
          </div>

          {/* Orders Table */}
          <div className="bg-white rounded-xl border border-gray-200 overflow-hidden">
            <table className="w-full text-sm">
              <thead className="bg-gray-50 border-b">
                <tr>
                  <th className="px-4 py-3 text-left font-medium text-gray-600">商机号</th>
                  <th className="px-4 py-3 text-left font-medium text-gray-600">客户名称</th>
                  <th className="px-4 py-3 text-left font-medium text-gray-600">商机名称</th>
                  <th className="px-4 py-3 text-center font-medium text-gray-600">产品项</th>
                  <th className="px-4 py-3 text-center font-medium text-gray-600">服务项</th>
                  <th className="px-4 py-3 text-left font-medium text-gray-600">创建人</th>
                  <th className="px-4 py-3 text-left font-medium text-gray-600">创建时间</th>
                  <th className="px-4 py-3 text-center font-medium text-gray-600">操作</th>
                </tr>
              </thead>
              <tbody>
                {orders.length === 0 ? (
                  <tr><td colSpan={8} className="px-4 py-12 text-center text-gray-400">
                    <FileSpreadsheet className="w-12 h-12 mx-auto mb-3 text-gray-300" />
                    暂无WBS订单，点击"新建WBS"开始
                  </td></tr>
                ) : orders.map(o => (
                  <tr key={o.id} className="border-b hover:bg-gray-50">
                    <td className="px-4 py-3 font-mono text-xs">{o.opportunity_no || '-'}</td>
                    <td className="px-4 py-3">{o.customer_name}</td>
                    <td className="px-4 py-3 max-w-[200px] truncate">{o.opportunity_name}</td>
                    <td className="px-4 py-3 text-center">{o.product_count}</td>
                    <td className="px-4 py-3 text-center">{o.service_count}</td>
                    <td className="px-4 py-3">{o.username}</td>
                    <td className="px-4 py-3 text-xs text-gray-500">{new Date(o.created_at).toLocaleDateString()}</td>
                    <td className="px-4 py-3 text-center">
                      <div className="flex items-center justify-center gap-1">
                        <button onClick={() => handleExport(o.id)} title="导出Excel" className="p-1.5 rounded hover:bg-green-50 text-green-600">
                          <Download className="w-4 h-4" />
                        </button>
                        <button onClick={() => handleDeleteOrder(o.id)} title="删除" className="p-1.5 rounded hover:bg-red-50 text-red-500">
                          <Trash2 className="w-4 h-4" />
                        </button>
                      </div>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
            {totalOrders > 10 && (
              <div className="flex items-center justify-between px-4 py-3 border-t bg-gray-50">
                <span className="text-xs text-gray-500">共 {totalOrders} 条</span>
                <div className="flex gap-1">
                  <button disabled={page <= 1} onClick={() => setPage(p => p - 1)} className="px-3 py-1 text-xs rounded border disabled:opacity-40">上一页</button>
                  <button disabled={page * 10 >= totalOrders} onClick={() => setPage(p => p + 1)} className="px-3 py-1 text-xs rounded border disabled:opacity-40">下一页</button>
                </div>
              </div>
            )}
          </div>
        </div>
      </div>
    );
  }

  // === CREATE MODE ===
  return (
    <div className="h-full overflow-auto p-6">
      <div className="max-w-6xl mx-auto">
        {/* Header */}
        <div className="flex items-center justify-between mb-6">
          <div className="flex items-center gap-3">
            <button onClick={() => setMode('list')} className="text-sm text-gray-500 hover:text-primary-600">&larr; 返回列表</button>
            <h2 className="text-lg font-bold text-gray-800">新建 WBS 订单</h2>
          </div>
        </div>

        {/* Steps */}
        <div className="flex items-center gap-2 mb-6 bg-white rounded-xl border px-4 py-3">
          {STEPS.map((s, i) => {
            const Icon = s.icon;
            const isActive = i === step;
            const isDone = i < step;
            return (
              <React.Fragment key={s.id}>
                {i > 0 && <ChevronRight className="w-4 h-4 text-gray-300" />}
                <button
                  onClick={() => setStep(i)}
                  className={`flex items-center gap-2 px-3 py-1.5 rounded-lg text-sm transition-colors ${
                    isActive ? 'bg-primary-50 text-primary-700 font-medium' :
                    isDone ? 'text-green-600' : 'text-gray-400'
                  }`}
                >
                  <Icon className="w-4 h-4" />
                  {s.label}
                  {isDone && <CheckCircle2 className="w-3.5 h-3.5 text-green-500" />}
                </button>
              </React.Fragment>
            );
          })}
        </div>

        {/* Step Content */}
        <div className="bg-white rounded-xl border p-6 min-h-[500px]">
          {/* Step 0: Opportunity Info */}
          {step === 0 && (
            <div>
              <h3 className="text-base font-semibold mb-4">商机基础信息</h3>
              <div className="grid grid-cols-2 gap-4 max-w-3xl">
                {[
                  ['opportunity_name', '商机名称', '请输入CRM中的商机名称'],
                  ['opportunity_no', '商机号', 'SJ-XXXXXXXXXXX（必填）'],
                  ['customer_name', '客户名称', '请输入客户名称（必填）'],
                  ['agent', '代理商', '请与CRM信息保持一致'],
                  ['deploy_location', '部署地点', '如：北京'],
                  ['sales', '销售', '负责销售姓名'],
                  ['pre_sales', '售前', '负责售前姓名'],
                  ['project_manager_email', '项目经理邮箱', 'xxx@easystack.cn（必填）'],
                  ['delivery_leader_email', '区域交付Leader邮箱', 'xxx@easystack.cn'],
                  ['sales_order', '销售订单', '如无则留空'],
                  ['contract_no', '合同号', '如无则留空'],
                ].map(([key, label, placeholder]) => (
                  <div key={key}>
                    <label className="block text-xs font-medium text-gray-600 mb-1">{label}</label>
                    <input
                      className="w-full px-3 py-2 border rounded-lg text-sm focus:ring-2 focus:ring-primary-500 focus:border-primary-500"
                      placeholder={placeholder}
                      value={opportunity[key] || ''}
                      onChange={e => setOpportunity(prev => ({ ...prev, [key]: e.target.value }))}
                    />
                  </div>
                ))}
              </div>
            </div>
          )}

          {/* Step 1: Products */}
          {step === 1 && (
            <div>
              <div className="flex items-center justify-between mb-4">
                <h3 className="text-base font-semibold">选择产品及数量</h3>
                <span className="text-sm text-primary-600 font-medium">已选 {selectedProductCount} 项</span>
              </div>
              {/* Filters */}
              <div className="flex items-center gap-3 mb-4">
                <div className="relative flex-1 max-w-xs">
                  <Search className="absolute left-3 top-2.5 w-4 h-4 text-gray-400" />
                  <input
                    className="w-full pl-9 pr-3 py-2 border rounded-lg text-sm"
                    placeholder="搜索产品名称或编码..."
                    value={searchProduct}
                    onChange={e => setSearchProduct(e.target.value)}
                  />
                </div>
                <select className="px-3 py-2 border rounded-lg text-sm" value={filterArch} onChange={e => setFilterArch(e.target.value)}>
                  <option value="all">全部架构</option>
                  <option value="X86">X86</option>
                  <option value="Arm">Arm</option>
                </select>
                <select className="px-3 py-2 border rounded-lg text-sm" value={filterCategory} onChange={e => setFilterCategory(e.target.value)}>
                  <option value="all">全部产品大类</option>
                  {productCategories.map(c => <option key={c} value={c}>{c}</option>)}
                </select>
              </div>
              {/* Products Table */}
              <div className="border rounded-lg overflow-hidden max-h-[400px] overflow-y-auto">
                <table className="w-full text-sm">
                  <thead className="bg-gray-50 border-b sticky top-0">
                    <tr>
                      <th className="px-3 py-2 text-left w-[300px]">产品名称</th>
                      <th className="px-3 py-2 text-left w-[100px]">产品编码</th>
                      <th className="px-3 py-2 text-center w-[60px]">架构</th>
                      <th className="px-3 py-2 text-center w-[60px]">类型</th>
                      <th className="px-3 py-2 text-center w-[80px]">单位</th>
                      <th className="px-3 py-2 text-center w-[100px]">数量</th>
                    </tr>
                  </thead>
                  <tbody>
                    {filteredProducts.map(p => (
                      <tr key={p.id} className={`border-b hover:bg-blue-50/50 ${selectedProducts[p.id] > 0 ? 'bg-green-50/50' : ''}`}>
                        <td className="px-3 py-2">
                          <div className="font-medium text-xs leading-tight">{p.name}</div>
                          <div className="text-[10px] text-gray-400 mt-0.5 line-clamp-1">{p.description}</div>
                        </td>
                        <td className="px-3 py-2 font-mono text-xs text-gray-600">{p.code}</td>
                        <td className="px-3 py-2 text-center">
                          <span className={`text-xs px-1.5 py-0.5 rounded ${p.arch === 'X86' ? 'bg-blue-100 text-blue-700' : 'bg-purple-100 text-purple-700'}`}>{p.arch}</span>
                        </td>
                        <td className="px-3 py-2 text-center text-xs">{p.type_class}</td>
                        <td className="px-3 py-2 text-center text-xs text-gray-500">{p.unit}</td>
                        <td className="px-3 py-2 text-center">
                          <input
                            type="number"
                            min="0"
                            className="w-16 px-2 py-1 border rounded text-center text-sm"
                            value={selectedProducts[p.id] || 0}
                            onChange={e => setSelectedProducts(prev => ({ ...prev, [p.id]: parseInt(e.target.value) || 0 }))}
                          />
                        </td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
            </div>
          )}

          {/* Step 2: Services */}
          {step === 2 && (
            <div>
              <div className="flex items-center justify-between mb-4">
                <h3 className="text-base font-semibold">选择服务及数量</h3>
                <span className="text-sm text-primary-600 font-medium">已选 {selectedServiceCount} 项</span>
              </div>
              <div className="flex items-center gap-3 mb-4">
                <div className="relative flex-1 max-w-xs">
                  <Search className="absolute left-3 top-2.5 w-4 h-4 text-gray-400" />
                  <input
                    className="w-full pl-9 pr-3 py-2 border rounded-lg text-sm"
                    placeholder="搜索服务名称或编码..."
                    value={searchService}
                    onChange={e => setSearchService(e.target.value)}
                  />
                </div>
              </div>
              {/* Services grouped by category */}
              <div className="space-y-4 max-h-[450px] overflow-y-auto">
                {serviceCategories.map(cat => {
                  const catServices = filteredServices.filter(s => s.category === cat);
                  if (catServices.length === 0) return null;
                  return (
                    <div key={cat}>
                      <h4 className="text-xs font-semibold text-gray-500 uppercase tracking-wide mb-2 px-1">{cat}</h4>
                      <div className="border rounded-lg overflow-hidden">
                        <table className="w-full text-sm">
                          <tbody>
                            {catServices.map(s => (
                              <tr key={s.id} className={`border-b last:border-0 hover:bg-blue-50/50 ${selectedServices[s.id] > 0 ? 'bg-green-50/50' : ''}`}>
                                <td className="px-3 py-2 w-[320px]">
                                  <div className="font-medium text-xs">{s.name}</div>
                                  <div className="text-[10px] text-gray-400 mt-0.5 line-clamp-1">{s.description}</div>
                                </td>
                                <td className="px-3 py-2 font-mono text-xs text-gray-600 w-[100px]">{s.code}</td>
                                <td className="px-3 py-2 text-center text-xs text-gray-500 w-[60px]">{s.unit}</td>
                                <td className="px-3 py-2 text-center w-[100px]">
                                  <input
                                    type="number"
                                    min="0"
                                    className="w-16 px-2 py-1 border rounded text-center text-sm"
                                    value={selectedServices[s.id] || 0}
                                    onChange={e => setSelectedServices(prev => ({ ...prev, [s.id]: parseInt(e.target.value) || 0 }))}
                                  />
                                </td>
                              </tr>
                            ))}
                          </tbody>
                        </table>
                      </div>
                    </div>
                  );
                })}
              </div>
            </div>
          )}

          {/* Step 3: Summary */}
          {step === 3 && (
            <div>
              <h3 className="text-base font-semibold mb-4">Order 汇总确认</h3>
              {/* Opportunity Summary */}
              <div className="mb-6 p-4 bg-gray-50 rounded-lg">
                <h4 className="text-xs font-semibold text-gray-500 mb-2">商机信息</h4>
                <div className="grid grid-cols-3 gap-2 text-sm">
                  <div><span className="text-gray-400">商机号: </span>{opportunity.opportunity_no || '-'}</div>
                  <div><span className="text-gray-400">客户: </span>{opportunity.customer_name || '-'}</div>
                  <div><span className="text-gray-400">部署地点: </span>{opportunity.deploy_location || '-'}</div>
                  <div className="col-span-3"><span className="text-gray-400">商机名称: </span>{opportunity.opportunity_name || '-'}</div>
                </div>
              </div>
              {/* Products Summary */}
              {selectedProductCount > 0 && (
                <div className="mb-4">
                  <h4 className="text-xs font-semibold text-gray-500 mb-2">产品明细 ({selectedProductCount} 项)</h4>
                  <div className="border rounded-lg overflow-hidden">
                    <table className="w-full text-xs">
                      <thead className="bg-gray-50">
                        <tr>
                          <th className="px-3 py-2 text-left">产品名称</th>
                          <th className="px-3 py-2 text-left">产品编码</th>
                          <th className="px-3 py-2 text-center">架构</th>
                          <th className="px-3 py-2 text-center">数量</th>
                          <th className="px-3 py-2 text-center">单位</th>
                        </tr>
                      </thead>
                      <tbody>
                        {Object.entries(selectedProducts).filter(([,q]) => q > 0).map(([id, qty]) => {
                          const p = catalog.products.find(x => x.id === id);
                          if (!p) return null;
                          return (
                            <tr key={id} className="border-t">
                              <td className="px-3 py-1.5">{p.name}</td>
                              <td className="px-3 py-1.5 font-mono">{p.code}</td>
                              <td className="px-3 py-1.5 text-center">{p.arch}</td>
                              <td className="px-3 py-1.5 text-center font-bold text-primary-600">{qty}</td>
                              <td className="px-3 py-1.5 text-center">{p.unit}</td>
                            </tr>
                          );
                        })}
                      </tbody>
                    </table>
                  </div>
                </div>
              )}
              {/* Services Summary */}
              {selectedServiceCount > 0 && (
                <div className="mb-4">
                  <h4 className="text-xs font-semibold text-gray-500 mb-2">服务明细 ({selectedServiceCount} 项)</h4>
                  <div className="border rounded-lg overflow-hidden">
                    <table className="w-full text-xs">
                      <thead className="bg-gray-50">
                        <tr>
                          <th className="px-3 py-2 text-left">服务名称</th>
                          <th className="px-3 py-2 text-left">服务编码</th>
                          <th className="px-3 py-2 text-center">数量</th>
                          <th className="px-3 py-2 text-center">单位</th>
                        </tr>
                      </thead>
                      <tbody>
                        {Object.entries(selectedServices).filter(([,q]) => q > 0).map(([id, qty]) => {
                          const s = catalog.services.find(x => x.id === id);
                          if (!s) return null;
                          return (
                            <tr key={id} className="border-t">
                              <td className="px-3 py-1.5">{s.name}</td>
                              <td className="px-3 py-1.5 font-mono">{s.code}</td>
                              <td className="px-3 py-1.5 text-center font-bold text-primary-600">{qty}</td>
                              <td className="px-3 py-1.5 text-center">{s.unit}</td>
                            </tr>
                          );
                        })}
                      </tbody>
                    </table>
                  </div>
                </div>
              )}
              {selectedProductCount === 0 && selectedServiceCount === 0 && (
                <div className="text-center py-8 text-gray-400">
                  <Package className="w-10 h-10 mx-auto mb-2" />
                  <p>尚未选择任何产品或服务，请返回上一步进行选择</p>
                </div>
              )}
            </div>
          )}
        </div>

        {/* Footer Navigation */}
        <div className="flex items-center justify-between mt-4">
          <button
            onClick={() => setStep(s => Math.max(0, s - 1))}
            disabled={step === 0}
            className="flex items-center gap-1 px-4 py-2 text-sm border rounded-lg disabled:opacity-40 hover:bg-gray-50"
          >
            <ChevronLeft className="w-4 h-4" /> 上一步
          </button>
          <div className="flex gap-2">
            {step < 3 ? (
              <button
                onClick={() => setStep(s => Math.min(3, s + 1))}
                className="flex items-center gap-1 px-4 py-2 text-sm bg-primary-600 text-white rounded-lg hover:bg-primary-700"
              >
                下一步 <ChevronRight className="w-4 h-4" />
              </button>
            ) : (
              <button
                onClick={handleSaveOrder}
                disabled={saving || (!opportunity.customer_name && !opportunity.opportunity_no)}
                className="flex items-center gap-2 px-5 py-2 text-sm bg-green-600 text-white rounded-lg hover:bg-green-700 disabled:opacity-50"
              >
                {saving ? '保存中...' : '保存并生成Order'}
                <FileSpreadsheet className="w-4 h-4" />
              </button>
            )}
          </div>
        </div>
      </div>
    </div>
  );
}
