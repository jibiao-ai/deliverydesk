import React, { useState, useEffect, useRef, useCallback } from 'react';
import {
  Send, Bot, User, Plus, Trash2, Loader2, X, Square, MessageSquare,
  Zap, ChevronDown, PanelLeftClose, PanelLeftOpen, Sparkles,
} from 'lucide-react';
import {
  getAgents, getConversations, createConversation, deleteConversation,
  getMessages, sendMessageStream, abortStream,
} from '../services/api';
import toast from 'react-hot-toast';

// ============================================================================
// Multi-Tab Chat Page
// ============================================================================

export default function ChatPage() {
  // ─── Global State ────────────────────────────────────────────────────
  const [agents, setAgents] = useState([]);
  const [convs, setConvs] = useState([]);
  const [sidebarOpen, setSidebarOpen] = useState(true);

  // ─── Tab State ───────────────────────────────────────────────────────
  // Each tab: { convId, agentName, title, msgs, input, streaming, streamContent, abortRef }
  const [tabs, setTabs] = useState([]);
  const [activeTabIdx, setActiveTabIdx] = useState(-1);

  // ─── Agent selector for new conversation ─────────────────────────────
  const [selectedAgentId, setSelectedAgentId] = useState(null);

  // ─── Load data ───────────────────────────────────────────────────────
  useEffect(() => {
    loadAgents();
    loadConvs();
  }, []);

  const loadAgents = async () => {
    try {
      const res = await getAgents();
      if (res.code === 0) {
        setAgents(res.data || []);
        if (!selectedAgentId && res.data?.length) setSelectedAgentId(res.data[0].id);
      }
    } catch (e) {}
  };

  const loadConvs = async () => {
    try {
      const res = await getConversations();
      if (res.code === 0) setConvs(res.data || []);
    } catch (e) {}
  };

  // ─── Open a conversation in a new tab (or focus if already open) ────
  const openConvTab = useCallback(async (conv) => {
    const existingIdx = tabs.findIndex((t) => t.convId === conv.id);
    if (existingIdx >= 0) {
      setActiveTabIdx(existingIdx);
      return;
    }

    // Load messages
    let msgs = [];
    try {
      const res = await getMessages(conv.id);
      if (res.code === 0) msgs = res.data || [];
    } catch (e) {}

    const agentObj = agents.find((a) => a.id === conv.agent_id);
    const newTab = {
      convId: conv.id,
      agentName: agentObj?.name || conv.title,
      title: conv.title,
      msgs,
      input: '',
      streaming: false,
      streamContent: '',
      abortRef: null,
    };
    setTabs((prev) => [...prev, newTab]);
    setActiveTabIdx(tabs.length); // point to the new tab
  }, [tabs, agents]);

  // ─── Create new conversation ─────────────────────────────────────────
  const handleNewConv = async () => {
    if (!selectedAgentId) {
      toast.error('请先选择智能体');
      return;
    }
    const agent = agents.find((a) => a.id == selectedAgentId);
    try {
      const res = await createConversation(selectedAgentId, `与${agent?.name || '智能体'}的对话`);
      if (res.code === 0) {
        loadConvs();
        openConvTab(res.data);
      }
    } catch (e) {
      toast.error('创建会话失败');
    }
  };

  // ─── Delete conversation ─────────────────────────────────────────────
  const handleDeleteConv = async (convId) => {
    try {
      await deleteConversation(convId);
      // Close tab if open
      setTabs((prev) => {
        const newTabs = prev.filter((t) => t.convId !== convId);
        // Adjust active tab index
        setActiveTabIdx((idx) => {
          if (newTabs.length === 0) return -1;
          if (idx >= newTabs.length) return newTabs.length - 1;
          return idx;
        });
        return newTabs;
      });
      loadConvs();
    } catch (e) {}
  };

  // ─── Close a tab ─────────────────────────────────────────────────────
  const closeTab = (tabIdx, e) => {
    e?.stopPropagation();
    setTabs((prev) => {
      const newTabs = [...prev];
      // If streaming, abort it first
      if (newTabs[tabIdx]?.abortRef) {
        newTabs[tabIdx].abortRef.abort();
        abortStream(newTabs[tabIdx].convId).catch(() => {});
      }
      newTabs.splice(tabIdx, 1);
      setActiveTabIdx((idx) => {
        if (newTabs.length === 0) return -1;
        if (tabIdx < idx) return idx - 1;
        if (tabIdx === idx) return Math.min(idx, newTabs.length - 1);
        return idx;
      });
      return newTabs;
    });
  };

  // ─── Update a specific tab's state ───────────────────────────────────
  const updateTab = useCallback((tabIdx, updates) => {
    setTabs((prev) => {
      const newTabs = [...prev];
      if (newTabs[tabIdx]) {
        newTabs[tabIdx] = { ...newTabs[tabIdx], ...updates };
      }
      return newTabs;
    });
  }, []);

  // ─── Send message (streaming) ────────────────────────────────────────
  const handleSend = useCallback((tabIdx) => {
    const tab = tabs[tabIdx];
    if (!tab || !tab.input.trim() || tab.streaming) return;

    const content = tab.input.trim();
    const convId = tab.convId;

    // Add user message to UI immediately
    const userMsg = { id: Date.now(), role: 'user', content };
    updateTab(tabIdx, {
      input: '',
      streaming: true,
      streamContent: '',
      msgs: [...tab.msgs, userMsg],
    });

    // Start streaming
    const streamRef = sendMessageStream(convId, content, {
      onToken: (token) => {
        setTabs((prev) => {
          const newTabs = [...prev];
          if (newTabs[tabIdx]) {
            newTabs[tabIdx] = {
              ...newTabs[tabIdx],
              streamContent: newTabs[tabIdx].streamContent + token,
            };
          }
          return newTabs;
        });
      },
      onDone: (data) => {
        setTabs((prev) => {
          const newTabs = [...prev];
          if (newTabs[tabIdx]) {
            const asstMsg = data.assistant_message || {
              id: Date.now() + 1,
              role: 'assistant',
              content: newTabs[tabIdx].streamContent,
            };
            newTabs[tabIdx] = {
              ...newTabs[tabIdx],
              streaming: false,
              streamContent: '',
              abortRef: null,
              msgs: [...newTabs[tabIdx].msgs, asstMsg],
            };
          }
          return newTabs;
        });
        loadConvs(); // refresh sidebar timestamps
      },
      onError: (errMsg) => {
        setTabs((prev) => {
          const newTabs = [...prev];
          if (newTabs[tabIdx]) {
            const errAsstMsg = {
              id: Date.now() + 2,
              role: 'assistant',
              content: newTabs[tabIdx].streamContent || errMsg || '发送失败',
            };
            newTabs[tabIdx] = {
              ...newTabs[tabIdx],
              streaming: false,
              streamContent: '',
              abortRef: null,
              msgs: [...newTabs[tabIdx].msgs, errAsstMsg],
            };
          }
          return newTabs;
        });
        if (errMsg) toast.error(errMsg);
      },
    });

    updateTab(tabIdx, { abortRef: streamRef });
  }, [tabs, updateTab]);

  // ─── Abort streaming ─────────────────────────────────────────────────
  const handleAbort = useCallback((tabIdx) => {
    const tab = tabs[tabIdx];
    if (!tab || !tab.streaming) return;

    // Abort client-side fetch
    if (tab.abortRef) {
      tab.abortRef.abort();
    }
    // Also tell the server to abort
    abortStream(tab.convId).catch(() => {});

    // Finalize with whatever content we have
    setTabs((prev) => {
      const newTabs = [...prev];
      if (newTabs[tabIdx]) {
        let finalContent = newTabs[tabIdx].streamContent || '';
        if (finalContent) {
          finalContent += '\n\n[回复已中断]';
        } else {
          finalContent = '[回复已中断]';
        }
        const abortMsg = { id: Date.now() + 3, role: 'assistant', content: finalContent };
        newTabs[tabIdx] = {
          ...newTabs[tabIdx],
          streaming: false,
          streamContent: '',
          abortRef: null,
          msgs: [...newTabs[tabIdx].msgs, abortMsg],
        };
      }
      return newTabs;
    });
  }, [tabs]);

  const activeTab = tabs[activeTabIdx] || null;

  // ─── Render ──────────────────────────────────────────────────────────
  return (
    <div className="h-full flex bg-gray-50">
      {/* ── Sidebar ── */}
      <div className={`${sidebarOpen ? 'w-72' : 'w-0'} transition-all duration-200 flex-shrink-0 overflow-hidden`}>
        <div className="w-72 h-full border-r border-gray-200 bg-white flex flex-col">
          {/* New conversation controls */}
          <div className="p-3 border-b border-gray-100 space-y-2">
            <div className="flex items-center gap-2">
              <select
                value={selectedAgentId || ''}
                onChange={(e) => setSelectedAgentId(Number(e.target.value) || null)}
                className="flex-1 px-3 py-2 border border-gray-200 rounded-lg text-sm bg-white focus:ring-2 focus:ring-primary-500/20 outline-none"
              >
                <option value="">选择智能体</option>
                {agents.filter(a => a.is_active).map((a) => (
                  <option key={a.id} value={a.id}>{a.name}</option>
                ))}
              </select>
            </div>
            <button
              onClick={handleNewConv}
              className="w-full flex items-center justify-center gap-2 px-3 py-2.5 bg-primary-600 text-white rounded-xl text-sm font-medium hover:bg-primary-700 transition-colors shadow-sm"
            >
              <Plus className="w-4 h-4" /> 新建对话
            </button>
          </div>

          {/* Conversation list */}
          <div className="flex-1 overflow-y-auto">
            {convs.length === 0 ? (
              <div className="p-6 text-center">
                <MessageSquare className="w-8 h-8 text-gray-200 mx-auto mb-2" />
                <p className="text-xs text-gray-400">暂无对话</p>
              </div>
            ) : (
              convs.map((conv) => {
                const isOpenInTab = tabs.some((t) => t.convId === conv.id);
                const isActive = activeTab?.convId === conv.id;
                return (
                  <div
                    key={conv.id}
                    onClick={() => openConvTab(conv)}
                    className={`flex items-center gap-2.5 px-4 py-3 cursor-pointer border-b border-gray-50 group transition-colors ${
                      isActive
                        ? 'bg-primary-50 border-l-2 border-l-primary-500'
                        : isOpenInTab
                        ? 'bg-blue-50/50'
                        : 'hover:bg-gray-50'
                    }`}
                  >
                    <div className={`w-7 h-7 rounded-lg flex items-center justify-center flex-shrink-0 ${
                      isActive ? 'bg-primary-100' : 'bg-gray-100'
                    }`}>
                      <Bot className={`w-3.5 h-3.5 ${isActive ? 'text-primary-600' : 'text-gray-400'}`} />
                    </div>
                    <div className="flex-1 min-w-0">
                      <p className={`text-sm truncate ${isActive ? 'font-semibold text-primary-700' : 'text-gray-700'}`}>
                        {conv.title}
                      </p>
                    </div>
                    {isOpenInTab && (
                      <div className="w-1.5 h-1.5 rounded-full bg-primary-400 flex-shrink-0" />
                    )}
                    <button
                      onClick={(e) => { e.stopPropagation(); handleDeleteConv(conv.id); }}
                      className="opacity-0 group-hover:opacity-100 p-1 text-gray-400 hover:text-red-500 transition-all"
                      title="删除会话"
                    >
                      <Trash2 className="w-3.5 h-3.5" />
                    </button>
                  </div>
                );
              })
            )}
          </div>
        </div>
      </div>

      {/* ── Main Area ── */}
      <div className="flex-1 flex flex-col min-w-0">
        {/* ── Tab Bar ── */}
        <div className="flex items-center bg-white border-b border-gray-200 pl-1 pr-2 min-h-[44px]">
          <button
            onClick={() => setSidebarOpen((v) => !v)}
            className="p-2 text-gray-400 hover:text-gray-600 hover:bg-gray-100 rounded-lg mr-1 flex-shrink-0 transition-colors"
            title={sidebarOpen ? '收起侧栏' : '展开侧栏'}
          >
            {sidebarOpen ? <PanelLeftClose className="w-4 h-4" /> : <PanelLeftOpen className="w-4 h-4" />}
          </button>

          <div className="flex-1 flex items-center overflow-x-auto gap-1 scrollbar-none">
            {tabs.map((tab, idx) => (
              <button
                key={tab.convId}
                onClick={() => setActiveTabIdx(idx)}
                className={`flex items-center gap-1.5 px-3 py-1.5 rounded-lg text-xs font-medium whitespace-nowrap transition-all flex-shrink-0 max-w-[180px] group ${
                  idx === activeTabIdx
                    ? 'bg-primary-50 text-primary-700 border border-primary-200'
                    : 'text-gray-500 hover:bg-gray-100 hover:text-gray-700 border border-transparent'
                }`}
              >
                {tab.streaming ? (
                  <Sparkles className="w-3 h-3 text-amber-500 animate-pulse flex-shrink-0" />
                ) : (
                  <Bot className="w-3 h-3 flex-shrink-0" />
                )}
                <span className="truncate">{tab.agentName}</span>
                <button
                  onClick={(e) => closeTab(idx, e)}
                  className="ml-0.5 p-0.5 rounded text-gray-400 hover:text-red-500 hover:bg-red-50 opacity-0 group-hover:opacity-100 transition-all flex-shrink-0"
                >
                  <X className="w-3 h-3" />
                </button>
              </button>
            ))}
          </div>

          {tabs.length > 0 && (
            <div className="flex items-center gap-1 flex-shrink-0 ml-1">
              <span className="text-[10px] text-gray-400 px-1">{tabs.length} 个对话</span>
            </div>
          )}
        </div>

        {/* ── Chat Content ── */}
        {activeTab ? (
          <ChatPanel
            key={activeTab.convId}
            tab={activeTab}
            tabIdx={activeTabIdx}
            updateTab={updateTab}
            handleSend={handleSend}
            handleAbort={handleAbort}
          />
        ) : (
          <EmptyState
            agents={agents}
            selectedAgentId={selectedAgentId}
            setSelectedAgentId={setSelectedAgentId}
            handleNewConv={handleNewConv}
          />
        )}
      </div>
    </div>
  );
}

