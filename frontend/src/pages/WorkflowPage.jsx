import React, { useState, useEffect, useCallback, useRef, useMemo } from 'react';
import {
  ReactFlow, Controls, Background, MiniMap, Panel,
  addEdge, applyNodeChanges, applyEdgeChanges,
  Handle, Position, useReactFlow, useViewport,
  ReactFlowProvider,
} from '@xyflow/react';
import '@xyflow/react/dist/style.css';
import {
  GitBranch, Plus, Save, Trash2, Play, Bot, Zap, ArrowRight,
  MessageSquare, Settings2, X, ChevronLeft, Loader2, Edit3,
  Copy, ToggleLeft, ToggleRight, AlertTriangle, ZoomIn, ZoomOut,
  CheckCircle, Clock, Terminal,
} from 'lucide-react';
import { getAgents, getSkills } from '../services/api';
import useStore from '../store/useStore';
import toast from 'react-hot-toast';

// ─── API helpers ────────────────────────────────────────────────────────────
const api = {
  list: () => fetch('/api/workflows', { headers: { Authorization: `Bearer ${useStore.getState().token}` } }).then(r => r.json()),
  get: (id) => fetch(`/api/workflows/${id}`, { headers: { Authorization: `Bearer ${useStore.getState().token}` } }).then(r => r.json()),
  create: (data) => fetch('/api/workflows', { method: 'POST', headers: { 'Content-Type': 'application/json', Authorization: `Bearer ${useStore.getState().token}` }, body: JSON.stringify(data) }).then(r => r.json()),
  update: (id, data) => fetch(`/api/workflows/${id}`, { method: 'PUT', headers: { 'Content-Type': 'application/json', Authorization: `Bearer ${useStore.getState().token}` }, body: JSON.stringify(data) }).then(r => r.json()),
  delete: (id) => fetch(`/api/workflows/${id}`, { method: 'DELETE', headers: { Authorization: `Bearer ${useStore.getState().token}` } }).then(r => r.json()),
};

// ─── Custom Node Components ─────────────────────────────────────────────────
function StartNode({ data }) {
  return (
    <div className="px-4 py-3 rounded-xl bg-gradient-to-br from-green-400 to-green-600 text-white shadow-lg shadow-green-200 border-2 border-green-300 min-w-[120px]">
      <div className="flex items-center gap-2">
        <Play className="w-4 h-4" />
        <span className="text-sm font-semibold">开始</span>
      </div>
      {data.label && <p className="text-xs mt-1 opacity-80">{data.label}</p>}
      <Handle type="source" position={Position.Right} className="!w-3 !h-3 !bg-white !border-2 !border-green-500" />
    </div>
  );
}

function AgentNode({ data }) {
  return (
    <div className="px-4 py-3 rounded-xl bg-white border-2 border-primary-300 shadow-lg shadow-primary-100 min-w-[160px] hover:shadow-xl transition-shadow">
      <Handle type="target" position={Position.Left} className="!w-3 !h-3 !bg-primary-500 !border-2 !border-white" />
      <div className="flex items-center gap-2 mb-1">
        <div className="w-6 h-6 rounded-lg bg-primary-100 flex items-center justify-center">
          <Bot className="w-3.5 h-3.5 text-primary-600" />
        </div>
        <span className="text-sm font-semibold text-gray-800">{data.label || '智能体'}</span>
      </div>
      {data.description && <p className="text-xs text-gray-400 truncate max-w-[140px]">{data.description}</p>}
      <Handle type="source" position={Position.Right} className="!w-3 !h-3 !bg-primary-500 !border-2 !border-white" />
    </div>
  );
}

function SkillNode({ data }) {
  return (
    <div className="px-4 py-3 rounded-xl bg-white border-2 border-amber-300 shadow-lg shadow-amber-100 min-w-[160px] hover:shadow-xl transition-shadow">
      <Handle type="target" position={Position.Left} className="!w-3 !h-3 !bg-amber-500 !border-2 !border-white" />
      <div className="flex items-center gap-2 mb-1">
        <div className="w-6 h-6 rounded-lg bg-amber-100 flex items-center justify-center">
          <Zap className="w-3.5 h-3.5 text-amber-600" />
        </div>
        <span className="text-sm font-semibold text-gray-800">{data.label || '技能'}</span>
      </div>
      {data.description && <p className="text-xs text-gray-400 truncate max-w-[140px]">{data.description}</p>}
      <Handle type="source" position={Position.Right} className="!w-3 !h-3 !bg-amber-500 !border-2 !border-white" />
    </div>
  );
}

