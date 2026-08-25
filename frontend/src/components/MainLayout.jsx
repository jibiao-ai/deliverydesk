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
import TotpPage from '../pages/TotpPage';
import WorktimePage from '../pages/WorktimePage';
import ProjectManagePage from '../pages/ProjectManagePage';
import OpsEnvironmentPage from '../pages/OpsEnvironmentPage';
import SettingsPage from '../pages/SettingsPage';
import KnowledgeBasePage from '../pages/KnowledgeBasePage';
import WBSServicePage from '../pages/WBSServicePage';
import WorkflowPage from '../pages/WorkflowPage';
import useStore from '../store/useStore';
import { Bell, Sun, Moon } from 'lucide-react';

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
  totp: TotpPage,
  worktime: WorktimePage,
  projects: ProjectManagePage,
  'ops-env': OpsEnvironmentPage,
  settings: SettingsPage,
  kb: KnowledgeBasePage,
  wbs: WBSServicePage,
  workflows: WorkflowPage,
};

const PAGE_META = {
  dashboard:        { title: '\u4eea\u8868\u76d8',     subtitle: '\u4e91\u4ea4\u4ed8\u670d\u52a1\u53f0\u6982\u89c8' },
  websites:         { title: '\u516c\u53f8\u7cfb\u7edf',   subtitle: '\u516c\u53f8\u5185\u90e8\u7cfb\u7edf\u4e0e\u4ea4\u4ed8\u8d44\u6e90\u5feb\u901f\u5bfc\u822a' },
  chat:             { title: '\u5373\u65f6\u5bf9\u8bdd',   subtitle: '\u4e0e AI \u667a\u80fd\u4f53\u5b9e\u65f6\u4ea4\u4e92' },
  agents:           { title: '\u667a\u80fd\u4f53',     subtitle: '\u7ba1\u7406\u548c\u914d\u7f6e AI \u667a\u80fd\u4f53' },
  skills:           { title: '\u6280\u80fd\u5546\u5e97',   subtitle: '\u67e5\u770b\u548c\u7ba1\u7406\u5e73\u53f0\u6280\u80fd' },
  'ai-models':      { title: '\u6a21\u578b\u914d\u7f6e',   subtitle: '\u914d\u7f6e AI \u670d\u52a1\u63d0\u4f9b\u5546\u53c2\u6570' },
  ldap:             { title: 'LDAP\u7ba1\u7406',   subtitle: '\u914d\u7f6e\u4f01\u4e1aLDAP\u8ba4\u8bc1\u670d\u52a1' },
  users:            { title: '\u7528\u6237\u7ba1\u7406',   subtitle: '\u7ba1\u7406\u5e73\u53f0\u7528\u6237\u8d26\u53f7\u548c\u6743\u9650' },
  'operation-logs': { title: '\u64cd\u4f5c\u65e5\u5fd7',   subtitle: '\u8bb0\u5f55\u5e73\u53f0\u5173\u952e\u64cd\u4f5c' },
  totp:             { title: '\u53cc\u56e0\u5b50\u7ba1\u7406', subtitle: '\u7533\u8bf7\u548c\u5ba1\u6838\u53cc\u56e0\u5b50\u8ba4\u8bc1\u5bc6\u7801' },
  worktime:         { title: '\u5de5\u65f6\u7ba1\u7406',   subtitle: 'Redmine \u5de5\u65f6\u6570\u636e\u7edf\u8ba1\u4e0e\u5206\u6790' },
  projects:         { title: '\u9879\u76ee\u7ba1\u7406',   subtitle: 'Redmine \u9879\u76ee\u7acb\u9879\u6570\u636e\u7edf\u8ba1\u770b\u677f' },
  'ops-env':        { title: '\u8fd0\u7ef4\u73af\u5883',   subtitle: 'Jira \u8fd0\u7ef4\u73af\u5883\u72b6\u6001\u4e0e\u5f03\u7528\u8ffd\u8e2a' },
  settings:         { title: '\u7cfb\u7edf\u8bbe\u7f6e',   subtitle: '\u7ba1\u7406\u96c6\u6210\u914d\u7f6e\u548c\u670d\u52a1\u53c2\u6570' },
  kb:               { title: '\u77e5\u8bc6\u5e93\u751f\u6210', subtitle: 'Jira \u5de5\u5355 \u2192 Confluence \u77e5\u8bc6\u5e93\uff08AI \u6da6\u8272\uff09' },
  wbs:              { title: 'WBS\u670d\u52a1', subtitle: '\u9879\u76ee\u5de5\u4f5c\u4efb\u52a1\u5206\u89e3\u4e0e\u4ea7\u54c1\u670d\u52a1\u62a5\u4ef7\u6c47\u603b' },
  workflows:        { title: '\u5de5\u4f5c\u6d41',     subtitle: '\u53ef\u89c6\u5316\u7f16\u6392\u667a\u80fd\u4f53\u4e0e\u6280\u80fd\u7684\u81ea\u52a8\u5316\u6d41\u7a0b' },
};

