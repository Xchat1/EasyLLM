<template>
  <div class="space-y-5 p-5">
    <section class="rounded-2xl border border-gray-800 bg-gradient-to-br from-sky-500/15 via-cyan-400/5 to-gray-950 p-5 shadow-2xl shadow-black/20">
      <div class="flex flex-col gap-5 lg:flex-row lg:items-end lg:justify-between">
        <div class="max-w-3xl space-y-3">
          <div class="inline-flex items-center gap-2 rounded-full border border-sky-500/20 bg-sky-500/10 px-3 py-1 text-xs text-sky-200">
            <span>📊</span>
            <span>EasyLLM</span>
          </div>
          <div>
            <h1 class="text-3xl font-semibold text-white">多平台总览</h1>
            <p class="mt-2 text-sm leading-6 text-gray-300">
              集中查看已接入平台、账号状态和 Codex 代理池运行情况。
            </p>
          </div>
        </div>

        <div class="stable-action-row lg:justify-end">
          <button class="btn btn-secondary" :disabled="loading" @click="loadDashboard">
            {{ loading ? '刷新中...' : '刷新总览' }}
          </button>
        </div>
      </div>
    </section>

    <div class="grid gap-4 md:grid-cols-2 xl:grid-cols-4">
      <div class="card p-4">
        <div class="text-xs uppercase tracking-wide text-gray-500">有账号平台</div>
        <div class="mt-2 text-2xl font-semibold text-white">{{ platformCards.length }}</div>
        <div class="mt-1 text-sm text-gray-400">仅展示已接入账号的平台</div>
      </div>
      <div class="card p-4">
        <div class="text-xs uppercase tracking-wide text-gray-500">账号总数</div>
        <div class="mt-2 text-2xl font-semibold text-white">{{ summary.total_accounts || 0 }}</div>
        <div class="mt-1 text-sm text-gray-400">{{ summary.active_accounts || 0 }} 个当前激活</div>
      </div>
      <div class="card p-4">
        <div class="text-xs uppercase tracking-wide text-gray-500">实例总数</div>
        <div class="mt-2 text-2xl font-semibold text-white">{{ summary.total_instances || 0 }}</div>
        <div class="mt-1 text-sm text-gray-400">{{ summary.running_instances || 0 }} 个运行中</div>
      </div>
      <div class="card p-4">
        <div class="text-xs uppercase tracking-wide text-gray-500">唤醒任务</div>
        <div class="mt-2 text-2xl font-semibold text-white">{{ summary.total_wakeup_tasks || 0 }}</div>
        <div class="mt-1 text-sm text-gray-400">{{ summary.enabled_wakeup_tasks || 0 }} 个已启用</div>
      </div>
    </div>

    <div class="grid gap-5 xl:grid-cols-[minmax(0,1.7fr)_360px]">
      <section class="space-y-4">
        <div class="flex items-center justify-between">
          <div>
            <h2 class="text-lg font-semibold text-white">平台矩阵</h2>
            <p class="mt-1 text-sm text-gray-500">只显示已有账号的平台，卡片压缩为关键指标。</p>
          </div>
        </div>

        <div v-if="platformCards.length === 0" class="card p-6 text-sm text-gray-500">
          暂无已接入账号的平台。导入或新增账号后会在这里显示。
        </div>

        <div v-else class="grid gap-3 sm:grid-cols-2 2xl:grid-cols-3">
          <button
            v-for="platform in platformCards"
            :key="platform.id"
            class="compact-platform-card"
            @click="router.push(platform.route)"
          >
            <div class="flex min-w-0 items-start justify-between gap-3">
              <div class="flex min-w-0 items-start gap-3">
                <PlatformIcon :platform="platform" size="md" />
                <div class="min-w-0">
                  <div class="flex min-w-0 items-center gap-2">
                    <div class="truncate font-medium text-white">{{ platform.label }}</div>
                    <span class="badge shrink-0" :class="platform.managementMode === 'legacy' ? 'badge-yellow' : 'badge-blue'">
                      {{ platform.managementMode === 'legacy' ? 'legacy' : '通用' }}
                    </span>
                  </div>
                  <div class="mt-1 truncate text-xs text-gray-500">{{ platform.description }}</div>
                </div>
              </div>
            </div>

            <div class="mt-3 grid grid-cols-4 gap-2 text-center">
              <div class="compact-platform-stat">
                <div class="text-[10px] text-gray-500">账号</div>
                <div class="mt-0.5 font-semibold text-white">{{ platform.accounts }}</div>
              </div>
              <div class="compact-platform-stat">
                <div class="text-[10px] text-gray-500">激活</div>
                <div class="mt-0.5 font-semibold text-white">{{ platform.active_accounts }}</div>
              </div>
              <div class="compact-platform-stat">
                <div class="text-[10px] text-gray-500">实例</div>
                <div class="mt-0.5 font-semibold text-white">{{ platform.instances }}</div>
              </div>
              <div class="compact-platform-stat">
                <div class="text-[10px] text-gray-500">任务</div>
                <div class="mt-0.5 font-semibold text-white">{{ platform.enabled_wakeup_tasks }}</div>
              </div>
            </div>

            <div class="mt-3 truncate rounded-lg border border-gray-800 bg-gray-950/50 px-2.5 py-1.5 text-xs text-gray-300">
              {{ platform.active_account_email || '未指定当前账号' }}
            </div>
          </button>
        </div>
      </section>

      <section class="space-y-4">
        <article class="card p-4">
          <div class="flex items-start justify-between gap-4">
            <div class="min-w-0">
              <h2 class="text-lg font-semibold text-white">Codex 代理池</h2>
              <p class="mt-1 text-sm text-gray-500">当前请求分发与账号池状态。</p>
            </div>
            <span class="badge shrink-0" :class="proxy.enabled ? 'badge-green' : 'badge-red'">
              {{ proxy.enabled ? '已启用' : '已关闭' }}
            </span>
          </div>

          <div class="mt-4 grid grid-cols-3 gap-2">
            <div class="codex-pool-stat">
              <div class="text-xs text-gray-500">策略</div>
              <div class="mt-1 truncate text-sm font-semibold text-white">{{ proxy.strategy || 'round_robin' }}</div>
            </div>
            <div class="codex-pool-stat">
              <div class="text-xs text-gray-500">账号池</div>
              <div class="mt-1 text-sm font-semibold text-white">{{ proxy.enabled_accounts || 0 }} / {{ proxy.accounts || 0 }}</div>
            </div>
            <div class="codex-pool-stat">
              <div class="text-xs text-gray-500">累计请求</div>
              <div class="mt-1 text-sm font-semibold text-white">{{ proxy.total_requests || 0 }}</div>
            </div>
          </div>
        </article>

        <article class="card p-5">
          <div>
            <h2 class="text-lg font-semibold text-white">运行状态</h2>
            <p class="mt-1 text-sm text-gray-500">来自当前 Go 服务的实时环境信息。</p>
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
              <dt>Goroutines</dt>
              <dd class="text-white">{{ sysInfo.goroutines || '-' }}</dd>
            </div>
            <div class="flex items-center justify-between gap-4">
              <dt>内存</dt>
              <dd class="text-white">{{ sysInfo.memory_alloc_mb || '-' }} MB</dd>
            </div>
            <div class="flex items-center justify-between gap-4">
              <dt>数据库</dt>
              <dd class="text-white">{{ sysInfo.db_type || '-' }}</dd>
            </div>
            <div class="flex items-center justify-between gap-4">
              <dt>端口</dt>
              <dd class="text-white">{{ sysInfo.server_port || 8022 }}</dd>
            </div>
          </dl>
        </article>
      </section>
    </div>

  </div>
