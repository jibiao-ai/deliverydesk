import React, { useEffect, useState } from 'react';
import { ExternalLink, Globe, Wrench, Network, Cloud, BookOpen, Settings, Target, HardDrive, Search, Loader2 } from 'lucide-react';
import { getWebsiteCategories } from '../services/api';

const ICON_MAP = {
  Wrench: Wrench,
  Network: Network,
  Globe: Globe,
  Cloud: Cloud,
  BookOpen: BookOpen,
  Settings: Settings,
  Target: Target,
  HardDrive: HardDrive,
};

const CATEGORY_COLORS = [
  { bg: 'bg-primary-50', border: 'border-primary-200', iconBg: 'bg-primary-100', iconColor: 'text-primary-600', hoverBorder: 'hover:border-primary-400' },
  { bg: 'bg-emerald-50', border: 'border-emerald-200', iconBg: 'bg-emerald-100', iconColor: 'text-emerald-600', hoverBorder: 'hover:border-emerald-400' },
  { bg: 'bg-purple-50', border: 'border-purple-200', iconBg: 'bg-purple-100', iconColor: 'text-purple-600', hoverBorder: 'hover:border-purple-400' },
  { bg: 'bg-orange-50', border: 'border-orange-200', iconBg: 'bg-orange-100', iconColor: 'text-orange-600', hoverBorder: 'hover:border-orange-400' },
  { bg: 'bg-pink-50', border: 'border-pink-200', iconBg: 'bg-pink-100', iconColor: 'text-pink-600', hoverBorder: 'hover:border-pink-400' },
  { bg: 'bg-cyan-50', border: 'border-cyan-200', iconBg: 'bg-cyan-100', iconColor: 'text-cyan-600', hoverBorder: 'hover:border-cyan-400' },
  { bg: 'bg-amber-50', border: 'border-amber-200', iconBg: 'bg-amber-100', iconColor: 'text-amber-600', hoverBorder: 'hover:border-amber-400' },
  { bg: 'bg-indigo-50', border: 'border-indigo-200', iconBg: 'bg-indigo-100', iconColor: 'text-indigo-600', hoverBorder: 'hover:border-indigo-400' },
];

export default function WebsitesPage() {
  const [categories, setCategories] = useState([]);
  const [loading, setLoading] = useState(true);
  const [search, setSearch] = useState('');

  useEffect(() => { loadCategories(); }, []);

  const loadCategories = async () => {
    try {
      const res = await getWebsiteCategories();
      if (res.code === 0) setCategories(res.data || []);
    } catch (err) { console.error(err); }
    finally { setLoading(false); }
  };

  const filteredCategories = categories.map(cat => ({
    ...cat,
    links: (cat.links || []).filter(link =>
      link.name.toLowerCase().includes(search.toLowerCase()) ||
      link.url.toLowerCase().includes(search.toLowerCase())
    ),
  })).filter(cat => cat.links.length > 0 || !search);

  const totalLinks = categories.reduce((sum, cat) => sum + (cat.links?.length || 0), 0);

  if (loading) {
    return (
      <div className="h-full flex items-center justify-center">
        <Loader2 className="w-6 h-6 animate-spin text-primary-600" />
      </div>
    );
  }

  return (
    <div className="h-full overflow-y-auto">
      <div className="p-4 sm:p-6 space-y-4 sm:space-y-6 w-full">
        {/* Search & Stats */}
        <div className="bg-white rounded-xl border border-gray-200 shadow-sm px-6 py-4">
          <div className="flex items-center gap-4">
            <div className="flex-1 relative">
              <Search className="w-4 h-4 text-gray-400 absolute left-3 top-1/2 -translate-y-1/2" />
              <input
                type="text"
                value={search}
                onChange={(e) => setSearch(e.target.value)}
                placeholder="搜索工具名称或链接..."
                className="w-full pl-10 pr-4 py-2 border border-gray-200 rounded-lg text-sm focus:ring-2 focus:ring-primary-500 focus:border-primary-500 outline-none"
              />
            </div>
            <div className="flex items-center gap-4 text-sm text-gray-500 flex-shrink-0">
              <span>共 <strong className="text-gray-700">{categories.length}</strong> 个分类</span>
              <span>共 <strong className="text-gray-700">{totalLinks}</strong> 个链接</span>
            </div>
          </div>
        </div>

        {/* Category Grid */}
        <div className="space-y-6">
          {filteredCategories.map((cat, catIdx) => {
            const color = CATEGORY_COLORS[catIdx % CATEGORY_COLORS.length];
            const IconComp = ICON_MAP[cat.icon] || Globe;

            return (
              <div key={cat.id} className="bg-white rounded-xl border border-gray-200 shadow-sm overflow-hidden">
                <div className={`px-6 py-4 border-b border-gray-100 flex items-center gap-3 ${color.bg}`}>
                  <div className={`w-9 h-9 rounded-lg flex items-center justify-center ${color.iconBg}`}>
                    <IconComp className={`w-5 h-5 ${color.iconColor}`} />
                  </div>
                  <div>
                    <h2 className="text-base font-semibold text-gray-800">{cat.name}</h2>
                    <p className="text-xs text-gray-400">{cat.links?.length || 0} 个链接</p>
                  </div>
                </div>
                <div className="p-4">
                  <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4 gap-3">
                    {(cat.links || []).map((link) => (
                      <a
                        key={link.id}
                        href={link.url}
                        target="_blank"
                        rel="noopener noreferrer"
                        className={`group flex items-center gap-3 px-4 py-3 rounded-xl border ${color.border} ${color.hoverBorder} hover:shadow-md transition-all`}
                      >
                        <div className={`w-8 h-8 rounded-lg flex items-center justify-center flex-shrink-0 ${color.iconBg} group-hover:scale-110 transition-transform`}>
                          <ExternalLink className={`w-4 h-4 ${color.iconColor}`} />
                        </div>
                        <div className="flex-1 min-w-0">
                          <p className="text-sm font-medium text-gray-700 group-hover:text-primary-600 truncate transition-colors">
                            {link.name}
                          </p>
                          <p className="text-xs text-gray-400 truncate">{new URL(link.url).hostname}</p>
                        </div>
                      </a>
                    ))}
                  </div>
                </div>
              </div>
            );
          })}
        </div>

        {filteredCategories.length === 0 && search && (
          <div className="text-center py-16">
            <Globe className="w-12 h-12 text-gray-200 mx-auto mb-3" />
            <p className="text-gray-400">未找到匹配 "{search}" 的链接</p>
          </div>
        )}
      </div>
    </div>
  );
}
