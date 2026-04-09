import React, { useEffect, useState, useMemo } from 'react';
import {
  ExternalLink, Globe, Search, Loader2,
  Monitor, Package, ClipboardCheck, ShieldCheck,
  Database, Network, Bug, LayoutGrid, BookOpen, Shield, Mail, HardDrive,
  Download, ArrowUpCircle, BookMarked, Receipt,
  FileText, AlertTriangle, GitBranch, Image, Building,
  ScrollText, GraduationCap, RefreshCw,
  FolderOpen, Star, ChevronRight, Link2,
} from 'lucide-react';
import { getWebsiteCategories } from '../services/api';

// ── Category-level icon mapping (matches backend Icon field) ────────────────
const CATEGORY_ICON_MAP = {
  Monitor: Monitor,
  Package: Package,
  Globe: Globe,
  ClipboardCheck: ClipboardCheck,
  ShieldCheck: ShieldCheck,
};

// ── Link-level icon mapping (matches backend Icon field on each link) ───────
const LINK_ICON_MAP = {
  database: Database,
  network: Network,
  redmine: Bug,
  jira: LayoutGrid,
  confluence: BookOpen,
  vpn: Shield,
  email: Mail,
  'cloud-storage': HardDrive,
  download: Download,
  upgrade: ArrowUpCircle,
  book: BookMarked,
  receipt: Receipt,
  manual: FileText,
  template: ScrollText,
  alert: AlertTriangle,
  migrate: GitBranch,
  image: Image,
  bank: Building,
  standard: ClipboardCheck,
  training: GraduationCap,
  change: RefreshCw,
};

// ── Per-category color themes ───────────────────────────────────────────────
const CATEGORY_THEMES = [
  {
    headerBg: 'from-blue-600 to-blue-500',
    iconBg: 'bg-blue-700/30',
    cardBorder: 'border-blue-100',
    cardHover: 'hover:border-blue-300 hover:shadow-blue-100/50',
    iconCircle: 'bg-blue-50 text-blue-600',
    accent: 'text-blue-600',
    badge: 'bg-blue-100 text-blue-700',
  },
  {
    headerBg: 'from-emerald-600 to-teal-500',
    iconBg: 'bg-emerald-700/30',
    cardBorder: 'border-emerald-100',
    cardHover: 'hover:border-emerald-300 hover:shadow-emerald-100/50',
    iconCircle: 'bg-emerald-50 text-emerald-600',
    accent: 'text-emerald-600',
    badge: 'bg-emerald-100 text-emerald-700',
  },
  {
    headerBg: 'from-violet-600 to-purple-500',
    iconBg: 'bg-violet-700/30',
    cardBorder: 'border-violet-100',
    cardHover: 'hover:border-violet-300 hover:shadow-violet-100/50',
    iconCircle: 'bg-violet-50 text-violet-600',
    accent: 'text-violet-600',
    badge: 'bg-violet-100 text-violet-700',
  },
  {
    headerBg: 'from-amber-600 to-orange-500',
    iconBg: 'bg-amber-700/30',
    cardBorder: 'border-amber-100',
    cardHover: 'hover:border-amber-300 hover:shadow-amber-100/50',
    iconCircle: 'bg-amber-50 text-amber-600',
    accent: 'text-amber-600',
    badge: 'bg-amber-100 text-amber-700',
  },
  {
    headerBg: 'from-rose-600 to-pink-500',
    iconBg: 'bg-rose-700/30',
    cardBorder: 'border-rose-100',
    cardHover: 'hover:border-rose-300 hover:shadow-rose-100/50',
    iconCircle: 'bg-rose-50 text-rose-600',
    accent: 'text-rose-600',
    badge: 'bg-rose-100 text-rose-700',
  },
];

// ── Helper: extract readable hostname ───────────────────────────────────────
function getHostname(url) {
  try {
    return new URL(url).hostname;
  } catch {
    return url;
  }
}

