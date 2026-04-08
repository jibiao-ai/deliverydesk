import React, { useEffect, useState } from 'react';
import {
  Cpu, Eye, EyeOff, CheckCircle, AlertCircle, Loader2, Star, StarOff,
  Settings, Trash2, Plus, Search, TestTube, Save, X, ChevronDown, ChevronUp,
  Key, Globe, Zap, Info, ExternalLink, RefreshCw
} from 'lucide-react';
import { getAIProviders, createAIProvider, updateAIProvider, deleteAIProvider, testAIProvider } from '../services/api';
import toast from 'react-hot-toast';

/* ─── provider icon/emoji mapping ─── */
const PROVIDER_ICONS = {
  openai:      { emoji: '🤖', color: 'bg-gray-800',    text: 'text-white' },
  deepseek:    { emoji: '🔍', color: 'bg-blue-600',    text: 'text-white' },
  qwen:        { emoji: '☁️', color: 'bg-purple-600',  text: 'text-white' },
  glm:         { emoji: '🧠', color: 'bg-indigo-600',  text: 'text-white' },
  minimax:     { emoji: '⚡', color: 'bg-yellow-500',  text: 'text-white' },
  siliconflow: { emoji: '💎', color: 'bg-cyan-600',    text: 'text-white' },
  moonshot:    { emoji: '🌙', color: 'bg-slate-700',   text: 'text-white' },
  ernie:       { emoji: '🔵', color: 'bg-blue-500',    text: 'text-white' },
  doubao:      { emoji: '🔥', color: 'bg-orange-500',  text: 'text-white' },
  hunyuan:     { emoji: '🌀', color: 'bg-teal-600',    text: 'text-white' },
  baichuan:    { emoji: '🕊️', color: 'bg-emerald-600', text: 'text-white' },
  anthropic:   { emoji: '🎭', color: 'bg-amber-700',   text: 'text-white' },
  gemini:      { emoji: '✨', color: 'bg-sky-500',     text: 'text-white' },
};

const DEFAULT_ICON = { emoji: '🤖', color: 'bg-gray-500', text: 'text-white' };

function getIcon(provider) {
  return PROVIDER_ICONS[provider.icon_url] || PROVIDER_ICONS[provider.name] || DEFAULT_ICON;
}

