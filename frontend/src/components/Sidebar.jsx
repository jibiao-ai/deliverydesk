import React from 'react';
import {
  LayoutDashboard, MessageSquare, Bot, Zap, Users, LogOut, Menu, Cpu, Globe, Shield, FileText, Server, Settings, Key, Clock, FolderKanban, Monitor, BookOpen, FileSpreadsheet, GitBranch,
} from 'lucide-react';
import useStore from '../store/useStore';
import { useNavigate } from 'react-router-dom';

// Menu visible to ALL users (including normal users)
const userMenuGroups = [
  {
    label: '\u4ea4\u4ed8\u5de5\u4f5c\u53f0',
    items: [
      { id: 'dashboard', label: '\u4eea\u8868\u76d8', icon: LayoutDashboard },
      { id: 'websites', label: '\u516c\u53f8\u7cfb\u7edf', icon: Globe },
      { id: 'chat', label: '\u5373\u65f6\u5bf9\u8bdd', icon: MessageSquare },
      { id: 'totp', label: '\u53cc\u56e0\u5b50\u7ba1\u7406', icon: Key },
      { id: 'kb', label: '\u77e5\u8bc6\u5e93\u751f\u6210', icon: BookOpen },
      { id: 'wbs', label: 'WBS\u670d\u52a1', icon: FileSpreadsheet },
    ],
  },
];

// Menu visible to ADMIN users only
const adminMenuGroups = [
  {
    label: '\u667a\u80fd\u5e94\u7528',
    items: [
      { id: 'agents', label: '\u667a\u80fd\u4f53', icon: Bot },
      { id: 'skills', label: '\u6280\u80fd\u5546\u5e97', icon: Zap },
      { id: 'workflows', label: '\u5de5\u4f5c\u6d41', icon: GitBranch },
      { id: 'ai-models', label: '\u6a21\u578b\u914d\u7f6e', icon: Cpu },
    ],
  },
  {
    label: '\u4e1a\u52a1\u5e94\u7528',
    items: [
      { id: 'worktime', label: '\u5de5\u65f6\u7ba1\u7406', icon: Clock },
      { id: 'projects', label: '\u9879\u76ee\u7ba1\u7406', icon: FolderKanban },
      { id: 'ops-env', label: '\u8fd0\u7ef4\u7ba1\u7406', icon: Monitor },
    ],
  },
  {
    label: '\u7cfb\u7edf\u7ba1\u7406',
    items: [
      { id: 'ldap', label: 'LDAP\u7ba1\u7406', icon: Server },
      { id: 'users', label: '\u7528\u6237\u7ba1\u7406', icon: Users },
      { id: 'settings', label: '\u7cfb\u7edf\u8bbe\u7f6e', icon: Settings },
      { id: 'operation-logs', label: '\u64cd\u4f5c\u65e5\u5fd7', icon: FileText },
    ],
  },
];

