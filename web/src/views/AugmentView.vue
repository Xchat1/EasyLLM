<template>
  <div class="p-6">
    <!-- Header -->
    <div class="flex items-center justify-between mb-6">
      <div>
        <h1 class="text-2xl font-bold text-white">🔑 Augment Token 管理</h1>
        <p class="text-gray-400 text-sm mt-1">管理 Augment 访问令牌，共 {{ tokens.length }} 个</p>
      </div>
      <div class="flex items-center gap-2">
        <button @click="checkAllStatus" class="btn btn-secondary btn-sm" :disabled="checking">
          {{ checking ? '检测中...' : '批量检测' }}
        </button>
        <button @click="showImportModal = true" class="btn btn-secondary btn-sm">批量导入 Session</button>
        <button @click="showOAuthModal = true" class="btn btn-secondary btn-sm">OAuth 登录</button>
        <button @click="showAddModal = true" class="btn btn-primary btn-sm">+ 添加 Token</button>
      </div>
    </div>

    <!-- Stats Bar -->
    <div class="grid grid-cols-4 gap-4 mb-6">
      <div class="card p-4">
        <div class="text-2xl font-bold text-white">{{ tokens.length }}</div>
        <div class="text-sm text-gray-400">总 Token 数</div>
      </div>
      <div class="card p-4">
        <div class="text-2xl font-bold text-green-400">{{ activeCount }}</div>
        <div class="text-sm text-gray-400">活跃</div>
      </div>
      <div class="card p-4">
        <div class="text-2xl font-bold text-red-400">{{ suspendedCount }}</div>
        <div class="text-sm text-gray-400">已封禁</div>
      </div>
      <div class="card p-4">
        <div class="text-2xl font-bold text-gray-400">{{ unknownCount }}</div>
        <div class="text-sm text-gray-400">未知状态</div>
      </div>
    </div>

    <!-- Filters -->
    <div class="flex items-center gap-3 mb-4">
      <input v-model="search" placeholder="搜索邮箱或标签..." class="input max-w-xs" />
      <select v-model="filterStatus" class="input max-w-xs">
        <option value="">全部状态</option>
        <option value="ACTIVE">活跃</option>
        <option value="SUSPENDED">已封禁</option>
        <option value="INVALID_TOKEN">无效</option>
      </select>
      <div class="ml-auto flex gap-2">
        <button v-if="selectedIds.length > 0" @click="deleteSelected" class="btn btn-danger btn-sm">
          删除选中 ({{ selectedIds.length }})
        </button>
        <button v-if="selectedIds.length > 0" @click="batchRefreshSessions" class="btn btn-secondary btn-sm">
          刷新 Session ({{ selectedIds.length }})
        </button>
        <button @click="exportJSON" class="btn btn-secondary btn-sm">导出 JSON</button>
      </div>
    </div>

    <!-- Token Table -->
    <div class="card overflow-hidden">
      <table class="w-full text-sm">
        <thead class="bg-gray-800/50">
          <tr>
            <th class="p-3 text-left w-8">
              <input type="checkbox" class="accent-blue-500" @change="toggleAll" :checked="allSelected" />
            </th>
            <th class="p-3 text-left text-gray-400 font-medium">邮箱</th>
            <th class="p-3 text-left text-gray-400 font-medium">状态</th>
            <th class="p-3 text-left text-gray-400 font-medium">标签</th>
            <th class="p-3 text-left text-gray-400 font-medium">Token (前20位)</th>
            <th class="p-3 text-left text-gray-400 font-medium">Session</th>
            <th class="p-3 text-left text-gray-400 font-medium">创建时间</th>
            <th class="p-3 text-left text-gray-400 font-medium">操作</th>
          </tr>
        </thead>
        <tbody>
          <tr v-if="loading" class="text-center">
            <td colspan="8" class="p-8 text-gray-500">加载中...</td>
          </tr>
          <tr v-else-if="filteredTokens.length === 0" class="text-center">
            <td colspan="8" class="p-8 text-gray-500">暂无数据</td>
          </tr>
          <tr
            v-for="token in filteredTokens"
            :key="token.id"
            class="border-b border-gray-700/50 hover:bg-gray-800/30 transition-colors"
          >
            <td class="p-3">
              <input type="checkbox" class="accent-blue-500"
                :checked="selectedIds.includes(token.id)"
                @change="toggleSelect(token.id)" />
            </td>
            <td class="p-3 text-gray-100">{{ token.email_note || '-' }}</td>
            <td class="p-3">
              <span :class="statusBadgeClass(token.ban_status)">
                {{ statusText(token.ban_status) }}
              </span>
            </td>
            <td class="p-3">
              <span v-if="token.tag_name"
                class="px-2 py-0.5 rounded text-xs"
                :style="{ backgroundColor: token.tag_color || '#4B5563', color: '#fff' }">
                {{ token.tag_name }}
              </span>
              <span v-else class="text-gray-600">-</span>
            </td>
            <td class="p-3 font-mono text-xs text-gray-400">{{ token.access_token?.slice(0, 20) }}...</td>
            <td class="p-3">
              <span v-if="token.auth_session" class="badge badge-blue">有 Session</span>
              <span v-else class="text-gray-600">-</span>
            </td>
            <td class="p-3 text-gray-400 text-xs">{{ formatDate(token.created_at) }}</td>
            <td class="p-3">
              <div class="flex items-center gap-1">
                <button @click="checkStatus(token)" class="btn btn-secondary btn-xs" title="检测状态">🔍</button>
                <button @click="getCreditInfo(token)" class="btn btn-secondary btn-xs" title="查看额度">💰</button>
                <button v-if="token.auth_session" @click="refreshSession(token)" class="btn btn-secondary btn-xs" title="刷新Session">🔄</button>
                <button @click="editToken(token)" class="btn btn-secondary btn-xs" title="编辑">✏️</button>
                <button @click="deleteToken(token)" class="btn btn-danger btn-xs" title="删除">🗑️</button>
              </div>
            </td>
          </tr>
        </tbody>
      </table>
    </div>

    <!-- Import Session Modal -->
    <div v-if="showImportModal" class="modal-overlay" @click.self="showImportModal = false">
      <div class="modal-content max-w-2xl">
        <div class="modal-header">
          <h3 class="font-semibold text-white">批量导入 Session</h3>
          <button @click="showImportModal = false" class="text-gray-400 hover:text-white">✕</button>
        </div>
        <div class="modal-body space-y-4">
          <div>
            <label class="label">Sessions (每行一个)</label>
            <textarea v-model="importSessions" class="input h-40 resize-none font-mono text-xs"
              placeholder="粘贴 session 值，每行一个..."></textarea>
          </div>
          <div v-if="importResults" class="bg-gray-900 rounded-lg p-3 text-sm">
            <p class="text-green-400">✅ 成功: {{ importResults.successful }} / {{ importResults.total }}</p>
            <p class="text-red-400">❌ 失败: {{ importResults.failed }}</p>
          </div>

          <!-- API Import Reference -->
          <div class="border border-gray-700 rounded-lg overflow-hidden">
            <button @click="showAPIRef = !showAPIRef"
              class="w-full flex items-center justify-between px-3 py-2 text-xs text-gray-400 hover:text-gray-200 hover:bg-gray-800 transition-colors">
              <span class="flex items-center gap-1.5">
                <svg class="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M8 9l3 3-3 3m5 0h3M5 20h14a2 2 0 002-2V6a2 2 0 00-2-2H5a2 2 0 00-2 2v12a2 2 0 002 2z"/>
                </svg>
                API 方式导入（程序化接入参考）
              </span>
              <svg class="w-3 h-3 transition-transform" :class="showAPIRef ? 'rotate-180' : ''" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 9l-7 7-7-7"/>
              </svg>
            </button>
            <div v-if="showAPIRef" class="px-3 pb-3 space-y-3 bg-gray-900/40">
              <div v-for="ref in importAPIEndpoints" :key="ref.path" class="space-y-1.5">
                <div class="flex items-center gap-2 pt-2">
                  <span class="text-xs font-bold bg-orange-500/20 text-orange-400 px-1.5 py-0.5 rounded font-mono">POST</span>
                  <code class="text-blue-300 text-xs font-mono">{{ augmentBaseURL + ref.path }}</code>
                  <button @click="copyAugmentText(augmentBaseURL + ref.path)"
                    class="ml-auto text-xs text-gray-600 hover:text-white bg-gray-700 hover:bg-gray-600 px-2 py-0.5 rounded">复制</button>
                </div>
                <p class="text-xs text-gray-500">{{ ref.desc }}</p>
                <div class="bg-gray-950 rounded p-2 relative group">
                  <pre class="text-xs text-gray-300 overflow-auto">{{ ref.example }}</pre>
                  <button @click="copyAugmentText(ref.example)"
                    class="absolute top-1.5 right-1.5 text-xs text-gray-600 hover:text-white bg-gray-800 hover:bg-gray-700 px-1.5 py-0.5 rounded opacity-0 group-hover:opacity-100 transition-opacity">复制</button>
                </div>
              </div>
            </div>
          </div>
        </div>
        <div class="modal-footer">
          <button @click="showImportModal = false" class="btn btn-secondary">取消</button>
          <button @click="doImportSessions" :disabled="importing" class="btn btn-primary">
            {{ importing ? '导入中...' : '开始导入' }}
          </button>
        </div>
      </div>
    </div>

    <!-- Add/Edit Token Modal -->
    <div v-if="showAddModal" class="modal-overlay" @click.self="showAddModal = false">
      <div class="modal-content">
        <div class="modal-header">
          <h3 class="font-semibold text-white">{{ editingToken ? '编辑 Token' : '添加 Token' }}</h3>
          <button @click="showAddModal = false" class="text-gray-400 hover:text-white">✕</button>
        </div>
        <div class="modal-body space-y-3">
          <div>
            <label class="label">Tenant URL *</label>
            <input v-model="tokenForm.tenant_url" class="input" placeholder="https://xxx.augmentcode.com/" />
          </div>
          <div>
            <label class="label">Access Token *</label>
            <textarea v-model="tokenForm.access_token" class="input h-20 resize-none font-mono text-xs" placeholder="eyJ..."></textarea>
          </div>
          <div>
            <label class="label">Email (备注)</label>
            <input v-model="tokenForm.email_note" class="input" placeholder="user@example.com" />
          </div>
          <div>
            <label class="label">Auth Session (可选)</label>
            <input v-model="tokenForm.auth_session" class="input" placeholder="session cookie 值" />
          </div>
          <div class="grid grid-cols-2 gap-3">
            <div>
              <label class="label">标签名</label>
              <input v-model="tokenForm.tag_name" class="input" placeholder="标签名" />
            </div>
            <div>
              <label class="label">标签颜色</label>
              <input v-model="tokenForm.tag_color" type="color" class="input h-10 p-1" />
            </div>
          </div>
        </div>
        <div class="modal-footer">
          <button @click="showAddModal = false" class="btn btn-secondary">取消</button>
          <button @click="saveToken" class="btn btn-primary">保存</button>
        </div>
      </div>
    </div>

    <!-- OAuth Modal -->
    <div v-if="showOAuthModal" class="modal-overlay" @click.self="showOAuthModal = false">
      <div class="modal-content">
        <div class="modal-header">
          <h3 class="font-semibold text-white">OAuth 登录</h3>
          <button @click="showOAuthModal = false" class="text-gray-400 hover:text-white">✕</button>
        </div>
        <div class="modal-body space-y-4">
          <div v-if="!oauthURL">
            <p class="text-gray-400 text-sm">点击下方按钮生成授权链接，在浏览器中完成登录后，将回调 URL 中的 code 参数粘贴到下方。</p>
            <button @click="startOAuth" class="btn btn-primary w-full mt-3">生成授权链接</button>
          </div>
          <div v-else class="space-y-3">
            <div class="bg-gray-900 rounded p-3">
              <p class="text-xs text-gray-400 mb-1">授权链接（在浏览器中打开）：</p>
              <a :href="oauthURL" target="_blank" class="text-blue-400 text-xs break-all hover:underline">{{ oauthURL }}</a>
              <button @click="copyText(oauthURL)" class="btn btn-secondary btn-xs mt-2">复制链接</button>
            </div>
            <div>
              <label class="label">回调中的 Code JSON</label>
              <textarea v-model="oauthCode" class="input h-24 resize-none font-mono text-xs"
                placeholder='{"code":"xxx","state":"xxx","tenant_url":"https://xxx/"}'></textarea>
            </div>
            <button @click="completeOAuth" :disabled="!oauthCode" class="btn btn-primary w-full">完成登录</button>
          </div>
        </div>
      </div>
    </div>

    <!-- Credit Info Modal -->
    <div v-if="creditInfo" class="modal-overlay" @click.self="creditInfo = null">
      <div class="modal-content">
        <div class="modal-header">
          <h3 class="font-semibold text-white">额度信息</h3>
          <button @click="creditInfo = null" class="text-gray-400 hover:text-white">✕</button>
        </div>
        <div class="modal-body">
          <pre class="text-xs text-gray-300 bg-gray-900 rounded p-3 overflow-auto max-h-96">{{ JSON.stringify(creditInfo, null, 2) }}</pre>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, computed, onMounted, inject } from 'vue'
