<template>
  <div class="dashboard-page flex flex-col gap-5 p-5">
    <section class="dashboard-hero">
      <div class="flex flex-col gap-5 lg:flex-row lg:items-end lg:justify-between">
        <div class="max-w-3xl space-y-3">
          <div class="dashboard-hero__badge">
            <span>🤖</span>
            <span>Codex</span>
          </div>
          <div>
            <h1 class="dashboard-hero__title">Codex 总览</h1>
            <p class="dashboard-hero__desc">
              查看 OpenAI OAuth 账号、API 账号、Codex 代理池和本机运行状态。
            </p>
          </div>
        </div>

        <div class="stable-action-row lg:justify-end">
          <button class="btn btn-secondary" :disabled="loading" @click="loadDashboard">
            {{ loading ? '刷新中...' : '刷新总览' }}
          </button>
          <button class="btn btn-primary" @click="router.push('/codex')">管理 Codex</button>
        </div>
      </div>
    </section>

    <div class="grid gap-4 md:grid-cols-2 xl:grid-cols-4">
      <div
        v-for="stat in dashboardStatCards"
        :key="stat.label"
        class="card dashboard-stat-card p-4 select-none"
      >
        <div class="dashboard-stat-card__label">{{ stat.label }}</div>
        <div class="dashboard-stat-card__value">{{ stat.value }}</div>
        <div class="dashboard-stat-card__sub">{{ stat.sub }}</div>
      </div>
    </div>

    <section class="relay-panel">
      <div class="flex flex-col gap-4 sm:flex-row sm:items-center sm:justify-between">
        <div>
          <div class="relay-panel__badge">
            <span>🔗</span>
            <span>Relay 模式</span>
          </div>
          <h2 class="relay-panel__title">Relay 调用统计</h2>
          <p class="relay-panel__desc">
            <span v-if="relayUsage.upstream_label" class="relay-panel__provider">{{ relayUsage.upstream_label }}</span>
            <span v-if="relayUsage.default_model" class="relay-panel__model"> · {{ relayUsage.default_model }}</span>
            <span> — Codex 经本地 Relay 转发的累计请求与 Token 消耗。</span>
            <span v-if="relayUsage.upstream_configured" class="relay-panel__status-ok">上游已配置</span>
            <span v-else class="relay-panel__status-warn">上游未配置</span>
            <span v-if="relayUsage.codex_injected" class="relay-panel__status-ok"> · Codex 已注入</span>
          </p>
        </div>
        <button class="btn btn-secondary shrink-0" @click="router.push('/relay')">Relay 配置</button>
      </div>

      <div class="mt-5 grid gap-4 sm:grid-cols-2 xl:grid-cols-5">
        <div class="card relay-stat-card p-4 select-none">
          <div class="relay-stat-card__label">请求次数</div>
          <div class="relay-stat-card__value">{{ relayUsage.usage?.request_count || 0 }}</div>
          <div class="relay-stat-card__sub">流式 {{ relayUsage.usage?.stream_count || 0 }}</div>
        </div>
        <div class="card relay-stat-card p-4 select-none">
          <div class="relay-stat-card__label">输入 Tokens</div>
          <div class="relay-stat-card__value">{{ formatTokens(relayUsage.usage?.input_tokens) }}</div>
          <div class="relay-stat-card__sub">缓存 {{ formatTokens(relayUsage.usage?.cached_tokens) }}</div>
        </div>
        <div class="card relay-stat-card p-4 select-none">
          <div class="relay-stat-card__label">输出 Tokens</div>
          <div class="relay-stat-card__value">{{ formatTokens(relayUsage.usage?.output_tokens) }}</div>
          <div class="relay-stat-card__sub">模型回复消耗</div>
        </div>
        <div class="card relay-stat-card relay-stat-card--highlight p-4 select-none">
          <div class="relay-stat-card__label">总 Tokens</div>
          <div class="relay-stat-card__value">{{ formatTokens(relayUsage.usage?.total_tokens) }}</div>
          <div class="relay-stat-card__sub">累计消耗</div>
        </div>
        <div class="card relay-stat-card p-4 select-none">
          <div class="relay-stat-card__label">最近调用</div>
          <div class="relay-stat-card__value relay-stat-card__value--sm" :title="relayUsage.usage?.last_model || '-'">
            {{ relayUsage.usage?.last_model || '暂无' }}
          </div>
          <div class="relay-stat-card__sub">{{ formatRelayTime(relayUsage.usage?.last_request_at) }}</div>
        </div>
      </div>

      <div class="mt-5">
        <div class="mb-3 flex items-center justify-between gap-3">
          <h3 class="relay-panel__subtitle">最近调用记录</h3>
          <span class="relay-panel__hint">每 10 秒自动刷新</span>
        </div>
        <div v-if="recentRelayCalls.length === 0" class="relay-call-empty">
          暂无 Relay 调用记录。通过小米 MiMo Relay 在 Codex 中发起对话后，记录会显示在这里。
        </div>
        <div v-else class="relay-call-wrap overflow-x-auto">
          <table class="relay-call-table w-full min-w-[720px] text-left text-sm">
            <thead>
              <tr>
                <th>时间</th>
                <th>上游</th>
                <th>Codex 模型</th>
                <th>上游模型</th>
                <th>类型</th>
                <th>输入</th>
                <th>输出</th>
                <th>总计</th>
              </tr>
            </thead>
            <tbody>
              <tr v-for="(call, index) in recentRelayCalls" :key="`${call.timestamp}-${index}`">
                <td class="cell-muted whitespace-nowrap">{{ formatRelayTime(call.timestamp) }}</td>
                <td class="cell-accent whitespace-nowrap">{{ call.provider || relayUsage.upstream_label || '-' }}</td>
                <td class="cell-text whitespace-nowrap">{{ call.codex_model || '-' }}</td>
                <td class="cell-secondary whitespace-nowrap">{{ call.upstream_model || '-' }}</td>
                <td class="whitespace-nowrap">
                  <span class="relay-type-tag">{{ call.stream ? '流式' : '非流式' }}</span>
                </td>
                <td class="cell-secondary whitespace-nowrap">{{ formatTokens(call.input_tokens) }}</td>
                <td class="cell-secondary whitespace-nowrap">{{ formatTokens(call.output_tokens) }}</td>
                <td class="cell-highlight whitespace-nowrap">{{ formatTokens(call.total_tokens) }}</td>
              </tr>
            </tbody>
          </table>
        </div>
      </div>
    </section>

    <div class="grid min-h-0 flex-1 items-stretch gap-5 xl:grid-cols-[minmax(0,1.4fr)_360px]">
      <section class="card flex min-h-[420px] flex-col p-5">
        <div class="flex items-start justify-between gap-4">
          <div>
            <h2 class="text-lg font-semibold text-white">账号概览</h2>
            <p class="mt-1 text-sm text-gray-500">当前 OpenAI OAuth 与 API 账号。</p>
          </div>
          <span class="badge badge-blue">Codex</span>
        </div>

        <div class="mt-5 flex flex-1 flex-col">
          <div class="account-overview-grid">
            <div
              v-for="account in paginatedAccounts"
              :key="account.id"
              class="account-overview-item"
              :class="account.is_codex_active ? 'account-overview-item--active' : ''"
            >
              <div class="min-w-0">
                <div class="truncate text-xs font-medium text-white" :title="account.email || account.model_provider || account.id">
                  {{ account.email || account.model_provider || account.id }}
                </div>
                <div class="mt-1 flex min-w-0 items-center gap-1.5">
                  <span class="shrink-0 rounded bg-gray-700/60 px-1.5 py-0.5 text-[10px] font-medium uppercase text-gray-300">
                    {{ account.account_type === 'api' ? 'API' : (account.plan || 'OAuth') }}
                  </span>
                  <span class="truncate text-[11px] text-gray-500">
                    {{ account.account_type === 'api' ? account.model || account.wire_api || 'API 账号' : 'OAuth 账号' }}
                  </span>
                </div>
              </div>
              <span class="badge shrink-0" :class="account.is_codex_active ? 'badge-green' : 'badge-gray'">
                {{ account.is_codex_active ? '当前' : account.status || 'active' }}
              </span>
            </div>
          </div>

          <div v-if="paginatedAccounts.length === 0" class="py-8 text-center text-sm text-gray-500">
            暂无账号，进入 Codex 管理页导入或添加账号。
          </div>

          <div v-if="accountOverviewTotalPages > 1" class="mt-auto flex flex-wrap items-center justify-center gap-2 pt-4 text-sm">
            <button
              class="btn btn-sm btn-secondary"
              :disabled="accountOverviewPage <= 1"
              @click="accountOverviewPage = Math.max(1, accountOverviewPage - 1)"
            >
              上一页
            </button>
            <span class="text-gray-400">
              {{ accountOverviewPage }} / {{ accountOverviewTotalPages }}
              <span class="ml-2 text-gray-600">({{ accountOverviewRangeText }}，共 {{ accounts.length }} 个)</span>
            </span>
            <button
              class="btn btn-sm btn-secondary"
              :disabled="accountOverviewPage >= accountOverviewTotalPages"
              @click="accountOverviewPage = Math.min(accountOverviewTotalPages, accountOverviewPage + 1)"
            >
              下一页
            </button>
          </div>
        </div>
      </section>

      <section class="space-y-4">
        <article class="card p-5">
          <div>
            <h2 class="text-lg font-semibold text-white">运行状态</h2>
            <p class="mt-1 text-sm text-gray-500">来自本机 Go 进程的实时环境信息。</p>
          </div>
          <dl class="mt-4 space-y-3 text-sm text-gray-400">
            <div class="flex items-center justify-between gap-4">
              <dt>版本</dt>
              <dd class="text-white">v{{ sysInfo.version || '-' }}</dd>
            </div>
            <div class="flex items-center justify-between gap-4">
              <dt>运行时间</dt>
              <dd class="text-white">{{ sysInfo.uptime || '-' }}</dd>
            </div>
            <div class="flex items-center justify-between gap-4">
              <dt>数据库</dt>
              <dd class="text-white">{{ sysInfo.db_type || '-' }}</dd>
            </div>
            <div class="flex items-center justify-between gap-4">
              <dt>端口</dt>
              <dd class="text-white">{{ sysInfo.server_port || 8022 }}</dd>
            </div>
            <div class="flex items-center justify-between gap-4">
              <dt>Goroutines</dt>
              <dd class="text-white">{{ sysInfo.goroutines || '-' }}</dd>
            </div>
            <div class="flex items-center justify-between gap-4">
              <dt>内存</dt>
              <dd class="text-white">{{ sysInfo.memory_alloc_mb || '-' }} MB</dd>
            </div>
          </dl>
        </article>
      </section>
    </div>
  </div>
