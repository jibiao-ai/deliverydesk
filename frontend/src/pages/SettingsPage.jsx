import React, { useState, useEffect } from 'react';
import { Settings, Save, Eye, EyeOff, RefreshCw, Server, Key, MessageSquare } from 'lucide-react';
import toast from 'react-hot-toast';
import { getSettings, updateSettings } from '../services/api';

const CATEGORY_META = {
  jira: { label: 'Jira 配置', icon: Server, description: '配置 Jira 集成参数，用于工单验证和自动关联' },
  totp: { label: 'TOTP / 双因子', icon: Key, description: '配置双因子认证服务参数（密码生成、自动审批等）' },
  wechat: { label: '企业微信', icon: MessageSquare, description: '配置企业微信通知集成参数' },
};

export default function SettingsPage() {
  const [loading, setLoading] = useState(false);
  const [saving, setSaving] = useState(false);
  const [settings, setSettings] = useState({});
  const [modified, setModified] = useState({});
  const [visiblePasswords, setVisiblePasswords] = useState({});

  const fetchSettings = async () => {
    setLoading(true);
    try {
      const res = await getSettings({});
      if (res?.code === 0) {
        setSettings(res.data || {});
        setModified({});
      }
    } catch (e) {
      toast.error('加载设置失败');
    }
    setLoading(false);
  };

  useEffect(() => { fetchSettings(); }, []);

  const handleChange = (id, category, key, value) => {
    setModified((prev) => ({ ...prev, [`${category}.${key}`]: { id, category, key, value } }));
  };

  const handleSave = async () => {
    const changes = Object.values(modified);
    if (changes.length === 0) {
      toast('没有修改', { icon: 'ℹ️' });
      return;
    }
    setSaving(true);
    try {
      const res = await updateSettings({ settings: changes });
      if (res?.code === 0) {
        toast.success('设置已保存');
        fetchSettings();
      } else {
        toast.error(res?.message || '保存失败');
      }
    } catch (e) {
      toast.error('保存失败');
    }
    setSaving(false);
  };

  const togglePassword = (key) => {
    setVisiblePasswords((prev) => ({ ...prev, [key]: !prev[key] }));
  };

  const categories = Object.keys(settings);
  const hasChanges = Object.keys(modified).length > 0;

  return (
    <div className="h-full flex flex-col overflow-hidden">
      {/* Header */}
      <div className="flex items-center justify-between px-6 py-3 border-b border-gray-200 bg-white flex-shrink-0">
        <div className="flex items-center gap-3">
          <div className="w-9 h-9 rounded-xl bg-primary-100 flex items-center justify-center">
            <Settings className="w-4.5 h-4.5 text-primary-600" />
          </div>
          <div>
            <h2 className="text-sm font-semibold text-gray-800">系统设置</h2>
            <p className="text-xs text-gray-400">管理集成配置和服务参数</p>
          </div>
        </div>
        <div className="flex items-center gap-2">
          <button
            onClick={fetchSettings}
            className="p-2 text-gray-400 hover:text-gray-600 hover:bg-gray-100 rounded-lg transition-colors"
          >
            <RefreshCw className={`w-4 h-4 ${loading ? 'animate-spin' : ''}`} />
          </button>
          <button
            onClick={handleSave}
            disabled={!hasChanges || saving}
            className={`flex items-center gap-1.5 px-4 py-2 rounded-lg text-sm font-medium transition-all ${
              hasChanges
                ? 'bg-primary-600 text-white hover:bg-primary-700 shadow-sm'
                : 'bg-gray-100 text-gray-400 cursor-not-allowed'
            }`}
          >
            <Save className="w-4 h-4" />
            {saving ? '保存中...' : '保存设置'}
          </button>
        </div>
      </div>

      {/* Settings Grid */}
      <div className="flex-1 overflow-auto px-6 py-4 space-y-6">
        {loading ? (
          <div className="flex items-center justify-center h-48 text-gray-400">加载中...</div>
        ) : (
          categories.map((category) => {
            const meta = CATEGORY_META[category] || { label: category, icon: Settings, description: '' };
            const Icon = meta.icon;
            const items = settings[category] || [];

            return (
              <div key={category} className="bg-white rounded-xl border border-gray-200 overflow-hidden">
                {/* Category Header */}
                <div className="px-6 py-4 border-b border-gray-100 bg-gray-50/50">
                  <div className="flex items-center gap-3">
                    <div className="w-8 h-8 rounded-lg bg-primary-50 flex items-center justify-center">
                      <Icon className="w-4 h-4 text-primary-600" />
                    </div>
                    <div>
                      <h3 className="text-sm font-semibold text-gray-800">{meta.label}</h3>
                      <p className="text-xs text-gray-400">{meta.description}</p>
                    </div>
                  </div>
                </div>

                {/* Settings Items */}
                <div className="divide-y divide-gray-50">
                  {items.map((item) => {
                    const fullKey = `${category}.${item.key}`;
                    const currentValue = modified[fullKey]?.value ?? item.value;
                    const isPassword = item.value_type === 'password';
                    const isBoolean = item.value_type === 'boolean';
                    const isModified = modified[fullKey] !== undefined;

                    return (
                      <div key={item.id} className="px-6 py-4 flex items-center gap-4">
                        <div className="flex-1 min-w-0">
                          <div className="flex items-center gap-2">
                            <span className="text-sm font-medium text-gray-700">{item.label}</span>
                            {isModified && (
                              <span className="px-1.5 py-0.5 bg-yellow-100 text-yellow-700 text-xs rounded">已修改</span>
                            )}
                          </div>
                          <span className="text-xs text-gray-400 font-mono">{item.key}</span>
                        </div>
                        <div className="w-80 flex items-center gap-2">
                          {isBoolean ? (
                            <label className="relative inline-flex items-center cursor-pointer">
                              <input
                                type="checkbox"
                                checked={currentValue === 'true'}
                                onChange={(e) => handleChange(item.id, category, item.key, e.target.checked ? 'true' : 'false')}
                                className="sr-only peer"
                              />
                              <div className="w-9 h-5 bg-gray-200 peer-focus:outline-none peer-focus:ring-2 peer-focus:ring-primary-300 rounded-full peer peer-checked:after:translate-x-full peer-checked:after:border-white after:content-[''] after:absolute after:top-[2px] after:left-[2px] after:bg-white after:border-gray-300 after:border after:rounded-full after:h-4 after:w-4 after:transition-all peer-checked:bg-primary-600"></div>
                              <span className="ml-2 text-sm text-gray-600">{currentValue === 'true' ? '启用' : '禁用'}</span>
                            </label>
                          ) : (
                            <div className="relative flex-1">
                              <input
                                type={isPassword && !visiblePasswords[fullKey] ? 'password' : 'text'}
                                value={currentValue}
                                onChange={(e) => handleChange(item.id, category, item.key, e.target.value)}
                                className="w-full px-3 py-2 border border-gray-300 rounded-lg text-sm focus:ring-2 focus:ring-primary-500 focus:border-primary-500 pr-10"
                                placeholder={`输入${item.label}`}
                              />
                              {isPassword && (
                                <button
                                  type="button"
                                  onClick={() => togglePassword(fullKey)}
                                  className="absolute right-2.5 top-1/2 -translate-y-1/2 text-gray-400 hover:text-gray-600"
                                >
                                  {visiblePasswords[fullKey] ? <EyeOff className="w-4 h-4" /> : <Eye className="w-4 h-4" />}
                                </button>
                              )}
                            </div>
                          )}
                        </div>
                      </div>
                    );
                  })}
                </div>
              </div>
            );
          })
        )}
      </div>
    </div>
  );
}
