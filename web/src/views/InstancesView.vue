<template>
  <div class="p-6 space-y-6">
    <section class="rounded-3xl border border-gray-800 bg-gradient-to-br from-emerald-500/12 via-teal-400/6 to-gray-950 p-6">
      <div class="flex flex-col gap-4 lg:flex-row lg:items-end lg:justify-between">
        <div>
          <h1 class="text-3xl font-semibold text-white">全局实例中心</h1>
          <p class="mt-2 text-sm leading-6 text-gray-300">
            统一查看和维护各个平台的多开实例，沿用 cockpit-tools 的实例编排思路。
          </p>
        </div>
        <div class="flex gap-2">
          <button class="btn btn-secondary" :disabled="loading" @click="loadData">
            {{ loading ? '刷新中...' : '刷新实例' }}
          </button>
          <button class="btn btn-secondary" @click="exportAll">导出全部</button>
          <button class="btn btn-primary" @click="openModal()">新增实例</button>
        </div>
      </div>
    </section>

    <div class="grid gap-4 md:grid-cols-3">
      <div class="card p-4">
        <div class="text-xs uppercase tracking-wide text-gray-500">实例总数</div>
        <div class="mt-2 text-3xl font-semibold text-white">{{ instances.length }}</div>
      </div>
      <div class="card p-4">
        <div class="text-xs uppercase tracking-wide text-gray-500">运行中</div>
        <div class="mt-2 text-3xl font-semibold text-white">{{ runningCount }}</div>
      </div>
      <div class="card p-4">
        <div class="text-xs uppercase tracking-wide text-gray-500">自动启动</div>
        <div class="mt-2 text-3xl font-semibold text-white">{{ autoStartCount }}</div>
      </div>
    </div>

    <section class="card overflow-hidden">
      <div class="flex flex-col gap-3 border-b border-gray-800 px-5 py-4 lg:flex-row lg:items-center lg:justify-between">
        <div>
          <h2 class="text-lg font-semibold text-white">实例清单</h2>
          <p class="mt-1 text-sm text-gray-500">按平台和状态筛选，多平台共用同一张实例表。</p>
        </div>
        <div class="flex flex-col gap-2 sm:flex-row">
          <select v-model="platformFilter" class="input sm:w-44">
            <option value="all">全部平台</option>
            <option v-for="platform in instancePlatforms" :key="platform.id" :value="platform.id">
              {{ platform.label }}
            </option>
          </select>
          <select v-model="stateFilter" class="input sm:w-40">
            <option value="all">全部状态</option>
            <option value="running">running</option>
            <option value="stopped">stopped</option>
            <option value="paused">paused</option>
          </select>
          <input v-model="search" class="input sm:w-64" placeholder="搜索实例名 / 路径 / 账号" />
        </div>
      </div>

      <div v-if="filteredInstances.length === 0" class="p-10 text-center text-sm text-gray-500">
        当前筛选条件下没有实例。
      </div>
      <div v-else class="overflow-x-auto">
        <table class="min-w-full text-sm">
          <thead class="bg-gray-900/80 text-left text-gray-400">
            <tr>
              <th class="px-5 py-3">实例</th>
              <th class="px-5 py-3">平台</th>
              <th class="px-5 py-3">账号</th>
              <th class="px-5 py-3">目录</th>
              <th class="px-5 py-3">状态</th>
              <th class="px-5 py-3 text-right">操作</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="instance in filteredInstances" :key="instance.id" class="border-t border-gray-800/80">
              <td class="px-5 py-4">
                <div class="font-medium text-white">{{ instance.name }}</div>
                <div class="mt-1 text-xs text-gray-500">{{ instance.launch_args || '默认启动参数' }}</div>
              </td>
              <td class="px-5 py-4 text-gray-300">{{ platformLabel(instance.platform) }}</td>
              <td class="px-5 py-4 text-gray-300">{{ accountLabel(instance.account_id) }}</td>
              <td class="px-5 py-4 text-gray-400">
                <div class="truncate max-w-xs">{{ instance.workspace_dir || '未配置工作目录' }}</div>
                <div class="mt-1 truncate max-w-xs text-xs text-gray-600">{{ instance.user_data_dir || '未配置用户目录' }}</div>
              </td>
              <td class="px-5 py-4">
                <span class="badge" :class="instance.state === 'running' ? 'badge-green' : instance.state === 'paused' ? 'badge-yellow' : 'badge-gray'">
                  {{ instance.state || 'stopped' }}
                </span>
              </td>
              <td class="px-5 py-4">
                <div class="flex justify-end gap-2">
                  <button class="btn btn-secondary btn-xs" @click="setState(instance, instance.state === 'running' ? 'paused' : 'running')">
                    {{ instance.state === 'running' ? '暂停' : '启动' }}
                  </button>
                  <button v-if="instance.state !== 'stopped'" class="btn btn-secondary btn-xs" @click="setState(instance, 'stopped')">停止</button>
                  <button class="btn btn-secondary btn-xs" @click="openModal(instance)">编辑</button>
                  <button class="btn btn-danger btn-xs" @click="remove(instance)">删除</button>
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
          <h3 class="text-white">{{ editing ? '编辑实例' : '新增实例' }}</h3>
          <button class="text-gray-500 hover:text-white" @click="closeModal">✕</button>
        </div>
        <div class="modal-body space-y-4">
          <div class="grid gap-4 md:grid-cols-2">
            <div>
              <label class="label">平台</label>
              <select v-model="form.platform" class="input" :disabled="!!editing">
                <option v-for="platform in instancePlatforms" :key="platform.id" :value="platform.id">
                  {{ platform.label }}
                </option>
              </select>
            </div>
            <div>
              <label class="label">实例名称</label>
              <input v-model="form.name" class="input" placeholder="例如：主开发窗口" />
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
              <label class="label">状态</label>
              <select v-model="form.state" class="input">
                <option value="stopped">stopped</option>
                <option value="running">running</option>
                <option value="paused">paused</option>
              </select>
            </div>
          </div>
          <div class="grid gap-4 md:grid-cols-2">
            <div>
              <label class="label">工作目录</label>
              <input v-model="form.workspace_dir" class="input" placeholder="/path/to/project" />
            </div>
            <div>
              <label class="label">用户目录</label>
              <input v-model="form.user_data_dir" class="input" placeholder="/path/to/user-data" />
            </div>
          </div>
          <div class="grid gap-4 md:grid-cols-3">
            <div>
              <label class="label">PID</label>
              <input v-model="form.pid" class="input" placeholder="选填" />
            </div>
            <div class="md:col-span-2 flex items-end">
              <label class="flex items-center gap-2 text-sm text-gray-300">
                <input v-model="form.auto_start" type="checkbox" class="h-4 w-4 rounded border-gray-600 bg-gray-900 text-blue-500" />
                启动 EasyLLM 后自动拉起
              </label>
            </div>
          </div>
          <div>
            <label class="label">启动参数</label>
            <textarea v-model="form.launch_args" class="input min-h-24 font-mono text-xs" placeholder="--new-window --disable-extensions" />
          </div>
          <div>
            <label class="label">备注</label>
            <textarea v-model="form.notes" class="input min-h-24" placeholder="记录这个实例的用途、策略或项目说明" />
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
import { getInstancePlatforms, getPlatformMeta } from '@/lib/platforms'