</template>

<script setup>
import { computed, inject, onMounted, onUnmounted, ref, watch } from 'vue'
import { useRouter } from 'vue-router'
import api, { openaiAPI, relayAPI, settingsAPI } from '@/api'
import { filterAPIAccounts, filterOAuthAccounts } from '@/lib/accounts'

const notify = inject('notify')
const router = useRouter()
const loading = ref(false)
const accounts = ref([])
const pool = ref({})
const sysInfo = ref({})
const relayUsage = ref({ usage: {}, recent_calls: [] })
let relayRefreshTimer = null
const accountOverviewPage = ref(1)
const ACCOUNT_OVERVIEW_PAGE_SIZE = 12

const oauthAccounts = computed(() => filterOAuthAccounts(accounts.value))
const apiAccounts = computed(() => filterAPIAccounts(accounts.value))
const activeOAuthCount = computed(() => oauthAccounts.value.filter((account) => account.is_codex_active).length)
const activeAPICount = computed(() => apiAccounts.value.filter((account) => account.is_codex_active).length)
const accountOverviewTotalPages = computed(() => Math.ceil(accounts.value.length / ACCOUNT_OVERVIEW_PAGE_SIZE) || 1)
const paginatedAccounts = computed(() => {
  const start = (accountOverviewPage.value - 1) * ACCOUNT_OVERVIEW_PAGE_SIZE
  return accounts.value.slice(start, start + ACCOUNT_OVERVIEW_PAGE_SIZE)
})
const accountOverviewRangeText = computed(() => {
  if (accounts.value.length === 0) return '显示 0'
  const start = (accountOverviewPage.value - 1) * ACCOUNT_OVERVIEW_PAGE_SIZE + 1
  const end = Math.min(accounts.value.length, start + ACCOUNT_OVERVIEW_PAGE_SIZE - 1)
  return `显示 ${start}-${end}`
})
const joinedProxyCount = computed(() => Number(pool.value.proxy_enabled_count ?? pool.value.pool_size ?? 0))
const effectivePoolSize = computed(() => (pool.value.proxy_pool_enabled ? Number(pool.value.pool_size ?? 0) : 0))
const poolStatusText = computed(() => {
  if (!pool.value.proxy_pool_enabled) {
    return joinedProxyCount.value > 0 ? `已关闭，已加入 ${joinedProxyCount.value} 个账号` : '已关闭'
  }
  return effectivePoolSize.value > 0 ? '已启用' : '已启用，暂无可用账号'
})
const recentRelayCalls = computed(() => {
  const calls = relayUsage.value?.recent_calls
  return Array.isArray(calls) ? calls : []
})
const dashboardStatCards = computed(() => [
  { label: 'OAuth 账号', value: oauthAccounts.value.length, sub: `${activeOAuthCount.value} 个当前激活` },
  { label: 'API 账号', value: apiAccounts.value.length, sub: `${activeAPICount.value} 个当前激活` },
  { label: '代理池', value: `${effectivePoolSize.value} / ${joinedProxyCount.value}`, sub: poolStatusText.value },
  { label: '累计请求', value: pool.value.total_requests || 0, sub: `策略 ${pool.value.strategy || 'round_robin'}` },
])

