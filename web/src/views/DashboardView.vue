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
            <span>调用监控</span>
          </div>
          <h2 class="relay-panel__title">Codex 与 Relay 调用</h2>
          <p class="relay-panel__desc">
            <span>本地代理和 Relay 转发记录集中在这里，自动刷新。</span>
            <span :class="pool.codex_api_service ? 'relay-panel__status-ok' : 'relay-panel__status-warn'">
              Codex {{ pool.codex_api_service ? '已启用' : '未启用' }}
            </span>
            <span :class="relayUsage.upstream_configured ? 'relay-panel__status-ok' : 'relay-panel__status-warn'">
              · Relay {{ relayUsage.upstream_configured ? '已配置' : '未配置' }}
            </span>
            <span v-if="relayUsage.codex_injected" class="relay-panel__status-ok"> · 已注入</span>
          </p>
        </div>
        <div class="stable-action-row sm:justify-end">
          <button class="btn btn-secondary" @click="router.push('/codex')">Codex 配置</button>
          <button class="btn btn-secondary" @click="router.push('/relay')">Relay 配置</button>
        </div>
      </div>

      <div class="mt-5 grid gap-3 sm:grid-cols-2 xl:grid-cols-4">
        <div class="dashboard-mini-stat">
          <div class="relay-stat-card__label">Codex 最近调用</div>
          <div class="relay-stat-card__value">{{ recentCodexCalls.length }}</div>
          <div class="relay-stat-card__sub">{{ pool.codex_api_service ? '服务运行中' : '服务未启用' }}</div>
        </div>
        <div class="dashboard-mini-stat">
          <div class="relay-stat-card__label">Relay 请求</div>
          <div class="relay-stat-card__value">{{ relayUsage.usage?.request_count || 0 }}</div>
          <div class="relay-stat-card__sub">流式 {{ relayUsage.usage?.stream_count || 0 }}</div>
        </div>
        <div class="dashboard-mini-stat dashboard-mini-stat--highlight">
          <div class="relay-stat-card__label">总 Tokens</div>
          <div class="relay-stat-card__value">{{ formatTokens(relayUsage.usage?.total_tokens) }}</div>
          <div class="relay-stat-card__sub">
            输入 {{ formatTokens(relayUsage.usage?.input_tokens) }} / 输出 {{ formatTokens(relayUsage.usage?.output_tokens) }}
          </div>
        </div>
        <div class="dashboard-mini-stat">
          <div class="relay-stat-card__label">最近调用</div>
          <div class="relay-stat-card__value relay-stat-card__value--sm" :title="relayUsage.usage?.last_model || '-'">
            {{ relayUsage.usage?.last_model || '暂无' }}
          </div>
          <div class="relay-stat-card__sub">{{ formatRelayTime(relayUsage.usage?.last_request_at) }}</div>
        </div>
      </div>

      <div class="mt-5">
        <div class="mb-3 flex flex-wrap items-center justify-between gap-3">
          <div>
            <h3 class="relay-panel__subtitle">最近调用</h3>
            <p class="relay-panel__hint mt-1">
              共 {{ combinedCallRows.length }} 条，Codex {{ recentCodexCalls.length }} 条，Relay {{ recentRelayCalls.length }} 条
            </p>
          </div>
          <div class="flex flex-wrap items-center justify-end gap-2">
            <span class="relay-panel__hint">每 10 秒刷新</span>
            <select
              v-if="combinedCallRows.length > 0 || callSourceFilter !== 'all'"
              v-model="callSourceFilter"
              class="input h-8 w-auto py-1 pl-2 pr-6 text-sm"
            >
              <option value="all">全部来源</option>
              <option value="codex">仅 Codex</option>
              <option value="relay">仅 Relay</option>
            </select>
            <button
              v-if="combinedCallRows.length > 0"
              class="btn btn-secondary py-1 text-sm h-8 px-3"
              @click="clearAllHistory"
            >
              清空
            </button>
          </div>
        </div>

        <div v-if="filteredCombinedCallRows.length === 0" class="relay-call-empty">
          <template v-if="combinedCallRows.length === 0">
            暂无调用记录。通过 Codex 本地服务或 Relay 发起对话后，记录会显示在这里。
          </template>
          <template v-else>
            暂无匹配该来源的调用记录。
          </template>
        </div>
        <div v-else class="relay-call-wrap overflow-x-auto">
          <table class="relay-call-table w-full min-w-[860px] text-left text-sm">
            <thead>
              <tr>
                <th>时间</th>
                <th>来源</th>
                <th>账号 / 上游</th>
                <th>模型</th>
                <th>类型</th>
                <th>状态 / 消耗</th>
                <th>详情</th>
              </tr>
            </thead>
            <tbody>
              <tr v-for="call in paginatedCombinedCalls" :key="call.id">
                <td class="cell-muted whitespace-nowrap">{{ formatRelayTime(call.timestamp) }}</td>
                <td class="whitespace-nowrap">
                  <span class="relay-type-tag" :class="call.sourceClass">{{ call.sourceLabel }}</span>
                </td>
                <td class="cell-accent whitespace-nowrap" :title="call.primary">
                  {{ call.primary }}
                </td>
                <td class="cell-text whitespace-nowrap" :title="call.model">
                  {{ call.model }}
                </td>
                <td class="whitespace-nowrap">
                  <span class="relay-type-tag">{{ call.typeLabel }}</span>
                </td>
                <td class="whitespace-nowrap" :title="call.statusTitle">
                  <span class="relay-type-tag" :class="call.statusClass">{{ call.statusLabel }}</span>
                </td>
                <td class="cell-secondary whitespace-nowrap" :title="call.detailTitle">
                  {{ call.detail }}
                </td>
              </tr>
            </tbody>
          </table>
        </div>
        <div v-if="combinedTotalPages > 1" class="mt-4 flex items-center justify-center gap-2">
          <button class="btn btn-sm btn-secondary" :disabled="combinedCurrentPage === 1" @click="combinedCurrentPage--">上一页</button>
          <span class="text-sm text-gray-400">第 {{ combinedCurrentPage }} / {{ combinedTotalPages }} 页</span>
          <button class="btn btn-sm btn-secondary" :disabled="combinedCurrentPage === combinedTotalPages" @click="combinedCurrentPage++">下一页</button>
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
        <article class="card runtime-status-card p-5">
          <div>
            <h2 class="text-lg font-semibold text-white">运行状态</h2>
            <p class="mt-1 text-sm text-gray-500">来自本机 Go 进程的实时环境信息。</p>
          </div>
          <dl class="runtime-status-list">
            <div class="runtime-status-row">
              <dt>版本</dt>
              <dd class="runtime-status-value">v{{ sysInfo.version || '-' }}</dd>
            </div>
            <div class="runtime-status-row">
              <dt>运行时间</dt>
              <dd class="runtime-status-value">{{ sysInfo.uptime || '-' }}</dd>
            </div>
            <div class="runtime-status-row">
              <dt>数据库</dt>
              <dd class="runtime-status-value">{{ sysInfo.db_type || '-' }}</dd>
            </div>
            <div class="runtime-status-row">
              <dt>端口</dt>
              <dd class="runtime-status-value">{{ sysInfo.server_port || 8022 }}</dd>
            </div>
            <div class="runtime-status-row">
              <dt>Goroutines</dt>
              <dd class="runtime-status-value">{{ sysInfo.goroutines || '-' }}</dd>
            </div>
            <div class="runtime-status-row">
              <dt>内存</dt>
              <dd class="runtime-status-value">{{ sysInfo.memory_alloc_mb || '-' }} MB</dd>
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
const confirmOperation = inject('confirmOperation')
const router = useRouter()
const loading = ref(false)
const accounts = ref([])
const pool = ref({})
const sysInfo = ref({})
const relayUsage = ref({ usage: {}, recent_calls: [] })
const codexCalls = ref({ recent_calls: [] })
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
const recentCodexCalls = computed(() => {
  const calls = codexCalls.value?.recent_calls
  return Array.isArray(calls) ? calls : []
})

