<template>
  <div class="p-6 space-y-6">
    <section class="rounded-3xl border border-gray-800 bg-gradient-to-br from-sky-500/15 via-cyan-400/5 to-gray-950 p-6 shadow-2xl shadow-black/20">
      <div class="flex flex-col gap-6 lg:flex-row lg:items-end lg:justify-between">
        <div class="max-w-3xl space-y-3">
          <div class="inline-flex items-center gap-2 rounded-full border border-sky-500/20 bg-sky-500/10 px-3 py-1 text-xs text-sky-200">
            <span>📊</span>
            <span>EasyLLM</span>
          </div>
          <div>
            <h1 class="text-3xl font-semibold text-white">多平台驾驶舱总览</h1>
            <p class="mt-2 text-sm leading-6 text-gray-300">
              对齐 cockpit-tools 的信息组织方式，把账号、实例、唤醒任务和运行状态集中到同一块看板里。
            </p>
          </div>
        </div>

        <div class="flex flex-wrap items-center gap-2">
          <button class="btn btn-secondary" :disabled="loading" @click="loadDashboard">
            {{ loading ? '刷新中...' : '刷新总览' }}
          </button>
          <router-link to="/instances" class="btn btn-secondary">查看实例</router-link>
          <router-link to="/wakeup" class="btn btn-primary">查看唤醒</router-link>
        </div>
      </div>
    </section>

    <div class="grid gap-4 md:grid-cols-2 xl:grid-cols-4">
      <div class="card p-5">
        <div class="text-xs uppercase tracking-wide text-gray-500">平台覆盖</div>
        <div class="mt-3 text-3xl font-semibold text-white">{{ summary.total_platforms || cockpitPlatforms.length }}</div>
        <div class="mt-1 text-sm text-gray-400">{{ summary.enabled_platforms || 0 }} 个已有数据</div>
      </div>
      <div class="card p-5">
        <div class="text-xs uppercase tracking-wide text-gray-500">账号总数</div>
        <div class="mt-3 text-3xl font-semibold text-white">{{ summary.total_accounts || 0 }}</div>
        <div class="mt-1 text-sm text-gray-400">{{ summary.active_accounts || 0 }} 个当前激活</div>
      </div>
      <div class="card p-5">
        <div class="text-xs uppercase tracking-wide text-gray-500">实例总数</div>
        <div class="mt-3 text-3xl font-semibold text-white">{{ summary.total_instances || 0 }}</div>
        <div class="mt-1 text-sm text-gray-400">{{ summary.running_instances || 0 }} 个运行中</div>
      </div>
      <div class="card p-5">
        <div class="text-xs uppercase tracking-wide text-gray-500">唤醒任务</div>
        <div class="mt-3 text-3xl font-semibold text-white">{{ summary.total_wakeup_tasks || 0 }}</div>
        <div class="mt-1 text-sm text-gray-400">{{ summary.enabled_wakeup_tasks || 0 }} 个已启用</div>
      </div>
    </div>

    <div class="grid gap-6 xl:grid-cols-[1.8fr_1fr]">
      <section class="space-y-4">
        <div class="flex items-center justify-between">
          <div>
            <h2 class="text-lg font-semibold text-white">平台矩阵</h2>
            <p class="mt-1 text-sm text-gray-500">每个平台卡片都汇总了账号、实例、激活账号和唤醒状态。</p>
          </div>
        </div>

        <div class="grid gap-4 md:grid-cols-2 2xl:grid-cols-3">
          <button
            v-for="platform in platformCards"
            :key="platform.id"
            class="card overflow-hidden text-left transition-transform hover:-translate-y-0.5"
            @click="router.push(platform.route)"
          >
            <div class="border-b border-gray-800 bg-gray-900/70 px-5 py-4">
              <div class="flex items-center justify-between">
                <div class="flex items-center gap-3">
                  <div class="text-2xl">{{ platform.icon }}</div>
                  <div>
                    <div class="font-medium text-white">{{ platform.label }}</div>
                    <div class="text-xs text-gray-500">{{ platform.description }}</div>
                  </div>
                </div>
                <span class="badge" :class="platform.managementMode === 'legacy' ? 'badge-yellow' : 'badge-blue'">
                  {{ platform.managementMode === 'legacy' ? 'legacy' : 'cockpit' }}
                </span>
              </div>
            </div>

            <div class="grid grid-cols-2 gap-3 p-5 text-sm">
              <div>
                <div class="text-xs uppercase tracking-wide text-gray-500">账号</div>
                <div class="mt-1 text-xl font-semibold text-white">{{ platform.accounts }}</div>
                <div class="mt-1 text-xs text-gray-500">{{ platform.active_accounts }} 激活</div>
              </div>
              <div>
                <div class="text-xs uppercase tracking-wide text-gray-500">实例</div>
                <div class="mt-1 text-xl font-semibold text-white">{{ platform.instances }}</div>
                <div class="mt-1 text-xs text-gray-500">{{ platform.running_instances }} 运行中</div>
              </div>
              <div class="col-span-2">
                <div class="text-xs uppercase tracking-wide text-gray-500">当前账号</div>
                <div class="mt-1 truncate text-sm text-gray-200">{{ platform.active_account_email || '未指定' }}</div>
              </div>
              <div class="col-span-2 flex items-center justify-between rounded-xl border border-gray-800 bg-gray-950/60 px-3 py-2">
                <span class="text-xs text-gray-500">唤醒任务</span>
                <span class="text-sm text-white">{{ platform.enabled_wakeup_tasks }} / {{ platform.wakeup_tasks }}</span>
              </div>
            </div>
          </button>
        </div>
      </section>

      <section class="space-y-4">
        <article class="card p-5">
          <div class="flex items-center justify-between">
            <div>
              <h2 class="text-lg font-semibold text-white">Codex 代理池</h2>
              <p class="mt-1 text-sm text-gray-500">沿用 EasyLLM 当前代理池与请求分发能力。</p>
            </div>
            <span class="badge" :class="proxy.enabled ? 'badge-green' : 'badge-red'">
              {{ proxy.enabled ? '已启用' : '已关闭' }}
            </span>
          </div>

          <div class="mt-4 grid grid-cols-2 gap-3">
            <div class="rounded-2xl border border-gray-800 bg-gray-950/60 p-4">
              <div class="text-xs text-gray-500">策略</div>
              <div class="mt-2 text-lg font-semibold text-white">{{ proxy.strategy || 'round_robin' }}</div>
            </div>
            <div class="rounded-2xl border border-gray-800 bg-gray-950/60 p-4">
              <div class="text-xs text-gray-500">账号池</div>
              <div class="mt-2 text-lg font-semibold text-white">{{ proxy.enabled_accounts || 0 }} / {{ proxy.accounts || 0 }}</div>
            </div>
            <div class="col-span-2 rounded-2xl border border-gray-800 bg-gray-950/60 p-4">
              <div class="text-xs text-gray-500">累计请求</div>
              <div class="mt-2 text-2xl font-semibold text-white">{{ proxy.total_requests || 0 }}</div>
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

    <div class="grid gap-6 xl:grid-cols-2">
      <section class="card p-5">
        <div class="flex items-center justify-between">
          <div>
            <h2 class="text-lg font-semibold text-white">运行中实例</h2>
            <p class="mt-1 text-sm text-gray-500">优先展示状态为 running 的实例。</p>
          </div>
          <router-link to="/instances" class="text-sm text-blue-400 hover:text-blue-300">进入实例页</router-link>
        </div>

        <div v-if="runningInstanceItems.length === 0" class="mt-6 text-sm text-gray-500">
          当前没有 running 实例。
        </div>
        <div v-else class="mt-5 space-y-3">
          <div
            v-for="instance in runningInstanceItems"
            :key="instance.id"
            class="rounded-2xl border border-gray-800 bg-gray-950/60 px-4 py-3"
          >
            <div class="flex items-center justify-between gap-4">
              <div>
                <div class="text-sm font-medium text-white">{{ instance.name }}</div>
                <div class="mt-1 text-xs text-gray-500">
                  {{ platformLabel(instance.platform) }} · {{ accountLabel(instance.account_id) }}
                </div>
              </div>
              <span class="badge badge-green">running</span>
            </div>
          </div>
        </div>
      </section>

      <section class="card p-5">
        <div class="flex items-center justify-between">
          <div>
            <h2 class="text-lg font-semibold text-white">已启用唤醒任务</h2>
            <p class="mt-1 text-sm text-gray-500">展示当前启用中的任务与调度信息。</p>
          </div>
          <router-link to="/wakeup" class="text-sm text-blue-400 hover:text-blue-300">进入唤醒页</router-link>
        </div>

        <div v-if="enabledWakeupItems.length === 0" class="mt-6 text-sm text-gray-500">
          当前没有启用中的唤醒任务。
        </div>
        <div v-else class="mt-5 space-y-3">
          <div
            v-for="task in enabledWakeupItems"
            :key="task.id"
            class="rounded-2xl border border-gray-800 bg-gray-950/60 px-4 py-3"
          >
            <div class="flex items-center justify-between gap-4">
              <div>
                <div class="text-sm font-medium text-white">{{ task.name }}</div>
                <div class="mt-1 text-xs text-gray-500">
                  {{ platformLabel(task.platform) }} · {{ task.schedule_type }} · {{ task.schedule_value }}
                </div>
              </div>
              <span class="badge badge-green">enabled</span>
            </div>
          </div>
        </div>
      </section>
    </div>
  </div>
