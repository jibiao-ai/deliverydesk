import React, { useEffect } from 'react';
import Sidebar from './Sidebar';
import DashboardPage from '../pages/DashboardPage';
import WebsitesPage from '../pages/WebsitesPage';
import ChatPage from '../pages/ChatPage';
import AgentsPage from '../pages/AgentsPage';
import SkillsPage from '../pages/SkillsPage';
import AIModelsPage from '../pages/AIModelsPage';
import LDAPPage from '../pages/LDAPPage';
import UsersPage from '../pages/UsersPage';
import OperationLogPage from '../pages/OperationLogPage';
import useStore from '../store/useStore';
import { Bell } from 'lucide-react';

const pageComponents = {
  dashboard: DashboardPage,
  websites: WebsitesPage,
  chat: ChatPage,
  agents: AgentsPage,
  skills: SkillsPage,
  'ai-models': AIModelsPage,
  ldap: LDAPPage,
  users: UsersPage,
  'operation-logs': OperationLogPage,
};

const PAGE_META = {
  dashboard:        { title: '仪表盘',     subtitle: '云交付服务台概览' },
  websites:         { title: '公司系统',   subtitle: '公司内部系统与交付资源快速导航' },
  chat:             { title: '即时对话',   subtitle: '与 AI 智能体实时交互' },
  agents:           { title: '智能体',     subtitle: '管理和配置 AI 智能体' },
  skills:           { title: '技能商店',   subtitle: '查看和管理平台技能' },
  'ai-models':      { title: '模型配置',   subtitle: '配置 AI 服务提供商参数' },
  ldap:             { title: 'LDAP管理',   subtitle: '配置企业LDAP认证服务' },
  users:            { title: '用户管理',   subtitle: '管理平台用户账号和权限' },
  'operation-logs': { title: '操作日志',   subtitle: '记录平台关键操作' },
};

const THEMES = [
  { id: 'light', label: '白色主题', bg: '#ffffff', border: '#e5e7eb' },
  { id: 'dark',  label: '暗色主题', bg: '#0f0e17', border: '#374151' },
];

// Pages that require admin role
const ADMIN_PAGES = new Set(['ai-models', 'skills', 'ldap', 'users', 'operation-logs']);

export default function MainLayout() {
  const activePage = useStore((s) => s.activePage);
  const setActivePage = useStore((s) => s.setActivePage);
  const theme = useStore((s) => s.theme);
  const setTheme = useStore((s) => s.setTheme);
  const user = useStore((s) => s.user);

  // Redirect non-admin users away from admin pages
  const effectivePage = (user?.role !== 'admin' && ADMIN_PAGES.has(activePage))
    ? 'dashboard'
    : activePage;

  // Auto-redirect if needed
  React.useEffect(() => {
    if (effectivePage !== activePage) {
      setActivePage(effectivePage);
    }
  }, [effectivePage, activePage, setActivePage]);

  const PageComponent = pageComponents[effectivePage] || DashboardPage;
  const meta = PAGE_META[effectivePage] || { title: effectivePage, subtitle: '' };

  useEffect(() => {
    document.documentElement.setAttribute('data-theme', theme);
  }, [theme]);

  return (
    <div className="flex h-screen bg-gray-50 overflow-hidden">
      <Sidebar />
      <div className="flex-1 flex flex-col overflow-hidden min-w-0">
        <header className="h-14 bg-white border-b border-gray-200 flex items-center px-6 flex-shrink-0 z-10">
          <div className="flex-1 min-w-0">
            <h1 className="text-lg font-semibold text-gray-800 leading-tight">{meta.title}</h1>
            {meta.subtitle && (
              <p className="text-sm text-gray-400 leading-tight hidden sm:block">{meta.subtitle}</p>
            )}
          </div>
          <div className="flex items-center gap-3 flex-shrink-0">
            <div className="flex items-center gap-1.5 bg-gray-50 border border-gray-200 rounded-lg px-2 py-1.5">
              {THEMES.map((t) => (
                <button
                  key={t.id}
                  title={t.label}
                  onClick={() => setTheme(t.id)}
                  className={`w-4 h-4 rounded-full transition-all ring-offset-1 ${
                    theme === t.id ? 'ring-2 ring-primary-600 scale-110' : 'hover:ring-2 hover:ring-gray-300'
                  }`}
                  style={{ backgroundColor: t.bg, border: `1.5px solid ${t.border}` }}
                />
              ))}
            </div>
            <button className="w-8 h-8 flex items-center justify-center rounded-lg text-gray-400 hover:text-gray-600 hover:bg-gray-100 transition-colors relative">
              <Bell className="w-4 h-4" />
              <span className="absolute top-1.5 right-1.5 w-1.5 h-1.5 bg-red-500 rounded-full" />
            </button>
            <div className="flex items-center gap-2">
              <div className="w-8 h-8 rounded-full flex items-center justify-center text-xs font-bold text-white flex-shrink-0 bg-primary-600">
                {(user?.username || 'U').slice(0, 1).toUpperCase()}
              </div>
              <span className="text-sm font-medium text-gray-700 hidden md:block">
                {user?.username || 'admin'}
              </span>
            </div>
          </div>
        </header>
        <main className="flex-1 overflow-hidden bg-gray-50">
          <PageComponent />
        </main>
      </div>
    </div>
  );
}
