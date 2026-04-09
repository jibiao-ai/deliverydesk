import React, { useState, useEffect, useCallback } from 'react';
import { Zap, Plus, Trash2, Edit3, Upload, FileText, RefreshCw, Loader2, Database, Globe2, Search, ChevronDown, ChevronUp, X } from 'lucide-react';
import { getSkills, createSkill, updateSkill, deleteSkill, uploadSkillDocument, reindexSkill } from '../services/api';
import toast from 'react-hot-toast';

const TYPE_LABELS = {
  delivery: { label: '交付技能', color: 'bg-blue-100 text-blue-700' },
  community: { label: '社区技能', color: 'bg-green-100 text-green-700' },
  knowledge: { label: '知识技能', color: 'bg-purple-100 text-purple-700' },
  ops: { label: '运维技能', color: 'bg-orange-100 text-orange-700' },
};

const STATUS_LABELS = {
  pending: { label: '等待处理', color: 'bg-gray-100 text-gray-600' },
  processing: { label: '处理中', color: 'bg-yellow-100 text-yellow-700' },
  ready: { label: '就绪', color: 'bg-green-100 text-green-700' },
  error: { label: '错误', color: 'bg-red-100 text-red-700' },
};

export default function SkillsPage() {
  const [skills, setSkills] = useState([]);
  const [loading, setLoading] = useState(true);
  const [showModal, setShowModal] = useState(false);
  const [editingSkill, setEditingSkill] = useState(null);
  const [expandedSkill, setExpandedSkill] = useState(null);
  const [uploading, setUploading] = useState({});
  const [reindexing, setReindexing] = useState({});
  const [form, setForm] = useState({ name: '', description: '', type: 'delivery', category: '' });

  const loadSkills = useCallback(async () => {
    setLoading(true);
    try {
      const res = await getSkills();
      if (res.code === 0) setSkills(res.data || []);
    } catch (e) { toast.error('加载技能失败'); }
    finally { setLoading(false); }
  }, []);

  useEffect(() => { loadSkills(); }, [loadSkills]);

  const handleCreate = () => {
    setEditingSkill(null);
    setForm({ name: '', description: '', type: 'delivery', category: '' });
    setShowModal(true);
  };

  const handleEdit = (sk) => {
    setEditingSkill(sk);
    setForm({ name: sk.name, description: sk.description, type: sk.type, category: sk.category || '' });
    setShowModal(true);
  };

  const handleSave = async () => {
    if (!form.name.trim()) { toast.error('请输入技能名称'); return; }
    try {
      if (editingSkill) {
        const res = await updateSkill(editingSkill.id, form);
        if (res.code === 0) { toast.success('更新成功'); loadSkills(); setShowModal(false); }
        else toast.error(res.message || '更新失败');
      } else {
        const res = await createSkill(form);
        if (res.code === 0) { toast.success('创建成功'); loadSkills(); setShowModal(false); }
        else toast.error(res.message || '创建失败');
      }
    } catch (e) { toast.error('操作失败'); }
  };

  const handleDelete = async (sk) => {
    if (!confirm(`确定删除技能「${sk.name}」？`)) return;
    try {
      const res = await deleteSkill(sk.id);
      if (res.code === 0) { toast.success('删除成功'); loadSkills(); }
      else toast.error(res.message || '删除失败');
    } catch (e) { toast.error('删除失败'); }
  };

  const handleUpload = async (skillId, e) => {
    const file = e.target.files[0];
    if (!file) return;
    const ext = file.name.split('.').pop().toLowerCase();
    if (!['docx', 'xlsx', 'txt', 'md'].includes(ext)) {
      toast.error('支持的文件格式: .docx, .xlsx, .txt, .md');
      return;
    }
    setUploading(prev => ({ ...prev, [skillId]: true }));
    try {
      const res = await uploadSkillDocument(skillId, file);
      if (res.code === 0) { toast.success(`文档「${file.name}」上传成功，正在索引...`); loadSkills(); }
      else toast.error(res.message || '上传失败');
    } catch (e) { toast.error('上传失败'); }
    finally { setUploading(prev => ({ ...prev, [skillId]: false })); e.target.value = ''; }
  };

  const handleReindex = async (skillId) => {
    setReindexing(prev => ({ ...prev, [skillId]: true }));
    try {
      const res = await reindexSkill(skillId);
      if (res.code === 0) { toast.success('重新索引完成'); loadSkills(); }
      else toast.error(res.message || '索引失败');
    } catch (e) { toast.error('索引失败'); }
    finally { setReindexing(prev => ({ ...prev, [skillId]: false })); }
  };

  return (
    <div className="h-full overflow-y-auto">
      <div className="p-4 sm:p-6 space-y-4">
        {/* Stats */}
        <div className="grid grid-cols-2 sm:grid-cols-4 gap-3">
          <div className="bg-white rounded-xl border border-gray-200 p-4">
            <p className="text-xs text-gray-500">总技能数</p>
            <p className="text-2xl font-bold text-gray-800">{skills.length}</p>
          </div>
          <div className="bg-white rounded-xl border border-gray-200 p-4">
            <p className="text-xs text-gray-500">交付技能</p>
            <p className="text-2xl font-bold text-blue-600">{skills.filter(s => s.type === 'delivery').length}</p>
          </div>
          <div className="bg-white rounded-xl border border-gray-200 p-4">
            <p className="text-xs text-gray-500">社区技能</p>
            <p className="text-2xl font-bold text-green-600">{skills.filter(s => s.type === 'community').length}</p>
          </div>
          <div className="bg-white rounded-xl border border-gray-200 p-4">
            <p className="text-xs text-gray-500">总文档块</p>
            <p className="text-2xl font-bold text-purple-600">{skills.reduce((acc, s) => acc + (s.chunk_count || 0), 0)}</p>
          </div>
        </div>

        {/* Header */}
        <div className="flex items-center justify-between">
          <h2 className="text-lg font-semibold text-gray-800">技能列表</h2>
          <button onClick={handleCreate}
            className="flex items-center gap-2 px-4 py-2 bg-primary-600 text-white rounded-lg text-sm hover:bg-primary-700">
            <Plus className="w-4 h-4" /> 新建技能
          </button>
        </div>

        {/* Skill List */}
        {loading ? (
          <div className="flex justify-center py-12"><Loader2 className="w-6 h-6 animate-spin text-primary-600" /></div>
        ) : skills.length === 0 ? (
          <div className="bg-white rounded-xl border border-gray-200 p-8 text-center text-gray-400">
            暂无技能，点击"新建技能"创建
          </div>
        ) : (
          <div className="space-y-3">
            {skills.map(sk => {
              const typeInfo = TYPE_LABELS[sk.type] || { label: sk.type, color: 'bg-gray-100 text-gray-600' };
              const isExpanded = expandedSkill === sk.id;
              return (
                <div key={sk.id} className="bg-white rounded-xl border border-gray-200 shadow-sm overflow-hidden">
                  <div className="p-4 flex items-start gap-3">
                    <div className={`w-10 h-10 rounded-lg flex items-center justify-center flex-shrink-0 ${
                      sk.type === 'delivery' ? 'bg-blue-50' : sk.type === 'community' ? 'bg-green-50' : 'bg-purple-50'
                    }`}>
                      {sk.type === 'community' ? <Globe2 className="w-5 h-5 text-green-600" /> :
                       sk.type === 'delivery' ? <Zap className="w-5 h-5 text-blue-600" /> :
                       <Database className="w-5 h-5 text-purple-600" />}
                    </div>
                    <div className="flex-1 min-w-0">
                      <div className="flex items-center gap-2 mb-1">
                        <h3 className="text-sm font-semibold text-gray-800 truncate">{sk.name}</h3>
                        <span className={`px-2 py-0.5 rounded-full text-[10px] font-medium ${typeInfo.color}`}>
                          {typeInfo.label}
                        </span>
                        {sk.is_active ? (
                          <span className="px-2 py-0.5 rounded-full text-[10px] font-medium bg-green-100 text-green-700">启用</span>
                        ) : (
                          <span className="px-2 py-0.5 rounded-full text-[10px] font-medium bg-gray-100 text-gray-500">禁用</span>
                        )}
                      </div>
                      <p className="text-xs text-gray-500 line-clamp-2">{sk.description}</p>
                      <div className="flex items-center gap-4 mt-2 text-xs text-gray-400">
                        <span><FileText className="w-3 h-3 inline mr-1" />{sk.doc_count || 0} 文档</span>
                        <span><Database className="w-3 h-3 inline mr-1" />{sk.chunk_count || 0} 文本块</span>
                        {sk.category && <span className="text-gray-300">分类: {sk.category}</span>}
                      </div>
                    </div>
                    <div className="flex items-center gap-1 flex-shrink-0">
                      <label className={`flex items-center gap-1 px-2 py-1.5 rounded-lg text-xs cursor-pointer ${
                        uploading[sk.id] ? 'bg-gray-100 text-gray-400' : 'bg-blue-50 text-blue-600 hover:bg-blue-100'
                      }`}>
                        {uploading[sk.id] ? <Loader2 className="w-3 h-3 animate-spin" /> : <Upload className="w-3 h-3" />}
                        上传文档
                        <input type="file" accept=".docx,.xlsx,.txt,.md" className="hidden"
                          disabled={uploading[sk.id]}
                          onChange={(e) => handleUpload(sk.id, e)} />
                      </label>
                      <button onClick={() => handleReindex(sk.id)} disabled={reindexing[sk.id]}
                        className="p-1.5 rounded-lg text-gray-400 hover:bg-gray-100 hover:text-gray-600" title="重新索引">
                        <RefreshCw className={`w-3.5 h-3.5 ${reindexing[sk.id] ? 'animate-spin' : ''}`} />
                      </button>
                      <button onClick={() => handleEdit(sk)}
                        className="p-1.5 rounded-lg text-gray-400 hover:bg-gray-100 hover:text-gray-600" title="编辑">
                        <Edit3 className="w-3.5 h-3.5" />
                      </button>
                      <button onClick={() => handleDelete(sk)}
                        className="p-1.5 rounded-lg text-gray-400 hover:bg-red-50 hover:text-red-500" title="删除">
                        <Trash2 className="w-3.5 h-3.5" />
                      </button>
                      <button onClick={() => setExpandedSkill(isExpanded ? null : sk.id)}
                        className="p-1.5 rounded-lg text-gray-400 hover:bg-gray-100 hover:text-gray-600" title="展开详情">
                        {isExpanded ? <ChevronUp className="w-3.5 h-3.5" /> : <ChevronDown className="w-3.5 h-3.5" />}
                      </button>
                    </div>
                  </div>
                  {isExpanded && (
                    <div className="border-t border-gray-100 bg-gray-50 p-4">
                      <h4 className="text-xs font-medium text-gray-600 mb-2">知识库文档</h4>
                      {sk.documents && sk.documents.length > 0 ? (
                        <div className="space-y-2">
                          {sk.documents.map(doc => {
                            const statusInfo = STATUS_LABELS[doc.status] || { label: doc.status, color: 'bg-gray-100 text-gray-600' };
                            return (
                              <div key={doc.id} className="flex items-center gap-3 bg-white rounded-lg px-3 py-2 border border-gray-100">
                                <FileText className="w-4 h-4 text-gray-400 flex-shrink-0" />
                                <span className="text-sm text-gray-700 flex-1 truncate">{doc.file_name}</span>
                                <span className={`px-2 py-0.5 rounded-full text-[10px] ${statusInfo.color}`}>{statusInfo.label}</span>
                                <span className="text-xs text-gray-400">{doc.chunks || 0} 块</span>
                                <span className="text-xs text-gray-300">{(doc.file_size / 1024).toFixed(1)} KB</span>
                              </div>
                            );
                          })}
                        </div>
                      ) : (
                        <p className="text-xs text-gray-400">暂无文档，请上传 .docx / .xlsx / .txt / .md 文件</p>
                      )}
                    </div>
                  )}
                </div>
              );
            })}
          </div>
        )}
      </div>

      {/* Create/Edit Modal */}
      {showModal && (
        <div className="fixed inset-0 bg-black/50 flex items-center justify-center z-50">
          <div className="bg-white rounded-2xl shadow-2xl w-full max-w-lg mx-4">
            <div className="flex items-center justify-between px-6 py-4 border-b border-gray-100">
              <h3 className="text-base font-semibold text-gray-800">{editingSkill ? '编辑技能' : '新建技能'}</h3>
              <button onClick={() => setShowModal(false)} className="p-1 rounded-lg hover:bg-gray-100"><X className="w-4 h-4" /></button>
            </div>
            <div className="p-6 space-y-4">
              <div>
                <label className="block text-xs font-medium text-gray-600 mb-1">技能名称</label>
                <input value={form.name} onChange={(e) => setForm({ ...form, name: e.target.value })}
                  placeholder="例如：交付技能的skills" className="w-full px-3 py-2 border border-gray-200 rounded-lg text-sm focus:ring-2 focus:ring-primary-500 outline-none" />
              </div>
              <div>
                <label className="block text-xs font-medium text-gray-600 mb-1">描述</label>
                <textarea value={form.description} onChange={(e) => setForm({ ...form, description: e.target.value })}
                  rows={3} placeholder="描述技能的功能和用途"
                  className="w-full px-3 py-2 border border-gray-200 rounded-lg text-sm focus:ring-2 focus:ring-primary-500 outline-none resize-none" />
              </div>
              <div className="grid grid-cols-2 gap-4">
                <div>
                  <label className="block text-xs font-medium text-gray-600 mb-1">类型</label>
                  <select value={form.type} onChange={(e) => setForm({ ...form, type: e.target.value })}
                    className="w-full px-3 py-2 border border-gray-200 rounded-lg text-sm bg-white">
                    <option value="delivery">交付技能</option>
                    <option value="knowledge">知识技能</option>
                    <option value="ops">运维技能</option>
                    <option value="community">社区技能</option>
                  </select>
                </div>
                <div>
                  <label className="block text-xs font-medium text-gray-600 mb-1">分类标签</label>
                  <input value={form.category} onChange={(e) => setForm({ ...form, category: e.target.value })}
                    placeholder="例如: delivery-skill" className="w-full px-3 py-2 border border-gray-200 rounded-lg text-sm focus:ring-2 focus:ring-primary-500 outline-none" />
                </div>
              </div>
            </div>
            <div className="flex justify-end gap-2 px-6 py-4 border-t border-gray-100">
              <button onClick={() => setShowModal(false)} className="px-4 py-2 text-sm text-gray-600 hover:bg-gray-100 rounded-lg">取消</button>
              <button onClick={handleSave} className="px-4 py-2 bg-primary-600 text-white rounded-lg text-sm hover:bg-primary-700">保存</button>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}
