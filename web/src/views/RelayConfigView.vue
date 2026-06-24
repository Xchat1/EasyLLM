<template>
  <div class="p-6 space-y-6">
    <section class="rounded-3xl border border-gray-800 bg-gradient-to-br from-slate-200/10 via-gray-200/5 to-gray-950 p-6">
      <h1 class="text-3xl font-semibold text-white">Relay 配置</h1>
      <p class="mt-2 max-w-3xl text-sm leading-6 text-gray-300">
        配置 Relay 模式，让 Codex 客户端 / Codex CLI 通过 EasyLLM 对接任意 OpenAI 兼容的上游提供商。支持添加多个渠道，自动轮询分流。
      </p>
    </section>

    <div v-if="loading" class="card p-5 text-center text-gray-400">
      加载中...
    </div>

    <div v-else class="space-y-6">

      <!-- ── 上游渠道 ─────────────────────────────────────────── -->
      <section class="card p-5 space-y-5">
        <div class="flex items-center justify-between">
          <div>
            <h2 class="text-lg font-semibold text-white">上游渠道</h2>
            <p class="mt-1 text-sm text-gray-500">添加一个或多个 OpenAI 兼容上游，EasyLLM 自动按策略轮询分流。</p>
          </div>
          <div class="flex items-center gap-3">
            <select v-model="upstreamStrategy" class="input input-sm w-36">
              <option value="round_robin">轮询（Round Robin）</option>
            </select>
            <button class="btn btn-sm btn-primary" @click="openAddUpstream">+ 添加渠道</button>
          </div>
        </div>

        <!-- 渠道列表 -->
        <div v-if="upstreams.length === 0" class="text-center py-6 text-gray-500 text-sm border border-dashed border-gray-700 rounded-xl">
          暂无上游渠道，点击「添加渠道」开始配置
        </div>
        <div v-else class="space-y-3">
          <div
            v-for="(u, idx) in upstreams"
            :key="u.id"
            class="upstream-card"
            :class="{ 'upstream-card--disabled': !u.enabled }"
          >
            <div class="flex items-center gap-3 min-w-0">
              <label class="relative inline-flex items-center cursor-pointer shrink-0" :title="u.enabled ? '禁用' : '启用'">
                <input type="checkbox" v-model="upstreams[idx].enabled" class="sr-only peer" />
                <div class="upstream-toggle peer-checked:bg-blue-500"></div>
              </label>
              <div class="min-w-0 flex-1">
                <div class="flex items-center gap-2 flex-wrap">
                  <span class="text-sm font-medium text-white truncate">{{ u.name || '未命名渠道' }}</span>
                  <span class="text-[10px] px-1.5 py-0.5 rounded bg-gray-800 text-gray-400 font-mono truncate max-w-[240px]">{{ u.upstream_url }}</span>
                </div>
                <div class="text-xs text-gray-500 mt-0.5">{{ u.api_key ? '已配置 API Key' : '⚠️ 未配置 API Key' }}</div>
              </div>
            </div>
            <div class="flex items-center gap-2 shrink-0">
              <button class="btn btn-sm btn-secondary" @click="editUpstream(idx)">编辑</button>
              <button class="btn btn-sm btn-danger" @click="removeUpstream(idx)">删除</button>
            </div>
          </div>
        </div>

        <!-- 内联添加/编辑表单 -->
        <div v-if="editForm.open" class="upstream-form space-y-4">
          <div class="flex items-center justify-between mb-1">
            <h3 class="text-sm font-semibold text-white">{{ editForm.isNew ? '添加渠道' : '编辑渠道' }}</h3>
            <button class="text-gray-500 hover:text-white text-lg leading-none" @click="closeEditForm">✕</button>
          </div>

          <div class="grid gap-4 md:grid-cols-2">
            <div>
              <label class="label">渠道名称</label>
              <input v-model="editForm.data.name" type="text" class="input" placeholder="如：DeepSeek 主力" />
            </div>
            <div>
              <label class="label">提供商快选</label>
              <select v-model="editForm.selectedProvider" class="input" @change="applyProvider">
                <option value="custom">自定义</option>
                <option value="openai">OpenAI</option>
                <option value="deepseek">DeepSeek</option>
                <option value="kimi">Kimi (Moonshot)</option>
                <option value="qwen">Qwen (DashScope)</option>
                <option value="mistral">Mistral</option>
                <option value="groq">Groq</option>
                <option value="xai">xAI (Grok)</option>
                <option value="openrouter">OpenRouter</option>
                <option value="codestral">Codestral</option>
                <option value="xiaomi">Xiaomi (MiMo)</option>
              </select>
            </div>
          </div>

          <div class="grid gap-4 md:grid-cols-2">
            <div>
              <label class="label">上游 Base URL</label>
              <input v-model="editForm.data.upstream_url" type="text" class="input" placeholder="https://api.openai.com/v1" />
            </div>
            <div>
              <label class="label">API Key</label>
              <input v-model="editForm.data.api_key" type="password" class="input" placeholder="sk-..." />
            </div>
          </div>

          <div class="grid gap-4 md:grid-cols-2">
            <div>
              <label class="label">认证请求头 <span class="text-gray-600 font-normal">（默认 Authorization）</span></label>
              <input v-model="editForm.data.auth_header" type="text" class="input" placeholder="Authorization" />
            </div>
            <div>
              <label class="label">认证值前缀 <span class="text-gray-600 font-normal">（默认 Bearer ）</span></label>
              <input v-model="editForm.data.auth_value_prefix" type="text" class="input" placeholder="Bearer " />
            </div>
          </div>

          <div class="flex items-center gap-2">
            <input id="upstream-enabled" v-model="editForm.data.enabled" type="checkbox" class="h-4 w-4 rounded border-gray-600 bg-gray-800 text-blue-500" />
            <label for="upstream-enabled" class="text-sm text-gray-300 cursor-pointer">启用此渠道</label>
          </div>

          <div class="flex gap-2 pt-1">
            <button class="btn btn-primary btn-sm" @click="saveEditForm">{{ editForm.isNew ? '添加' : '保存' }}</button>
            <button class="btn btn-secondary btn-sm" @click="closeEditForm">取消</button>
          </div>
        </div>
      </section>

      <!-- ── 模型映射（全局） ────────────────────────────────── -->
      <section class="card p-5 space-y-5">
        <div>
          <h2 class="text-lg font-semibold text-white">模型映射 <span class="text-sm font-normal text-gray-500">（全局，适用于所有渠道）</span></h2>
          <p class="mt-1 text-sm text-gray-500">将 Codex 模型名称映射为上游模型名称。</p>
        </div>

        <div>
          <label class="label" for="relay-default-model">默认模型</label>
          <input
            id="relay-default-model"
            v-model="globalConfig.default_model"
            type="text"
            class="input"
            placeholder="gpt-5.5"
          />
          <p class="mt-1 text-xs text-gray-500">未在映射表中的 Codex 模型名将自动使用此默认上游模型</p>
        </div>

        <div>
          <label class="label" for="relay-model-map">模型映射 (JSON)</label>
          <textarea
            id="relay-model-map"
            v-model="globalConfig.model_map_json"
            class="input font-mono text-sm"
            rows="4"
            placeholder='{"codex-model": "upstream-model", "gpt-5.4": "deepseek-v4-pro"}'
          />
          <p class="mt-1 text-xs text-gray-500">JSON 格式，或逗号分隔的 key:value 对</p>
        </div>
      </section>

      <!-- ── 工具过滤 ─────────────────────────────────────────── -->
      <section class="card p-5 space-y-5">
        <div>
          <h2 class="text-lg font-semibold text-white">工具过滤</h2>
          <p class="mt-1 text-sm text-gray-500">拒绝列表中的工具将不会转发到上游。</p>
        </div>
        <div>
          <label class="label" for="relay-tool-denylist">工具拒绝列表</label>
          <input
            id="relay-tool-denylist"
            v-model="globalConfig.tool_denylist_str"
            type="text"
            class="input"
            placeholder="web_search,image_generation"
          />
          <p class="mt-1 text-xs text-gray-500">逗号分隔的工具名称列表</p>
        </div>
      </section>

      <!-- ── 会话管理 ─────────────────────────────────────────── -->
      <section class="card p-5 space-y-5">
        <div>
          <h2 class="text-lg font-semibold text-white">会话管理</h2>
          <p class="mt-1 text-sm text-gray-500">配置会话历史存储的限制。</p>
        </div>

        <div class="grid gap-4 md:grid-cols-3">
          <div>
            <label class="label" for="relay-max-sessions">最大会话数</label>
            <input id="relay-max-sessions" v-model.number="globalConfig.max_sessions" type="number" class="input" min="1" max="10000" />
          </div>
          <div>
            <label class="label" for="relay-max-session-bytes">最大历史字节</label>
            <input id="relay-max-session-bytes" v-model.number="globalConfig.max_session_bytes" type="number" class="input" min="1048576" />
            <p class="mt-1 text-xs text-gray-500">默认 536870912 (512MB)</p>
          </div>
          <div>
            <label class="label" for="relay-session-ttl">会话 TTL (小时)</label>
            <input id="relay-session-ttl" v-model.number="globalConfig.session_ttl_hours" type="number" class="input" min="1" max="8760" />
            <p class="mt-1 text-xs text-gray-500">默认 168 小时 (7 天)</p>
          </div>
        </div>
      </section>

      <!-- ── 会话统计 ─────────────────────────────────────────── -->
      <section class="card p-5 space-y-5">
        <div class="flex items-center justify-between">
          <div>
            <h2 class="text-lg font-semibold text-white">会话统计</h2>
            <p class="mt-1 text-sm text-gray-500">当前会话历史存储状态。</p>
          </div>
          <button class="btn btn-sm btn-secondary" @click="loadStats">刷新</button>
        </div>

        <div v-if="stats" class="grid gap-3 text-sm text-gray-400 md:grid-cols-4">
          <div class="settings-stat"><dt>会话数量</dt><dd>{{ stats.session_count || 0 }}</dd></div>
          <div class="settings-stat"><dt>Reasoning 条目</dt><dd>{{ stats.reasoning_count || 0 }}</dd></div>
          <div class="settings-stat"><dt>Turn 条目</dt><dd>{{ stats.turn_count || 0 }}</dd></div>
          <div class="settings-stat"><dt>存储字节</dt><dd>{{ formatBytes(stats.stored_bytes) }}</dd></div>
        </div>

        <div>
          <button class="btn btn-sm btn-danger" @click="clearSessions">清空会话历史</button>
        </div>
      </section>

      <!-- ── 服务配置 ─────────────────────────────────────────── -->
      <section class="card p-5 space-y-5">
        <div>
          <h2 class="text-lg font-semibold text-white">服务配置</h2>
          <p class="mt-1 text-sm text-gray-500">启动 Relay 服务并注入本机 Codex 配置。</p>
        </div>

        <div class="bg-gray-800 rounded-xl p-4 flex items-center justify-between">
          <div class="min-w-0 flex-1">
            <div class="flex items-center gap-2">
              <div class="text-sm font-medium text-white">Codex Relay 注入</div>
              <span
                class="text-[10px] px-2 py-0.5 rounded-full font-medium"
                :class="relayInjected ? 'bg-green-500/20 text-green-300' : 'bg-gray-700 text-gray-500'"
              >
                {{ relayInjected ? '已注入' : '未注入' }}
              </span>
            </div>
            <div class="text-xs text-gray-400 mt-0.5">
              自动写入本机 <code class="text-blue-300">~/.codex/auth.json</code> 和 <code class="text-blue-300">config.toml</code>，Codex 直接走 EasyLLM 分流。
            </div>
          </div>
          <button
            @click="injectCodexConfig"
            :disabled="injecting || upstreams.filter(u => u.enabled).length === 0"
            class="btn btn-sm btn-primary shrink-0"
            :title="upstreams.filter(u => u.enabled).length === 0 ? '请先添加并启用至少一个渠道' : '启动 Relay 并注入本机 Codex 配置'"
          >
            {{ injecting ? '注入中...' : '启动并注入 Codex' }}
          </button>
        </div>

        <div class="grid md:grid-cols-2 gap-2 text-xs">
          <div class="flex items-center justify-between bg-gray-900/60 rounded-lg px-3 py-2 min-w-0">
            <span class="text-gray-500 shrink-0">Relay URL</span>
            <code class="text-blue-300 font-mono truncate mx-3">{{ relayServiceURL }}</code>
            <button @click="copyText(relayServiceURL)" class="text-gray-500 hover:text-white shrink-0">复制</button>
          </div>
          <div class="flex items-center justify-between bg-gray-900/60 rounded-lg px-3 py-2 min-w-0">
            <span class="text-gray-500 shrink-0">Wire API</span>
            <code class="text-emerald-300 font-mono truncate mx-3">responses</code>
            <span class="text-gray-600 shrink-0">model_provider=relay</span>
          </div>
        </div>
      </section>

      <!-- ── 实时日志 ─────────────────────────────────────────── -->
      <section class="card p-5 space-y-4">
        <div class="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
          <div>
            <div class="flex items-center gap-2">
              <h2 class="text-lg font-semibold text-white">实时日志</h2>
              <span
                class="text-[10px] px-2 py-0.5 rounded-full font-medium"
                :class="logConnected ? 'bg-green-500/20 text-green-300' : 'bg-gray-700 text-gray-500'"
              >
                {{ logConnected ? '已连接' : '未连接' }}
              </span>
            </div>
            <p class="mt-1 text-sm text-gray-500">Relay 请求、上游响应与 Token 消耗的实时记录。</p>
          </div>
          <div class="flex flex-wrap gap-2">
            <button class="btn btn-sm btn-secondary" @click="toggleLogStream">{{ logConnected ? '暂停' : '连接' }}</button>
            <button class="btn btn-sm btn-secondary" @click="loadLogs">刷新</button>
            <button class="btn btn-sm btn-danger" @click="clearLogs">清空</button>
          </div>
        </div>

        <div ref="logPanelRef" class="relay-log-panel">
          <div v-if="logs.length === 0" class="relay-log-empty">暂无日志，在 Codex 中发起请求后将在此显示。</div>
          <div
            v-for="entry in logs"
            :key="entry.id"
            class="relay-log-line"
            :class="`relay-log-line--${entry.level || 'info'}`"
          >
            <span class="relay-log-time">{{ formatLogTime(entry.timestamp) }}</span>
            <span class="relay-log-level">{{ (entry.level || 'info').toUpperCase() }}</span>
            <span v-if="entry.model" class="relay-log-model">{{ entry.model }}</span>
            <span class="relay-log-message">{{ entry.message }}</span>
          </div>
        </div>
      </section>

      <!-- ── 保存按钮 ─────────────────────────────────────────── -->
      <section class="card p-5">
        <div class="flex items-center justify-between">
          <div>
            <p v-if="saveMessage" class="text-sm" :class="saveMessageType === 'success' ? 'text-green-400' : 'text-red-400'">
              {{ saveMessage }}
            </p>
          </div>
          <button class="btn btn-primary" @click="saveConfig" :disabled="saving">
            {{ saving ? '保存中...' : '保存配置' }}
          </button>
        </div>
      </section>
    </div>
  </div>