// ============================================================================
// Chat Panel — renders messages, input, and streaming state for one tab
// ============================================================================

function ChatPanel({ tab, tabIdx, updateTab, handleSend, handleAbort }) {
  const messagesEndRef = useRef(null);
  const inputRef = useRef(null);

  // Auto-scroll on new messages or streaming content
  useEffect(() => {
    messagesEndRef.current?.scrollIntoView({ behavior: 'smooth' });
  }, [tab.msgs, tab.streamContent]);

  // Focus input when switching tabs
  useEffect(() => {
    inputRef.current?.focus();
  }, [tabIdx]);

  const onKeyDown = (e) => {
    if (e.key === 'Enter' && !e.shiftKey) {
      e.preventDefault();
      handleSend(tabIdx);
    }
  };

  return (
    <div className="flex-1 flex flex-col min-h-0">
      {/* Header */}
      <div className="px-6 py-2.5 border-b border-gray-100 bg-white flex items-center gap-2">
        <Bot className="w-4 h-4 text-primary-500" />
        <h3 className="text-sm font-medium text-gray-700 truncate">{tab.title}</h3>
        {tab.streaming && (
          <span className="flex items-center gap-1 text-xs text-amber-600 bg-amber-50 px-2 py-0.5 rounded-full">
            <Sparkles className="w-3 h-3 animate-pulse" /> 回复中...
          </span>
        )}
      </div>

      {/* Messages */}
      <div className="flex-1 overflow-y-auto px-6 py-4 space-y-4">
        {tab.msgs.length === 0 && !tab.streaming && (
          <div className="flex items-center justify-center h-full">
            <div className="text-center">
              <Zap className="w-10 h-10 text-gray-200 mx-auto mb-3" />
              <p className="text-sm text-gray-400">发送消息开始对话</p>
            </div>
          </div>
        )}

        {tab.msgs.map((msg, i) => (
          <MessageBubble key={msg.id || i} msg={msg} />
        ))}

        {/* Streaming content (assistant is still typing) */}
        {tab.streaming && tab.streamContent && (
          <div className="flex gap-3">
            <div className="w-8 h-8 rounded-xl bg-gradient-to-br from-primary-100 to-primary-50 flex items-center justify-center flex-shrink-0">
              <Bot className="w-4 h-4 text-primary-600" />
            </div>
            <div className="max-w-[75%] px-4 py-3 rounded-2xl rounded-tl-md bg-white border border-gray-200 shadow-sm text-sm text-gray-800">
              <div className="whitespace-pre-wrap">{tab.streamContent}<span className="inline-block w-1.5 h-4 bg-primary-500 animate-pulse ml-0.5 rounded-sm" /></div>
            </div>
          </div>
        )}

        {/* Typing indicator when streaming starts but no content yet */}
        {tab.streaming && !tab.streamContent && (
          <div className="flex gap-3">
            <div className="w-8 h-8 rounded-xl bg-gradient-to-br from-primary-100 to-primary-50 flex items-center justify-center flex-shrink-0">
              <Bot className="w-4 h-4 text-primary-600" />
            </div>
            <div className="bg-white border border-gray-200 shadow-sm px-4 py-3 rounded-2xl rounded-tl-md">
              <div className="flex gap-1.5 items-center">
                <span className="w-2 h-2 rounded-full bg-primary-400 animate-bounce" style={{ animationDelay: '0ms' }} />
                <span className="w-2 h-2 rounded-full bg-primary-400 animate-bounce" style={{ animationDelay: '150ms' }} />
                <span className="w-2 h-2 rounded-full bg-primary-400 animate-bounce" style={{ animationDelay: '300ms' }} />
              </div>
            </div>
          </div>
        )}

        <div ref={messagesEndRef} />
      </div>

      {/* Input Area */}
      <div className="px-6 py-3 border-t border-gray-100 bg-white">
        <div className="flex items-end gap-3">
          <textarea
            ref={inputRef}
            value={tab.input}
            onChange={(e) => updateTab(tabIdx, { input: e.target.value })}
            onKeyDown={onKeyDown}
            placeholder={tab.streaming ? '智能体正在回复中...' : '输入消息，Enter 发送，Shift+Enter 换行'}
            disabled={false}
            rows={1}
            className="flex-1 px-4 py-2.5 border border-gray-200 rounded-xl text-sm focus:ring-2 focus:ring-primary-500/20 focus:border-primary-400 outline-none resize-none max-h-32 min-h-[42px] transition-colors"
            style={{ height: 'auto', minHeight: '42px' }}
            onInput={(e) => {
              e.target.style.height = 'auto';
              e.target.style.height = Math.min(e.target.scrollHeight, 128) + 'px';
            }}
          />

          {tab.streaming ? (
            /* ── STOP Button ── */
            <button
              onClick={() => handleAbort(tabIdx)}
              className="px-4 py-2.5 bg-red-500 text-white rounded-xl hover:bg-red-600 transition-colors flex items-center gap-2 text-sm font-medium shadow-sm min-w-[80px] justify-center"
              title="中断回复"
            >
              <Square className="w-4 h-4 fill-current" />
              <span>中断</span>
            </button>
          ) : (
            /* ── SEND Button ── */
            <button
              onClick={() => handleSend(tabIdx)}
              disabled={!tab.input.trim()}
              className="px-4 py-2.5 bg-primary-600 text-white rounded-xl hover:bg-primary-700 disabled:opacity-40 disabled:cursor-not-allowed transition-colors flex items-center gap-2 text-sm font-medium shadow-sm min-w-[80px] justify-center"
            >
              <Send className="w-4 h-4" />
              <span>发送</span>
            </button>
          )}
        </div>
      </div>
    </div>
  );
}

