<template>
  <div class="p-6 space-y-6">
    <section class="rounded-3xl border border-gray-800 bg-gradient-to-br from-slate-200/10 via-gray-200/5 to-gray-950 p-6">
      <h1 class="text-3xl font-semibold text-white">设置中心</h1>
      <p class="mt-2 max-w-3xl text-sm leading-6 text-gray-300">
        集中管理通用设置、自动刷新、启动路径和运行状态，同时保留代理、数据库和安全配置能力。
      </p>
    </section>

    <div class="flex flex-wrap gap-2">
      <button v-for="tab in tabs" :key="tab.id" class="btn btn-sm" :class="activeTab === tab.id ? 'btn-primary' : 'btn-secondary'" @click="activeTab = tab.id">
        {{ tab.label }}
      </button>
    </div>

    <section v-if="activeTab === 'experience'" class="card p-5 space-y-5">
      <div>
        <h2 class="text-lg font-semibold text-white">通用体验</h2>
        <p class="mt-1 text-sm text-gray-500">统一配置语言、主题、关闭行为与切号联动策略。</p>
      </div>

      <div class="grid gap-4 md:grid-cols-3">
        <div>
          <label class="label">语言</label>
          <select v-model="general.language" class="input">
            <option value="zh-CN">简体中文</option>
            <option value="en">English</option>
            <option value="ja">日本語</option>
          </select>
        </div>
        <div>
          <label class="label">主题</label>
          <select v-model="general.theme" class="input">
            <option value="system">跟随系统</option>
            <option value="dark">暗色</option>
            <option value="light">亮色</option>
          </select>
        </div>
        <div>
          <label class="label">关闭行为</label>
          <select v-model="general.close_behavior" class="input">
            <option value="ask">每次询问</option>
            <option value="minimize">最小化到后台</option>
            <option value="quit">直接退出</option>
          </select>
        </div>
      </div>

      <div class="grid gap-4 md:grid-cols-2">
        <label class="flex items-center justify-between rounded-2xl border border-gray-800 bg-gray-950/60 px-4 py-3">
          <div>
            <div class="text-sm font-medium text-white">隐私模式</div>
            <div class="mt-1 text-xs text-gray-500">在总览和列表里优先显示脱敏信息。</div>
          </div>
          <input v-model="general.privacy_mode" type="checkbox" class="h-4 w-4 rounded border-gray-600 bg-gray-900 text-blue-500" />
        </label>

        <label class="flex items-center justify-between rounded-2xl border border-gray-800 bg-gray-950/60 px-4 py-3">
          <div>
            <div class="text-sm font-medium text-white">切换 Codex 时自动拉起</div>
            <div class="mt-1 text-xs text-gray-500">切号时自动拉起对应的客户端。</div>
          </div>
          <input v-model="general.integrations.codex_launch_on_switch" type="checkbox" class="h-4 w-4 rounded border-gray-600 bg-gray-900 text-blue-500" />
        </label>

        <label class="flex items-center justify-between rounded-2xl border border-gray-800 bg-gray-950/60 px-4 py-3">
          <div>
            <div class="text-sm font-medium text-white">同步 OpenCode 账号</div>
            <div class="mt-1 text-xs text-gray-500">切换后同步 OpenCode 本地认证状态。</div>
          </div>
          <input v-model="general.integrations.opencode_sync_on_switch" type="checkbox" class="h-4 w-4 rounded border-gray-600 bg-gray-900 text-blue-500" />
        </label>

        <label class="flex items-center justify-between rounded-2xl border border-gray-800 bg-gray-950/60 px-4 py-3">
          <div>
            <div class="text-sm font-medium text-white">覆盖 OpenCode auth</div>
            <div class="mt-1 text-xs text-gray-500">同步时覆盖现有 auth 文件。</div>
          </div>
          <input v-model="general.integrations.opencode_auth_overwrite_on_switch" type="checkbox" class="h-4 w-4 rounded border-gray-600 bg-gray-900 text-blue-500" />
        </label>
      </div>

      <div class="flex justify-end">
        <button class="btn btn-primary" @click="saveGeneralSettings">保存通用设置</button>
      </div>
    </section>

    <section v-else-if="activeTab === 'refresh'" class="card p-5 space-y-5">
      <div>
        <h2 class="text-lg font-semibold text-white">自动刷新策略</h2>
        <p class="mt-1 text-sm text-gray-500">全局默认值会作为各平台的兜底间隔，支持单独覆盖。</p>
      </div>

      <div class="grid gap-4 md:grid-cols-3">
        <div>
          <label class="label">全局默认刷新分钟数</label>
          <input v-model.number="general.auto_refresh_minutes" type="number" min="1" class="input" />
        </div>
      </div>

      <div class="grid gap-4 md:grid-cols-2 xl:grid-cols-3">
        <div v-for="platform in cockpitPlatforms" :key="platform.id" class="rounded-2xl border border-gray-800 bg-gray-950/60 p-4">
          <div class="flex items-center gap-2">
            <span class="text-lg">{{ platform.icon }}</span>
            <span class="font-medium text-white">{{ platform.label }}</span>
          </div>
          <label class="label mt-4">刷新分钟数</label>
          <input
            v-model.number="general.refresh_intervals[platform.id]"
            type="number"
            min="1"
            class="input"
          />
        </div>
      </div>

      <div class="flex justify-end">
        <button class="btn btn-primary" @click="saveGeneralSettings">保存刷新策略</button>
      </div>
    </section>

    <section v-else-if="activeTab === 'paths'" class="card p-5 space-y-5">
      <div>
        <h2 class="text-lg font-semibold text-white">启动路径</h2>
        <p class="mt-1 text-sm text-gray-500">集中管理各客户端的可执行文件路径，用于环境联动拉起。</p>
      </div>

      <div class="grid gap-4 md:grid-cols-2">
        <div v-for="entry in pathEntries" :key="entry.key">
          <label class="label">{{ entry.label }}</label>
          <input v-model="general.app_paths[entry.key]" class="input" :placeholder="entry.placeholder" />
        </div>
      </div>

      <div class="flex justify-end">
        <button class="btn btn-primary" @click="saveGeneralSettings">保存路径配置</button>
      </div>
    </section>

    <section v-else-if="activeTab === 'runtime'" class="space-y-6">
      <div class="grid gap-6 xl:grid-cols-[1.2fr_1fr]">
        <article class="card p-5">
          <div>
            <h2 class="text-lg font-semibold text-white">运行状态</h2>
            <p class="mt-1 text-sm text-gray-500">当前服务和数据面的一些关键指标。</p>
          </div>
          <dl class="mt-5 grid gap-3 text-sm text-gray-400 md:grid-cols-2">
            <div class="flex items-center justify-between gap-4 rounded-2xl border border-gray-800 bg-gray-950/60 px-4 py-3">
              <dt>版本</dt>
              <dd class="text-white">v{{ sysInfo.version || '-' }}</dd>
            </div>
            <div class="flex items-center justify-between gap-4 rounded-2xl border border-gray-800 bg-gray-950/60 px-4 py-3">
              <dt>运行时间</dt>
              <dd class="text-white">{{ sysInfo.uptime || '-' }}</dd>
            </div>
            <div class="flex items-center justify-between gap-4 rounded-2xl border border-gray-800 bg-gray-950/60 px-4 py-3">
              <dt>数据库</dt>
              <dd class="text-white">{{ sysInfo.db_type || '-' }}</dd>
            </div>
            <div class="flex items-center justify-between gap-4 rounded-2xl border border-gray-800 bg-gray-950/60 px-4 py-3">
              <dt>端口</dt>
              <dd class="text-white">{{ sysInfo.server_port || 8022 }}</dd>
            </div>
            <div class="flex items-center justify-between gap-4 rounded-2xl border border-gray-800 bg-gray-950/60 px-4 py-3">
              <dt>Goroutines</dt>
              <dd class="text-white">{{ sysInfo.goroutines || '-' }}</dd>
            </div>
            <div class="flex items-center justify-between gap-4 rounded-2xl border border-gray-800 bg-gray-950/60 px-4 py-3">
              <dt>内存</dt>
              <dd class="text-white">{{ sysInfo.memory_alloc_mb || '-' }} MB</dd>
            </div>
          </dl>
        </article>

        <article class="card p-5">
          <div>
            <h2 class="text-lg font-semibold text-white">总览摘要</h2>
            <p class="mt-1 text-sm text-gray-500">从统一平台总览接口拉取的实时摘要。</p>
          </div>
          <div class="mt-5 grid grid-cols-2 gap-3">
            <div class="rounded-2xl border border-gray-800 bg-gray-950/60 p-4">
              <div class="text-xs text-gray-500">账号</div>
              <div class="mt-2 text-2xl font-semibold text-white">{{ overview.summary?.total_accounts || 0 }}</div>
            </div>
            <div class="rounded-2xl border border-gray-800 bg-gray-950/60 p-4">
              <div class="text-xs text-gray-500">实例</div>
              <div class="mt-2 text-2xl font-semibold text-white">{{ overview.summary?.total_instances || 0 }}</div>
            </div>
            <div class="rounded-2xl border border-gray-800 bg-gray-950/60 p-4">
              <div class="text-xs text-gray-500">唤醒</div>
              <div class="mt-2 text-2xl font-semibold text-white">{{ overview.summary?.total_wakeup_tasks || 0 }}</div>
            </div>
            <div class="rounded-2xl border border-gray-800 bg-gray-950/60 p-4">
              <div class="text-xs text-gray-500">Codex 代理</div>
              <div class="mt-2 text-2xl font-semibold text-white">{{ overview.proxy?.enabled_accounts || 0 }} / {{ overview.proxy?.accounts || 0 }}</div>
            </div>
          </div>
        </article>
      </div>

      <article class="card p-5 space-y-5">
        <div>
          <h2 class="text-lg font-semibold text-white">系统开关与连接</h2>
          <p class="mt-1 text-sm text-gray-500">保留 EasyLLM 自身的网络和数据库配置能力。</p>
        </div>

        <div class="grid gap-4 md:grid-cols-3">
          <label class="flex items-center justify-between rounded-2xl border border-gray-800 bg-gray-950/60 px-4 py-3">
            <div>
              <div class="text-sm font-medium text-white">日志记录</div>
              <div class="mt-1 text-xs text-gray-500">记录代理请求与管理操作。</div>
            </div>
            <input v-model="switches.log_enabled" type="checkbox" class="h-4 w-4 rounded border-gray-600 bg-gray-900 text-blue-500" />
          </label>
          <label class="flex items-center justify-between rounded-2xl border border-gray-800 bg-gray-950/60 px-4 py-3">
            <div>
              <div class="text-sm font-medium text-white">IP 黑名单</div>
              <div class="mt-1 text-xs text-gray-500">限制指定来源访问代理接口。</div>
            </div>
            <input v-model="switches.ip_blacklist_enabled" type="checkbox" class="h-4 w-4 rounded border-gray-600 bg-gray-900 text-blue-500" />
          </label>
          <label class="flex items-center justify-between rounded-2xl border border-gray-800 bg-gray-950/60 px-4 py-3">
            <div>
              <div class="text-sm font-medium text-white">HTTP 代理</div>
              <div class="mt-1 text-xs text-gray-500">上游请求统一经过代理服务器。</div>
            </div>
            <input v-model="switches.proxy_enabled" type="checkbox" class="h-4 w-4 rounded border-gray-600 bg-gray-900 text-blue-500" />
          </label>
        </div>

        <div class="grid gap-4 md:grid-cols-2">
          <div>
            <label class="label">代理主机</label>
            <input v-model="proxy.host" class="input" placeholder="127.0.0.1" />
          </div>
          <div>
            <label class="label">代理端口</label>
            <input v-model.number="proxy.port" type="number" class="input" placeholder="7890" />
          </div>
          <div>
            <label class="label">数据库类型</label>
            <select v-model="database.type" class="input">
              <option value="sqlite">SQLite</option>
              <option value="postgres">PostgreSQL</option>
            </select>
          </div>
          <div>
            <label class="label">数据库路径 / DSN</label>
            <input
              v-model="databasePathOrDSN"
              class="input"
              :placeholder="database.type === 'sqlite' ? './data/easyllm.db' : 'postgres dsn'"
            />
          </div>
        </div>

        <div class="flex justify-end gap-2">
          <button class="btn btn-secondary" @click="saveSwitches">保存开关</button>
          <button class="btn btn-secondary" @click="saveProxy">保存代理</button>
          <button class="btn btn-primary" @click="saveDatabase">保存数据库</button>
        </div>
      </article>
    </section>

    <section v-else class="card p-5 space-y-5">
      <div>
        <h2 class="text-lg font-semibold text-white">访问安全</h2>
        <p class="mt-1 text-sm text-gray-500">继续保留 EasyLLM 自身的登录密码能力。</p>
      </div>

      <div class="max-w-xl space-y-4">
        <div v-if="passwordSet">
          <label class="label">当前密码</label>
          <input v-model="pwForm.oldPassword" type="password" class="input" placeholder="输入当前密码" />
        </div>
        <div>
          <label class="label">{{ passwordSet ? '新密码' : '设置密码' }}</label>
          <input v-model="pwForm.newPassword" type="password" class="input" placeholder="至少 4 位" />
        </div>
        <div>
          <label class="label">确认密码</label>
          <input v-model="pwForm.confirmPassword" type="password" class="input" placeholder="再次输入" />
        </div>
        <div v-if="pwError" class="rounded-2xl border border-red-700 bg-red-900/20 px-4 py-3 text-sm text-red-300">
          {{ pwError }}
        </div>
      </div>

      <div class="flex justify-end">
        <button class="btn btn-primary" :disabled="pwSaving" @click="savePassword">
          {{ pwSaving ? '保存中...' : passwordSet ? '修改密码' : '设置密码' }}
        </button>
      </div>
    </section>
  </div>
