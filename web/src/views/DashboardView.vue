<template>
  <div class="dashboard-page flex flex-col gap-5 p-5">
    <section class="rounded-2xl border border-gray-800 bg-gradient-to-br from-sky-500/15 via-cyan-400/5 to-gray-950 p-5 shadow-2xl shadow-black/20">
      <div class="flex flex-col gap-5 lg:flex-row lg:items-end lg:justify-between">
        <div class="max-w-3xl space-y-3">
          <div class="inline-flex items-center gap-2 rounded-full border border-sky-500/20 bg-sky-500/10 px-3 py-1 text-xs text-sky-200">
            <span>🤖</span>
            <span>Codex</span>
          </div>
          <div>
            <h1 class="text-3xl font-semibold text-white">Codex 总览</h1>
            <p class="mt-2 text-sm leading-6 text-gray-300">
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
      <div class="card p-4 select-none">
        <div class="text-xs uppercase tracking-wide text-gray-500">OAuth 账号</div>
        <div class="mt-2 text-2xl font-semibold text-white">{{ oauthAccounts.length }}</div>
        <div class="mt-1 text-sm text-gray-400">{{ activeOAuthCount }} 个当前激活</div>
      </div>
      <div class="card p-4 select-none">
        <div class="text-xs uppercase tracking-wide text-gray-500">API 账号</div>
        <div class="mt-2 text-2xl font-semibold text-white">{{ apiAccounts.length }}</div>
        <div class="mt-1 text-sm text-gray-400">{{ activeAPICount }} 个当前激活</div>
      </div>
      <div class="card p-4 select-none">
        <div class="text-xs uppercase tracking-wide text-gray-500">代理池</div>
        <div class="mt-2 text-2xl font-semibold text-white">{{ effectivePoolSize }} / {{ joinedProxyCount }}</div>
        <div class="mt-1 text-sm text-gray-400">{{ poolStatusText }}</div>
      </div>
      <div class="card p-4 select-none">
        <div class="text-xs uppercase tracking-wide text-gray-500">累计请求</div>
        <div class="mt-2 text-2xl font-semibold text-white">{{ pool.total_requests || 0 }}</div>
        <div class="mt-1 text-sm text-gray-400">策略 {{ pool.strategy || 'round_robin' }}</div>
      </div>
    </div>

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
import { computed, inject, onMounted, ref, watch } from 'vue'
import { useRouter } from 'vue-router'
import api, { openaiAPI, settingsAPI } from '@/api'
import { filterAPIAccounts, filterOAuthAccounts } from '@/lib/accounts'

const notify = inject('notify')
const router = useRouter()
const loading = ref(false)
const accounts = ref([])
const pool = ref({})
const sysInfo = ref({})
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

watch(accountOverviewTotalPages, (totalPages) => {
  if (accountOverviewPage.value > totalPages) {
    accountOverviewPage.value = totalPages
  }
})

onMounted(loadDashboard)

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
  } catch (error) {
    notify?.(error.message || '加载总览失败', 'error')
  } finally {
    loading.value = false
  }
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
</style>