import { augmentAPI } from '@/api'

const notify = inject('notify')
const tokens = ref([])
const loading = ref(false)
const checking = ref(false)
const importing = ref(false)

const showAPIRef = ref(false)
const augmentBaseURL = computed(() => `${window.location.protocol}//${window.location.hostname}:${window.location.port || 8022}`)
const importAPIEndpoints = [
  {
    path: '/api/import/session',
    desc: '导入单个 Augment session（兼容原 ATM 接口）',
    example: `curl -X POST ${window.location.protocol}//${window.location.hostname}:${window.location.port || 8022}/api/import/session \\
  -H "Content-Type: application/json" \\
  -d '{"session":"your_session_here"}'`
  },
  {
    path: '/api/import/sessions',
    desc: '批量导入多个 Augment sessions（兼容原 ATM 接口）',
    example: `curl -X POST ${window.location.protocol}//${window.location.hostname}:${window.location.port || 8022}/api/import/sessions \\
  -H "Content-Type: application/json" \\
  -d '{"sessions":["session1","session2"]}'`
  }
]

function copyAugmentText(text) {
  navigator.clipboard.writeText(text).then(() => notify('已复制', 'success'))
}
const search = ref('')
const filterStatus = ref('')
const selectedIds = ref([])
const showImportModal = ref(false)
const showAddModal = ref(false)
const showOAuthModal = ref(false)
const importSessions = ref('')
const importResults = ref(null)
const editingToken = ref(null)
const tokenForm = ref({})
const oauthURL = ref('')
const oauthCode = ref('')
const creditInfo = ref(null)