const THEMES = [
  { id: 'light', label: '\u6d45\u8272\u6a21\u5f0f', icon: Sun },
  { id: 'dark',  label: '\u6df1\u8272\u6a21\u5f0f', icon: Moon },
];

// Pages that require admin role
const ADMIN_PAGES = new Set(['agents', 'ai-models', 'skills', 'ldap', 'users', 'operation-logs', 'settings', 'worktime', 'projects', 'ops-env', 'workflows']);

export default function MainLayout() {
  const activePage = useStore((s) => s.activePage);
  const setActivePage = useStore((s) => s.setActivePage);
  const theme = useStore((s) => s.theme);
  const setTheme = useStore((s) => s.setTheme);
  const user = useStore((s) => s.user);
  const isDark = theme === 'dark';

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
    <div className={`flex h-screen overflow-hidden ${isDark ? 'bg-slate-900' : 'bg-gray-50'}`}>
      <Sidebar />
      <div className="flex-1 flex flex-col overflow-hidden min-w-0">
        <header className={`h-14 border-b flex items-center px-6 flex-shrink-0 z-10 ${
          isDark ? 'bg-slate-800 border-slate-700' : 'bg-white border-gray-200'
        }`}>
          <div className="flex-1 min-w-0">
            <h1 className={`text-lg font-semibold leading-tight ${isDark ? 'text-slate-100' : 'text-gray-800'}`}>{meta.title}</h1>
            {meta.subtitle && (
              <p className={`text-sm leading-tight hidden sm:block ${isDark ? 'text-slate-400' : 'text-gray-400'}`}>{meta.subtitle}</p>
            )}
          </div>
          <div className="flex items-center gap-3 flex-shrink-0">
            {/* Theme toggle */}
            <div className={`flex items-center gap-0.5 rounded-lg px-1 py-1 ${isDark ? 'bg-slate-700 border border-slate-600' : 'bg-gray-50 border border-gray-200'}`}>
              {THEMES.map((t) => {
                const Icon = t.icon;
                const isActive = theme === t.id;
                return (
                  <button
                    key={t.id}
                    title={t.label}
                    onClick={() => setTheme(t.id)}
                    className={`flex items-center justify-center w-7 h-7 rounded-md transition-all ${
                      isActive
                        ? isDark
                          ? 'bg-slate-600 text-primary-300 shadow-sm'
                          : 'bg-white text-primary-600 shadow-sm'
                        : isDark
                          ? 'text-slate-400 hover:text-slate-200'
                          : 'text-gray-400 hover:text-gray-600'
                    }`}
                  >
                    <Icon className="w-4 h-4" />
                  </button>
                );
              })}
            </div>
            <button className={`w-8 h-8 flex items-center justify-center rounded-lg transition-colors relative ${
              isDark ? 'text-slate-400 hover:text-slate-200 hover:bg-slate-700' : 'text-gray-400 hover:text-gray-600 hover:bg-gray-100'
            }`}>
              <Bell className="w-4 h-4" />
              <span className="absolute top-1.5 right-1.5 w-1.5 h-1.5 bg-red-500 rounded-full" />
            </button>
            <div className="flex items-center gap-2">
              <div className="w-8 h-8 rounded-full flex items-center justify-center text-xs font-bold text-white flex-shrink-0 bg-primary-600">
                {(user?.username || 'U').slice(0, 1).toUpperCase()}
              </div>
              <span className={`text-sm font-medium hidden md:block ${isDark ? 'text-slate-200' : 'text-gray-700'}`}>
                {user?.username || 'admin'}
              </span>
            </div>
          </div>
        </header>
        <main className={`flex-1 overflow-hidden ${isDark ? 'bg-slate-900' : 'bg-gray-50'}`}>
          <PageComponent />
        </main>
      </div>
    </div>
  );
}