// ── Single link card component ──────────────────────────────────────────────
function LinkCard({ link, theme }) {
  const LinkIcon = LINK_ICON_MAP[link.icon] || ExternalLink;
  return (
    <a
      href={link.url}
      target="_blank"
      rel="noopener noreferrer"
      className={`group relative flex items-start gap-3 p-4 rounded-xl border bg-white ${theme.cardBorder} ${theme.cardHover} hover:shadow-lg transition-all duration-200`}
    >
      {/* Icon */}
      <div className={`w-10 h-10 rounded-xl flex items-center justify-center flex-shrink-0 ${theme.iconCircle} group-hover:scale-110 transition-transform duration-200`}>
        <LinkIcon className="w-5 h-5" />
      </div>

      {/* Text */}
      <div className="flex-1 min-w-0 pt-0.5">
        <p className="text-sm font-semibold text-gray-800 group-hover:text-primary-600 truncate transition-colors leading-tight">
          {link.name}
        </p>
        <p className="text-xs text-gray-400 truncate mt-1 flex items-center gap-1">
          <Link2 className="w-3 h-3 flex-shrink-0" />
          {getHostname(link.url)}
        </p>
      </div>

      {/* External arrow */}
      <ExternalLink className="w-3.5 h-3.5 text-gray-300 group-hover:text-primary-500 flex-shrink-0 mt-1 opacity-0 group-hover:opacity-100 transition-all duration-200 -translate-x-1 group-hover:translate-x-0" />
    </a>
  );
}

// ── Category section component ──────────────────────────────────────────────
function CategorySection({ category, themeIdx }) {
  const theme = CATEGORY_THEMES[themeIdx % CATEGORY_THEMES.length];
  const CatIcon = CATEGORY_ICON_MAP[category.icon] || Globe;
  const links = category.links || [];

  return (
    <div className="bg-white rounded-2xl border border-gray-200 shadow-sm overflow-hidden">
      {/* Category header with gradient */}
      <div className={`bg-gradient-to-r ${theme.headerBg} px-6 py-4`}>
        <div className="flex items-center gap-3">
          <div className={`w-10 h-10 rounded-xl flex items-center justify-center ${theme.iconBg} backdrop-blur-sm`}>
            <CatIcon className="w-5 h-5 text-white" />
          </div>
          <div className="flex-1 min-w-0">
            <h2 className="text-base font-bold text-white">{category.name}</h2>
            <p className="text-xs text-white/70 mt-0.5">{links.length} 个系统</p>
          </div>
          <span className="text-xs font-medium text-white/80 bg-white/15 px-2.5 py-1 rounded-full backdrop-blur-sm">
            {category.name}
          </span>
        </div>
      </div>

      {/* Link grid */}
      <div className="p-5">
        {links.length > 0 ? (
          <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4 gap-3">
            {links.map((link) => (
              <LinkCard key={link.id} link={link} theme={theme} />
            ))}
          </div>
        ) : (
          <div className="text-center py-8 text-gray-300">
            <FolderOpen className="w-8 h-8 mx-auto mb-2" />
            <p className="text-sm">暂无系统链接</p>
          </div>
        )}
      </div>
    </div>
  );
}

