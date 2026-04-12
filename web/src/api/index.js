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

// OpenAI API
export const openaiAPI = {
  list: () => api.get('/openai/accounts'),
  exportJSON: () => api.get('/openai/export'),
  refreshAll: () => api.post('/openai/accounts/refresh-all'),
  generateOAuthUrl: (data) => api.post('/openai/oauth/generate-url', data),
  getOAuthSession: (id) => api.get(`/openai/oauth/sessions/${id}`),
  cancelOAuthSession: (id) => api.delete(`/openai/oauth/sessions/${id}`),
  exchangeOAuthCode: (data) => api.post('/openai/oauth/exchange-code', data),
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

// Antigravity API
export const antigravityAPI = {
  list: () => api.get('/antigravity/accounts'),
  add: (data) => api.post('/antigravity/accounts', data),
  update: (id, data) => api.put(`/antigravity/accounts/${id}`, data),
  delete: (id) => api.delete(`/antigravity/accounts/${id}`),
  deleteMany: (ids) => api.delete('/antigravity/accounts', { data: { ids } }),
  activate: (id) => api.post(`/antigravity/accounts/${id}/activate`),
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

export const cockpitAPI = {
  listAllAccounts: () => api.get('/cockpit/accounts'),
  definitions: () => api.get('/cockpit/definitions'),
  overview: () => api.get('/cockpit/overview'),
  listPlatformAccounts: (platform) => api.get(`/cockpit/platforms/${platform}/accounts`),
  importPlatformAccounts: (platform, data) => api.post(`/cockpit/platforms/${platform}/accounts/import`, data),
  exportPlatformAccounts: (platform) => api.get(`/cockpit/platforms/${platform}/accounts/export`),
  addPlatformAccount: (platform, data) => api.post(`/cockpit/platforms/${platform}/accounts`, data),
  updatePlatformAccount: (platform, id, data) => api.put(`/cockpit/platforms/${platform}/accounts/${id}`, data),
  deletePlatformAccount: (platform, id) => api.delete(`/cockpit/platforms/${platform}/accounts/${id}`),
  deleteManyPlatformAccounts: (platform, ids) =>
    api.delete(`/cockpit/platforms/${platform}/accounts`, { data: { ids } }),
  activatePlatformAccount: (platform, id) => api.post(`/cockpit/platforms/${platform}/accounts/${id}/activate`),
  listAllInstances: () => api.get('/cockpit/instances'),
  listPlatformInstances: (platform) => api.get(`/cockpit/platforms/${platform}/instances`),
  exportPlatformInstances: (platform) => api.get(`/cockpit/platforms/${platform}/instances/export`),
  addPlatformInstance: (platform, data) => api.post(`/cockpit/platforms/${platform}/instances`, data),
  updatePlatformInstance: (platform, id, data) => api.put(`/cockpit/platforms/${platform}/instances/${id}`, data),
  deletePlatformInstance: (platform, id) => api.delete(`/cockpit/platforms/${platform}/instances/${id}`),
  updatePlatformInstanceState: (platform, id, state) =>
    api.post(`/cockpit/platforms/${platform}/instances/${id}/state`, { state }),
  listWakeupTasks: (platform) =>
    api.get('/cockpit/wakeup/tasks', { params: platform ? { platform } : {} }),
  addWakeupTask: (data) => api.post('/cockpit/wakeup/tasks', data),
  updateWakeupTask: (id, data) => api.put(`/cockpit/wakeup/tasks/${id}`, data),
  deleteWakeupTask: (id) => api.delete(`/cockpit/wakeup/tasks/${id}`),
  toggleWakeupTask: (id) => api.post(`/cockpit/wakeup/tasks/${id}/toggle`),
  getGeneralSettings: () => api.get('/cockpit/settings/general'),
  updateGeneralSettings: (data) => api.put('/cockpit/settings/general', data),
}

export { longApi }
export default api