function ConditionNode({ data }) {
  return (
    <div className="px-4 py-3 rounded-xl bg-white border-2 border-purple-300 shadow-lg shadow-purple-100 min-w-[140px] hover:shadow-xl transition-shadow">
      <Handle type="target" position={Position.Left} className="!w-3 !h-3 !bg-purple-500 !border-2 !border-white" />
      <div className="flex items-center gap-2 mb-1">
        <div className="w-6 h-6 rounded-lg bg-purple-100 flex items-center justify-center">
          <GitBranch className="w-3.5 h-3.5 text-purple-600" />
        </div>
        <span className="text-sm font-semibold text-gray-800">{data.label || '条件判断'}</span>
      </div>
      {data.description && <p className="text-xs text-gray-400 truncate max-w-[120px]">{data.description}</p>}
      <Handle type="source" position={Position.Right} id="yes" className="!w-3 !h-3 !bg-green-500 !border-2 !border-white !top-[35%]" />
      <Handle type="source" position={Position.Right} id="no" className="!w-3 !h-3 !bg-red-500 !border-2 !border-white !top-[65%]" />
    </div>
  );
}

function EndNode({ data }) {
  return (
    <div className="px-4 py-3 rounded-xl bg-gradient-to-br from-red-400 to-red-600 text-white shadow-lg shadow-red-200 border-2 border-red-300 min-w-[120px]">
      <Handle type="target" position={Position.Left} className="!w-3 !h-3 !bg-white !border-2 !border-red-500" />
      <div className="flex items-center gap-2">
        <MessageSquare className="w-4 h-4" />
        <span className="text-sm font-semibold">{data.label || '结束'}</span>
      </div>
    </div>
  );
}

const nodeTypes = {
  start: StartNode,
  agent: AgentNode,
  skill: SkillNode,
  condition: ConditionNode,
  end: EndNode,
};

// ─── Default edge style ─────────────────────────────────────────────────────
const defaultEdgeOptions = {
  animated: true,
  style: { strokeWidth: 2, stroke: '#94a3b8' },
  type: 'smoothstep',
};

// ─── Zoom Display Panel ─────────────────────────────────────────────────────
function ZoomIndicator() {
  const { zoom } = useViewport();
  const { zoomIn, zoomOut, fitView } = useReactFlow();
  const pct = Math.round(zoom * 100);
  return (
    <Panel position="top-right" className="!m-3">
      <div className="flex items-center gap-1 bg-white rounded-lg border border-gray-200 shadow-sm px-2 py-1">
        <button onClick={() => zoomOut()} className="p-1 rounded hover:bg-gray-100 text-gray-500">
          <ZoomOut className="w-3.5 h-3.5" />
        </button>
        <span className="text-xs font-medium text-gray-600 w-10 text-center">{pct}%</span>
        <button onClick={() => zoomIn()} className="p-1 rounded hover:bg-gray-100 text-gray-500">
          <ZoomIn className="w-3.5 h-3.5" />
        </button>
      </div>
    </Panel>
  );
}