</template>

<script setup>
import { computed, inject, onMounted, ref } from 'vue'
import { authAPI, cockpitAPI, settingsAPI } from '@/api'
import { cockpitPlatforms } from '@/lib/platforms'

const notify = inject('notify')

const tabs = [
  { id: 'experience', label: '通用体验' },
  { id: 'refresh', label: '自动刷新' },
  { id: 'paths', label: '启动路径' },
  { id: 'runtime', label: '运行状态' },
  { id: 'security', label: '安全' },
]

const pathEntries = [
  { key: 'opencode', label: 'OpenCode 路径', placeholder: '/Applications/OpenCode.app' },
  { key: 'antigravity', label: 'Antigravity 路径', placeholder: '/Applications/Antigravity.app' },
  { key: 'codex', label: 'Codex 路径', placeholder: '/Applications/Codex.app' },
  { key: 'vscode', label: 'VS Code 路径', placeholder: '/Applications/Visual Studio Code.app' },
  { key: 'windsurf', label: 'Windsurf 路径', placeholder: '/Applications/Windsurf.app' },
  { key: 'kiro', label: 'Kiro 路径', placeholder: '/Applications/Kiro.app' },
  { key: 'cursor', label: 'Cursor 路径', placeholder: '/Applications/Cursor.app' },
  { key: 'gemini', label: 'Gemini CLI 路径', placeholder: '/usr/local/bin/gemini' },
  { key: 'codebuddy', label: 'CodeBuddy 路径', placeholder: '/Applications/CodeBuddy.app' },
  { key: 'codebuddy-cn', label: 'CodeBuddy CN 路径', placeholder: '/Applications/CodeBuddyCN.app' },
  { key: 'qoder', label: 'Qoder 路径', placeholder: '/Applications/Qoder.app' },
  { key: 'trae', label: 'Trae 路径', placeholder: '/Applications/Trae.app' },
  { key: 'zed', label: 'Zed 路径', placeholder: '/Applications/Zed.app' },
  { key: 'workbuddy', label: 'Workbuddy 路径', placeholder: '/Applications/Workbuddy.app' },
]

