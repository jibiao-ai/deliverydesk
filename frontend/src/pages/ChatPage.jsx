import React, { useState, useEffect, useRef } from 'react';
import { Send, Bot, User, Plus, Trash2, Loader2 } from 'lucide-react';
import { getAgents, getConversations, createConversation, deleteConversation, getMessages, sendMessage } from '../services/api';
import useStore from '../store/useStore';
import toast from 'react-hot-toast';

export default function ChatPage() {
  const [agents, setAgents] = useState([]);
  const [convs, setConvs] = useState([]);
  const [currentConv, setCurrentConv] = useState(null);
  const [msgs, setMsgs] = useState([]);
  const [input, setInput] = useState('');
  const [sending, setSending] = useState(false);
  const [selectedAgent, setSelectedAgent] = useState(null);
  const messagesEndRef = useRef(null);

  useEffect(() => {
    loadAgents();
    loadConvs();
  }, []);

  useEffect(() => {
    if (currentConv) loadMessages(currentConv.id);
  }, [currentConv]);

  useEffect(() => {
    messagesEndRef.current?.scrollIntoView({ behavior: 'smooth' });
  }, [msgs]);

  const loadAgents = async () => {
    try { const res = await getAgents(); if (res.code === 0) { setAgents(res.data || []); if (!selectedAgent && res.data?.length) setSelectedAgent(res.data[0]); } } catch (e) {}
  };

  const loadConvs = async () => {
    try { const res = await getConversations(); if (res.code === 0) setConvs(res.data || []); } catch (e) {}
  };

  const loadMessages = async (convId) => {
    try { const res = await getMessages(convId); if (res.code === 0) setMsgs(res.data || []); } catch (e) {}
  };

  const handleNewConv = async () => {
    if (!selectedAgent) { toast.error('请先选择智能体'); return; }
    try {
      const res = await createConversation(selectedAgent.id, `与${selectedAgent.name}的对话`);
      if (res.code === 0) { setCurrentConv(res.data); setMsgs([]); loadConvs(); }
    } catch (e) { toast.error('创建会话失败'); }
  };

  const handleSend = async () => {
    if (!input.trim() || !currentConv || sending) return;
    const content = input.trim();
    setInput('');
    setSending(true);
    setMsgs(prev => [...prev, { role: 'user', content, id: Date.now() }]);
    try {
      const res = await sendMessage(currentConv.id, content);
      if (res.code === 0) {
        setMsgs(prev => [...prev.filter(m => m.role !== 'assistant' || m.id), res.data.assistant_message]);
      }
    } catch (e) { toast.error('发送失败'); }
    finally { setSending(false); }
  };

  const handleDeleteConv = async (id) => {
    try {
      await deleteConversation(id);
      if (currentConv?.id === id) { setCurrentConv(null); setMsgs([]); }
      loadConvs();
    } catch (e) {}
  };

  return (
    <div className="h-full flex">
      {/* Sidebar: conversations */}
      <div className="w-64 border-r border-gray-200 bg-white flex flex-col flex-shrink-0">
        <div className="p-3 border-b border-gray-100">
          <select value={selectedAgent?.id || ''} onChange={(e) => setSelectedAgent(agents.find(a => a.id == e.target.value))}
            className="w-full px-3 py-2 border border-gray-200 rounded-lg text-sm mb-2 bg-white">
            <option value="">选择智能体</option>
            {agents.map(a => <option key={a.id} value={a.id}>{a.name}</option>)}
          </select>
          <button onClick={handleNewConv} className="w-full flex items-center justify-center gap-2 px-3 py-2 bg-blue-600 text-white rounded-lg text-sm hover:bg-blue-700">
            <Plus className="w-4 h-4" /> 新建对话
          </button>
        </div>
        <div className="flex-1 overflow-y-auto">
          {convs.map(conv => (
            <div key={conv.id}
              onClick={() => setCurrentConv(conv)}
              className={`flex items-center gap-2 px-3 py-2.5 cursor-pointer border-b border-gray-50 group ${currentConv?.id === conv.id ? 'bg-blue-50 text-blue-600' : 'hover:bg-gray-50'}`}>
              <Bot className="w-4 h-4 flex-shrink-0" />
              <span className="text-sm truncate flex-1">{conv.title}</span>
              <button onClick={(e) => { e.stopPropagation(); handleDeleteConv(conv.id); }}
                className="opacity-0 group-hover:opacity-100 p-1 text-gray-400 hover:text-red-500">
                <Trash2 className="w-3 h-3" />
              </button>
            </div>
          ))}
        </div>
      </div>

      {/* Chat area */}
      <div className="flex-1 flex flex-col">
        {currentConv ? (
          <>
            <div className="px-6 py-3 border-b border-gray-100 bg-white">
              <h3 className="text-sm font-medium text-gray-700">{currentConv.title}</h3>
            </div>
            <div className="flex-1 overflow-y-auto p-6 space-y-4">
              {msgs.map((msg, i) => (
                <div key={msg.id || i} className={`flex gap-3 ${msg.role === 'user' ? 'justify-end' : ''}`}>
                  {msg.role !== 'user' && (
                    <div className="w-8 h-8 rounded-full bg-blue-100 flex items-center justify-center flex-shrink-0">
                      <Bot className="w-4 h-4 text-blue-600" />
                    </div>
                  )}
                  <div className={`max-w-[70%] px-4 py-3 rounded-2xl text-sm ${
                    msg.role === 'user' ? 'bg-blue-600 text-white' : 'bg-gray-100 text-gray-800'
                  }`}>
                    <div className="chat-message whitespace-pre-wrap">{msg.content}</div>
                  </div>
                  {msg.role === 'user' && (
                    <div className="w-8 h-8 rounded-full bg-blue-600 flex items-center justify-center flex-shrink-0">
                      <User className="w-4 h-4 text-white" />
                    </div>
                  )}
                </div>
              ))}
              {sending && (
                <div className="flex gap-3">
                  <div className="w-8 h-8 rounded-full bg-blue-100 flex items-center justify-center">
                    <Bot className="w-4 h-4 text-blue-600" />
                  </div>
                  <div className="bg-gray-100 px-4 py-3 rounded-2xl">
                    <div className="flex gap-1"><span className="typing-dot" /><span className="typing-dot" /><span className="typing-dot" /></div>
                  </div>
                </div>
              )}
              <div ref={messagesEndRef} />
            </div>
            <div className="px-6 py-4 border-t border-gray-100 bg-white">
              <div className="flex gap-3">
                <input value={input} onChange={(e) => setInput(e.target.value)}
                  onKeyDown={(e) => e.key === 'Enter' && !e.shiftKey && handleSend()}
                  placeholder="输入消息..." disabled={sending}
                  className="flex-1 px-4 py-2.5 border border-gray-200 rounded-xl text-sm focus:ring-2 focus:ring-blue-500 outline-none" />
                <button onClick={handleSend} disabled={sending || !input.trim()}
                  className="px-4 py-2.5 bg-blue-600 text-white rounded-xl hover:bg-blue-700 disabled:opacity-50 flex items-center gap-2 text-sm">
                  {sending ? <Loader2 className="w-4 h-4 animate-spin" /> : <Send className="w-4 h-4" />}
                </button>
              </div>
            </div>
          </>
        ) : (
          <div className="flex-1 flex items-center justify-center">
            <div className="text-center">
              <Bot className="w-16 h-16 text-gray-200 mx-auto mb-4" />
              <p className="text-gray-400 mb-2">选择一个会话或创建新的对话</p>
              <button onClick={handleNewConv} className="text-sm px-4 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700">
                <Plus className="w-4 h-4 inline mr-1" /> 开始对话
              </button>
            </div>
          </div>
        )}
      </div>
    </div>
  );
}