/* ─── Provider Card ─── */
function ProviderCard({ provider, onEdit, onTest, onToggle, onSetDefault, onDelete }) {
  const icon = getIcon(provider);
  const [testing, setTesting] = useState(false);

  const handleTest = async () => {
    setTesting(true);
    try {
      const res = await testAIProvider(provider.id);
      if (res.code === 0) {
        toast.success(res.data?.message || '连接测试成功');
      } else {
        toast.error(res.message || '测试失败');
      }
    } catch (err) {
      toast.error(err?.message || '连接测试失败');
    }
    setTesting(false);
  };

  return (
    <div className={`relative bg-white rounded-xl border-2 transition-all duration-200 hover:shadow-lg group ${
      provider.is_enabled ? 'border-gray-200 hover:border-primary-300' : 'border-gray-100 opacity-60'
    } ${provider.is_default ? 'ring-2 ring-primary-400 ring-offset-2' : ''}`}>
      {/* Default badge */}
      {provider.is_default && (
        <div className="absolute -top-2.5 left-4 px-2.5 py-0.5 bg-primary-500 text-white text-xs font-medium rounded-full shadow-sm">
          默认模型
        </div>
      )}

      <div className="p-5">
        {/* Header */}
        <div className="flex items-start justify-between mb-4">
          <div className="flex items-center gap-3">
            <div className={`w-11 h-11 rounded-xl ${icon.color} flex items-center justify-center text-xl shadow-sm`}>
              {icon.emoji}
            </div>
            <div>
              <h3 className="font-semibold text-gray-900 text-base">{provider.label}</h3>
              <p className="text-xs text-gray-400 font-mono">{provider.name}</p>
            </div>
          </div>
          <div className="flex items-center gap-1">
            {provider.configured ? (
              <span className="flex items-center gap-1 text-xs text-green-600 bg-green-50 px-2 py-1 rounded-full">
                <CheckCircle className="w-3.5 h-3.5" /> 已配置
              </span>
            ) : (
              <span className="flex items-center gap-1 text-xs text-amber-600 bg-amber-50 px-2 py-1 rounded-full">
                <AlertCircle className="w-3.5 h-3.5" /> 未配置
              </span>
            )}
          </div>
        </div>

        {/* Description */}
        <p className="text-sm text-gray-500 mb-3 line-clamp-2">{provider.description}</p>

        {/* Model Info */}
        <div className="space-y-2 mb-4">
          <div className="flex items-center gap-2 text-sm">
            <Cpu className="w-4 h-4 text-gray-400" />
            <span className="text-gray-500">推荐模型:</span>
            <span className="font-mono text-primary-600 bg-primary-50 px-2 py-0.5 rounded text-xs">{provider.model}</span>
          </div>
          <div className="flex items-center gap-2 text-sm">
            <Globe className="w-4 h-4 text-gray-400" />
            <span className="text-gray-500">API 地址:</span>
            <span className="font-mono text-gray-600 text-xs truncate max-w-[200px]" title={provider.base_url}>
              {provider.base_url}
            </span>
          </div>
          <div className="flex items-center gap-2 text-sm">
            <Key className="w-4 h-4 text-gray-400" />
            <span className="text-gray-500">API Key:</span>
            <span className="font-mono text-gray-600 text-xs">
              {provider.api_key || '未设置'}
            </span>
          </div>
        </div>

        {/* Actions */}
        <div className="flex items-center justify-between pt-3 border-t border-gray-100">
          <div className="flex items-center gap-1">
            <button
              onClick={() => onEdit(provider)}
              className="p-2 text-gray-400 hover:text-primary-600 hover:bg-primary-50 rounded-lg transition-colors"
              title="编辑配置"
            >
              <Settings className="w-4 h-4" />
            </button>
            <button
              onClick={handleTest}
              disabled={!provider.configured || testing}
              className="p-2 text-gray-400 hover:text-green-600 hover:bg-green-50 rounded-lg transition-colors disabled:opacity-30 disabled:cursor-not-allowed"
              title="测试连接"
            >
              {testing ? <Loader2 className="w-4 h-4 animate-spin" /> : <TestTube className="w-4 h-4" />}
            </button>
            <button
              onClick={() => onSetDefault(provider)}
              className={`p-2 rounded-lg transition-colors ${
                provider.is_default
                  ? 'text-yellow-500 bg-yellow-50'
                  : 'text-gray-400 hover:text-yellow-500 hover:bg-yellow-50'
              }`}
              title={provider.is_default ? '取消默认' : '设为默认'}
            >
              {provider.is_default ? <Star className="w-4 h-4 fill-current" /> : <StarOff className="w-4 h-4" />}
            </button>
            <button
              onClick={() => onDelete(provider)}
              className="p-2 text-gray-400 hover:text-red-600 hover:bg-red-50 rounded-lg transition-colors"
              title="删除"
            >
              <Trash2 className="w-4 h-4" />
            </button>
          </div>
          <button
            onClick={() => onToggle(provider)}
            className={`relative w-11 h-6 rounded-full transition-colors ${
              provider.is_enabled ? 'bg-primary-500' : 'bg-gray-300'
            }`}
            title={provider.is_enabled ? '点击禁用' : '点击启用'}
          >
            <span className={`absolute top-0.5 w-5 h-5 rounded-full bg-white shadow-sm transition-transform ${
              provider.is_enabled ? 'translate-x-5.5 left-[22px]' : 'left-0.5'
            }`} />
          </button>
        </div>
      </div>
    </div>
  );
}