const notify = inject('notify')

const instancePlatforms = getInstancePlatforms()
const loading = ref(false)
const instances = ref([])
const accounts = ref([])

const platformFilter = ref('all')
const stateFilter = ref('all')
const search = ref('')

const showModal = ref(false)
const editing = ref(null)
const form = ref(createForm())

const runningCount = computed(() => instances.value.filter((item) => item.state === 'running').length)
const autoStartCount = computed(() => instances.value.filter((item) => item.auto_start).length)

const accountMap = computed(() => Object.fromEntries(accounts.value.map((item) => [item.id, item])))
const accountOptions = computed(() => accounts.value.filter((item) => item.platform === form.value.platform))

const filteredInstances = computed(() => {
  const query = search.value.trim().toLowerCase()
  return instances.value.filter((instance) => {
    if (platformFilter.value !== 'all' && instance.platform !== platformFilter.value) return false
    if (stateFilter.value !== 'all' && instance.state !== stateFilter.value) return false
    if (!query) return true
    const text = [
      instance.name,
      instance.workspace_dir,
      instance.user_data_dir,
      instance.launch_args,
      accountLabel(instance.account_id),
      platformLabel(instance.platform),
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
    const [instanceData, accountData] = await Promise.all([
      cockpitAPI.listAllInstances(),
      cockpitAPI.listAllAccounts(),
    ])
    instances.value = instanceData
    accounts.value = accountData
  } catch (error) {
    notify?.(error.message || '加载实例失败', 'error')
  } finally {
    loading.value = false
  }
}

function createForm(instance = null) {
  return {
    platform: instance?.platform || instancePlatforms[0]?.id || 'antigravity',
    name: instance?.name || '',
    account_id: instance?.account_id || '',
    workspace_dir: instance?.workspace_dir || '',
    user_data_dir: instance?.user_data_dir || '',
    state: instance?.state || 'stopped',
    pid: instance?.pid ?? '',
    auto_start: !!instance?.auto_start,
    launch_args: instance?.launch_args || '',
    notes: instance?.notes || '',
  }
}

function openModal(instance = null) {
  editing.value = instance
  form.value = createForm(instance)
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
      notify?.('请先填写实例名称', 'error')
      return
    }
    const targetPlatform = editing.value ? editing.value.platform : form.value.platform
    const payload = {
      name: form.value.name.trim(),
      account_id: normalizeText(form.value.account_id),
      workspace_dir: normalizeText(form.value.workspace_dir),
      user_data_dir: normalizeText(form.value.user_data_dir),
      state: form.value.state || 'stopped',
      pid: normalizeInteger(form.value.pid),
      auto_start: !!form.value.auto_start,
      launch_args: normalizeText(form.value.launch_args),
      notes: normalizeText(form.value.notes),
    }
    if (editing.value) {
      await cockpitAPI.updatePlatformInstance(targetPlatform, editing.value.id, payload)
    } else {
      await cockpitAPI.addPlatformInstance(targetPlatform, payload)
    }
    closeModal()
    notify?.('实例已保存', 'success')
    await loadData()
  } catch (error) {
    notify?.(error.message || '保存失败', 'error')
  }
}

async function remove(instance) {
  if (!confirm(`确认删除实例 ${instance.name} 吗？`)) return
  try {
    await cockpitAPI.deletePlatformInstance(instance.platform, instance.id)
    notify?.('实例已删除', 'success')
    await loadData()
  } catch (error) {
    notify?.(error.message || '删除失败', 'error')
  }
}

async function setState(instance, state) {
  try {
    await cockpitAPI.updatePlatformInstanceState(instance.platform, instance.id, state)
    notify?.(`实例状态已切换为 ${state}`, 'success')
    await loadData()
  } catch (error) {
    notify?.(error.message || '状态切换失败', 'error')
  }
}

function exportAll() {
  const text = JSON.stringify(instances.value, null, 2)
  const blob = new Blob([text], { type: 'application/json;charset=utf-8' })
  const url = URL.createObjectURL(blob)
  const link = document.createElement('a')
  link.href = url
  link.download = 'easyllm-instances.json'
  link.click()
  URL.revokeObjectURL(url)
}

function normalizeText(value) {
  const text = typeof value === 'string' ? value.trim() : ''
  return text || null
}

function normalizeInteger(value) {
  if (value === '' || value == null) return null
  const num = Number(value)
  return Number.isFinite(num) ? Math.trunc(num) : null
}

function platformLabel(platformId) {
  return getPlatformMeta(platformId)?.label || platformId || '未知平台'
}

function accountLabel(accountId) {
  if (!accountId) return '未绑定账号'
  return accountMap.value[accountId]?.email || '账号已删除'
}
</script>