</template>

<script setup>
import { ref, reactive, onMounted, onUnmounted, nextTick } from 'vue'
import { relayAPI } from '@/api'
import { localRelayServiceURL } from '@/lib/runtime'

// ── State ──────────────────────────────────────────────────────
const loading = ref(false)
const saving = ref(false)
const saveMessage = ref('')
const saveMessageType = ref('success')
const stats = ref(null)
const logs = ref([])
const logConnected = ref(false)
const logPanelRef = ref(null)
const autoReconnect = ref(true)
let logEventSource = null
let logReconnectTimer = null
const MAX_LOG_LINES = 500

const relayInjected = ref(false)
const injecting = ref(false)
const relayServiceURL = ref(localRelayServiceURL())

// Multi-upstream
const upstreams = ref([])
const upstreamStrategy = ref('round_robin')

// Global options
const globalConfig = ref({
  default_model: '',
  model_map_json: '',
  tool_denylist_str: '',
  max_sessions: 256,
  max_session_bytes: 536870912,
  session_ttl_hours: 168,
})

// ── Upstream edit form ──────────────────────────────────────────
const editForm = reactive({
  open: false,
  isNew: true,
  editIdx: -1,
  selectedProvider: 'custom',
  data: emptyUpstream(),
})

function emptyUpstream() {
  return {
    id: '',
    name: '',
    enabled: true,
    upstream_url: '',
    api_key: '',
    auth_header: '',
    auth_value_prefix: '',
  }
}