watch(accountOverviewTotalPages, (totalPages) => {
  if (accountOverviewPage.value > totalPages) {
    accountOverviewPage.value = totalPages
  }
})

onMounted(() => {
  loadDashboard()
  relayRefreshTimer = setInterval(loadRelayUsage, 10000)
})

onUnmounted(() => {
  if (relayRefreshTimer) {
    clearInterval(relayRefreshTimer)
    relayRefreshTimer = null
  }
})

async function loadRelayUsage() {
  try {
    const relayData = await relayAPI.getUsage()
    relayUsage.value = relayData || { usage: {}, recent_calls: [] }
  } catch (error) {
    console.error('Failed to load relay usage:', error)
  }
}

async function loadDashboard() {
  loading.value = true
  try {
    const [accountData, poolData, systemData] = await Promise.all([
      openaiAPI.list(),
      api.get('/openai/service-config'),
      settingsAPI.systemInfo(),
    ])
    accounts.value = Array.isArray(accountData) ? accountData : []
    pool.value = poolData || {}
    sysInfo.value = systemData || {}
    await loadRelayUsage()
  } catch (error) {
    notify?.(error.message || '加载总览失败', 'error')
  } finally {
    loading.value = false
  }
}

function formatTokens(value) {
  const n = Number(value) || 0
  if (n >= 1_000_000) return `${(n / 1_000_000).toFixed(2)}M`
  if (n >= 10_000) return `${(n / 1_000).toFixed(1)}K`
  return n.toLocaleString()
}