/* ─── Edit / Create Modal ─── */
function ProviderModal({ provider, isNew, onClose, onSave }) {
  const [form, setForm] = useState({
    name: '',
    label: '',
    api_key: '',
    base_url: '',
    model: '',
    is_default: false,
    is_enabled: true,
    description: '',
    icon_url: '',
  });
  const [saving, setSaving] = useState(false);

  useEffect(() => {
    if (provider && !isNew) {
      setForm({
        name: provider.name || '',
        label: provider.label || '',
        api_key: '', // Don't pre-fill masked key
        base_url: provider.base_url || '',
        model: provider.model || '',
        is_default: provider.is_default || false,
        is_enabled: provider.is_enabled ?? true,
        description: provider.description || '',
        icon_url: provider.icon_url || provider.name || '',
      });
    }
  }, [provider, isNew]);

  const handleSave = async () => {
    if (isNew && (!form.name || !form.label || !form.base_url || !form.model)) {
      toast.error('请填写厂商标识、名称、API地址和模型');
      return;
    }
    setSaving(true);
    try {
      const data = { ...form };
      // Don't send empty api_key on update
      if (!isNew && !data.api_key) {
        delete data.api_key;
      }
      let res;
      if (isNew) {
        res = await createAIProvider(data);
      } else {
        res = await updateAIProvider(provider.id, data);
      }
      if (res.code === 0) {
        toast.success(isNew ? '厂商添加成功' : '配置更新成功');
        onSave();
      } else {
        toast.error(res.message || '操作失败');
      }
    } catch (err) {
      toast.error(err?.message || '操作失败');
    }
    setSaving(false);
  };

  const iconKeys = Object.keys(PROVIDER_ICONS);

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/40 backdrop-blur-sm" onClick={onClose}>
      <div className="bg-white rounded-2xl shadow-2xl w-full max-w-lg mx-4 max-h-[90vh] overflow-y-auto" onClick={e => e.stopPropagation()}>
        <div className="sticky top-0 bg-white border-b border-gray-100 px-6 py-4 rounded-t-2xl flex items-center justify-between">
          <h2 className="text-lg font-semibold text-gray-900 flex items-center gap-2">
            <Cpu className="w-5 h-5 text-primary-500" />
            {isNew ? '添加 AI 模型厂商' : `编辑 ${provider?.label}`}
          </h2>
          <button onClick={onClose} className="p-2 hover:bg-gray-100 rounded-lg transition-colors">
            <X className="w-5 h-5 text-gray-400" />
          </button>
        </div>

        <div className="p-6 space-y-5">
          {/* Icon selector (only for new) */}
          {isNew && (
            <div>
              <label className="block text-sm font-medium text-gray-700 mb-2">选择图标标识</label>
              <div className="flex flex-wrap gap-2">
                {iconKeys.map(key => {
                  const ic = PROVIDER_ICONS[key];
                  return (
                    <button
                      key={key}
                      onClick={() => setForm(f => ({ ...f, icon_url: key }))}
                      className={`w-10 h-10 rounded-lg flex items-center justify-center text-lg transition-all ${
                        form.icon_url === key ? 'ring-2 ring-primary-500 scale-110 shadow-md' : 'hover:scale-105'
                      } ${ic.color}`}
                      title={key}
                    >
                      {ic.emoji}
                    </button>
                  );
                })}
              </div>
            </div>
          )}

          {/* Name (identifier) */}
          <div>
            <label className="block text-sm font-medium text-gray-700 mb-1.5">
              厂商标识 <span className="text-red-500">*</span>
              <span className="text-xs text-gray-400 ml-1">(唯一英文标识，如 openai)</span>
            </label>
            <input
              value={form.name}
              onChange={e => setForm(f => ({ ...f, name: e.target.value }))}
              disabled={!isNew}
              className="w-full px-4 py-2.5 bg-gray-50 border border-gray-200 rounded-xl focus:ring-2 focus:ring-primary-500 focus:border-primary-500 outline-none text-sm font-mono disabled:opacity-50"
              placeholder="例如: openai, deepseek, qwen"
            />
          </div>

          {/* Label (display name) */}
          <div>
            <label className="block text-sm font-medium text-gray-700 mb-1.5">
              显示名称 <span className="text-red-500">*</span>
            </label>
            <input
              value={form.label}
              onChange={e => setForm(f => ({ ...f, label: e.target.value }))}
              className="w-full px-4 py-2.5 bg-gray-50 border border-gray-200 rounded-xl focus:ring-2 focus:ring-primary-500 focus:border-primary-500 outline-none text-sm"
              placeholder="例如: OpenAI, 通义千问, 智谱 GLM"
            />
          </div>

          {/* Base URL */}
          <div>
            <label className="block text-sm font-medium text-gray-700 mb-1.5">
              API 基础地址 <span className="text-red-500">*</span>
            </label>
            <input
              value={form.base_url}
              onChange={e => setForm(f => ({ ...f, base_url: e.target.value }))}
              className="w-full px-4 py-2.5 bg-gray-50 border border-gray-200 rounded-xl focus:ring-2 focus:ring-primary-500 focus:border-primary-500 outline-none text-sm font-mono"
              placeholder="https://api.openai.com/v1"
            />
          </div>

          {/* Model */}
          <div>
            <label className="block text-sm font-medium text-gray-700 mb-1.5">
              推荐模型 <span className="text-red-500">*</span>
            </label>
            <input
              value={form.model}
              onChange={e => setForm(f => ({ ...f, model: e.target.value }))}
              className="w-full px-4 py-2.5 bg-gray-50 border border-gray-200 rounded-xl focus:ring-2 focus:ring-primary-500 focus:border-primary-500 outline-none text-sm font-mono"
              placeholder="例如: gpt-4o, deepseek-chat, qwen-plus"
            />
          </div>

          {/* API Key */}
          <div>
            <label className="block text-sm font-medium text-gray-700 mb-1.5">
              API Key
              {!isNew && <span className="text-xs text-gray-400 ml-1">(留空保持原值不变)</span>}
            </label>
            <input
              value={form.api_key}
              onChange={e => setForm(f => ({ ...f, api_key: e.target.value }))}
              type="password"
              className="w-full px-4 py-2.5 bg-gray-50 border border-gray-200 rounded-xl focus:ring-2 focus:ring-primary-500 focus:border-primary-500 outline-none text-sm font-mono"
              placeholder={isNew ? '输入 API Key' : '留空保持原密钥不变'}
            />
          </div>

          {/* Description */}
          <div>
            <label className="block text-sm font-medium text-gray-700 mb-1.5">说明</label>
            <input
              value={form.description}
              onChange={e => setForm(f => ({ ...f, description: e.target.value }))}
              className="w-full px-4 py-2.5 bg-gray-50 border border-gray-200 rounded-xl focus:ring-2 focus:ring-primary-500 focus:border-primary-500 outline-none text-sm"
              placeholder="模型厂商简要说明"
            />
          </div>

          {/* Toggles */}
          <div className="flex items-center gap-6 pt-2">
            <label className="flex items-center gap-2 cursor-pointer">
              <input
                type="checkbox"
                checked={form.is_enabled}
                onChange={e => setForm(f => ({ ...f, is_enabled: e.target.checked }))}
                className="w-4 h-4 text-primary-600 border-gray-300 rounded focus:ring-primary-500"
              />
              <span className="text-sm text-gray-700">启用</span>
            </label>
            <label className="flex items-center gap-2 cursor-pointer">
              <input
                type="checkbox"
                checked={form.is_default}
                onChange={e => setForm(f => ({ ...f, is_default: e.target.checked }))}
                className="w-4 h-4 text-primary-600 border-gray-300 rounded focus:ring-primary-500"
              />
              <span className="text-sm text-gray-700">设为默认</span>
            </label>
          </div>
        </div>

        {/* Footer */}
        <div className="sticky bottom-0 bg-white border-t border-gray-100 px-6 py-4 rounded-b-2xl flex items-center justify-end gap-3">
          <button
            onClick={onClose}
            className="px-5 py-2.5 text-sm text-gray-600 bg-gray-100 hover:bg-gray-200 rounded-xl transition-colors"
          >
            取消
          </button>
          <button
            onClick={handleSave}
            disabled={saving}
            className="px-5 py-2.5 text-sm text-white bg-primary-500 hover:bg-primary-600 rounded-xl transition-colors flex items-center gap-2 disabled:opacity-50"
          >
            {saving ? <Loader2 className="w-4 h-4 animate-spin" /> : <Save className="w-4 h-4" />}
            {isNew ? '添加' : '保存'}
          </button>
        </div>
      </div>
    </div>
  );
}