const activeTab = ref('experience')

const general = ref(defaultGeneralSettings())
const switches = ref({ log_enabled: true, ip_blacklist_enabled: false, proxy_enabled: false })
const proxy = ref({ enabled: false, host: '', port: 0, username: '', password: '' })
const database = ref({ type: 'sqlite', sqlite_path: './data/easyllm.db', dsn: '' })
const sysInfo = ref({})
const overview = ref({ summary: {}, proxy: {} })
const passwordSet = ref(false)

const pwForm = ref({ oldPassword: '', newPassword: '', confirmPassword: '' })
const pwError = ref('')
const pwSaving = ref(false)

const databasePathOrDSN = computed({
  get() {
    return database.value.type === 'sqlite' ? database.value.sqlite_path : database.value.dsn
  },
  set(value) {
    if (database.value.type === 'sqlite') {
      database.value.sqlite_path = value
    } else {
      database.value.dsn = value
    }
  },
})

onMounted(loadSettings)

async function loadSettings() {
  try {
    const [generalData, switchData, proxyData, databaseData, sysData, overviewData, authData] = await Promise.all([
      cockpitAPI.getGeneralSettings(),
      settingsAPI.getSwitches(),
      settingsAPI.getProxy(),
      settingsAPI.getDatabase(),
      settingsAPI.systemInfo(),
      cockpitAPI.overview(),
      authAPI.check(),
    ])
    general.value = mergeGeneralSettings(generalData)
    switches.value = switchData
    proxy.value = { ...proxy.value, ...proxyData }
    database.value = { ...database.value, ...databaseData }
    sysInfo.value = sysData
    overview.value = overviewData
    passwordSet.value = !!authData.password_set
  } catch (error) {
    notify?.(error.message || '加载设置失败', 'error')
  }
}