function formatRelayTime(iso) {
  if (!iso) return '暂无记录'
  const date = new Date(iso)
  if (Number.isNaN(date.getTime())) return iso
  return date.toLocaleString()
}
</script>

<style scoped>
.dashboard-page {
  min-height: 100vh;
}

.stable-action-row {
  @apply flex items-center gap-2 overflow-x-auto pb-1;
}

.stable-action-row > * {
  @apply shrink-0 whitespace-nowrap;
}

.account-overview-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(180px, 1fr));
  gap: 0.6rem;
}

.account-overview-item {
  @apply flex min-h-[58px] min-w-0 items-center justify-between gap-2 rounded-lg border px-3 py-2 transition-colors;
  background: var(--app-surface-muted);
  border-color: var(--app-border-soft);
}

.account-overview-item:hover {
  background: var(--app-control-bg);
  border-color: var(--app-border);
}

.account-overview-item--active {
  background: color-mix(in srgb, var(--app-success) 12%, transparent);
  border-color: color-mix(in srgb, var(--app-success) 42%, transparent);
}

.dashboard-hero {
  border-radius: 1rem;
  border: 1px solid var(--app-border-soft);
  background:
    linear-gradient(135deg, var(--app-accent-tint), transparent 58%),
    var(--app-surface-muted);
  padding: 1.25rem;
  box-shadow: var(--app-shadow-lg);
}

.dashboard-hero__badge {
  display: inline-flex;
  align-items: center;
  gap: 0.5rem;
  border-radius: 9999px;
  border: 1px solid color-mix(in srgb, var(--app-accent) 25%, transparent);
  background: var(--app-accent-tint);
  padding: 0.25rem 0.75rem;
  font-size: 0.75rem;
  color: var(--app-accent);
}

.dashboard-hero__title {
  margin-top: 0.75rem;
  font-size: 1.875rem;
  font-weight: 600;
  color: var(--app-text);
}