const callSourceFilter = ref('all')
const combinedCurrentPage = ref(1)
const COMBINED_PAGE_SIZE = 10

const combinedCallRows = computed(() => {
  const codexRows = recentCodexCalls.value.map((call, index) => ({
    id: `codex-${call.timestamp || index}-${index}`,
    source: 'codex',
    sourceLabel: 'Codex',
    sourceClass: 'relay-type-tag--codex',
    timestamp: call.timestamp,
    primary: call.account_email || call.account_id || '-',
    model: call.model || '-',
    typeLabel: formatCodexCallType(call),
    statusLabel: call.status_code || '-',
    statusClass: statusClass(call.status_code),
    statusTitle: call.error || '',
    detail: `${call.path || '-'} · ${formatDuration(call.duration_ms)}`,
    detailTitle: `${call.path || '-'} · ${formatDuration(call.duration_ms)}`,
  }))

  const relayRows = recentRelayCalls.value.map((call, index) => {
    const provider = call.provider || relayUsage.value.upstream_label || '-'
    const codexModel = call.codex_model || '-'
    const upstreamModel = call.upstream_model || '-'
    const model = upstreamModel !== '-' ? `${codexModel} -> ${upstreamModel}` : codexModel
    const tokenDetail = `输入 ${formatTokens(call.input_tokens)} / 输出 ${formatTokens(call.output_tokens)}`
    return {
      id: `relay-${call.timestamp || index}-${index}`,
      source: 'relay',
      sourceLabel: 'Relay',
      sourceClass: 'relay-type-tag--relay',
      timestamp: call.timestamp,
      primary: provider,
      model,
      typeLabel: call.stream ? '流式' : '非流式',
      statusLabel: formatTokens(call.total_tokens),
      statusClass: 'relay-type-tag--token',
      statusTitle: tokenDetail,
      detail: tokenDetail,
      detailTitle: tokenDetail,
    }
  })

  return [...codexRows, ...relayRows].sort((a, b) => {
    const at = new Date(a.timestamp || 0).getTime()
    const bt = new Date(b.timestamp || 0).getTime()
    return (Number.isNaN(bt) ? 0 : bt) - (Number.isNaN(at) ? 0 : at)
  })
})