</template>

<script setup>
import { computed, inject, ref, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { cockpitAPI, settingsAPI } from '@/api'
import PlatformIcon from '@/components/PlatformIcon.vue'
import { cockpitPlatforms } from '@/lib/platforms'

const notify = inject('notify')
const router = useRouter()

const loading = ref(false)
const overview = ref({ summary: {}, platforms: [], proxy: {} })
const sysInfo = ref({})

const summary = computed(() => overview.value.summary || {})
const proxy = computed(() => overview.value.proxy || {})
const platformCards = computed(() => {
  const statsMap = Object.fromEntries((overview.value.platforms || []).map((item) => [item.definition.id, item]))
  return cockpitPlatforms
    .map((platform) => ({
      ...platform,
      ...(statsMap[platform.id] || {
        accounts: 0,
        active_accounts: 0,
        active_account_email: '',
        instances: 0,
        running_instances: 0,
        wakeup_tasks: 0,
        enabled_wakeup_tasks: 0,
      }),
    }))
    .filter((platform) => Number(platform.accounts || 0) > 0)
})

onMounted(loadDashboard)

async function loadDashboard() {
  loading.value = true
  try {
    const [overviewData, systemData] = await Promise.all([
      cockpitAPI.overview(),
      settingsAPI.systemInfo(),
    ])
    overview.value = overviewData
    sysInfo.value = systemData
  } catch (error) {
    notify?.(error.message || '加载总览失败', 'error')
  } finally {
    loading.value = false
  }
}
</script>

<style scoped>
.stable-action-row {
  @apply flex items-center gap-2 overflow-x-auto pb-1;
}

.stable-action-row > * {
  @apply shrink-0 whitespace-nowrap;
}

.compact-platform-card {
  @apply rounded-xl border p-3 text-left transition-colors;
  background: var(--app-surface);
  border-color: var(--app-border);
  color: var(--app-text);
}

.compact-platform-card:hover {
  background: var(--app-surface-elevated);
  border-color: var(--app-accent-soft);
  box-shadow: 0 10px 24px var(--app-accent-shadow);
}

.compact-platform-stat {
  @apply min-w-0 rounded-lg border px-2 py-1.5;
  background: var(--app-surface-muted);
  border-color: var(--app-border-soft);
}

.codex-pool-stat {
  @apply min-w-0 rounded-lg border p-3;
  background: var(--app-surface-muted);
  border-color: var(--app-border-soft);
}
</style>
