<template>
  <div class="p-6 space-y-6">
    <section class="rounded-3xl border border-gray-800 bg-gradient-to-br from-amber-500/12 via-orange-400/6 to-gray-950 p-6">
      <div class="flex flex-col gap-4 lg:flex-row lg:items-end lg:justify-between">
        <div>
          <h1 class="text-3xl font-semibold text-white">唤醒任务中心</h1>
          <p class="mt-2 text-sm leading-6 text-gray-300">
            按 cockpit-tools 的任务台账方式统一管理唤醒计划、绑定账号和调度表达。
          </p>
        </div>
        <div class="flex gap-2">
          <button class="btn btn-secondary" :disabled="loading" @click="loadData">
            {{ loading ? '刷新中...' : '刷新任务' }}
          </button>
          <button class="btn btn-secondary" @click="exportAll">导出全部</button>
          <button class="btn btn-primary" @click="openModal()">新增任务</button>
        </div>
      </div>
    </section>

    <div class="grid gap-4 md:grid-cols-3">
      <div class="card p-4">
        <div class="text-xs uppercase tracking-wide text-gray-500">任务总数</div>
        <div class="mt-2 text-3xl font-semibold text-white">{{ tasks.length }}</div>
      </div>
      <div class="card p-4">
        <div class="text-xs uppercase tracking-wide text-gray-500">已启用</div>
        <div class="mt-2 text-3xl font-semibold text-white">{{ enabledCount }}</div>
      </div>
      <div class="card p-4">
        <div class="text-xs uppercase tracking-wide text-gray-500">覆盖平台</div>
        <div class="mt-2 text-3xl font-semibold text-white">{{ coveredPlatforms }}</div>
      </div>
    </div>

    <section class="card overflow-hidden">
      <div class="flex flex-col gap-3 border-b border-gray-800 px-5 py-4 lg:flex-row lg:items-center lg:justify-between">
        <div>
          <h2 class="text-lg font-semibold text-white">任务清单</h2>
          <p class="mt-1 text-sm text-gray-500">当前对外暴露的是统一任务模型，方便后续继续接入真实执行器。</p>
        </div>
        <div class="flex flex-col gap-2 sm:flex-row">
          <select v-model="platformFilter" class="input sm:w-44">
            <option value="all">全部平台</option>
            <option v-for="platform in wakeupPlatforms" :key="platform.id" :value="platform.id">
              {{ platform.label }}
            </option>
          </select>
          <select v-model="enabledFilter" class="input sm:w-40">
            <option value="all">全部状态</option>
            <option value="enabled">enabled</option>
            <option value="disabled">disabled</option>
          </select>
          <input v-model="search" class="input sm:w-64" placeholder="搜索任务名 / 调度 / 模型" />
        </div>
      </div>

      <div v-if="filteredTasks.length === 0" class="p-10 text-center text-sm text-gray-500">
        当前筛选条件下没有唤醒任务。
      </div>
      <div v-else class="overflow-x-auto">
        <table class="min-w-full text-sm">
          <thead class="bg-gray-900/80 text-left text-gray-400">
            <tr>
              <th class="px-5 py-3">任务</th>
              <th class="px-5 py-3">平台</th>
              <th class="px-5 py-3">账号</th>
              <th class="px-5 py-3">调度</th>
              <th class="px-5 py-3">状态</th>
              <th class="px-5 py-3 text-right">操作</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="task in filteredTasks" :key="task.id" class="border-t border-gray-800/80">
              <td class="px-5 py-4">
                <div class="font-medium text-white">{{ task.name }}</div>
                <div class="mt-1 text-xs text-gray-500">{{ task.model || '未指定模型' }}</div>
              </td>
              <td class="px-5 py-4 text-gray-300">{{ platformLabel(task.platform) }}</td>
              <td class="px-5 py-4 text-gray-300">{{ accountLabel(task.account_id) }}</td>
              <td class="px-5 py-4 text-gray-400">{{ scheduleText(task) }}</td>
              <td class="px-5 py-4">
                <span class="badge" :class="task.enabled ? 'badge-green' : 'badge-gray'">
                  {{ task.enabled ? 'enabled' : 'disabled' }}
                </span>
              </td>
              <td class="px-5 py-4">
                <div class="flex justify-end gap-2">
                  <button class="btn btn-secondary btn-xs" @click="openModal(task)">编辑</button>
                  <button class="btn btn-secondary btn-xs" @click="toggle(task)">
                    {{ task.enabled ? '停用' : '启用' }}
                  </button>
                  <button class="btn btn-danger btn-xs" @click="remove(task)">删除</button>
                </div>
              </td>
            </tr>
          </tbody>
        </table>
      </div>
    </section>

    <div v-if="showModal" class="modal-overlay" @click.self="closeModal">
      <div class="modal-content max-w-2xl">
        <div class="modal-header">
          <h3 class="text-white">{{ editing ? '编辑唤醒任务' : '新增唤醒任务' }}</h3>
          <button class="text-gray-500 hover:text-white" @click="closeModal">✕</button>
        </div>
        <div class="modal-body space-y-4">
          <div class="grid gap-4 md:grid-cols-2">
            <div>
              <label class="label">平台</label>
              <select v-model="form.platform" class="input">
                <option v-for="platform in wakeupPlatforms" :key="platform.id" :value="platform.id">
                  {{ platform.label }}
                </option>
              </select>
            </div>
            <div>
              <label class="label">任务名</label>
              <input v-model="form.name" class="input" placeholder="例如：工作日前置唤醒" />
            </div>
          </div>
          <div class="grid gap-4 md:grid-cols-2">
            <div>
              <label class="label">绑定账号</label>
              <select v-model="form.account_id" class="input">
                <option value="">不绑定</option>
                <option v-for="account in accountOptions" :key="account.id" :value="account.id">
                  {{ account.email }}
                </option>
              </select>
            </div>
            <div>
              <label class="label">模型</label>
              <input v-model="form.model" class="input" placeholder="gpt-5.4 / gemini-2.5-pro" />
            </div>
          </div>
          <div class="grid gap-4 md:grid-cols-2">
            <div>
              <label class="label">调度类型</label>
              <select v-model="form.schedule_type" class="input">
                <option value="daily">daily</option>
                <option value="weekly">weekly</option>
                <option value="interval">interval</option>
                <option value="manual">manual</option>
              </select>
            </div>
            <div>
              <label class="label">调度表达</label>
              <input v-model="form.schedule_value" class="input" placeholder="08:00 / Mon-Fri 09:00 / every 4h" />
            </div>
          </div>
          <div>
            <label class="label">Prompt</label>
            <textarea v-model="form.prompt" class="input min-h-28" placeholder="hi / warmup / keep session alive" />
          </div>
          <div class="flex items-center gap-2">
            <input v-model="form.enabled" type="checkbox" class="h-4 w-4 rounded border-gray-600 bg-gray-900 text-blue-500" />
            <label class="text-sm text-gray-300">保存后立即启用</label>
          </div>
        </div>
        <div class="modal-footer">
          <button class="btn btn-secondary" @click="closeModal">取消</button>
          <button class="btn btn-primary" @click="save">保存</button>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { computed, inject, onMounted, ref } from 'vue'