// ─── Node Config Modal (friendly form, no JSON/special chars) ────────────────
function NodeConfigModal({ node, onClose, onSave }) {
  const [label, setLabel] = useState(node.data.label || '');
  const [description, setDescription] = useState(node.data.description || '');
  // Friendly params as key-value pairs
  const initParams = () => {
    if (node.data.paramList && Array.isArray(node.data.paramList)) return node.data.paramList;
    return [{ key: '', value: '' }];
  };
  const [paramList, setParamList] = useState(initParams);
  const [condField, setCondField] = useState(node.data.condField || '');
  const [condOp, setCondOp] = useState(node.data.condOp || '>');
  const [condValue, setCondValue] = useState(node.data.condValue || '');
  const [inputText, setInputText] = useState(node.data.inputText || '');
  const [sourceRef, setSourceRef] = useState(node.data.sourceRef || '上一节点输出');

  const typeLabels = { agent: '智能体', skill: '技能', condition: '条件判断', start: '开始节点', end: '结束节点' };

  const handleSave = () => {
    const newData = { ...node.data, label, description, paramList, condField, condOp, condValue, inputText, sourceRef };
    // Build readable params string for display in run logs
    const filledParams = paramList.filter(p => p.key);
    if (filledParams.length > 0) {
      newData.params = filledParams.map(p => `${p.key}=${p.value}`).join(', ');
    }
    if (condField) {
      newData.condition = `${condField} ${condOp} ${condValue}`;
    }
    onSave(node.id, newData);
    onClose();
  };

  const addParam = () => setParamList([...paramList, { key: '', value: '' }]);
  const removeParam = (idx) => setParamList(paramList.filter((_, i) => i !== idx));
  const updateParam = (idx, field, val) => {
    const list = [...paramList];
    list[idx] = { ...list[idx], [field]: val };
    setParamList(list);
  };

  return (
    <div className="fixed inset-0 bg-black/40 backdrop-blur-sm z-50 flex items-center justify-center p-4" onClick={onClose}>
      <div className="bg-white rounded-2xl shadow-2xl w-full max-w-md overflow-hidden max-h-[85vh] flex flex-col" onClick={(e) => e.stopPropagation()}>
        <div className="h-1 bg-gradient-to-r from-primary-400 to-primary-600" />
        <div className="p-6 overflow-y-auto flex-1">
          <div className="flex items-center gap-3 mb-5">
            <div className="w-10 h-10 rounded-xl bg-primary-100 flex items-center justify-center">
              {node.type === 'agent' && <Bot className="w-5 h-5 text-primary-600" />}
              {node.type === 'skill' && <Zap className="w-5 h-5 text-amber-600" />}
              {node.type === 'condition' && <GitBranch className="w-5 h-5 text-purple-600" />}
              {node.type === 'start' && <Play className="w-5 h-5 text-green-600" />}
              {node.type === 'end' && <MessageSquare className="w-5 h-5 text-red-600" />}
            </div>
            <div>
              <h3 className="text-lg font-semibold text-gray-800">配置{typeLabels[node.type] || '节点'}</h3>
              <p className="text-xs text-gray-400">设置节点参数和运行条件</p>
            </div>
          </div>

          <div className="space-y-4">
            <div>
              <label className="block text-sm font-medium text-gray-700 mb-1">节点名称</label>
              <input
                value={label}
                onChange={(e) => setLabel(e.target.value)}
                className="w-full px-3 py-2 border border-gray-300 rounded-lg text-sm focus:ring-2 focus:ring-primary-500 focus:border-primary-500"
                placeholder="输入节点名称"
              />
            </div>
            <div>
              <label className="block text-sm font-medium text-gray-700 mb-1">功能说明</label>
              <input
                value={description}
                onChange={(e) => setDescription(e.target.value)}
                className="w-full px-3 py-2 border border-gray-300 rounded-lg text-sm focus:ring-2 focus:ring-primary-500 focus:border-primary-500"
                placeholder="简述该节点的作用"
              />
            </div>

            {node.type === 'condition' && (
              <div className="space-y-3">
                <label className="block text-sm font-medium text-gray-700">判断规则</label>
                <div className="flex items-center gap-2">
                  <input
                    value={condField}
                    onChange={(e) => setCondField(e.target.value)}
                    className="flex-1 px-3 py-2 border border-gray-300 rounded-lg text-sm focus:ring-2 focus:ring-primary-500"
                    placeholder="判断字段"
                  />
                  <select
                    value={condOp}
                    onChange={(e) => setCondOp(e.target.value)}
                    className="px-2 py-2 border border-gray-300 rounded-lg text-sm bg-white"
                  >
                    <option value=">">大于</option>
                    <option value=">=">大于等于</option>
                    <option value="<">小于</option>
                    <option value="<=">小于等于</option>
                    <option value="==">等于</option>
                    <option value="!=">不等于</option>
                    <option value="contains">包含</option>
                  </select>
                  <input
                    value={condValue}
                    onChange={(e) => setCondValue(e.target.value)}
                    className="w-24 px-3 py-2 border border-gray-300 rounded-lg text-sm focus:ring-2 focus:ring-primary-500"
                    placeholder="值"
                  />
                </div>
                <p className="text-xs text-gray-400">满足条件走「是」分支，不满足走「否」分支</p>
              </div>
            )}

            {(node.type === 'agent' || node.type === 'skill') && (
              <div className="space-y-3">
                <div>
                  <label className="block text-sm font-medium text-gray-700 mb-1">数据来源</label>
                  <select
                    value={sourceRef}
                    onChange={(e) => setSourceRef(e.target.value)}
                    className="w-full px-3 py-2 border border-gray-300 rounded-lg text-sm bg-white focus:ring-2 focus:ring-primary-500"
                  >
                    <option value="上一节点输出">上一节点输出</option>
                    <option value="用户输入">用户输入</option>
                    <option value="固定值">固定值</option>
                  </select>
                </div>
                <div>
                  <div className="flex items-center justify-between mb-1">
                    <label className="block text-sm font-medium text-gray-700">参数配置</label>
                    <button onClick={addParam} className="text-xs text-primary-600 hover:text-primary-700 font-medium">+ 添加参数</button>
                  </div>
                  <div className="space-y-2">
                    {paramList.map((p, idx) => (
                      <div key={idx} className="flex items-center gap-2">
                        <input
                          value={p.key}
                          onChange={(e) => updateParam(idx, 'key', e.target.value)}
                          className="flex-1 px-2.5 py-1.5 border border-gray-300 rounded-lg text-sm focus:ring-1 focus:ring-primary-500"
                          placeholder="参数名"
                        />
                        <span className="text-gray-400 text-xs">=</span>
                        <input
                          value={p.value}
                          onChange={(e) => updateParam(idx, 'value', e.target.value)}
                          className="flex-1 px-2.5 py-1.5 border border-gray-300 rounded-lg text-sm focus:ring-1 focus:ring-primary-500"
                          placeholder="参数值"
                        />
                        {paramList.length > 1 && (
                          <button onClick={() => removeParam(idx)} className="p-1 text-gray-400 hover:text-red-500 rounded">
                            <X className="w-3.5 h-3.5" />
                          </button>
                        )}
                      </div>
                    ))}
                  </div>
                  <p className="text-xs text-gray-400 mt-1.5">参数值支持填写文本，系统自动传递给节点</p>
                </div>
              </div>
            )}

            {node.type === 'start' && (
              <div>
                <label className="block text-sm font-medium text-gray-700 mb-1">触发输入内容</label>
                <textarea
                  value={inputText}
                  onChange={(e) => setInputText(e.target.value)}
                  className="w-full px-3 py-2 border border-gray-300 rounded-lg text-sm focus:ring-2 focus:ring-primary-500 focus:border-primary-500 resize-none"
                  rows={3}
                  placeholder="输入启动工作流时传入的内容"
                />
              </div>
            )}

            {node.type === 'end' && (
              <div>
                <label className="block text-sm font-medium text-gray-700 mb-1">输出格式</label>
                <select className="w-full px-3 py-2 border border-gray-300 rounded-lg text-sm bg-white focus:ring-2 focus:ring-primary-500">
                  <option>文本输出</option>
                  <option>结构化数据</option>
                  <option>文件下载</option>
                </select>
              </div>
            )}
          </div>

          <div className="flex justify-end gap-3 mt-6">
            <button onClick={onClose} className="px-4 py-2 text-sm text-gray-600 border border-gray-300 rounded-lg hover:bg-gray-50">取消</button>
            <button onClick={handleSave} className="px-4 py-2 text-sm text-white bg-primary-600 rounded-lg hover:bg-primary-700">确认保存</button>
          </div>
        </div>
      </div>
    </div>
  );
}

