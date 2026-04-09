import React, { useEffect, useState } from 'react';
import { Server, Plus, Edit2, Trash2, CheckCircle, XCircle, Loader2, X, Wifi, Shield } from 'lucide-react';
import { getLDAPConfigs, createLDAPConfig, updateLDAPConfig, deleteLDAPConfig, testLDAPConfig } from '../services/api';
import toast from 'react-hot-toast';

export default function LDAPPage() {
  const [configs, setConfigs] = useState([]);
  const [loading, setLoading] = useState(true);
  const [showModal, setShowModal] = useState(false);
  const [editConfig, setEditConfig] = useState(null);
  const [form, setForm] = useState({ name: '', host: '', port: 389, use_tls: false, bind_dn: '', bind_password: '', base_dn: '', user_ou: '', user_filter: '(uid=%s)', attr_username: 'uid', attr_email: 'mail', attr_display: 'cn', is_enabled: true, is_default: false });
  const [saving, setSaving] = useState(false);
  const [testing, setTesting] = useState({});

  useEffect(() => { loadConfigs(); }, []);

  const loadConfigs = async () => {
    try { const res = await getLDAPConfigs(); if (res.code === 0) setConfigs(res.data || []); }
    catch (e) {} finally { setLoading(false); }
  };

  const openCreate = () => {
    setEditConfig(null);
    setForm({ name: '', host: '', port: 389, use_tls: false, bind_dn: '', bind_password: '', base_dn: '', user_ou: '', user_filter: '(uid=%s)', attr_username: 'uid', attr_email: 'mail', attr_display: 'cn', is_enabled: true, is_default: false });
    setShowModal(true);
  };

  const openEdit = (cfg) => {
    setEditConfig(cfg);
    setForm({ ...cfg, bind_password: '' });
    setShowModal(true);
  };

  const handleSave = async () => {
    setSaving(true);
    try {
      if (editConfig) {
        const res = await updateLDAPConfig(editConfig.id, form);
        if (res.code === 0) { toast.success('LDAP 配置已更新'); setShowModal(false); loadConfigs(); }
      } else {
        const res = await createLDAPConfig(form);
        if (res.code === 0) { toast.success('LDAP 配置已创建'); setShowModal(false); loadConfigs(); }
      }
    } catch (e) { toast.error('保存失败'); }
    finally { setSaving(false); }
  };

  const handleDelete = async (id) => {
    if (!confirm('确认删除此LDAP配置？')) return;
    try { await deleteLDAPConfig(id); toast.success('已删除'); loadConfigs(); } catch (e) { toast.error('删除失败'); }
  };

  const handleTest = async (id) => {
    setTesting(p => ({ ...p, [id]: true }));
    try {
      const res = await testLDAPConfig(id);
      if (res.code === 0) toast.success(res.data?.message || '连接成功');
      else toast.error(res.message || '连接失败');
    } catch (e) { toast.error('测试失败'); }
    finally { setTesting(p => ({ ...p, [id]: false })); }
  };

  if (loading) return <div className="h-full flex items-center justify-center"><Loader2 className="w-6 h-6 animate-spin text-primary-600" /></div>;

  return (
    <div className="h-full overflow-y-auto">
      <div className="p-4 sm:p-6 space-y-4">
        <div className="bg-white rounded-xl border border-gray-200 shadow-sm">
          <div className="px-6 py-4 flex items-center justify-between border-b border-gray-100">
            <div>
              <h2 className="text-lg font-semibold text-gray-800">LDAP 认证配置</h2>
              <p className="text-sm text-gray-400">管理企业LDAP/Active Directory服务对接，普通用户可通过LDAP登录</p>
            </div>
            <button onClick={openCreate} className="flex items-center gap-2 bg-primary-600 hover:bg-primary-700 text-white px-4 py-2 rounded-lg text-sm font-medium">
              <Plus className="w-4 h-4" /> 新增 LDAP
            </button>
          </div>
          <div className="p-6">
            {configs.length === 0 ? (
              <div className="text-center py-12">
                <Server className="w-12 h-12 text-gray-200 mx-auto mb-3" />
                <p className="text-gray-400 mb-3">暂未配置LDAP服务</p>
                <p className="text-xs text-gray-300">配置LDAP后，企业用户可使用域账号登录本平台</p>
              </div>
            ) : (
              <div className="space-y-3">
                {configs.map(cfg => (
                  <div key={cfg.id} className="flex items-center gap-4 px-5 py-4 rounded-xl border border-gray-100 hover:border-gray-200 hover:shadow-sm transition-all">
                    <div className="w-10 h-10 bg-primary-100 rounded-xl flex items-center justify-center">
                      <Server className="w-5 h-5 text-primary-600" />
                    </div>
                    <div className="flex-1 min-w-0">
                      <div className="flex items-center gap-2">
                        <span className="font-semibold text-gray-800 text-sm">{cfg.name}</span>
                        {cfg.is_default && <span className="text-xs px-2 py-0.5 rounded-full bg-primary-50 text-primary-600 border border-primary-200">默认</span>}
                        {cfg.is_enabled ? <CheckCircle className="w-3.5 h-3.5 text-green-500" /> : <XCircle className="w-3.5 h-3.5 text-gray-400" />}
                      </div>
                      <p className="text-xs text-gray-400 mt-0.5">{cfg.host}:{cfg.port} | BaseDN: {cfg.base_dn}{cfg.user_ou ? ` | 用户OU: ${cfg.user_ou}` : ''}</p>
                    </div>
                    <div className="flex gap-1">
                      <button onClick={() => handleTest(cfg.id)} disabled={testing[cfg.id]}
                        className="p-1.5 text-gray-400 hover:text-primary-500 hover:bg-primary-50 rounded-lg" title="测试连接">
                        {testing[cfg.id] ? <Loader2 className="w-4 h-4 animate-spin" /> : <Wifi className="w-4 h-4" />}
                      </button>
                      <button onClick={() => openEdit(cfg)} className="p-1.5 text-gray-400 hover:text-primary-600 hover:bg-primary-50 rounded-lg" title="编辑">
                        <Edit2 className="w-4 h-4" />
                      </button>
                      <button onClick={() => handleDelete(cfg.id)} className="p-1.5 text-gray-400 hover:text-red-500 hover:bg-red-50 rounded-lg" title="删除">
                        <Trash2 className="w-4 h-4" />
                      </button>
                    </div>
                  </div>
                ))}
              </div>
            )}
          </div>
        </div>

        {/* Info card */}
        <div className="bg-primary-50 rounded-xl border border-primary-200 p-5">
          <div className="flex items-start gap-3">
            <Shield className="w-5 h-5 text-primary-600 mt-0.5 flex-shrink-0" />
            <div>
              <h3 className="text-sm font-semibold text-primary-800">LDAP 认证说明</h3>
              <ul className="text-xs text-primary-600 mt-1 space-y-1 list-disc list-inside">
                <li>管理员可在此页面配置企业LDAP/AD服务器信息</li>
                <li>配置完成后，前往「用户管理」页面点击「同步LDAP用户」将LDAP用户拉取到平台</li>
                <li>同步后的LDAP用户默认为普通用户角色，管理员可修改其角色</li>
                <li>LDAP用户使用域账号密码在登录页选择「LDAP登录」进行认证</li>
                <li>可配置「用户OU」字段来限定只同步特定组织单元的用户（如 ou=Technology,dc=easystack,dc=cn）</li>
                <li>单次同步最多支持 1000 个用户，支持多个LDAP源，可设置默认LDAP服务器</li>
              </ul>
            </div>
          </div>
        </div>
      </div>

      {/* Modal */}
      {showModal && (
        <div className="fixed inset-0 bg-black/40 z-50 flex items-center justify-center p-4">
          <div className="bg-white rounded-2xl shadow-2xl w-full max-w-xl max-h-[90vh] flex flex-col">
            <div className="flex items-center justify-between px-6 py-4 border-b border-gray-100">
              <h2 className="text-lg font-semibold text-gray-800">{editConfig ? '编辑 LDAP' : '新增 LDAP'}</h2>
              <button onClick={() => setShowModal(false)} className="p-1.5 text-gray-400 hover:text-gray-600 rounded"><X className="w-5 h-5" /></button>
            </div>
            <div className="p-6 overflow-y-auto space-y-4">
              <div className="grid grid-cols-2 gap-4">
                <div>
                  <label className="block text-sm font-medium text-gray-600 mb-1">名称</label>
                  <input value={form.name} onChange={e => setForm({...form, name: e.target.value})} className="w-full px-3 py-2 border border-gray-200 rounded-lg text-sm outline-none focus:ring-2 focus:ring-primary-500" placeholder="如：公司LDAP" />
                </div>
                <div>
                  <label className="block text-sm font-medium text-gray-600 mb-1">服务器地址</label>
                  <input value={form.host} onChange={e => setForm({...form, host: e.target.value})} className="w-full px-3 py-2 border border-gray-200 rounded-lg text-sm outline-none focus:ring-2 focus:ring-primary-500" placeholder="ldap.example.com" />
                </div>
                <div>
                  <label className="block text-sm font-medium text-gray-600 mb-1">端口</label>
                  <input type="number" value={form.port} onChange={e => setForm({...form, port: parseInt(e.target.value)||389})} className="w-full px-3 py-2 border border-gray-200 rounded-lg text-sm outline-none focus:ring-2 focus:ring-primary-500" />
                </div>
                <div>
                  <label className="block text-sm font-medium text-gray-600 mb-1">Base DN</label>
                  <input value={form.base_dn} onChange={e => setForm({...form, base_dn: e.target.value})} className="w-full px-3 py-2 border border-gray-200 rounded-lg text-sm outline-none focus:ring-2 focus:ring-primary-500" placeholder="dc=example,dc=com" />
                </div>
              </div>
              <div>
                <label className="block text-sm font-medium text-gray-600 mb-1">用户 OU <span className="text-xs text-gray-400 font-normal">(可选，指定同步的组织单元)</span></label>
                <input value={form.user_ou} onChange={e => setForm({...form, user_ou: e.target.value})} className="w-full px-3 py-2 border border-gray-200 rounded-lg text-sm outline-none focus:ring-2 focus:ring-primary-500" placeholder="如：ou=Technology,dc=easystack,dc=cn（留空则同步 BaseDN 下所有用户）" />
              </div>
              <div>
                <label className="block text-sm font-medium text-gray-600 mb-1">Bind DN</label>
                <input value={form.bind_dn} onChange={e => setForm({...form, bind_dn: e.target.value})} className="w-full px-3 py-2 border border-gray-200 rounded-lg text-sm outline-none focus:ring-2 focus:ring-primary-500" placeholder="cn=admin,dc=example,dc=com" />
              </div>
              <div>
                <label className="block text-sm font-medium text-gray-600 mb-1">Bind Password {editConfig && <span className="text-xs text-gray-400">(留空不修改)</span>}</label>
                <input type="password" value={form.bind_password} onChange={e => setForm({...form, bind_password: e.target.value})} className="w-full px-3 py-2 border border-gray-200 rounded-lg text-sm outline-none focus:ring-2 focus:ring-primary-500" />
              </div>
              <div className="grid grid-cols-3 gap-4">
                <div>
                  <label className="block text-sm font-medium text-gray-600 mb-1">用户名属性</label>
                  <input value={form.attr_username} onChange={e => setForm({...form, attr_username: e.target.value})} className="w-full px-3 py-2 border border-gray-200 rounded-lg text-sm outline-none" />
                </div>
                <div>
                  <label className="block text-sm font-medium text-gray-600 mb-1">邮箱属性</label>
                  <input value={form.attr_email} onChange={e => setForm({...form, attr_email: e.target.value})} className="w-full px-3 py-2 border border-gray-200 rounded-lg text-sm outline-none" />
                </div>
                <div>
                  <label className="block text-sm font-medium text-gray-600 mb-1">显示名属性</label>
                  <input value={form.attr_display} onChange={e => setForm({...form, attr_display: e.target.value})} className="w-full px-3 py-2 border border-gray-200 rounded-lg text-sm outline-none" />
                </div>
              </div>
              <div className="flex items-center gap-6">
                <label className="flex items-center gap-2 cursor-pointer">
                  <input type="checkbox" checked={form.use_tls} onChange={e => setForm({...form, use_tls: e.target.checked})} className="w-4 h-4 rounded" />
                  <span className="text-sm text-gray-600">使用 TLS</span>
                </label>
                <label className="flex items-center gap-2 cursor-pointer">
                  <input type="checkbox" checked={form.is_enabled} onChange={e => setForm({...form, is_enabled: e.target.checked})} className="w-4 h-4 rounded" />
                  <span className="text-sm text-gray-600">启用</span>
                </label>
                <label className="flex items-center gap-2 cursor-pointer">
                  <input type="checkbox" checked={form.is_default} onChange={e => setForm({...form, is_default: e.target.checked})} className="w-4 h-4 rounded" />
                  <span className="text-sm text-gray-600">设为默认</span>
                </label>
              </div>
            </div>
            <div className="px-6 py-4 border-t border-gray-100 flex justify-end gap-3">
              <button onClick={() => setShowModal(false)} className="px-4 py-2 border border-gray-200 text-gray-600 hover:bg-gray-50 rounded-lg text-sm">取消</button>
              <button onClick={handleSave} disabled={saving} className="px-5 py-2 bg-primary-600 hover:bg-primary-700 text-white rounded-lg text-sm font-medium disabled:opacity-50 flex items-center gap-2">
                {saving && <Loader2 className="w-4 h-4 animate-spin" />}
                {editConfig ? '保存修改' : '创建配置'}
              </button>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}