import { cockpitAPI } from '@/api'
import { getPlatformMeta, getWakeupPlatforms } from '@/lib/platforms'

const notify = inject('notify')
const wakeupPlatforms = getWakeupPlatforms()

const loading = ref(false)
const tasks = ref([])
const accounts = ref([])

const platformFilter = ref('all')
const enabledFilter = ref('all')
const search = ref('')

const showModal = ref(false)
const editing = ref(null)
const form = ref(createForm())

const accountMap = computed(() => Object.fromEntries(accounts.value.map((item) => [item.id, item])))
const accountOptions = computed(() => accounts.value.filter((item) => item.platform === form.value.platform))
const enabledCount = computed(() => tasks.value.filter((item) => item.enabled).length)
const coveredPlatforms = computed(() => new Set(tasks.value.map((item) => item.platform)).size)

const filteredTasks = computed(() => {
  const query = search.value.trim().toLowerCase()
  return tasks.value.filter((task) => {
    if (platformFilter.value !== 'all' && task.platform !== platformFilter.value) return false
    if (enabledFilter.value === 'enabled' && !task.enabled) return false
    if (enabledFilter.value === 'disabled' && task.enabled) return false
    if (!query) return true
    const text = [
      task.name,
      task.model,
      task.schedule_type,
      task.schedule_value,
      task.prompt,
      platformLabel(task.platform),
      accountLabel(task.account_id),
    ]
      .filter(Boolean)
      .join(' ')
      .toLowerCase()
    return text.includes(query)
  })
})