// ============================================================================
// Message Bubble
// ============================================================================

function MessageBubble({ msg }) {
  const isUser = msg.role === 'user';
  return (
    <div className={`flex gap-3 ${isUser ? 'justify-end' : ''}`}>
      {!isUser && (
        <div className="w-8 h-8 rounded-xl bg-gradient-to-br from-primary-100 to-primary-50 flex items-center justify-center flex-shrink-0">
          <Bot className="w-4 h-4 text-primary-600" />
        </div>
      )}
      <div
        className={`max-w-[75%] px-4 py-3 text-sm ${
          isUser
            ? 'bg-primary-600 text-white rounded-2xl rounded-tr-md shadow-sm'
            : 'bg-white border border-gray-200 text-gray-800 rounded-2xl rounded-tl-md shadow-sm'
        }`}
      >
        <div className="whitespace-pre-wrap break-words">{msg.content}</div>
      </div>
      {isUser && (
        <div className="w-8 h-8 rounded-xl bg-primary-600 flex items-center justify-center flex-shrink-0 shadow-sm">
          <User className="w-4 h-4 text-white" />
        </div>
      )}
    </div>
  );
}

// ============================================================================
// Empty State — shown when no tab is selected
// ============================================================================

function EmptyState({ agents, selectedAgentId, setSelectedAgentId, handleNewConv }) {
  return (
    <div className="flex-1 flex items-center justify-center bg-gradient-to-br from-gray-50 to-white">
      <div className="text-center max-w-md px-6">
        <div className="w-20 h-20 rounded-2xl bg-primary-50 flex items-center justify-center mx-auto mb-5">
          <Bot className="w-10 h-10 text-primary-400" />
        </div>
        <h2 className="text-xl font-semibold text-gray-700 mb-2">多智能体即时对话</h2>
        <p className="text-sm text-gray-400 mb-6 leading-relaxed">
          支持同时与多个智能体进行对话，每个对话独立运行。
          智能体回复时可随时点击「中断」按钮停止回复。
        </p>
        <div className="flex flex-col items-center gap-3">
          <select
            value={selectedAgentId || ''}
            onChange={(e) => setSelectedAgentId(Number(e.target.value) || null)}
            className="px-4 py-2.5 border border-gray-200 rounded-xl text-sm bg-white focus:ring-2 focus:ring-primary-500/20 outline-none w-64"
          >
            <option value="">选择智能体</option>
            {agents.filter(a => a.is_active).map((a) => (
              <option key={a.id} value={a.id}>{a.name}</option>
            ))}
          </select>
          <button
            onClick={handleNewConv}
            className="flex items-center gap-2 px-6 py-2.5 bg-primary-600 text-white rounded-xl hover:bg-primary-700 transition-colors text-sm font-medium shadow-sm"
          >
            <Plus className="w-4 h-4" /> 开始新对话
          </button>
        </div>
        <div className="mt-8 grid grid-cols-3 gap-4 text-xs text-gray-400">
          <div className="flex flex-col items-center gap-1.5">
            <div className="w-9 h-9 rounded-lg bg-blue-50 flex items-center justify-center">
              <MessageSquare className="w-4 h-4 text-blue-400" />
            </div>
            <span>多标签对话</span>
          </div>
          <div className="flex flex-col items-center gap-1.5">
            <div className="w-9 h-9 rounded-lg bg-amber-50 flex items-center justify-center">
              <Sparkles className="w-4 h-4 text-amber-400" />
            </div>
            <span>流式回复</span>
          </div>
          <div className="flex flex-col items-center gap-1.5">
            <div className="w-9 h-9 rounded-lg bg-red-50 flex items-center justify-center">
              <Square className="w-4 h-4 text-red-400" />
            </div>
            <span>随时中断</span>
          </div>
        </div>
      </div>
    </div>
  );
}
