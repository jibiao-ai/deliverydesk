import React, { useState } from 'react';
import { useNavigate } from 'react-router-dom';
import { login } from '../services/api';
import useStore from '../store/useStore';
import toast from 'react-hot-toast';
import { Bot, Cloud, Shield, Cpu, Zap, Lock, User, Eye, EyeOff, Server } from 'lucide-react';

export default function LoginPage() {
  const [username, setUsername] = useState('admin');
  const [password, setPassword] = useState('');
  const [showPassword, setShowPassword] = useState(false);
  const [loading, setLoading] = useState(false);
  const [authType, setAuthType] = useState('local');
  const setAuth = useStore((s) => s.setAuth);
  const navigate = useNavigate();

  const handleLogin = async (e) => {
    e.preventDefault();
    setLoading(true);
    try {
      const res = await login(username, password, authType);
      if (res.code === 0) {
        setAuth(res.data.user, res.data.token);
        toast.success('登录成功');
        navigate('/', { replace: true });
      } else {
        toast.error(res.message || '登录失败');
      }
    } catch (err) {
      // err may be the backend response object { code, message } or a network error
      const msg = err?.message || err?.response?.data?.message || '网络错误，请检查后端服务是否运行';
      toast.error(msg);
    } finally {
      setLoading(false);
    }
  };

  const features = [
    { icon: Cloud,  label: '交付资源导航', desc: '常用工具、文档、系统一站式快速访问' },
    { icon: Cpu,    label: 'AI 智能助手',   desc: '集成大模型 API，智能解答交付问题' },
    { icon: Zap,    label: '技能赋能',      desc: '交付黄页、技能库，提升团队能力' },
  ];

  return (
    <div className="min-h-screen flex">
      <div className="hidden md:flex md:w-2/5 flex-col justify-between p-10 relative overflow-hidden" style={{ background: '#513CC8' }}>
        <div className="absolute -top-24 -right-24 w-72 h-72 rounded-full opacity-15" style={{ background: '#ffffff' }} />
        <div className="absolute -bottom-16 -left-16 w-56 h-56 rounded-full opacity-15" style={{ background: '#ffffff' }} />
        <div className="relative z-10">
          <div className="flex items-center gap-3 mb-8">
            <div className="w-12 h-12 rounded-2xl bg-white/15 flex items-center justify-center">
              <Shield className="w-7 h-7 text-white" />
            </div>
            <div>
              <h1 className="text-xl font-bold text-white">DeliveryDesk <span className="text-sm font-normal px-1.5 py-0.5 rounded" style={{ background: 'rgba(255,255,255,0.2)' }}>AI</span></h1>
              <p className="text-xs" style={{ color: 'rgba(255,255,255,0.65)' }}>Cloud Delivery Workbench</p>
            </div>
          </div>
          <div className="mb-2">
            <h2 className="text-3xl font-bold text-white mb-3 leading-snug">云交付服务台<br />智能工作平台</h2>
            <p className="text-sm leading-relaxed" style={{ color: 'rgba(255,255,255,0.85)' }}>
              集 AI 智能体、交付资源导航、LDAP 企业认证于一体，让交付更简单、更高效。
            </p>
          </div>
        </div>
        <div className="relative z-10 space-y-4">
          {features.map((f, i) => {
            const Icon = f.icon;
            return (
              <div key={i} className="flex items-start gap-4">
                <div className="w-9 h-9 rounded-xl flex items-center justify-center flex-shrink-0" style={{ background: 'rgba(255,255,255,0.15)' }}>
                  <Icon className="w-4 h-4 text-white" />
                </div>
                <div>
                  <p className="text-sm font-medium text-white">{f.label}</p>
                  <p className="text-xs mt-0.5" style={{ color: 'rgba(255,255,255,0.65)' }}>{f.desc}</p>
                </div>
              </div>
            );
          })}
        </div>
        <p className="relative z-10 text-xs" style={{ color: 'rgba(255,255,255,0.5)' }}>© 2024-2026 DeliveryDesk. All rights reserved.</p>
      </div>

      <div className="flex-1 flex flex-col items-center justify-center bg-white px-8 py-12">
        <div className="w-full max-w-sm">
          <div className="mb-8">
            <h2 className="text-2xl font-bold text-gray-800 mb-1">欢迎回来</h2>
            <p className="text-sm text-gray-400">请输入账号和密码登录平台</p>
          </div>

          {/* Auth Type Toggle */}
          <div className="flex gap-2 mb-5 bg-gray-100 rounded-xl p-1">
            <button onClick={() => setAuthType('local')}
              className={`flex-1 flex items-center justify-center gap-1.5 py-2 rounded-lg text-sm font-medium transition-all ${
                authType === 'local' ? 'bg-white text-primary-600 shadow-sm' : 'text-gray-500'
              }`}>
              <User className="w-3.5 h-3.5" /> 本地登录
            </button>
            <button onClick={() => setAuthType('ldap')}
              className={`flex-1 flex items-center justify-center gap-1.5 py-2 rounded-lg text-sm font-medium transition-all ${
                authType === 'ldap' ? 'bg-white text-primary-600 shadow-sm' : 'text-gray-500'
              }`}>
              <Server className="w-3.5 h-3.5" /> LDAP登录
            </button>
          </div>

          <form onSubmit={handleLogin} className="space-y-5">
            <div>
              <label className="block text-sm font-medium text-gray-700 mb-1.5">用户名</label>
              <div className="relative">
                <User className="w-4 h-4 text-gray-400 absolute left-3 top-1/2 -translate-y-1/2" />
                <input type="text" value={username} onChange={(e) => setUsername(e.target.value)}
                  className="w-full pl-10 pr-4 py-2.5 border border-gray-200 rounded-xl text-sm focus:outline-none transition"
                  onFocus={e => { e.target.style.borderColor = '#513CC8'; e.target.style.boxShadow = '0 0 0 3px rgba(81,60,200,0.1)'; }}
                  onBlur={e => { e.target.style.borderColor = '#e5e7eb'; e.target.style.boxShadow = 'none'; }}
                  placeholder="请输入用户名" required />
              </div>
            </div>
            <div>
              <label className="block text-sm font-medium text-gray-700 mb-1.5">密码</label>
              <div className="relative">
                <Lock className="w-4 h-4 text-gray-400 absolute left-3 top-1/2 -translate-y-1/2" />
                <input type={showPassword ? 'text' : 'password'} value={password} onChange={(e) => setPassword(e.target.value)}
                  className="w-full pl-10 pr-10 py-2.5 border border-gray-200 rounded-xl text-sm focus:outline-none transition"
                  onFocus={e => { e.target.style.borderColor = '#513CC8'; e.target.style.boxShadow = '0 0 0 3px rgba(81,60,200,0.1)'; }}
                  onBlur={e => { e.target.style.borderColor = '#e5e7eb'; e.target.style.boxShadow = 'none'; }}
                  placeholder="请输入密码" required />
                <button type="button" onClick={() => setShowPassword(!showPassword)}
                  className="absolute right-3 top-1/2 -translate-y-1/2 text-gray-400 hover:text-gray-600" tabIndex={-1}>
                  {showPassword ? <EyeOff className="w-4 h-4" /> : <Eye className="w-4 h-4" />}
                </button>
              </div>
            </div>
            <button type="submit" disabled={loading}
              className="w-full py-2.5 rounded-xl text-white font-medium text-sm transition-all disabled:opacity-50"
              style={{ background: loading ? '#7757db' : '#513CC8' }}
              onMouseEnter={e => { if (!loading) e.currentTarget.style.background = '#4230a0'; }}
              onMouseLeave={e => { if (!loading) e.currentTarget.style.background = '#513CC8'; }}>
              {loading ? (
                <span className="flex items-center justify-center gap-2">
                  <span className="w-4 h-4 border-2 border-white/30 border-t-white rounded-full animate-spin" />
                  登录中...
                </span>
              ) : '登 录'}
            </button>
          </form>

          <div className="mt-8 pt-6 border-t border-gray-100">
            <div className="flex items-center justify-center gap-6 text-xs text-gray-400">
              <div className="flex items-center gap-1.5"><Shield className="w-3.5 h-3.5" /><span>安全可信</span></div>
              <div className="flex items-center gap-1.5"><Server className="w-3.5 h-3.5" /><span>LDAP 认证</span></div>
              <div className="flex items-center gap-1.5"><Bot className="w-3.5 h-3.5" /><span>AI 驱动</span></div>
            </div>
            <p className="text-center text-xs text-gray-300 mt-4">Powered by DeliveryDesk AI v3.2.0</p>
          </div>
        </div>
      </div>
    </div>
  );
}