/* ─── Delete Confirm Modal ─── */
function DeleteModal({ provider, onClose, onConfirm }) {
  const [deleting, setDeleting] = useState(false);
  const handleConfirm = async () => {
    setDeleting(true);
    try {
      const res = await deleteAIProvider(provider.id);
      if (res.code === 0) {
        toast.success(`已删除 ${provider.label}`);
        onConfirm();
      } else {
        toast.error(res.message || '删除失败');
      }
    } catch (err) {
      toast.error(err?.message || '删除失败');
    }
    setDeleting(false);
  };

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/40 backdrop-blur-sm" onClick={onClose}>
      <div className="bg-white rounded-2xl shadow-2xl w-full max-w-sm mx-4 p-6" onClick={e => e.stopPropagation()}>
        <div className="flex items-center gap-3 mb-4">
          <div className="w-10 h-10 bg-red-100 rounded-full flex items-center justify-center">
            <Trash2 className="w-5 h-5 text-red-600" />
          </div>
          <div>
            <h3 className="font-semibold text-gray-900">确认删除</h3>
            <p className="text-sm text-gray-500">删除后无法恢复</p>
          </div>
        </div>
        <p className="text-sm text-gray-600 mb-6">
          确定要删除 AI 模型厂商 <strong>{provider.label}</strong> ({provider.name}) 吗？
        </p>
        <div className="flex items-center justify-end gap-3">
          <button onClick={onClose} className="px-4 py-2 text-sm text-gray-600 bg-gray-100 hover:bg-gray-200 rounded-xl transition-colors">
            取消
          </button>
          <button
            onClick={handleConfirm}
            disabled={deleting}
            className="px-4 py-2 text-sm text-white bg-red-500 hover:bg-red-600 rounded-xl transition-colors flex items-center gap-2 disabled:opacity-50"
          >
            {deleting ? <Loader2 className="w-4 h-4 animate-spin" /> : <Trash2 className="w-4 h-4" />}
            删除
          </button>
        </div>
      </div>
    </div>
  );
}