// ─── Run Output Modal ───────────────────────────────────────────────────────
function RunOutputModal({ nodes, edges, onClose }) {
  const [running, setRunning] = useState(true);
  const [logs, setLogs] = useState([]);
  const [result, setResult] = useState(null);

  useEffect(() => {
    // Simulate workflow execution
    const simulateRun = async () => {
      setLogs([]);
      const sortedNodes = [...nodes].sort((a, b) => a.position.x - b.position.x);

      for (let i = 0; i < sortedNodes.length; i++) {
        const node = sortedNodes[i];
        const ts = new Date().toLocaleTimeString('zh-CN');
        await new Promise((r) => setTimeout(r, 600 + Math.random() * 400));

        if (node.type === 'start') {
          setLogs((prev) => [...prev, { ts, type: 'info', msg: `[开始] 工作流触发 - ${node.data.label}` }]);
        } else if (node.type === 'agent') {
          setLogs((prev) => [...prev, { ts, type: 'process', msg: `[智能体] 执行: ${node.data.label}${node.data.params ? ` (参数: ${node.data.params.slice(0, 50)})` : ''}` }]);
        } else if (node.type === 'skill') {
          setLogs((prev) => [...prev, { ts, type: 'process', msg: `[技能] 调用: ${node.data.label}${node.data.params ? ` (参数: ${node.data.params.slice(0, 50)})` : ''}` }]);
        } else if (node.type === 'condition') {
          setLogs((prev) => [...prev, { ts, type: 'branch', msg: `[条件] 判断: ${node.data.condition || node.data.label} → 结果: Yes` }]);
        } else if (node.type === 'end') {
          setLogs((prev) => [...prev, { ts, type: 'success', msg: `[完成] 工作流执行成功 - ${node.data.label}` }]);
        }
      }

      setResult({ status: 'success', message: '工作流执行完成', nodeCount: sortedNodes.length, edgeCount: edges.length });
      setRunning(false);
    };

    simulateRun();
  }, [nodes, edges]);

  const logColors = { info: 'text-blue-600', process: 'text-gray-700', branch: 'text-purple-600', success: 'text-green-600', error: 'text-red-600' };

  return (
    <div className="fixed inset-0 bg-black/40 backdrop-blur-sm z-50 flex items-center justify-center p-4" onClick={onClose}>
      <div className="bg-white rounded-2xl shadow-2xl w-full max-w-lg overflow-hidden max-h-[80vh] flex flex-col" onClick={(e) => e.stopPropagation()}>
        <div className="h-1 bg-gradient-to-r from-green-400 via-blue-500 to-purple-600" />
        <div className="p-5 border-b border-gray-100 flex items-center justify-between flex-shrink-0">
          <div className="flex items-center gap-3">
            <div className="w-10 h-10 rounded-xl bg-green-50 flex items-center justify-center">
              {running ? <Loader2 className="w-5 h-5 text-green-600 animate-spin" /> : <CheckCircle className="w-5 h-5 text-green-600" />}
            </div>
            <div>
              <h3 className="text-base font-semibold text-gray-800">{running ? '正在执行...' : '执行完成'}</h3>
              <p className="text-xs text-gray-400">{running ? '工作流节点逐步运行中' : `共 ${result?.nodeCount} 个节点, ${result?.edgeCount} 条连线`}</p>
            </div>
          </div>
          <button onClick={onClose} className="p-1.5 rounded-lg hover:bg-gray-100 text-gray-400"><X className="w-5 h-5" /></button>
        </div>
        <div className="flex-1 overflow-auto p-4">
          <div className="bg-gray-900 rounded-xl p-4 font-mono text-xs space-y-1.5 min-h-[200px]">
            {logs.map((log, i) => (
              <div key={i} className="flex gap-2">
                <span className="text-gray-500 flex-shrink-0">{log.ts}</span>
                <span className={logColors[log.type] || 'text-gray-300'}>{log.msg}</span>
              </div>
            ))}
            {running && <span className="inline-block w-2 h-4 bg-green-400 animate-pulse" />}
          </div>
        </div>
        <div className="p-4 border-t border-gray-100 flex justify-end flex-shrink-0">
          <button onClick={onClose} className="px-4 py-2 text-sm font-medium text-gray-600 bg-gray-100 rounded-lg hover:bg-gray-200">
            {running ? '后台运行' : '关闭'}
          </button>
        </div>
      </div>
    </div>
  );
}

