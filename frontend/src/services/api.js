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
    if (error.response?.status === 401) {
      localStorage.removeItem('token');
      localStorage.removeItem('user');
      window.location.href = '/login';
    }
    return Promise.reject(error.response?.data || error);
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

// Skills
export const getSkills = () => api.get('/skills');
export const getAgentSkills = (agentId) => api.get(`/agents/${agentId}/skills`);

// Users (Admin)
export const getUsers = () => api.get('/users');
export const createUser = (data) => api.post('/users', data);
export const updateUser = (id, data) => api.put(`/users/${id}`, data);
export const deleteUser = (id) => api.delete(`/users/${id}`);

// LDAP Configuration (Admin)
export const getLDAPConfigs = () => api.get('/ldap-configs');
export const createLDAPConfig = (data) => api.post('/ldap-configs', data);
export const updateLDAPConfig = (id, data) => api.put(`/ldap-configs/${id}`, data);
export const deleteLDAPConfig = (id) => api.delete(`/ldap-configs/${id}`);
export const testLDAPConfig = (id) => api.post(`/ldap-configs/${id}/test`);

// AI Providers
export const getAIProviders = () => api.get('/ai-providers');
export const updateAIProvider = (id, data) => api.put(`/ai-providers/${id}`, data);
export const testAIProvider = (id) => api.post(`/ai-providers/${id}/test`);

// Website Links
export const getWebsiteCategories = () => api.get('/website-categories');

// Operation Logs
export const getOperationLogs = (params) => api.get('/operation-logs', { params });

export default api;