const filteredTokens = computed(() => {
  let list = tokens.value
  if (search.value) {
    const q = search.value.toLowerCase()
    list = list.filter(t =>
      (t.email_note || '').toLowerCase().includes(q) ||
      (t.tag_name || '').toLowerCase().includes(q)
    )
  }
  if (filterStatus.value) {
    list = list.filter(t => {
      const status = getBanStatus(t.ban_status)
      return status === filterStatus.value
    })
  }
  return list
})

const activeCount = computed(() => tokens.value.filter(t => getBanStatus(t.ban_status) === 'ACTIVE').length)
const suspendedCount = computed(() => tokens.value.filter(t => ['SUSPENDED', 'INVALID_TOKEN'].includes(getBanStatus(t.ban_status))).length)
const unknownCount = computed(() => tokens.value.filter(t => !t.ban_status).length)
const allSelected = computed(() => selectedIds.value.length === filteredTokens.value.length && filteredTokens.value.length > 0)

function getBanStatus(banStatus) {
  if (!banStatus) return ''
  if (typeof banStatus === 'string') return banStatus
  if (typeof banStatus === 'object') return banStatus.status || ''
  return ''
}

function statusText(banStatus) {
  const s = getBanStatus(banStatus)
  const map = { ACTIVE: '活跃', SUSPENDED: '已封禁', INVALID_TOKEN: '无效', ERROR: '错误' }
  return map[s] || (s || '未知')
}