.dashboard-hero__desc {
  margin-top: 0.5rem;
  font-size: 0.875rem;
  line-height: 1.5rem;
  color: var(--app-text-secondary);
}

.dashboard-stat-card__label,
.relay-stat-card__label {
  font-size: 0.75rem;
  text-transform: uppercase;
  letter-spacing: 0.04em;
  color: var(--app-text-muted);
}

.dashboard-stat-card__value,
.relay-stat-card__value {
  margin-top: 0.5rem;
  font-size: 1.5rem;
  font-weight: 600;
  color: var(--app-text);
}

.relay-stat-card__value--sm {
  font-size: 0.875rem;
  font-weight: 500;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.dashboard-stat-card__sub,
.relay-stat-card__sub {
  margin-top: 0.25rem;
  font-size: 0.875rem;
  color: var(--app-text-muted);
}

.relay-panel {
  border-radius: 1rem;
  border: 1px solid var(--app-border-soft);
  background:
    linear-gradient(135deg, color-mix(in srgb, var(--app-accent-tint) 85%, transparent), transparent 62%),
    var(--app-surface-muted);
  padding: 1.25rem;
  box-shadow: var(--app-shadow-lg);
}

.relay-panel__badge {
  display: inline-flex;
  align-items: center;
  gap: 0.5rem;
  border-radius: 9999px;
  border: 1px solid color-mix(in srgb, var(--app-accent) 25%, transparent);
  background: var(--app-accent-tint);
  padding: 0.25rem 0.75rem;
  font-size: 0.75rem;
  color: var(--app-accent);
}

.relay-panel__title {
  margin-top: 0.75rem;
  font-size: 1.125rem;
  font-weight: 600;
  color: var(--app-text);
}

.relay-panel__subtitle {
  font-size: 0.875rem;
  font-weight: 500;
  color: var(--app-text);
}

.relay-panel__desc {
  margin-top: 0.25rem;
  font-size: 0.875rem;
  color: var(--app-text-muted);
}

.relay-panel__provider {
  color: var(--app-accent);
}

.relay-panel__model {
  color: var(--app-text-secondary);
}

.relay-panel__status-ok {
  color: var(--app-success);
}

.relay-panel__status-warn {
  color: var(--app-warning);
}

.relay-panel__hint {
  font-size: 0.75rem;
  color: var(--app-text-muted);
}

.relay-stat-card--highlight {
  border-color: color-mix(in srgb, var(--app-accent) 35%, transparent);
  background: color-mix(in srgb, var(--app-accent-tint) 75%, var(--app-surface));
}

.relay-stat-card--highlight .relay-stat-card__label {
  color: var(--app-accent);
}

.relay-call-empty,
.relay-call-wrap {
  border-radius: 0.75rem;
  border: 1px solid var(--app-border);
  background: var(--app-surface);
}

.relay-call-empty {
  padding: 2rem 1rem;
  text-align: center;
  font-size: 0.875rem;
  color: var(--app-text-muted);
}

.relay-type-tag {
  display: inline-block;
  border-radius: 0.25rem;
  background: var(--app-control-bg);
  padding: 0.125rem 0.5rem;
  font-size: 0.75rem;
  color: var(--app-text-secondary);
}

.relay-call-table .cell-muted {
  color: var(--app-text-muted);
}

.relay-call-table .cell-accent {
  color: var(--app-accent);
}

.relay-call-table .cell-text {
  color: var(--app-text);
}

.relay-call-table .cell-secondary {
  color: var(--app-text-secondary);
}

.relay-call-table .cell-highlight {
  color: var(--app-accent);
  font-weight: 500;
}

.relay-call-table thead th {
  padding: 0.75rem 1rem;
  font-size: 0.75rem;
  font-weight: 600;
  text-transform: uppercase;
  letter-spacing: 0.04em;
  color: var(--app-text-muted);
  background: var(--app-surface-muted);
  border-bottom: 1px solid var(--app-border);
}

.relay-call-table tbody td {
  padding: 0.75rem 1rem;
  border-bottom: 1px solid color-mix(in srgb, var(--app-border) 60%, transparent);
}

.relay-call-table tbody tr:last-child td {
  border-bottom: none;
}

.relay-call-table tbody tr:hover {
  background: color-mix(in srgb, var(--app-control-bg) 70%, transparent);
}
</style>