function defaultGeneralSettings() {
  return {
    language: 'zh-CN',
    theme: 'system',
    close_behavior: 'ask',
    privacy_mode: false,
    auto_refresh_minutes: 5,
    refresh_intervals: Object.fromEntries(cockpitPlatforms.map((platform) => [platform.id, 10])),
    app_paths: Object.fromEntries(pathEntries.map((entry) => [entry.key, ''])),
    integrations: {
      codex_launch_on_switch: true,
      opencode_sync_on_switch: false,
      opencode_auth_overwrite_on_switch: false,
    },
  }
}

function mergeGeneralSettings(data) {
  const defaults = defaultGeneralSettings()
  return {
    ...defaults,
    ...data,
    refresh_intervals: {
      ...defaults.refresh_intervals,
      ...(data?.refresh_intervals || {}),
    },
    app_paths: {
      ...defaults.app_paths,
      ...(data?.app_paths || {}),
    },
    integrations: {
      ...defaults.integrations,
      ...(data?.integrations || {}),
    },
  }
}

async function saveGeneralSettings() {
  try {
    general.value = mergeGeneralSettings(await cockpitAPI.updateGeneralSettings(general.value))
    notify?.('设置已保存', 'success')
  } catch (error) {
    notify?.(error.message || '保存失败', 'error')
  }
}

