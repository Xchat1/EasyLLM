import axios from 'axios'

const api = axios.create({
  baseURL: '/api/v1',
  timeout: 30000,
  headers: {
    'Content-Type': 'application/json'
  }
})

const longApi = axios.create({
  baseURL: '/api/v1',
  timeout: 660000,
  headers: {
    'Content-Type': 'application/json'
  }
})

function attachToken(config) {
  const token = localStorage.getItem('easyllm_token')
  if (token) {
    config.headers.Authorization = `Bearer ${token}`
  }
  return config
}

api.interceptors.request.use(attachToken)
longApi.interceptors.request.use(attachToken)

function handleResponseError(error) {
  // 401 redirect removed
  const message = error.response?.data?.error || error.message || 'Unknown error'
  return Promise.reject(new Error(message))
}

longApi.interceptors.response.use(response => response.data, handleResponseError)
api.interceptors.response.use(response => response.data, handleResponseError)

// Auth API (uses raw axios to avoid redirect loops)
const rawApi = axios.create({ baseURL: '/api/v1', timeout: 10000, headers: { 'Content-Type': 'application/json' } })

export const authAPI = {
  check: () => rawApi.get('/auth/check').then(r => r.data),
  login: (password) => rawApi.post('/auth/login', { password }).then(r => r.data),
  setup: (password) => rawApi.post('/auth/setup', { password }).then(r => r.data),
  logout: () => api.post('/auth/logout'),
  changePassword: (old_password, new_password) => api.post('/auth/change-password', { old_password, new_password }),
}

// OpenAI API
export const openaiAPI = {
  list: () => api.get('/openai/accounts'),
  exportJSON: () => api.get('/openai/export'),
  refreshAll: () => longApi.post('/openai/accounts/refresh-all'),
  generateOAuthUrl: (data) => api.post('/openai/oauth/generate-url', data),
  getOAuthSession: (id) => api.get(`/openai/oauth/sessions/${id}`),
  cancelOAuthSession: (id) => api.delete(`/openai/oauth/sessions/${id}`),
  exchangeOAuthCode: (data) => api.post('/openai/oauth/exchange-code', data),
  add: (data) => api.post('/openai/accounts', data),
  update: (id, data) => api.put(`/openai/accounts/${id}`, data),
  delete: (id) => api.delete(`/openai/accounts/${id}`),
  deleteMany: (ids) => api.delete('/openai/accounts', { data: { ids } }),
  // API account test
  testAPIAccount: (id) => api.post(`/openai/api-accounts/${id}/test`),
  // Codex
  listCodex: () => api.get('/openai/codex/accounts'),
  addCodex: (data) => api.post('/openai/codex/accounts', data),
  updateCodex: (id, data) => api.put(`/openai/codex/accounts/${id}`, data),
  deleteCodex: (id) => api.delete(`/openai/codex/accounts/${id}`),
  toggleCodex: (id) => api.post(`/openai/codex/accounts/${id}/toggle`),
  getCodexPool: () => api.get('/openai/codex/pool'),
  refreshCodexPool: () => api.post('/openai/codex/pool/refresh'),
  getAvailableModels: (refresh = false) =>
    api.get('/openai/available-models', { params: refresh ? { refresh: '1' } : {} }),
}

// Settings API
export const settingsAPI = {
  get: () => api.get('/settings'),
  update: (data) => api.put('/settings', data),
  getSwitches: () => api.get('/settings/switches'),
  updateSwitches: (data) => api.put('/settings/switches', data),
  getIPBlacklist: () => api.get('/settings/ip-blacklist'),
  updateIPBlacklist: (data) => api.put('/settings/ip-blacklist', data),
  getProxy: () => api.get('/settings/proxy'),
  updateProxy: (data) => api.put('/settings/proxy', data),
  getDatabase: () => api.get('/settings/database'),
  updateDatabase: (data) => api.put('/settings/database', data),
  health: () => api.get('/health'),
  systemInfo: () => api.get('/system/info'),
  apiServerStatus: () => api.get('/api-server/status'),
}

// Relay Config API
export const relayAPI = {
  getConfig: () => api.get('/relay/config'),
  updateConfig: (data) => api.put('/relay/config', data),
  clearSessions: () => api.post('/relay/sessions/clear'),
  getSessionStats: () => api.get('/relay/sessions/stats'),
  getUsage: () => api.get('/relay/usage'),
  getLogs: (limit = 100) => api.get('/relay/logs', { params: { limit } }),
  clearLogs: () => api.delete('/relay/logs'),
  injectCodex: (data) => api.post('/relay/inject-codex', data),
}

export { longApi }
export default api
