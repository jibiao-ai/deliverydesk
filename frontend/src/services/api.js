import axios from 'axios';

const API_BASE = '/api';

const api = axios.create({
  baseURL: API_BASE,
  timeout: 120000,
  headers: { 'Content-Type': 'application/json' },
});

api.interceptors.request.use((config) => {
  const token = localStorage.getItem('token');
  if (token) {
    config.headers.Authorization = `Bearer ${token}`;
  }
  return config;
});

api.interceptors.response.use(
  (response) => response.data,
  async (error) => {
    const requestUrl = error.config?.url || '';
    if (error.response?.status === 401 && !requestUrl.includes('/login')) {
      localStorage.removeItem('token');
      localStorage.removeItem('user');
      window.location.href = '/login';
    }
    if (error.response?.data) {
      return error.response.data;
    }
    return Promise.reject(error);
  }
);

// Auth
export const login = (username, password, authType) =>
  api.post('/login', { username, password, auth_type: authType });
export const getProfile = () => api.get('/profile');

// Dashboard
export const getDashboard = () => api.get('/dashboard');

// Agents
export const getAgents = () => api.get('/agents');
export const getAgent = (id) => api.get(`/agents/${id}`);
export const createAgent = (data) => api.post('/agents', data);
export const updateAgent = (id, data) => api.put(`/agents/${id}`, data);
export const deleteAgent = (id) => api.delete(`/agents/${id}`);

// Conversations
export const getConversations = () => api.get('/conversations');
export const createConversation = (agentId, title) =>
  api.post('/conversations', { agent_id: agentId, title });
export const deleteConversation = (id) => api.delete(`/conversations/${id}`);

// Messages
export const getMessages = (conversationId) =>
  api.get(`/conversations/${conversationId}/messages`);
export const sendMessage = (conversationId, content) =>
  api.post(`/conversations/${conversationId}/messages`, { content });

// Streaming message via SSE (Server-Sent Events)
// Returns an object { abort } where abort() cancels the stream.
export const sendMessageStream = (conversationId, content, { onToken, onDone, onError }) => {
  const abortController = new AbortController();
  const token = localStorage.getItem('token');

  fetch(`${API_BASE}/conversations/${conversationId}/messages/stream`, {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
      Authorization: `Bearer ${token}`,
    },
    body: JSON.stringify({ content }),
    signal: abortController.signal,
  })
    .then(async (response) => {
      if (!response.ok) {
        const text = await response.text();
        throw new Error(text || `HTTP ${response.status}`);
      }
      const reader = response.body.getReader();
      const decoder = new TextDecoder();
      let buffer = '';

      while (true) {
        const { done, value } = await reader.read();
        if (done) break;

        buffer += decoder.decode(value, { stream: true });
        const lines = buffer.split('\n');
        buffer = lines.pop() || ''; // keep incomplete line in buffer

        for (const line of lines) {
          if (!line.startsWith('data: ')) continue;
          const jsonStr = line.slice(6).trim();
          if (!jsonStr) continue;
          try {
            const data = JSON.parse(jsonStr);
            if (data.done) {
              onDone?.(data);
            } else if (data.error) {
              onError?.(data.error);
            } else if (data.token !== undefined) {
              onToken?.(data.token);
            }
          } catch (e) {
            // ignore malformed JSON
          }
        }
      }
      // Process any remaining buffer
      if (buffer.startsWith('data: ')) {
        try {
          const data = JSON.parse(buffer.slice(6).trim());
          if (data.done) onDone?.(data);
        } catch (e) {}
      }
    })
    .catch((err) => {
      if (err.name === 'AbortError') {
        // Client-initiated abort — not an error
        return;
      }
      onError?.(err.message || 'Stream failed');
    });

  return {
    abort: () => abortController.abort(),
  };
};

// Abort a streaming response for a conversation
export const abortStream = (conversationId) =>
  api.post(`/conversations/${conversationId}/abort`);

// Skills
export const getSkills = () => api.get('/skills');
export const getSkill = (id) => api.get(`/skills/${id}`);
export const createSkill = (data) => api.post('/skills', data);
export const updateSkill = (id, data) => api.put(`/skills/${id}`, data);
export const deleteSkill = (id) => api.delete(`/skills/${id}`);
export const uploadSkillDocument = (skillId, file) => {
  const formData = new FormData();
  formData.append('file', file);
  return api.post(`/skills/${skillId}/upload`, formData, {
    headers: { 'Content-Type': 'multipart/form-data' },
    timeout: 300000,
  });
};
export const reindexSkill = (id) => api.post(`/skills/${id}/reindex`);
export const getAgentSkills = (agentId) => api.get(`/agents/${agentId}/skills`);

// Published Agents (External)
export const getPublishedAgents = () => api.get('/published-agents');
export const chatWithPublishedAgent = (agentId, message) =>
  api.post(`/published-agents/${agentId}/chat`, { message });

// Users (Admin)
export const getUsers = (params) => api.get('/users', { params });
export const getUserStats = () => api.get('/users/stats');
export const createUser = (data) => api.post('/users', data);
export const updateUser = (id, data) => api.put(`/users/${id}`, data);
export const deleteUser = (id) => api.delete(`/users/${id}`);

// LDAP Configuration (Admin)
export const getLDAPConfigs = () => api.get('/ldap-configs');
export const createLDAPConfig = (data) => api.post('/ldap-configs', data);
export const updateLDAPConfig = (id, data) => api.put(`/ldap-configs/${id}`, data);
export const deleteLDAPConfig = (id) => api.delete(`/ldap-configs/${id}`);
export const testLDAPConfig = (id) => api.post(`/ldap-configs/${id}/test`);
export const syncLDAPUsers = () => api.post('/ldap-configs/sync-users');

// AI Providers
export const getAIProviders = () => api.get('/ai-providers');
export const createAIProvider = (data) => api.post('/ai-providers', data);
export const updateAIProvider = (id, data) => api.put(`/ai-providers/${id}`, data);
export const deleteAIProvider = (id) => api.delete(`/ai-providers/${id}`);
export const testAIProvider = (id) => api.post(`/ai-providers/${id}/test`);

// Website Links
export const getWebsiteCategories = () => api.get('/website-categories');

// Operation Logs
export const getOperationLogs = (params) => api.get('/operation-logs', { params });

export default api;