function statusBadgeClass(banStatus) {
  const s = getBanStatus(banStatus)
  if (s === 'ACTIVE') return 'badge badge-green'
  if (s === 'SUSPENDED') return 'badge badge-red'
  if (s === 'INVALID_TOKEN') return 'badge badge-yellow'
  if (s === 'ERROR') return 'badge badge-red'
  return 'badge badge-gray'
}

function formatDate(d) {
  if (!d) return '-'
  return new Date(d).toLocaleDateString('zh-CN', { year: 'numeric', month: '2-digit', day: '2-digit' })
}

function toggleAll(e) {
  if (e.target.checked) {
    selectedIds.value = filteredTokens.value.map(t => t.id)
  } else {
    selectedIds.value = []
  }
}

function toggleSelect(id) {
  const idx = selectedIds.value.indexOf(id)
  if (idx === -1) selectedIds.value.push(id)
  else selectedIds.value.splice(idx, 1)
}

async function loadTokens() {
  loading.value = true
  try {
    tokens.value = await augmentAPI.list()
  } catch (e) {
    notify(e.message, 'error')
  } finally {
    loading.value = false
  }
}

async function checkStatus(token) {
  try {
    const result = await augmentAPI.checkStatus(token.id)
    const idx = tokens.value.findIndex(t => t.id === token.id)
    if (idx !== -1) {
      tokens.value[idx].ban_status = result
    }
    notify(`状态: ${result.status}`, 'info')
  } catch (e) {
    notify(e.message, 'error')
  }
}