/* ─── Main Page ─── */
export default function AIModelsPage() {
  const [providers, setProviders] = useState([]);
  const [loading, setLoading] = useState(true);
  const [search, setSearch] = useState('');
  const [filter, setFilter] = useState('all'); // all | enabled | configured
  const [editingProvider, setEditingProvider] = useState(null);
  const [isNewProvider, setIsNewProvider] = useState(false);
  const [deletingProvider, setDeletingProvider] = useState(null);

  useEffect(() => {
    loadProviders();
  }, []);

  const loadProviders = async () => {
    setLoading(true);
    try {
      const res = await getAIProviders();
      if (res.code === 0) {
        setProviders(res.data || []);
      }
    } catch (err) {
      toast.error('加载模型厂商失败');
    }
    setLoading(false);
  };

  const handleToggle = async (provider) => {
    try {
      const res = await updateAIProvider(provider.id, {
        is_enabled: !provider.is_enabled,
        is_default: provider.is_default,
      });
      if (res.code === 0) {
        toast.success(provider.is_enabled ? `已禁用 ${provider.label}` : `已启用 ${provider.label}`);
        loadProviders();
      }
    } catch (err) {
      toast.error('操作失败');
    }
  };

  const handleSetDefault = async (provider) => {
    try {
      const res = await updateAIProvider(provider.id, {
        is_default: !provider.is_default,
        is_enabled: true,
      });
      if (res.code === 0) {
        toast.success(provider.is_default ? `已取消默认` : `已将 ${provider.label} 设为默认`);
        loadProviders();
      }
    } catch (err) {
      toast.error('操作失败');
    }
  };

  const filteredProviders = providers.filter(p => {
    const matchSearch = !search ||
      p.label.toLowerCase().includes(search.toLowerCase()) ||
      p.name.toLowerCase().includes(search.toLowerCase()) ||
      p.description?.toLowerCase().includes(search.toLowerCase()) ||
      p.model?.toLowerCase().includes(search.toLowerCase());

    if (filter === 'enabled') return matchSearch && p.is_enabled;
    if (filter === 'configured') return matchSearch && p.configured;
    return matchSearch;
  });

  const stats = {
    total: providers.length,
    enabled: providers.filter(p => p.is_enabled).length,
    configured: providers.filter(p => p.configured).length,
    defaultName: providers.find(p => p.is_default)?.label || '未设置',
  };

  return (
    <div className="h-full overflow-y-auto">
      <div className="p-4 sm:p-6 space-y-6">

        {/* Stats Cards */}
        <div className="grid grid-cols-2 lg:grid-cols-4 gap-4">
          <div className="bg-white rounded-xl border border-gray-200 p-4 shadow-sm">
            <div className="flex items-center justify-between">
              <div>
                <p className="text-xs text-gray-500 mb-1">总厂商数</p>
                <p className="text-2xl font-bold text-gray-900">{stats.total}</p>
              </div>
              <div className="w-10 h-10 bg-primary-50 rounded-xl flex items-center justify-center">
                <Cpu className="w-5 h-5 text-primary-500" />
              </div>
            </div>
          </div>
          <div className="bg-white rounded-xl border border-gray-200 p-4 shadow-sm">
            <div className="flex items-center justify-between">
              <div>
                <p className="text-xs text-gray-500 mb-1">已启用</p>
                <p className="text-2xl font-bold text-green-600">{stats.enabled}</p>
              </div>
              <div className="w-10 h-10 bg-green-50 rounded-xl flex items-center justify-center">
                <CheckCircle className="w-5 h-5 text-green-500" />
              </div>
            </div>
          </div>
          <div className="bg-white rounded-xl border border-gray-200 p-4 shadow-sm">
            <div className="flex items-center justify-between">
              <div>
                <p className="text-xs text-gray-500 mb-1">已配置 Key</p>
                <p className="text-2xl font-bold text-purple-600">{stats.configured}</p>
              </div>
              <div className="w-10 h-10 bg-purple-50 rounded-xl flex items-center justify-center">
                <Key className="w-5 h-5 text-purple-500" />
              </div>
            </div>
          </div>
          <div className="bg-white rounded-xl border border-gray-200 p-4 shadow-sm">
            <div className="flex items-center justify-between">
              <div>
                <p className="text-xs text-gray-500 mb-1">默认模型</p>
                <p className="text-lg font-bold text-yellow-600 truncate">{stats.defaultName}</p>
              </div>
              <div className="w-10 h-10 bg-yellow-50 rounded-xl flex items-center justify-center">
                <Star className="w-5 h-5 text-yellow-500" />
              </div>
            </div>
          </div>
        </div>

        {/* Toolbar */}
        <div className="flex flex-col sm:flex-row items-start sm:items-center justify-between gap-3">
          <div className="flex items-center gap-3 flex-wrap">
            {/* Search */}
            <div className="relative">
              <Search className="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-gray-400" />
              <input
                value={search}
                onChange={e => setSearch(e.target.value)}
                placeholder="搜索厂商名称、模型..."
                className="pl-9 pr-4 py-2.5 bg-white border border-gray-200 rounded-xl text-sm focus:ring-2 focus:ring-primary-500 focus:border-primary-500 outline-none w-64"
              />
            </div>
            {/* Filter */}
            <div className="flex items-center bg-white border border-gray-200 rounded-xl overflow-hidden">
              {[
                { key: 'all', label: '全部' },
                { key: 'enabled', label: '已启用' },
                { key: 'configured', label: '已配置' },
              ].map(f => (
                <button
                  key={f.key}
                  onClick={() => setFilter(f.key)}
                  className={`px-4 py-2.5 text-sm transition-colors ${
                    filter === f.key
                      ? 'bg-primary-500 text-white'
                      : 'text-gray-600 hover:bg-gray-50'
                  }`}
                >
                  {f.label}
                </button>
              ))}
            </div>
          </div>

          <div className="flex items-center gap-2">
            <button
              onClick={loadProviders}
              className="p-2.5 text-gray-500 hover:text-primary-600 bg-white border border-gray-200 hover:border-primary-300 rounded-xl transition-colors"
              title="刷新"
            >
              <RefreshCw className="w-4 h-4" />
            </button>
            <button
              onClick={() => { setIsNewProvider(true); setEditingProvider({}); }}
              className="px-4 py-2.5 text-sm text-white bg-primary-500 hover:bg-primary-600 rounded-xl transition-colors flex items-center gap-2 shadow-sm"
            >
              <Plus className="w-4 h-4" />
              添加厂商
            </button>
          </div>
        </div>

        {/* Info Banner */}
        <div className="bg-primary-50 border border-primary-200 rounded-xl p-4 flex items-start gap-3">
          <Info className="w-5 h-5 text-primary-500 mt-0.5 flex-shrink-0" />
          <div>
            <p className="text-sm text-primary-800 font-medium">AI 模型厂商配置</p>
            <p className="text-xs text-primary-600 mt-1">
              配置各厂商的 API Key 后，智能体即可调用对应的大语言模型。支持 {stats.total} 家主流 AI 模型提供商，包括 OpenAI、DeepSeek、通义千问、智谱GLM、MiniMax、硅基流动、Moonshot、百度文心一言、火山引擎、腾讯混元、百川智能、Anthropic Claude、Google Gemini 等。
            </p>
          </div>
        </div>

        {/* Provider Grid */}
        {loading ? (
          <div className="flex items-center justify-center py-20">
            <Loader2 className="w-8 h-8 text-primary-500 animate-spin" />
            <span className="ml-3 text-gray-500">加载中...</span>
          </div>
        ) : filteredProviders.length === 0 ? (
          <div className="text-center py-20">
            <Cpu className="w-12 h-12 text-gray-300 mx-auto mb-3" />
            <p className="text-gray-500">
              {search || filter !== 'all' ? '没有匹配的模型厂商' : '暂无模型厂商数据'}
            </p>
          </div>
        ) : (
          <div className="grid grid-cols-1 md:grid-cols-2 xl:grid-cols-3 gap-5">
            {filteredProviders.map(provider => (
              <ProviderCard
                key={provider.id}
                provider={provider}
                onEdit={p => { setIsNewProvider(false); setEditingProvider(p); }}
                onToggle={handleToggle}
                onSetDefault={handleSetDefault}
                onDelete={p => setDeletingProvider(p)}
              />
            ))}
          </div>
        )}

        {/* Provider Reference Table */}
        <div className="bg-white rounded-xl border border-gray-200 shadow-sm overflow-hidden">
          <div className="px-6 py-4 border-b border-gray-100 flex items-center gap-2">
            <Zap className="w-5 h-5 text-primary-500" />
            <h3 className="font-semibold text-gray-900">支持的 AI 模型厂商参考</h3>
          </div>
          <div className="overflow-x-auto">
            <table className="w-full text-sm">
              <thead>
                <tr className="bg-gray-50">
                  <th className="px-4 py-3 text-left text-xs font-semibold text-gray-500 uppercase">厂商</th>
                  <th className="px-4 py-3 text-left text-xs font-semibold text-gray-500 uppercase">标识</th>
                  <th className="px-4 py-3 text-left text-xs font-semibold text-gray-500 uppercase">推荐模型</th>
                  <th className="px-4 py-3 text-left text-xs font-semibold text-gray-500 uppercase">说明</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-gray-100">
                {[
                  { name: 'OpenAI', icon: '🤖', model: 'gpt-4o', desc: 'GPT-4o / GPT-4 / GPT-3.5 系列' },
                  { name: 'DeepSeek', icon: '🔍', model: 'deepseek-chat', desc: '深度求索，高性价比国产大模型' },
                  { name: '通义千问', icon: '☁️', model: 'qwen-plus', desc: '阿里云 Qwen-Plus / Qwen-Max 系列' },
                  { name: '智谱 GLM', icon: '🧠', model: 'glm-4', desc: '智谱 AI GLM-4 / GLM-4-Flash 系列' },
                  { name: 'MiniMax', icon: '⚡', model: 'abab6.5s-chat', desc: 'MiniMax abab 系列' },
                  { name: '硅基流动', icon: '💎', model: 'Qwen/Qwen2.5-7B-Instruct', desc: '支持 Qwen / DeepSeek / GLM 开源模型推理' },
                  { name: 'Moonshot (Kimi)', icon: '🌙', model: 'moonshot-v1-8k', desc: '超长上下文，8k / 32k / 128k' },
                  { name: '百度文心一言', icon: '🔵', model: 'ernie-4.5-8k', desc: 'ERNIE 4.5 / 4.0 / Speed 系列' },
                  { name: '火山引擎（豆包）', icon: '🔥', model: 'doubao-pro-4k', desc: '字节豆包 doubao-pro / lite 系列' },
                  { name: '腾讯混元', icon: '🌀', model: 'hunyuan-pro', desc: '混元 pro / standard 系列' },
                  { name: '百川智能', icon: '🕊️', model: 'Baichuan4', desc: 'Baichuan4 / Baichuan3-Turbo 系列' },
                  { name: 'Anthropic Claude', icon: '🎭', model: 'claude-3-5-sonnet-20241022', desc: 'claude-3-5-sonnet / haiku / opus' },
                  { name: 'Google Gemini', icon: '✨', model: 'gemini-2.0-flash', desc: 'gemini-2.0-flash / 1.5-pro 系列' },
                ].map((row, idx) => (
                  <tr key={idx} className="hover:bg-gray-50 transition-colors">
                    <td className="px-4 py-3 font-medium text-gray-900">{row.name}</td>
                    <td className="px-4 py-3 text-xl">{row.icon}</td>
                    <td className="px-4 py-3">
                      <span className="font-mono text-xs bg-primary-50 text-primary-700 px-2 py-1 rounded">{row.model}</span>
                    </td>
                    <td className="px-4 py-3 text-gray-500">{row.desc}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </div>
      </div>

      {/* Modals */}
      {editingProvider && (
        <ProviderModal
          provider={editingProvider}
          isNew={isNewProvider}
          onClose={() => { setEditingProvider(null); setIsNewProvider(false); }}
          onSave={() => { setEditingProvider(null); setIsNewProvider(false); loadProviders(); }}
        />
      )}
      {deletingProvider && (
        <DeleteModal
          provider={deletingProvider}
          onClose={() => setDeletingProvider(null)}
          onConfirm={() => { setDeletingProvider(null); loadProviders(); }}
        />
      )}
    </div>
  );
}