async function saveSwitches() {
  try {
    await settingsAPI.updateSwitches(switches.value)
    notify?.('开关已保存', 'success')
  } catch (error) {
    notify?.(error.message || '保存失败', 'error')
  }
}

async function saveProxy() {
  try {
    await settingsAPI.updateProxy({
      ...proxy.value,
      enabled: switches.value.proxy_enabled,
    })
    notify?.('代理设置已保存', 'success')
  } catch (error) {
    notify?.(error.message || '保存失败', 'error')
  }
}

async function saveDatabase() {
  try {
    await settingsAPI.updateDatabase({
      ...database.value,
      dsn: database.value.dsn === '[configured]' ? '' : database.value.dsn,
    })
    notify?.('数据库配置已保存，重启后生效', 'success')
  } catch (error) {
    notify?.(error.message || '保存失败', 'error')
  }
}

async function savePassword() {
  pwError.value = ''
  if (!pwForm.value.newPassword || pwForm.value.newPassword.length < 4) {
    pwError.value = '新密码至少需要 4 位'
    return
  }
  if (pwForm.value.newPassword !== pwForm.value.confirmPassword) {
    pwError.value = '两次输入的密码不一致'
    return
  }

  pwSaving.value = true
  try {
    if (passwordSet.value) {
      await authAPI.changePassword(pwForm.value.oldPassword, pwForm.value.newPassword)
    } else {
      const result = await authAPI.setup(pwForm.value.newPassword)
      if (result?.token) {
        localStorage.setItem('easyllm_token', result.token)
      }
      passwordSet.value = true
    }
    pwForm.value = { oldPassword: '', newPassword: '', confirmPassword: '' }
    notify?.('密码已保存', 'success')
  } catch (error) {
    pwError.value = error.message || '保存失败'
  } finally {
    pwSaving.value = false
  }
}
</script>