export default function Sidebar() {
  const { activePage, setActivePage, user, logout, sidebarCollapsed, toggleSidebar, theme } = useStore();
  const navigate = useNavigate();
  const isDark = theme === 'dark';

  const handleLogout = () => {
    logout();
    navigate('/login');
  };

  const allGroups = user?.role === 'admin' ? [...userMenuGroups, ...adminMenuGroups] : userMenuGroups;

  return (
    <div
      className={`flex flex-col transition-all duration-300 flex-shrink-0 border-r ${
        isDark ? 'border-slate-700' : 'border-gray-200'
      } ${sidebarCollapsed ? 'w-16' : 'w-56'}`}
      style={{ background: isDark ? '#1e293b' : '#ffffff' }}
    >
      <div className={`flex items-center h-16 px-3 flex-shrink-0 border-b ${isDark ? 'border-slate-700' : 'border-gray-200'}`}>
        <button
          onClick={toggleSidebar}
          className={`flex items-center justify-center w-8 h-8 rounded-lg transition-colors flex-shrink-0 ${
            isDark ? 'text-slate-400 hover:text-primary-400 hover:bg-slate-700' : 'text-gray-500 hover:text-primary-600 hover:bg-primary-50'
          }`}
        >
          <Menu className="w-5 h-5" />
        </button>
        {!sidebarCollapsed && (
          <div className="ml-2 flex items-center gap-2 overflow-hidden">
            <div className="w-7 h-7 rounded-lg bg-primary-600 flex items-center justify-center flex-shrink-0">
              <Shield className="w-4 h-4 text-white" />
            </div>
            <span className="text-sm font-semibold whitespace-nowrap">
              <span className={isDark ? 'text-slate-200' : 'text-gray-800'}>Delivery</span><span style={{ color: isDark ? '#a78bfa' : '#513CC8' }}>Desk</span>
              <span className="text-xs ml-1 px-1 py-0.5 rounded" style={{ background: isDark ? '#312e81' : '#ddd5f6', color: isDark ? '#a78bfa' : '#513CC8', fontSize: '10px' }}>AI</span>
            </span>
          </div>
        )}
      </div>

      <nav className="flex-1 overflow-y-auto py-2" style={{ scrollbarWidth: 'none' }}>
        {allGroups.map((group, groupIdx) => (
          <div key={groupIdx} className="mb-1">
            {!sidebarCollapsed && (
              <div className={`px-4 pt-4 pb-1 text-xs uppercase tracking-widest font-medium ${isDark ? 'text-slate-500' : 'text-gray-400'}`}
                style={{ letterSpacing: '0.1em' }}>
                {group.label}
              </div>
            )}
            {sidebarCollapsed && groupIdx > 0 && (
              <div className={`mx-3 my-2 border-t ${isDark ? 'border-slate-700' : 'border-gray-100'}`} />
            )}
            {group.items.map((item) => {
              const Icon = item.icon;
              const isActive = activePage === item.id;
              return (
                <button
                  key={item.id}
                  onClick={() => setActivePage(item.id)}
                  title={sidebarCollapsed ? item.label : undefined}
                  className={`w-full flex items-center h-9 text-sm transition-all duration-150 relative ${
                    sidebarCollapsed ? 'justify-center px-0' : 'px-4'
                  } cursor-pointer ${
                    isActive
                      ? isDark
                        ? 'bg-slate-700/60 text-primary-300 font-medium'
                        : 'bg-primary-50 text-primary-600 font-medium'
                      : isDark
                        ? 'text-slate-300 hover:bg-slate-700 hover:text-slate-100'
                        : 'text-gray-600 hover:bg-gray-50 hover:text-gray-900'
                  }`}
                >
                  {isActive && !sidebarCollapsed && (
                    <span className={`absolute left-0 top-0 bottom-0 w-0.5 rounded-r ${isDark ? 'bg-primary-400' : 'bg-primary-600'}`} />
                  )}
                  <Icon className="w-4 h-4 flex-shrink-0" />
                  {!sidebarCollapsed && (
                    <span className="ml-2.5 whitespace-nowrap text-sm">{item.label}</span>
                  )}
                </button>
              );
            })}
          </div>
        ))}
      </nav>

      <div className={`flex-shrink-0 border-t ${isDark ? 'border-slate-700' : 'border-gray-200'} p-3`}>
        {sidebarCollapsed ? (
          <button onClick={handleLogout}
            className={`w-full flex items-center justify-center h-9 rounded-lg transition-colors ${isDark ? 'text-slate-400 hover:bg-red-900/30 hover:text-red-400' : 'text-gray-400 hover:bg-red-50 hover:text-red-500'}`}
            title="\u9000\u51fa\u767b\u5f55">
            <LogOut className="w-4 h-4" />
          </button>
        ) : (
          <div className="flex items-center gap-2">
            <div className="w-8 h-8 rounded-full flex items-center justify-center flex-shrink-0 text-xs font-bold text-white bg-primary-600">
              {(user?.username || 'U').slice(0, 1).toUpperCase()}
            </div>
            <div className="flex-1 min-w-0">
              <p className={`text-sm font-medium truncate ${isDark ? 'text-slate-200' : 'text-gray-700'}`}>{user?.username || 'admin'}</p>
              <span className="text-xs px-1.5 py-0.5 rounded"
                style={{ background: user?.role === 'admin' ? (isDark ? '#312e81' : '#ddd5f6') : (isDark ? '#334155' : '#f3f4f6'), color: user?.role === 'admin' ? (isDark ? '#a78bfa' : '#513CC8') : (isDark ? '#94a3b8' : '#6b7280'), fontSize: '10px' }}>
                {user?.role === 'admin' ? '\u7ba1\u7406\u5458' : '\u7528\u6237'}
              </span>
            </div>
            <button onClick={handleLogout}
              className={`p-1.5 rounded-lg transition-colors flex-shrink-0 ${isDark ? 'text-slate-400 hover:bg-red-900/30 hover:text-red-400' : 'text-gray-400 hover:bg-red-50 hover:text-red-500'}`}
              title="\u9000\u51fa\u767b\u5f55">
              <LogOut className="w-4 h-4" />
            </button>
          </div>
        )}
        {!sidebarCollapsed && (
          <p className={`text-center text-[10px] mt-2 ${isDark ? 'text-slate-600' : 'text-gray-300'}`}>v3.2.0</p>
        )}
      </div>
    </div>
  );
}