async function checkAllStatus() {
  checking.value = true
  try {
    const results = await augmentAPI.checkAll()
    for (const r of results) {
      const idx = tokens.value.findIndex(t => t.id === r.token_id)
      if (idx !== -1) tokens.value[idx].ban_status = r.status
    }
    notify('批量检测完成', 'success')
  } catch (e) {
    notify(e.message, 'error')
  } finally {
    checking.value = false
  }
}

async function getCreditInfo(token) {
  try {
    const info = await augmentAPI.getCreditInfo(token.id)
    creditInfo.value = info
  } catch (e) {
    notify(e.message, 'error')
  }
}

async function refreshSession(token) {
  try {
    await augmentAPI.refreshSession(token.id)
    notify('Session 刷新成功', 'success')
  } catch (e) {
    notify(e.message, 'error')
  }
}

async function batchRefreshSessions() {
  try {
    const result = await augmentAPI.batchRefreshSessions(selectedIds.value)
    notify(`刷新完成`, 'success')
  } catch (e) {
    notify(e.message, 'error')
  }
}

function editToken(token) {
  editingToken.value = token
  tokenForm.value = { ...token }
  showAddModal.value = true
}

async function saveToken() {
  try {
    if (editingToken.value) {
      await augmentAPI.update(editingToken.value.id, tokenForm.value)
      notify('更新成功', 'success')
    } else {
      await augmentAPI.add(tokenForm.value)
      notify('添加成功', 'success')
    }
    showAddModal.value = false
    editingToken.value = null
    tokenForm.value = {}
    await loadTokens()
  } catch (e) {
    notify(e.message, 'error')
  }
}

async function deleteToken(token) {
  if (!confirm(`确认删除 ${token.email_note || token.id}?`)) return
  try {
    await augmentAPI.delete(token.id)
    tokens.value = tokens.value.filter(t => t.id !== token.id)
    notify('删除成功', 'success')
  } catch (e) {
    notify(e.message, 'error')
  }
}

async function deleteSelected() {
  if (!confirm(`确认删除选中的 ${selectedIds.value.length} 个 Token?`)) return
  try {
    await augmentAPI.deleteMany(selectedIds.value)
    tokens.value = tokens.value.filter(t => !selectedIds.value.includes(t.id))
    selectedIds.value = []
    notify('删除成功', 'success')
  } catch (e) {
    notify(e.message, 'error')
  }
}

async function doImportSessions() {
  const sessions = importSessions.value.split('\n')
    .map(s => s.trim())
    .filter(s => s.length >= 10)

  if (sessions.length === 0) {
    notify('没有有效的 Session', 'error')
    return
  }

  importing.value = true
  importResults.value = null
  try {
    const result = await augmentAPI.importSessions(sessions)
    importResults.value = result
    notify(`成功导入 ${result.successful}/${result.total}`, result.successful > 0 ? 'success' : 'error')
    await loadTokens()
  } catch (e) {
    notify(e.message, 'error')
  } finally {
    importing.value = false
  }
}

async function startOAuth() {
  try {
    const result = await augmentAPI.startOAuth()
    oauthURL.value = result.auth_url
  } catch (e) {
    notify(e.message, 'error')
  }
}

async function completeOAuth() {
  try {
    await augmentAPI.completeOAuth(oauthCode.value)
    showOAuthModal.value = false
    oauthURL.value = ''
    oauthCode.value = ''
    notify('OAuth 登录成功', 'success')
    await loadTokens()
  } catch (e) {
    notify(e.message, 'error')
  }
}

async function exportJSON() {
  try {
    const data = await augmentAPI.exportJSON()
    const blob = new Blob([JSON.stringify(data, null, 2)], { type: 'application/json' })
    const url = URL.createObjectURL(blob)
    const a = document.createElement('a')
    a.href = url
    a.download = 'augment-tokens.json'
    a.click()
    URL.revokeObjectURL(url)
  } catch (e) {
    notify(e.message, 'error')
  }
}

function copyText(text) {
  navigator.clipboard.writeText(text).then(() => notify('已复制', 'success'))
}

onMounted(loadTokens)
</script>