function openAddUpstream() {
  editForm.open = true
  editForm.isNew = true
  editForm.editIdx = -1
  editForm.selectedProvider = 'custom'
  editForm.data = emptyUpstream()
}

function editUpstream(idx) {
  editForm.open = true
  editForm.isNew = false
  editForm.editIdx = idx
  editForm.selectedProvider = 'custom'
  editForm.data = { ...upstreams.value[idx] }
  // Auto-detect provider
  for (const [key, cfg] of Object.entries(providerConfigs)) {
    if (cfg.base_url === editForm.data.upstream_url) {
      editForm.selectedProvider = key
      break
    }
  }
}

function closeEditForm() {
  editForm.open = false
}

function saveEditForm() {
  if (!editForm.data.upstream_url) {
    return
  }
  const entry = { ...editForm.data }
  if (!entry.id) {
    entry.id = Date.now().toString(36) + Math.random().toString(36).slice(2, 6)
  }
  if (!entry.name) {
    entry.name = entry.upstream_url.replace(/^https?:\/\//, '').split('/')[0]
  }
  if (editForm.isNew) {
    upstreams.value.push(entry)
  } else {
    upstreams.value[editForm.editIdx] = entry
  }
  closeEditForm()
}

function removeUpstream(idx) {
  upstreams.value.splice(idx, 1)
}

// Provider quick-fill
const providerConfigs = {
  openai:     { base_url: 'https://api.openai.com/v1',                         auth_header: '', auth_value_prefix: '' },
  deepseek:   { base_url: 'https://api.deepseek.com/v1',                        auth_header: '', auth_value_prefix: '' },
  kimi:       { base_url: 'https://api.moonshot.cn/v1',                         auth_header: '', auth_value_prefix: '' },
  qwen:       { base_url: 'https://dashscope.aliyuncs.com/compatible-mode/v1',  auth_header: '', auth_value_prefix: '' },
  mistral:    { base_url: 'https://api.mistral.ai/v1',                          auth_header: '', auth_value_prefix: '' },
  groq:       { base_url: 'https://api.groq.com/openai/v1',                     auth_header: '', auth_value_prefix: '' },
  xai:        { base_url: 'https://api.x.ai/v1',                                auth_header: '', auth_value_prefix: '' },
  openrouter: { base_url: 'https://openrouter.ai/api/v1',                       auth_header: '', auth_value_prefix: '' },
  codestral:  { base_url: 'https://api.mistral.ai/v1',                          auth_header: '', auth_value_prefix: '' },
  xiaomi:     { base_url: 'https://token-plan-cn.xiaomimimo.com/v1',            auth_header: 'api-key', auth_value_prefix: '' },
}

function applyProvider() {
  if (editForm.selectedProvider === 'custom') return
  const cfg = providerConfigs[editForm.selectedProvider]
  if (!cfg) return
  editForm.data.upstream_url = cfg.base_url
  editForm.data.auth_header = cfg.auth_header || ''
  editForm.data.auth_value_prefix = cfg.auth_value_prefix || ''
}

// ── Load / Save ─────────────────────────────────────────────────
async function loadConfig() {
  loading.value = true
  try {
    const data = await relayAPI.getConfig()

    // Multi-upstream
    if (Array.isArray(data.upstreams) && data.upstreams.length > 0) {
      upstreams.value = data.upstreams
    } else if (data.upstream_url) {
      // Migrate legacy single-upstream
      upstreams.value = [{
        id: 'default',
        name: '默认',
        enabled: true,
        upstream_url: data.upstream_url,
        api_key: data.api_key || '',
        auth_header: data.auth_header || '',
        auth_value_prefix: data.auth_value_prefix || '',
      }]
    } else {
      upstreams.value = []
    }
    upstreamStrategy.value = data.upstream_strategy || 'round_robin'

    // Global options
    globalConfig.value = {
      default_model: data.default_model || '',
      model_map_json: data.model_map_json || '',
      tool_denylist_str: data.tool_denylist_str || '',
      max_sessions: data.max_sessions || 256,
      max_session_bytes: data.max_session_bytes || 536870912,
      session_ttl_hours: data.session_ttl_hours || 168,
    }

    relayServiceURL.value = data.relay_url || localRelayServiceURL()
    relayInjected.value = !!data.codex_injected

    await loadStats()
  } catch (err) {
    console.error('Failed to load relay config:', err)
    relayServiceURL.value = localRelayServiceURL()
  } finally {
    loading.value = false
  }
}

async function saveConfig() {
  saving.value = true
  saveMessage.value = ''
  try {
    await relayAPI.updateConfig({
      upstreams: upstreams.value,
      upstream_strategy: upstreamStrategy.value,
      ...globalConfig.value,
    })
    saveMessage.value = '配置已保存'
    saveMessageType.value = 'success'
    await loadStats()
  } catch (err) {
    saveMessage.value = '保存失败: ' + (err.message || '未知错误')
    saveMessageType.value = 'error'
  } finally {
    saving.value = false
  }
}

// ── Session stats ────────────────────────────────────────────────
async function loadStats() {
  try {
    stats.value = await relayAPI.getSessionStats()
  } catch (err) {
    console.error('Failed to load session stats:', err)
  }
}

async function clearSessions() {
  if (!confirm('确定要清空所有会话历史吗？')) return
  try {
    await relayAPI.clearSessions()
    saveMessage.value = '会话历史已清空'
    saveMessageType.value = 'success'
    await loadStats()
  } catch (err) {
    saveMessage.value = '清空失败: ' + (err.message || '未知错误')
    saveMessageType.value = 'error'
  }
}

// ── Inject Codex ────────────────────────────────────────────────
async function injectCodexConfig() {
  const enabled = upstreams.value.filter(u => u.enabled)
  if (enabled.length === 0) {
    saveMessage.value = '请先添加并启用至少一个上游渠道'
    saveMessageType.value = 'error'
    return
  }
  // Save first
  await saveConfig()

  injecting.value = true
  try {
    // Pass empty body — backend selects the first enabled upstream automatically
    const data = await relayAPI.injectCodex({
      default_model: globalConfig.value.default_model || '',
    })
    if (data.success) {
      relayInjected.value = !!data.codex_injected
      relayServiceURL.value = data.relay_url || localRelayServiceURL()
      const restarted = data.codex_app_restarted ? '已重启' : (data.codex_app_started ? '已启动' : '')
      saveMessage.value = restarted
        ? `Codex 配置注入成功，Codex 客户端${restarted}。`
        : 'Codex 配置注入成功！请重启 Codex 客户端或 Codex CLI。'
      saveMessageType.value = 'success'
    } else {
      throw new Error(data.message || '注入失败')
    }
  } catch (err) {
    saveMessage.value = '注入失败: ' + err.message
    saveMessageType.value = 'error'
  } finally {
    injecting.value = false
  }
}

// ── Utilities ────────────────────────────────────────────────────
function copyText(text) {
  navigator.clipboard.writeText(text).then(() => {
    saveMessage.value = '已复制到剪贴板'
    saveMessageType.value = 'success'
    setTimeout(() => { saveMessage.value = '' }, 2000)
  })
}

function formatBytes(bytes) {
  if (!bytes || bytes === 0) return '0 B'
  const k = 1024
  const sizes = ['B', 'KB', 'MB', 'GB']
  const i = Math.floor(Math.log(bytes) / Math.log(k))
  return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + ' ' + sizes[i]
}

function formatLogTime(iso) {
  if (!iso) return '--:--:--'
  const date = new Date(iso)
  if (Number.isNaN(date.getTime())) return iso
  return date.toLocaleTimeString()
}

// ── Log stream ───────────────────────────────────────────────────
function appendLog(entry) {
  if (!entry?.id) return
  if (logs.value.some((item) => item.id === entry.id)) return
  logs.value.push(entry)
  if (logs.value.length > MAX_LOG_LINES) {
    logs.value = logs.value.slice(-MAX_LOG_LINES)
  }
  nextTick(scrollLogToBottom)
}

function scrollLogToBottom() {
  const panel = logPanelRef.value
  if (panel) panel.scrollTop = panel.scrollHeight
}

function disconnectLogStream() {
  if (logReconnectTimer) { clearTimeout(logReconnectTimer); logReconnectTimer = null }
  if (logEventSource) { logEventSource.close(); logEventSource = null }
  logConnected.value = false
}

function connectLogStream() {
  disconnectLogStream()
  logEventSource = new EventSource('/api/v1/relay/logs/stream')

  logEventSource.addEventListener('ready', (event) => {
    try {
      const data = JSON.parse(event.data)
      if (Array.isArray(data.entries) && data.entries.length > 0) {
        logs.value = data.entries.slice(-MAX_LOG_LINES)
        nextTick(scrollLogToBottom)
      }
    } catch {}
  })

  logEventSource.addEventListener('log', (event) => {
    try { appendLog(JSON.parse(event.data)) } catch {}
  })

  logEventSource.onopen = () => { logConnected.value = true }

  logEventSource.onerror = () => {
    logConnected.value = false
    disconnectLogStream()
    if (autoReconnect.value) {
      logReconnectTimer = setTimeout(connectLogStream, 3000)
    }
  }
}

function toggleLogStream() {
  autoReconnect.value = !logConnected.value
  if (logConnected.value) {
    autoReconnect.value = false
    disconnectLogStream()
    return
  }
  autoReconnect.value = true
  connectLogStream()
}

async function loadLogs() {
  try {
    const data = await relayAPI.getLogs(MAX_LOG_LINES)
    logs.value = Array.isArray(data.entries) ? data.entries : []
    nextTick(scrollLogToBottom)
  } catch {}
}

async function clearLogs() {
  if (!confirm('确定要清空 Relay 日志吗？')) return
  try {
    await relayAPI.clearLogs()
    logs.value = []
  } catch (err) {
    saveMessage.value = '清空日志失败: ' + (err.message || '未知错误')
    saveMessageType.value = 'error'
  }
}

onMounted(() => {
  loadConfig()
  loadLogs()
  connectLogStream()
})

onUnmounted(() => {
  autoReconnect.value = false
  disconnectLogStream()
})
</script>

<style scoped>
/* ── Upstream card ───────────────────────────────────── */
.upstream-card {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 0.75rem;
  border-radius: 0.75rem;
  border: 1px solid var(--app-border);
  background: var(--app-control-bg);
  padding: 0.75rem 1rem;
  transition: border-color 0.15s;
}
.upstream-card:hover {
  border-color: var(--app-accent-soft);
}
.upstream-card--disabled {
  opacity: 0.5;
}

.upstream-toggle {
  position: relative;
  width: 2rem;
  height: 1.125rem;
  background: var(--app-border);
  border-radius: 999px;
  transition: background 0.15s;
}
.upstream-toggle::after {
  content: '';
  position: absolute;
  top: 2px;
  left: 2px;
  width: 0.875rem;
  height: 0.875rem;
  background: white;
  border-radius: 50%;
  transition: transform 0.15s;
}
.peer:checked ~ .upstream-toggle::after {
  transform: translateX(0.875rem);
}

/* ── Add/edit form ───────────────────────────────────── */
.upstream-form {
  border: 1px solid var(--app-accent-soft);
  border-radius: 0.75rem;
  background: var(--app-surface-muted);
  padding: 1.25rem;
  margin-top: 0.5rem;
}

/* ── Log panel ───────────────────────────────────────── */
.relay-log-panel {
  max-height: 360px;
  overflow-y: auto;
  border-radius: 0.75rem;
  border: 1px solid var(--app-border);
  background: var(--app-surface-muted);
  padding: 0.75rem;
  font-family: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, monospace;
  font-size: 0.75rem;
}
.relay-log-empty {
  padding: 2rem 0;
  text-align: center;
  color: var(--app-text-muted);
}
.relay-log-line {
  display: flex;
  flex-wrap: wrap;
  align-items: baseline;
  gap: 0.25rem 0.5rem;
  border-bottom: 1px solid color-mix(in srgb, var(--app-border) 55%, transparent);
  padding: 0.375rem 0;
}
.relay-log-line:last-child { border-bottom: none; }
.relay-log-time { flex-shrink: 0; color: var(--app-text-muted); }
.relay-log-level {
  flex-shrink: 0;
  border-radius: 0.25rem;
  padding: 0.125rem 0.25rem;
  font-size: 10px;
  font-weight: 600;
  letter-spacing: 0.04em;
}
.relay-log-model {
  flex-shrink: 0;
  border-radius: 0.25rem;
  background: var(--app-control-bg);
  padding: 0.125rem 0.375rem;
  font-size: 10px;
  color: var(--app-accent);
}
.relay-log-message { min-width: 0; flex: 1; word-break: break-all; color: var(--app-text-secondary); }
.relay-log-line--info .relay-log-level { background: var(--app-control-bg); color: var(--app-text-muted); }
.relay-log-line--warn .relay-log-level { background: color-mix(in srgb, #f59e0b 15%, transparent); color: color-mix(in srgb, #f59e0b 85%, white); }
.relay-log-line--warn .relay-log-message { color: var(--app-text-primary); }
.relay-log-line--error .relay-log-level { background: color-mix(in srgb, #ef4444 15%, transparent); color: color-mix(in srgb, #ef4444 85%, white); }
.relay-log-line--error .relay-log-message { color: var(--app-text-primary); }
</style>