onMounted(loadData)

async function loadData() {
  loading.value = true
  try {
    const [taskData, accountData] = await Promise.all([
      cockpitAPI.listWakeupTasks(),
      cockpitAPI.listAllAccounts(),
    ])
    tasks.value = taskData
    accounts.value = accountData
  } catch (error) {
    notify?.(error.message || '加载任务失败', 'error')
  } finally {
    loading.value = false
  }
}

function createForm(task = null) {
  return {
    platform: task?.platform || wakeupPlatforms[0]?.id || 'antigravity',
    name: task?.name || '',
    account_id: task?.account_id || '',
    model: task?.model || '',
    prompt: task?.prompt || 'hi',
    schedule_type: task?.schedule_type || 'daily',
    schedule_value: task?.schedule_value || '08:00',
    enabled: task?.enabled ?? true,
  }
}

function openModal(task = null) {
  editing.value = task
  form.value = createForm(task)
  showModal.value = true
}

function closeModal() {
  editing.value = null
  form.value = createForm()
  showModal.value = false
}

async function save() {
  try {
    if (!form.value.name.trim()) {
      notify?.('请先填写任务名', 'error')
      return
    }
    const payload = {
      platform: form.value.platform,
      name: form.value.name.trim(),
      account_id: normalizeText(form.value.account_id),
      model: normalizeText(form.value.model),
      prompt: normalizeText(form.value.prompt),
      schedule_type: form.value.schedule_type || 'daily',
      schedule_value: form.value.schedule_value?.trim() || '08:00',
      enabled: !!form.value.enabled,
    }
    if (editing.value) {
      await cockpitAPI.updateWakeupTask(editing.value.id, payload)
    } else {
      await cockpitAPI.addWakeupTask(payload)
    }
    closeModal()
    notify?.('任务已保存', 'success')
    await loadData()
  } catch (error) {
    notify?.(error.message || '保存失败', 'error')
  }
}

async function toggle(task) {
  try {
    await cockpitAPI.toggleWakeupTask(task.id)
    notify?.(task.enabled ? '任务已停用' : '任务已启用', 'success')
    await loadData()
  } catch (error) {
    notify?.(error.message || '操作失败', 'error')
  }
}

async function remove(task) {
  if (!confirm(`确认删除任务 ${task.name} 吗？`)) return
  try {
    await cockpitAPI.deleteWakeupTask(task.id)
    notify?.('任务已删除', 'success')
    await loadData()
  } catch (error) {
    notify?.(error.message || '删除失败', 'error')
  }
}

function exportAll() {
  const text = JSON.stringify(tasks.value, null, 2)
  const blob = new Blob([text], { type: 'application/json;charset=utf-8' })
  const url = URL.createObjectURL(blob)
  const link = document.createElement('a')
  link.href = url
  link.download = 'easyllm-wakeup-tasks.json'
  link.click()
  URL.revokeObjectURL(url)
}

function normalizeText(value) {
  const text = typeof value === 'string' ? value.trim() : ''
  return text || null
}

function platformLabel(platformId) {
  return getPlatformMeta(platformId)?.label || platformId || '未知平台'
}

function accountLabel(accountId) {
  if (!accountId) return '未绑定账号'
  return accountMap.value[accountId]?.email || '账号已删除'
}

function scheduleText(task) {
  return `${task.schedule_type || 'daily'} · ${task.schedule_value || '未定义'}`
}
</script>
