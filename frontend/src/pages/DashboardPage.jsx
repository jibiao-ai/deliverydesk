import React, { useEffect, useState } from 'react';
import { MessageSquare, Bot, Zap, Globe, Cpu, ArrowRight, Plus } from 'lucide-react';
import { getDashboard } from '../services/api';
import useStore from '../store/useStore';

export default function DashboardPage() {
  const [stats, setStats] = useState(null);
  const [loading, setLoading] = useState(true);
  const setActivePage = useStore((s) => s.setActivePage);

  useEffect(() => { loadDashboard(); }, []);

  const loadDashboard = async () => {
    try {
      const res = await getDashboard();
      if (res.code === 0) setStats(res.data);
    } catch (err) { console.error(err); }
    finally { setLoading(false); }
  };

  const statCards = [
    { label: '公司系统链接', value: stats?.website_links ?? '--', icon: Globe, iconBg: 'bg-primary-100', iconColor: 'text-primary-600', trend: '已收录' },
    { label: 'AI 模型数', value: stats?.ai_models ?? '--', icon: Cpu, iconBg: 'bg-purple-100', iconColor: 'text-purple-600', trend: '已配置' },
    { label: '活跃 Agent 数', value: stats?.agents ?? '--', icon: Bot, iconBg: 'bg-emerald-100', iconColor: 'text-emerald-600', trend: '运行中' },
    { label: '对话数', value: stats?.conversations ?? '--', icon: MessageSquare, iconBg: 'bg-orange-100', iconColor: 'text-orange-600', trend: '累计' },
  ];

  const quickActions = [
    { label: '公司系统', desc: '快速访问公司内部系统和工具', icon: Globe, iconBg: 'bg-primary-50', iconColor: 'text-primary-600', page: 'websites' },
    { label: '智能对话', desc: '与 AI 智能体开始新会话', icon: MessageSquare, iconBg: 'bg-emerald-50', iconColor: 'text-emerald-600', page: 'chat' },
    { label: '配置模型', desc: '添加或更新 AI 模型参数', icon: Cpu, iconBg: 'bg-purple-50', iconColor: 'text-purple-600', page: 'ai-models' },
    { label: '查看技能', desc: '浏览平台可用技能列表', icon: Zap, iconBg: 'bg-amber-50', iconColor: 'text-amber-600', page: 'skills' },
  ];

  const recentConvs = stats?.recent_conversations || [];

  return (
    <div className="h-full overflow-y-auto">
      <div className="p-4 sm:p-6 space-y-4 sm:space-y-6 w-full">
        <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-3 sm:gap-4">
          {statCards.map((card, i) => {
            const Icon = card.icon;
            return (
              <div key={i} className="bg-white rounded-xl border border-gray-200 shadow-sm p-5 flex flex-col gap-3 hover:shadow-md transition-shadow">
                <div className="flex items-start justify-between">
                  <div className={`w-10 h-10 rounded-xl flex items-center justify-center ${card.iconBg}`}>
                    <Icon className={`w-5 h-5 ${card.iconColor}`} />
                  </div>
                  <span className="flex items-center gap-0.5 text-xs text-emerald-600">{card.trend}</span>
                </div>
                <p className="text-3xl font-bold text-gray-800">{loading ? '--' : card.value}</p>
                <p className="text-sm text-gray-500">{card.label}</p>
              </div>
            );
          })}
        </div>

        <div className="grid grid-cols-1 lg:grid-cols-3 gap-4 sm:gap-6">
          <div className="lg:col-span-2 bg-white rounded-xl border border-gray-200 shadow-sm">
            <div className="px-6 py-4 border-b border-gray-100 flex items-center justify-between">
              <div>
                <h2 className="text-base font-semibold text-gray-800">最近对话</h2>
                <p className="text-sm text-gray-400 mt-0.5">近期 AI 对话记录</p>
              </div>
              <button onClick={() => setActivePage('chat')} className="flex items-center gap-1 text-sm font-medium text-primary-600">
                查看全部 <ArrowRight className="w-3.5 h-3.5" />
              </button>
            </div>
            <div className="p-6">
              {recentConvs.length > 0 ? (
                <div className="space-y-1">
                  {recentConvs.slice(0, 6).map((conv, i) => (
                    <div key={conv.id || i} className="flex items-center gap-3 px-3 py-2.5 rounded-lg hover:bg-gray-50 cursor-pointer transition-colors group"
                      onClick={() => setActivePage('chat')}>
                      <div className="w-8 h-8 rounded-full flex items-center justify-center flex-shrink-0 bg-primary-50">
                        <MessageSquare className="w-4 h-4 text-primary-600" />
                      </div>
                      <div className="flex-1 min-w-0">
                        <p className="text-sm font-medium text-gray-700 truncate group-hover:text-primary-600">{conv.title || `对话 ${i + 1}`}</p>
                      </div>
                      <ArrowRight className="w-3.5 h-3.5 text-gray-300 group-hover:text-primary-600" />
                    </div>
                  ))}
                </div>
              ) : (
                <div className="text-center py-10">
                  <div className="w-12 h-12 bg-gray-100 rounded-full flex items-center justify-center mx-auto mb-3">
                    <MessageSquare className="w-6 h-6 text-gray-300" />
                  </div>
                  <p className="text-sm text-gray-400 mb-3">暂无对话记录</p>
                  <button onClick={() => setActivePage('chat')}
                    className="text-sm font-medium px-4 py-2 rounded-lg text-white bg-primary-600">
                    <span className="flex items-center gap-1.5"><Plus className="w-4 h-4" /> 开始第一个对话</span>
                  </button>
                </div>
              )}
            </div>
          </div>

          <div className="bg-white rounded-xl border border-gray-200 shadow-sm">
            <div className="px-6 py-4 border-b border-gray-100">
              <h2 className="text-base font-semibold text-gray-800">快捷操作</h2>
              <p className="text-sm text-gray-400 mt-0.5">常用功能入口</p>
            </div>
            <div className="p-4 space-y-2">
              {quickActions.map((action, i) => {
                const Icon = action.icon;
                return (
                  <button key={i} onClick={() => setActivePage(action.page)}
                    className="w-full flex items-center gap-3 px-4 py-3 rounded-xl border border-gray-100 hover:border-primary-600 hover:bg-primary-50 transition-all text-left group">
                    <div className={`w-9 h-9 rounded-lg flex items-center justify-center flex-shrink-0 ${action.iconBg} group-hover:scale-110 transition-transform`}>
                      <Icon className={`w-4 h-4 ${action.iconColor}`} />
                    </div>
                    <div className="flex-1 min-w-0">
                      <p className="text-sm font-medium text-gray-700 group-hover:text-primary-600">{action.label}</p>
                      <p className="text-xs text-gray-400 truncate">{action.desc}</p>
                    </div>
                    <ArrowRight className="w-3.5 h-3.5 text-gray-300 group-hover:text-primary-600" />
                  </button>
                );
              })}
            </div>
          </div>
        </div>
      </div>
    </div>
  );
}