// ─── Workflow Editor (inner, with useReactFlow) ─────────────────────────────
function WorkflowEditor({ workflow, agents, skills, onSave, onBack }) {
  const initialFlow = useMemo(() => {
    if (workflow?.flow_data) {
      try { return JSON.parse(workflow.flow_data); } catch { /* ignore */ }
    }
    return {
      nodes: [
        { id: 'start-1', type: 'start', position: { x: 50, y: 200 }, data: { label: '触发' } },
        { id: 'end-1', type: 'end', position: { x: 600, y: 200 }, data: { label: '输出' } },
      ],
      edges: [],
    };
  }, [workflow]);

  const [nodes, setNodes] = useState(initialFlow.nodes);
  const [edges, setEdges] = useState(initialFlow.edges);
  const [saving, setSaving] = useState(false);
  const [name, setName] = useState(workflow?.name || '');
  const [description, setDescription] = useState(workflow?.description || '');
  const [configNode, setConfigNode] = useState(null);
  const [showRunOutput, setShowRunOutput] = useState(false);
  const reactFlowInstance = useReactFlow();
  const idCounter = useRef(100);

  const onNodesChange = useCallback((changes) => setNodes((nds) => applyNodeChanges(changes, nds)), []);
  const onEdgesChange = useCallback((changes) => setEdges((eds) => applyEdgeChanges(changes, eds)), []);
  const onConnect = useCallback((connection) => setEdges((eds) => addEdge(connection, eds)), []);

  // Node double-click → open config modal
  const onNodeDoubleClick = useCallback((_, node) => {
    setConfigNode(node);
  }, []);

  // Save node config
  const handleNodeConfigSave = (nodeId, newData) => {
    setNodes((nds) => nds.map((n) => n.id === nodeId ? { ...n, data: newData } : n));
  };

  const addNode = (type, item) => {
    idCounter.current += 1;
    const id = `${type}-${idCounter.current}`;
    const center = reactFlowInstance.screenToFlowPosition({ x: 400, y: 300 });
    const newNode = {
      id,
      type,
      position: { x: center.x + Math.random() * 50, y: center.y + Math.random() * 50 },
      data: {
        label: item?.name || (type === 'condition' ? '条件判断' : type === 'end' ? '输出' : '节点'),
        description: item?.description || '',
        refId: item?.id,
      },
    };
    setNodes((nds) => [...nds, newNode]);
  };

  const handleSave = async () => {
    if (!name.trim()) { toast.error('请输入工作流名称'); return; }
    setSaving(true);
    const flowData = JSON.stringify({ nodes, edges });
    try {
      const res = workflow?.id
        ? await api.update(workflow.id, { name, description, flow_data: flowData })
        : await api.create({ name, description, flow_data: flowData, is_active: true });
      if (res.code === 0) {
        toast.success('保存成功');
        onSave(res.data);
      } else {
        toast.error(res.message || '保存失败');
      }
    } catch { toast.error('保存失败'); }
    setSaving(false);
  };

  // DnD from sidebar
  const onDragOver = useCallback((e) => { e.preventDefault(); e.dataTransfer.dropEffect = 'move'; }, []);
  const onDrop = useCallback((e) => {
    e.preventDefault();
    const raw = e.dataTransfer.getData('application/reactflow');
    if (!raw) return;
    try {
      const { type, item } = JSON.parse(raw);
      const position = reactFlowInstance.screenToFlowPosition({ x: e.clientX, y: e.clientY });
      idCounter.current += 1;
      const id = `${type}-${idCounter.current}`;
      setNodes((nds) => [...nds, {
        id, type, position,
        data: { label: item?.name || type, description: item?.description || '', refId: item?.id },
      }]);
    } catch { /* ignore */ }
  }, [reactFlowInstance]);

  return (
    <div className="h-full flex flex-col">
      {/* Top toolbar */}
      <div className="flex items-center gap-3 px-4 py-2 border-b border-gray-200 bg-white flex-shrink-0">
        <button onClick={onBack} className="p-1.5 rounded-lg hover:bg-gray-100 text-gray-500">
          <ChevronLeft className="w-5 h-5" />
        </button>
        <input
          value={name}
          onChange={(e) => setName(e.target.value)}
          placeholder="工作流名称"
          className="text-sm font-semibold border-none focus:ring-0 bg-transparent w-48 px-2 py-1 rounded hover:bg-gray-50 focus:bg-gray-50"
        />
        <input
          value={description}
          onChange={(e) => setDescription(e.target.value)}
          placeholder="描述（可选）"
          className="text-xs text-gray-400 border-none focus:ring-0 bg-transparent flex-1 px-2 py-1 rounded hover:bg-gray-50 focus:bg-gray-50"
        />
        <button
          onClick={() => setShowRunOutput(true)}
          className="flex items-center gap-1.5 px-3 py-1.5 bg-green-600 text-white rounded-lg text-sm font-medium hover:bg-green-700"
        >
          <Play className="w-4 h-4" />
          运行
        </button>
        <button
          onClick={handleSave}
          disabled={saving}
          className="flex items-center gap-1.5 px-3 py-1.5 bg-primary-600 text-white rounded-lg text-sm font-medium hover:bg-primary-700 disabled:opacity-50"
        >
          {saving ? <Loader2 className="w-4 h-4 animate-spin" /> : <Save className="w-4 h-4" />}
          保存
        </button>
      </div>

      <div className="flex-1 flex overflow-hidden">
        {/* Left panel: node palette */}
        <div className="w-56 border-r border-gray-200 bg-gray-50 overflow-y-auto flex-shrink-0">
          <div className="p-3">
            <p className="text-xs font-semibold text-gray-500 uppercase tracking-wide mb-2">节点类型</p>
            <div className="space-y-1.5">
              <DragItem type="condition" label="条件判断" icon={<GitBranch className="w-3.5 h-3.5 text-purple-600" />} color="border-purple-200 bg-purple-50" />
              <DragItem type="end" label="结束节点" icon={<MessageSquare className="w-3.5 h-3.5 text-red-500" />} color="border-red-200 bg-red-50" />
            </div>

            <p className="text-xs font-semibold text-gray-500 uppercase tracking-wide mt-4 mb-2">智能体</p>
            <div className="space-y-1.5 max-h-40 overflow-y-auto">
              {agents.map((a) => (
                <DragItem key={a.id} type="agent" item={a} label={a.name} icon={<Bot className="w-3.5 h-3.5 text-primary-600" />} color="border-primary-200 bg-primary-50" />
              ))}
              {agents.length === 0 && <p className="text-xs text-gray-400 px-2">暂无智能体</p>}
            </div>

            <p className="text-xs font-semibold text-gray-500 uppercase tracking-wide mt-4 mb-2">技能</p>
            <div className="space-y-1.5 max-h-40 overflow-y-auto">
              {skills.map((s) => (
                <DragItem key={s.id} type="skill" item={s} label={s.name} icon={<Zap className="w-3.5 h-3.5 text-amber-600" />} color="border-amber-200 bg-amber-50" />
              ))}
              {skills.length === 0 && <p className="text-xs text-gray-400 px-2">暂无技能</p>}
            </div>
          </div>
        </div>

        {/* Canvas */}
        <div className="flex-1">
          <ReactFlow
            nodes={nodes}
            edges={edges}
            onNodesChange={onNodesChange}
            onEdgesChange={onEdgesChange}
            onConnect={onConnect}
            onNodeDoubleClick={onNodeDoubleClick}
            onDragOver={onDragOver}
            onDrop={onDrop}
            nodeTypes={nodeTypes}
            defaultEdgeOptions={defaultEdgeOptions}
            defaultViewport={{ x: 80, y: 60, zoom: 0.8 }}
            minZoom={0.3}
            maxZoom={2}
            deleteKeyCode={['Backspace', 'Delete']}
            className="bg-gray-50"
          >
            <Background gap={20} size={1} color="#e2e8f0" />
            <Controls position="bottom-right" showZoom={false} className="!shadow-lg !rounded-xl !border !border-gray-200" />
            <ZoomIndicator />
            <MiniMap
              position="bottom-left"
              className="!shadow-lg !rounded-xl !border !border-gray-200"
              nodeColor={(n) => {
                if (n.type === 'agent') return '#7c3aed';
                if (n.type === 'skill') return '#f59e0b';
                if (n.type === 'start') return '#22c55e';
                if (n.type === 'end') return '#ef4444';
                return '#a855f7';
              }}
            />
          </ReactFlow>
        </div>
      </div>

      {/* Node config modal */}
      {configNode && (
        <NodeConfigModal
          node={configNode}
          onClose={() => setConfigNode(null)}
          onSave={handleNodeConfigSave}
        />
      )}

      {/* Run output modal */}
      {showRunOutput && (
        <RunOutputModal
          nodes={nodes}
          edges={edges}
          onClose={() => setShowRunOutput(false)}
        />
      )}
    </div>
  );
}

