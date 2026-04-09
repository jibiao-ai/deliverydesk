import React from 'react';
import {
  LayoutDashboard, MessageSquare, Bot, Zap, Users, LogOut, Menu, Cpu, Globe, Shield, FileText, Server,
} from 'lucide-react';
import useStore from '../store/useStore';
import { useNavigate } from 'react-router-dom';

const menuGroups = [
  {
    label: '交付工作台',
    items: [
      { id: 'dashboard', label: '仪表盘', icon: LayoutDashboard },
      { id: 'websites', label: '公司系统', icon: Globe },
      { id: 'chat', label: '即时对话', icon: MessageSquare },
      { id: 'agents', label: '智能体', icon: Bot },
    ],
  },
  {
    label: '配置管理',
    items: [
      { id: 'ai-models', label: '模型配置', icon: Cpu },
      { id: 'skills', label: '技能商店', icon: Zap },
    ],
  },
];

const adminGroup = {
  label: '系统管理',
  items: [
    { id: 'ldap', label: 'LDAP管理', icon: Server },
    { id: 'users', label: '用户管理', icon: Users },
    { id: 'operation-logs', label: '操作日志', icon: FileText },
  ],
};

export default function Sidebar() {
  const { activePage, setActivePage, user, logout, sidebarCollapsed, toggleSidebar } = useStore();
  const navigate = useNavigate();

  const handleLogout = () => {
    logout();
    navigate('/login');
  };

  const allGroups = user?.role === 'admin' ? [...menuGroups, adminGroup] : menuGroups;

  return (
    <div
      className={`flex flex-col transition-all duration-300 flex-shrink-0 border-r border-gray-200 ${
        sidebarCollapsed ? 'w-16' : 'w-56'
      }`}
      style={{ background: '#ffffff' }}
    >
      <div className="flex items-center h-16 px-3 flex-shrink-0 border-b border-gray-200">
        <button
          onClick={toggleSidebar}
          className="flex items-center justify-center w-8 h-8 rounded-lg transition-colors flex-shrink-0 text-gray-500 hover:text-primary-600 hover:bg-primary-50"
        >
          <Menu className="w-5 h-5" />
        </button>
        {!sidebarCollapsed && (
          <div className="ml-2 flex items-center gap-2 overflow-hidden">
            <div className="w-7 h-7 rounded-lg bg-primary-600 flex items-center justify-center flex-shrink-0">
              <Shield className="w-4 h-4 text-white" />
            </div>
            <span className="text-sm font-semibold whitespace-nowrap text-gray-800">
              <span className="text-gray-800">Delivery</span><span style={{ color: '#513CC8' }}>Desk</span>
              <span className="text-xs ml-1 px-1 py-0.5 rounded" style={{ background: '#ddd5f6', color: '#513CC8', fontSize: '10px' }}>AI</span>
            </span>
          </div>
        )}
      </div>

      <nav className="flex-1 overflow-y-auto py-2" style={{ scrollbarWidth: 'none' }}>
        {allGroups.map((group, groupIdx) => (
          <div key={groupIdx} className="mb-1">
            {!sidebarCollapsed && (
              <div className="px-4 pt-4 pb-1 text-xs uppercase tracking-widest font-medium text-gray-400"
                style={{ letterSpacing: '0.1em' }}>
                {group.label}
              </div>
            )}
            {sidebarCollapsed && groupIdx > 0 && (
              <div className="mx-3 my-2 border-t border-gray-100" />
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
                      ? 'bg-primary-50 text-primary-600 font-medium'
                      : 'text-gray-600 hover:bg-gray-50 hover:text-gray-900'
                  }`}
                >
                  {isActive && !sidebarCollapsed && (
                    <span className="absolute left-0 top-0 bottom-0 w-0.5 rounded-r bg-primary-600" />
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

      <div className="flex-shrink-0 border-t border-gray-200 p-3">
        {sidebarCollapsed ? (
          <button onClick={handleLogout}
            className="w-full flex items-center justify-center h-9 rounded-lg transition-colors text-gray-400 hover:bg-red-50 hover:text-red-500"
            title="退出登录">
            <LogOut className="w-4 h-4" />
          </button>
        ) : (
          <div className="flex items-center gap-2">
            <div className="w-8 h-8 rounded-full flex items-center justify-center flex-shrink-0 text-xs font-bold text-white bg-primary-600">
              {(user?.username || 'U').slice(0, 1).toUpperCase()}
            </div>
            <div className="flex-1 min-w-0">
              <p className="text-sm font-medium truncate text-gray-700">{user?.username || 'admin'}</p>
              <span className="text-xs px-1.5 py-0.5 rounded"
                style={{ background: user?.role === 'admin' ? '#ddd5f6' : '#f3f4f6', color: user?.role === 'admin' ? '#513CC8' : '#6b7280', fontSize: '10px' }}>
                {user?.role === 'admin' ? '管理员' : '用户'}
              </span>
            </div>
            <button onClick={handleLogout}
              className="p-1.5 rounded-lg transition-colors flex-shrink-0 text-gray-400 hover:bg-red-50 hover:text-red-500"
              title="退出登录">
              <LogOut className="w-4 h-4" />
            </button>
          </div>
        )}
      </div>
    </div>
  );
}