const filteredCombinedCallRows = computed(() => {
  if (callSourceFilter.value === 'all') return combinedCallRows.value
  return combinedCallRows.value.filter((call) => call.source === callSourceFilter.value)
})

const combinedTotalPages = computed(() => Math.ceil(filteredCombinedCallRows.value.length / COMBINED_PAGE_SIZE) || 1)

const paginatedCombinedCalls = computed(() => {
  const start = (combinedCurrentPage.value - 1) * COMBINED_PAGE_SIZE
  return filteredCombinedCallRows.value.slice(start, start + COMBINED_PAGE_SIZE)
})

watch(callSourceFilter, () => {
  combinedCurrentPage.value = 1
})

watch(combinedTotalPages, (totalPages) => {
  if (combinedCurrentPage.value > totalPages) {
    combinedCurrentPage.value = totalPages
  }
})

async function clearAllHistory() {
  const confirmed = await requestOperationConfirm({
    title: '清空调用记录',
    message: '确认清空最近的调用记录吗？',
    details: '会同时清空 Relay 调用历史和 Codex 本地服务调用记录。',
    confirmText: '清空记录',
    tone: 'danger',
  })
  if (!confirmed) return
  try {
    await Promise.all([
      api.delete('/relay/usage/history'),
      openaiAPI.clearCodexCalls(),
    ])
    await refreshCallSections()
    notify('调用记录已清空')
  } catch (err) {
    console.error('Failed to clear call history:', err)
  }
}

async function requestOperationConfirm(options) {
  if (typeof confirmOperation !== 'function') return false
  return await confirmOperation(options)
}

const dashboardStatCards = computed(() => [
  { label: 'OAuth 账号', value: oauthAccounts.value.length, sub: `${activeOAuthCount.value} 个当前激活` },
  { label: 'API 账号', value: apiAccounts.value.length, sub: `${activeAPICount.value} 个当前激活` },
  { label: '代理池', value: `${effectivePoolSize.value} / ${joinedProxyCount.value}`, sub: poolStatusText.value },
  { label: 'Codex 代理请求', value: pool.value.total_requests || 0, sub: `策略 ${pool.value.strategy || 'round_robin'}` },
])