// Draggable palette item
function DragItem({ type, item, label, icon, color }) {
  const onDragStart = (e) => {
    e.dataTransfer.setData('application/reactflow', JSON.stringify({ type, item }));
    e.dataTransfer.effectAllowed = 'move';
  };
  return (
    <div
      draggable
      onDragStart={onDragStart}
      className={`flex items-center gap-2 px-3 py-2 rounded-lg border cursor-grab active:cursor-grabbing text-xs font-medium text-gray-700 hover:shadow-sm transition-shadow ${color}`}
    >
      {icon}
      <span className="truncate">{label}</span>
    </div>
  );
}

// ─── Workflow List View ──────────────────────────────────────────────────────
function WorkflowList({ workflows, loading, onCreate, onEdit, onDelete, onToggle }) {
  if (loading) {
    return (
      <div className="flex items-center justify-center h-64">
        <Loader2 className="w-6 h-6 animate-spin text-primary-500" />
      </div>
    );
  }

  return (
    <div className="p-6">
      <div className="flex items-center justify-between mb-6">
        <div>
          <h2 className="text-lg font-semibold text-gray-800">工作流列表</h2>
          <p className="text-sm text-gray-400">将智能体与技能通过可视化流程关联在一起</p>
        </div>
        <button
          onClick={onCreate}
          className="flex items-center gap-1.5 px-4 py-2 bg-primary-600 text-white rounded-xl text-sm font-medium hover:bg-primary-700 transition-colors shadow-sm"
        >
          <Plus className="w-4 h-4" />
          新建工作流
        </button>
      </div>

      {workflows.length === 0 ? (
        <div className="text-center py-20">
          <div className="w-16 h-16 rounded-2xl bg-gray-100 flex items-center justify-center mx-auto mb-4">
            <GitBranch className="w-8 h-8 text-gray-300" />
          </div>
          <p className="text-gray-500 mb-2">暂无工作流</p>
          <p className="text-sm text-gray-400">创建一个工作流来连接智能体和技能</p>
        </div>
      ) : (
        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
          {workflows.map((wf) => (
            <div key={wf.id} className="bg-white rounded-xl border border-gray-200 p-5 hover:shadow-md transition-shadow group">
              <div className="flex items-start justify-between mb-3">
                <div className="flex items-center gap-2">
                  <div className="w-9 h-9 rounded-xl bg-primary-50 flex items-center justify-center">
                    <GitBranch className="w-4.5 h-4.5 text-primary-600" />
                  </div>
                  <div>
                    <h3 className="text-sm font-semibold text-gray-800">{wf.name}</h3>
                    <p className="text-xs text-gray-400">{wf.description || '无描述'}</p>
                  </div>
                </div>
                <button
                  onClick={() => onToggle(wf)}
                  className={`p-1 rounded ${wf.is_active ? 'text-green-500' : 'text-gray-300'}`}
                  title={wf.is_active ? '已启用' : '已禁用'}
                >
                  {wf.is_active ? <ToggleRight className="w-5 h-5" /> : <ToggleLeft className="w-5 h-5" />}
                </button>
              </div>
              <div className="flex items-center justify-between mt-4">
                <span className="text-xs text-gray-400">
                  更新于 {new Date(wf.updated_at).toLocaleString('zh-CN', { month: '2-digit', day: '2-digit', hour: '2-digit', minute: '2-digit' })}
                </span>
                <div className="flex items-center gap-1 opacity-0 group-hover:opacity-100 transition-opacity">
                  <button onClick={() => onEdit(wf)} className="p-1.5 rounded-lg hover:bg-gray-100 text-gray-400 hover:text-primary-600" title="编辑">
                    <Edit3 className="w-4 h-4" />
                  </button>
                  <button onClick={() => onDelete(wf)} className="p-1.5 rounded-lg hover:bg-red-50 text-gray-400 hover:text-red-500" title="删除">
                    <Trash2 className="w-4 h-4" />
                  </button>
                </div>
              </div>
            </div>
          ))}
        </div>
      )}
    </div>
  );
}