</template>

<script setup>
import { computed, inject, ref, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { cockpitAPI, settingsAPI } from '@/api'
import { cockpitPlatforms, getPlatformMeta } from '@/lib/platforms'

const notify = inject('notify')
const router = useRouter()

const loading = ref(false)
const overview = ref({ summary: {}, platforms: [], proxy: {} })
const sysInfo = ref({})
const instances = ref([])
const wakeupTasks = ref([])
const allAccounts = ref([])

const summary = computed(() => overview.value.summary || {})
const proxy = computed(() => overview.value.proxy || {})
const platformCards = computed(() => {
  const statsMap = Object.fromEntries((overview.value.platforms || []).map((item) => [item.definition.id, item]))
  return cockpitPlatforms.map((platform) => ({
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
})

const runningInstanceItems = computed(() => instances.value.filter((item) => item.state === 'running').slice(0, 6))
const enabledWakeupItems = computed(() => wakeupTasks.value.filter((item) => item.enabled).slice(0, 6))

onMounted(loadDashboard)

async function loadDashboard() {
  loading.value = true
  try {
    const [overviewData, systemData, instanceData, wakeupData, accountData] = await Promise.all([
      cockpitAPI.overview(),
      settingsAPI.systemInfo(),
      cockpitAPI.listAllInstances(),
      cockpitAPI.listWakeupTasks(),
      cockpitAPI.listAllAccounts(),
    ])
    overview.value = overviewData
    sysInfo.value = systemData
    instances.value = instanceData
    wakeupTasks.value = wakeupData
    allAccounts.value = accountData
  } catch (error) {
    notify?.(error.message || '加载总览失败', 'error')
  } finally {
    loading.value = false
  }
}

function platformLabel(platformId) {
  return getPlatformMeta(platformId)?.label || platformId || '未知平台'
}

function accountLabel(accountId) {
  if (!accountId) return '未绑定账号'
  return allAccounts.value.find((item) => item.id === accountId)?.email || '账号已删除'
}
</script>