watch(accountOverviewTotalPages, (totalPages) => {
  if (accountOverviewPage.value > totalPages) {
    accountOverviewPage.value = totalPages
  }
})

onMounted(() => {
  loadDashboard()
  relayRefreshTimer = setInterval(() => refreshCallSections({ includePool: true }), 10000)
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

async function loadCodexCalls() {
  try {
    const data = await openaiAPI.getCodexCalls()
    codexCalls.value = data || { recent_calls: [] }
  } catch (error) {
    console.error('Failed to load Codex calls:', error)
  }
}

async function loadPoolConfig() {
  try {
    const data = await api.get('/openai/service-config')
    pool.value = data || {}
  } catch (error) {
    console.error('Failed to load service config:', error)
  }
}

async function refreshCallSections(options = {}) {
  const tasks = [loadRelayUsage(), loadCodexCalls()]
  if (options.includePool) {
    tasks.push(loadPoolConfig())
  }
  await Promise.all(tasks)
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
    await refreshCallSections()
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

function formatDuration(value) {
  const ms = Number(value) || 0
  if (ms >= 1000) return `${(ms / 1000).toFixed(1)}s`
  return `${ms}ms`
}

function formatCodexCallType(call) {
  if (call?.path?.includes('/ws')) return 'WebSocket'
  if (call?.passthrough) return '直通'
  return call?.stream ? '流式' : '非流式'
}

function statusClass(status) {
  const code = Number(status) || 0
  if (code >= 200 && code < 400) return 'relay-type-tag--ok'
  if (code >= 400) return 'relay-type-tag--error'
  return ''
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

.runtime-status-card {
  min-height: 420px;
  height: 100%;
}

.runtime-status-list {
  display: grid;
  gap: 0.75rem;
  margin-top: 1rem;
  color: var(--app-text-muted);
  font-size: 0.875rem;
}

.runtime-status-row {
  display: grid;
  grid-template-columns: minmax(7rem, 1fr) minmax(0, auto);
  align-items: center;
  column-gap: 1rem;
  min-height: 1.5rem;
}

.runtime-status-value {
  min-width: 0;
  color: var(--app-text);
  font-variant-numeric: tabular-nums;
  overflow-wrap: anywhere;
  text-align: right;
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

.dashboard-mini-stat {
  min-width: 0;
  border-radius: 0.75rem;
  border: 1px solid var(--app-border-soft);
  background: color-mix(in srgb, var(--app-surface) 82%, transparent);
  padding: 1rem;
}

.dashboard-mini-stat--highlight {
  border-color: color-mix(in srgb, var(--app-accent) 32%, transparent);
  background: color-mix(in srgb, var(--app-accent-tint) 58%, var(--app-surface));
}

.dashboard-mini-stat--highlight .relay-stat-card__label {
  color: var(--app-accent);
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

.relay-type-tag--ok {
  background: color-mix(in srgb, var(--app-success) 14%, var(--app-control-bg));
  color: var(--app-success);
}

.relay-type-tag--error {
  background: color-mix(in srgb, var(--app-danger) 14%, var(--app-control-bg));
  color: var(--app-danger);
}

.relay-type-tag--codex {
  background: color-mix(in srgb, var(--app-accent) 14%, var(--app-control-bg));
  color: var(--app-accent);
}

.relay-type-tag--relay,
.relay-type-tag--token {
  background: color-mix(in srgb, var(--app-success) 14%, var(--app-control-bg));
  color: var(--app-success);
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

@media (max-width: 720px) {
  .dashboard-page {
    gap: 0.875rem;
    padding: 0.875rem;
  }

  .dashboard-hero,
  .relay-panel {
    padding: 1rem;
  }

  .dashboard-hero__title {
    font-size: 1.5rem;
  }

  .dashboard-hero__desc {
    font-size: 0.8125rem;
    line-height: 1.35rem;
  }

}
</style>
