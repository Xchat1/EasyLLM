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
  timeout: 120000,
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
  if (error.response?.status === 401) {
    const path = window.location.pathname
    if (path !== '/login') {
      localStorage.removeItem('easyllm_token')
      window.location.href = '/login'
    }
  }
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

// Augment API
export const augmentAPI = {
  list: () => api.get('/augment/tokens'),
  add: (data) => api.post('/augment/tokens', data),
  update: (id, data) => api.put(`/augment/tokens/${id}`, data),
  delete: (id) => api.delete(`/augment/tokens/${id}`),
  deleteMany: (ids) => api.delete('/augment/tokens', { data: { ids } }),
  checkStatus: (id) => api.post(`/augment/tokens/${id}/check`),
  checkAll: () => api.post('/augment/tokens/check-all'),
  getCreditInfo: (id) => api.post(`/augment/tokens/${id}/credit`),
  refreshSession: (id) => api.post(`/augment/tokens/${id}/refresh-session`),
  batchRefreshSessions: (ids) => api.post('/augment/tokens/batch-refresh-sessions', { ids }),
  startOAuth: () => api.post('/augment/oauth/start'),
  completeOAuth: (code) => api.post('/augment/oauth/complete', { code }),
  importSession: (session, detailed = true) => api.post('/augment/import/session', { session, detailed_response: detailed }),
  importSessions: (sessions, detailed = true) => api.post('/augment/import/sessions', { sessions, detailed_response: detailed }),
  exportJSON: () => api.get('/augment/export'),
  importJSON: (tokens) => api.post('/augment/import/json', tokens),
  sync: (req) => api.post('/augment/sync', req),
}

// OpenAI API
export const openaiAPI = {
  list: () => api.get('/openai/accounts'),
  exportJSON: () => api.get('/openai/export'),
  refreshAll: () => api.post('/openai/accounts/refresh-all'),
  add: (data) => api.post('/openai/accounts', data),
  update: (id, data) => api.put(`/openai/accounts/${id}`, data),
  delete: (id) => api.delete(`/openai/accounts/${id}`),
  deleteMany: (ids) => api.delete('/openai/accounts', { data: { ids } }),
  // Codex
  listCodex: () => api.get('/openai/codex/accounts'),
  addCodex: (data) => api.post('/openai/codex/accounts', data),
  updateCodex: (id, data) => api.put(`/openai/codex/accounts/${id}`, data),
  deleteCodex: (id) => api.delete(`/openai/codex/accounts/${id}`),
  toggleCodex: (id) => api.post(`/openai/codex/accounts/${id}/toggle`),
  getCodexPool: () => api.get('/openai/codex/pool'),
  refreshCodexPool: () => api.post('/openai/codex/pool/refresh'),
  getCodexLogs: (params) => api.get('/openai/codex/logs', { params }),
  clearCodexLogs: () => api.delete('/openai/codex/logs'),
  getAvailableModels: (refresh = false) =>
    api.get('/openai/available-models', { params: refresh ? { refresh: '1' } : {} }),
}

// Cursor API
export const cursorAPI = {
  list: () => api.get('/cursor/accounts'),
  add: (data) => api.post('/cursor/accounts', data),
  update: (id, data) => api.put(`/cursor/accounts/${id}`, data),
  delete: (id) => api.delete(`/cursor/accounts/${id}`),
  deleteMany: (ids) => api.delete('/cursor/accounts', { data: { ids } }),
  activate: (id) => api.post(`/cursor/accounts/${id}/activate`),
  import: (accounts) => api.post('/cursor/import', accounts),
}

// Windsurf API
export const windsurfAPI = {
  list: () => api.get('/windsurf/accounts'),
  add: (data) => api.post('/windsurf/accounts', data),
  update: (id, data) => api.put(`/windsurf/accounts/${id}`, data),
  delete: (id) => api.delete(`/windsurf/accounts/${id}`),
  deleteMany: (ids) => api.delete('/windsurf/accounts', { data: { ids } }),
  activate: (id) => api.post(`/windsurf/accounts/${id}/activate`),
}

// Antigravity API
export const antigravityAPI = {
  list: () => api.get('/antigravity/accounts'),
  add: (data) => api.post('/antigravity/accounts', data),
  update: (id, data) => api.put(`/antigravity/accounts/${id}`, data),
  delete: (id) => api.delete(`/antigravity/accounts/${id}`),
  deleteMany: (ids) => api.delete('/antigravity/accounts', { data: { ids } }),
  activate: (id) => api.post(`/antigravity/accounts/${id}/activate`),
}

// Claude API
export const claudeAPI = {
  list: () => api.get('/claude/accounts'),
  add: (data) => api.post('/claude/accounts', data),
  update: (id, data) => api.put(`/claude/accounts/${id}`, data),
  delete: (id) => api.delete(`/claude/accounts/${id}`),
  deleteMany: (ids) => api.delete('/claude/accounts', { data: { ids } }),
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

export { longApi }
export default api