// ── Main page component ─────────────────────────────────────────────────────
export default function WebsitesPage() {
  const [categories, setCategories] = useState([]);
  const [loading, setLoading] = useState(true);
  const [search, setSearch] = useState('');

  useEffect(() => {
    loadCategories();
  }, []);

  const loadCategories = async () => {
    try {
      const res = await getWebsiteCategories();
      if (res.code === 0) {
        setCategories(res.data || []);
      }
    } catch (err) {
      console.error('Failed to load categories:', err);
    } finally {
      setLoading(false);
    }
  };

  // Filtered view: search across link names and URLs
  const filteredCategories = useMemo(() => {
    if (!search.trim()) return categories;
    const q = search.toLowerCase();
    return categories
      .map((cat) => ({
        ...cat,
        links: (cat.links || []).filter(
          (link) =>
            link.name.toLowerCase().includes(q) ||
            link.url.toLowerCase().includes(q)
        ),
      }))
      .filter((cat) => cat.links.length > 0);
  }, [categories, search]);

  const totalLinks = useMemo(
    () => categories.reduce((sum, cat) => sum + (cat.links?.length || 0), 0),
    [categories]
  );

  // ── Loading state ──
  if (loading) {
    return (
      <div className="h-full flex items-center justify-center">
        <div className="text-center">
          <Loader2 className="w-8 h-8 animate-spin text-primary-600 mx-auto" />
          <p className="text-sm text-gray-400 mt-3">加载公司系统列表...</p>
        </div>
      </div>
    );
  }

  // ── Main render ──
  return (
    <div className="h-full overflow-y-auto">
      <div className="p-4 sm:p-6 space-y-5 w-full max-w-[1600px] mx-auto">
        {/* ── Top bar: stats + search ── */}
        <div className="bg-white rounded-2xl border border-gray-200 shadow-sm px-6 py-4">
          <div className="flex flex-col sm:flex-row items-start sm:items-center gap-3 sm:gap-4">
            {/* Stats */}
            <div className="flex items-center gap-4 text-sm flex-shrink-0 order-2 sm:order-1">
              <div className="flex items-center gap-1.5">
                <div className="w-6 h-6 rounded-lg bg-primary-50 flex items-center justify-center">
                  <Star className="w-3.5 h-3.5 text-primary-600" />
                </div>
                <span className="text-gray-500">
                  <strong className="text-gray-800">{categories.length}</strong> 个分类
                </span>
              </div>
              <div className="w-px h-4 bg-gray-200" />
              <div className="flex items-center gap-1.5">
                <div className="w-6 h-6 rounded-lg bg-primary-50 flex items-center justify-center">
                  <Globe className="w-3.5 h-3.5 text-primary-600" />
                </div>
                <span className="text-gray-500">
                  <strong className="text-gray-800">{totalLinks}</strong> 个系统
                </span>
              </div>
            </div>

            {/* Search */}
            <div className="flex-1 relative w-full sm:w-auto order-1 sm:order-2">
              <Search className="w-4 h-4 text-gray-400 absolute left-3 top-1/2 -translate-y-1/2 pointer-events-none" />
              <input
                type="text"
                value={search}
                onChange={(e) => setSearch(e.target.value)}
                placeholder="搜索系统名称、链接地址..."
                className="w-full pl-10 pr-4 py-2.5 border border-gray-200 rounded-xl text-sm focus:ring-2 focus:ring-primary-500/20 focus:border-primary-500 outline-none transition-all bg-gray-50 focus:bg-white"
              />
              {search && (
                <button
                  onClick={() => setSearch('')}
                  className="absolute right-3 top-1/2 -translate-y-1/2 text-gray-400 hover:text-gray-600 text-xs"
                >
                  清除
                </button>
              )}
            </div>
          </div>
        </div>

        {/* ── Category sections ── */}
        <div className="space-y-5">
          {filteredCategories.map((cat, idx) => (
            <CategorySection key={cat.id} category={cat} themeIdx={idx} />
          ))}
        </div>

        {/* ── Empty search result ── */}
        {filteredCategories.length === 0 && search && (
          <div className="text-center py-20 bg-white rounded-2xl border border-gray-200 shadow-sm">
            <Search className="w-12 h-12 text-gray-200 mx-auto mb-4" />
            <p className="text-gray-400 text-sm">
              未找到与 "<span className="text-gray-600 font-medium">{search}</span>" 匹配的系统
            </p>
            <button
              onClick={() => setSearch('')}
              className="mt-3 text-sm text-primary-600 hover:text-primary-700 font-medium transition-colors"
            >
              清除搜索
            </button>
          </div>
        )}

        {/* ── Empty state (no data at all) ── */}
        {categories.length === 0 && !loading && (
          <div className="text-center py-20 bg-white rounded-2xl border border-gray-200 shadow-sm">
            <Globe className="w-16 h-16 text-gray-200 mx-auto mb-4" />
            <h3 className="text-lg font-semibold text-gray-500 mb-1">暂无系统数据</h3>
            <p className="text-sm text-gray-400">请联系管理员添加公司系统链接</p>
          </div>
        )}
      </div>
    </div>
  );
}