// ─── Main Page Component ────────────────────────────────────────────────────
export default function WorkflowPage() {
  const [workflows, setWorkflows] = useState([]);
  const [loading, setLoading] = useState(true);
  const [editing, setEditing] = useState(null); // null = list view, object = editor
  const [agents, setAgents] = useState([]);
  const [skills, setSkills] = useState([]);
  const [deleteConfirm, setDeleteConfirm] = useState(null);
  const [deleting, setDeleting] = useState(false);

  const loadWorkflows = async () => {
    setLoading(true);
    try {
      const res = await api.list();
      if (res.code === 0) setWorkflows(res.data || []);
    } catch { /* ignore */ }
    setLoading(false);
  };

  const loadResources = async () => {
    try {
      const [agentRes, skillRes] = await Promise.all([getAgents(), getSkills()]);
      if (agentRes.code === 0) setAgents(agentRes.data || []);
      if (skillRes.code === 0) setSkills(skillRes.data || []);
    } catch { /* ignore */ }
  };

  useEffect(() => { loadWorkflows(); loadResources(); }, []);

  const handleCreate = () => setEditing({});
  const handleEdit = (wf) => setEditing(wf);
  const handleBack = () => { setEditing(null); loadWorkflows(); };
  const handleSave = () => { setEditing(null); loadWorkflows(); };

  const handleToggle = async (wf) => {
    const res = await api.update(wf.id, { is_active: !wf.is_active });
    if (res.code === 0) {
      toast.success(wf.is_active ? '已禁用' : '已启用');
      loadWorkflows();
    }
  };

  const handleDelete = (wf) => setDeleteConfirm(wf);
  const confirmDelete = async () => {
    if (!deleteConfirm) return;
    setDeleting(true);
    try {
      const res = await api.delete(deleteConfirm.id);
      if (res.code === 0) { toast.success('删除成功'); loadWorkflows(); }
      else toast.error(res.message || '删除失败');
    } catch { toast.error('删除失败'); }
    setDeleting(false);
    setDeleteConfirm(null);
  };

  if (editing !== null) {
    return (
      <ReactFlowProvider>
        <WorkflowEditor
          workflow={editing.id ? editing : null}
          agents={agents}
          skills={skills}
          onSave={handleSave}
          onBack={handleBack}
        />
      </ReactFlowProvider>
    );
  }

  return (
    <div className="h-full overflow-auto">
      <WorkflowList
        workflows={workflows}
        loading={loading}
        onCreate={handleCreate}
        onEdit={handleEdit}
        onDelete={handleDelete}
        onToggle={handleToggle}
      />

      {/* Delete confirmation modal */}
      {deleteConfirm && (
        <div className="fixed inset-0 bg-black/40 backdrop-blur-sm z-50 flex items-center justify-center p-4" onClick={() => setDeleteConfirm(null)}>
          <div className="bg-white rounded-2xl shadow-2xl w-full max-w-sm overflow-hidden" onClick={(e) => e.stopPropagation()}>
            <div className="h-1 bg-gradient-to-r from-red-400 via-red-500 to-red-600" />
            <div className="p-6 text-center">
              <div className="w-14 h-14 rounded-full bg-red-50 flex items-center justify-center mx-auto mb-4">
                <AlertTriangle className="w-7 h-7 text-red-500" />
              </div>
              <h3 className="text-lg font-semibold text-gray-800 mb-2">确认删除工作流</h3>
              <p className="text-sm text-gray-500">确定要删除工作流 <strong>{deleteConfirm.name}</strong> 吗？</p>
              <p className="text-xs text-gray-400 mt-2">此操作不可撤销</p>
            </div>
            <div className="px-6 pb-6 flex items-center gap-3 justify-center">
              <button onClick={() => setDeleteConfirm(null)} className="px-5 py-2.5 text-sm font-medium text-gray-600 bg-gray-100 rounded-xl hover:bg-gray-200">取消</button>
              <button onClick={confirmDelete} disabled={deleting} className="px-5 py-2.5 text-sm font-medium text-white bg-red-500 rounded-xl hover:bg-red-600 disabled:opacity-60 flex items-center gap-2">
                {deleting ? <Loader2 className="w-4 h-4 animate-spin" /> : <Trash2 className="w-4 h-4" />}
                确认删除
              </button>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}
