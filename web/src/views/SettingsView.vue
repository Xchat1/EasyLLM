<template>
  <div class="p-6 space-y-6">
    <section class="rounded-3xl border border-gray-800 bg-gradient-to-br from-slate-200/10 via-gray-200/5 to-gray-950 p-6">
      <h1 class="text-3xl font-semibold text-white">设置中心</h1>
      <p class="mt-2 max-w-3xl text-sm leading-6 text-gray-300">
        只保留 EasyLLM / Codex 本地编码对接所需的外观、网络、SQLite 和安全配置。
      </p>
    </section>

    <div class="flex flex-wrap gap-2">
      <button v-for="tab in tabs" :key="tab.id" class="btn btn-sm" :class="activeTab === tab.id ? 'btn-primary' : 'btn-secondary'" @click="activeTab = tab.id">
        {{ tab.label }}
      </button>
    </div>

    <section v-if="activeTab === 'appearance'" class="card p-5 space-y-5">
      <div>
        <h2 class="text-lg font-semibold text-white">外观</h2>
        <p class="mt-1 text-sm text-gray-500">前端本地保存，不再依赖额外平台设置接口。</p>
      </div>

      <div class="grid gap-4 md:grid-cols-3">
        <div class="md:col-span-2">
          <label class="label">外观模式</label>
          <div class="theme-segmented">
            <button
              v-for="mode in themeModes"
              :key="mode.id"
              type="button"
              class="theme-segmented-option"
              :class="{ 'theme-segmented-option-active': appearance.mode === mode.id }"
              @click="setThemeModePreference(mode.id)"
            >
              {{ mode.label }}
            </button>
          </div>
        </div>
        <div class="md:col-span-3">
          <label class="label">Apple 风格强调色</label>
          <div class="accent-theme-grid">
            <button
              v-for="accent in accentThemes"
              :key="accent.id"
              type="button"
              class="accent-theme-option"
              :class="{ 'accent-theme-option-active': appearance.accent === accent.id }"
              :style="{ '--accent-swatch': accent.swatch }"
              @click="setAccentThemePreference(accent.id)"
            >
              <span class="accent-theme-dot" />
              <span>{{ accent.label }}</span>
            </button>
          </div>
        </div>
      </div>
    </section>

    <section v-else-if="activeTab === 'runtime'" class="space-y-6">
      <div class="grid gap-6 xl:grid-cols-[1.2fr_1fr]">
        <article class="card p-5">
          <div>
            <h2 class="text-lg font-semibold text-white">运行状态</h2>
            <p class="mt-1 text-sm text-gray-500">当前服务和数据面的关键指标。</p>
          </div>
          <dl class="mt-5 grid gap-3 text-sm text-gray-400 md:grid-cols-2">
            <div class="settings-stat">
              <dt>版本</dt>
              <dd>v{{ sysInfo.version || '-' }}</dd>
            </div>
            <div class="settings-stat">
              <dt>运行时间</dt>
              <dd>{{ sysInfo.uptime || '-' }}</dd>
            </div>
            <div class="settings-stat">
              <dt>数据库</dt>
              <dd>{{ sysInfo.db_type || 'sqlite' }}</dd>
            </div>
            <div class="settings-stat">
              <dt>端口</dt>
              <dd>{{ sysInfo.server_port || 8022 }}</dd>
            </div>
            <div class="settings-stat">
              <dt>Goroutines</dt>
              <dd>{{ sysInfo.goroutines || '-' }}</dd>
            </div>
            <div class="settings-stat">
              <dt>内存</dt>
              <dd>{{ sysInfo.memory_alloc_mb || '-' }} MB</dd>
            </div>
          </dl>
        </article>

        <article class="card p-5">
          <div>
            <h2 class="text-lg font-semibold text-white">Codex 账号</h2>
            <p class="mt-1 text-sm text-gray-500">当前只统计 OpenAI / Codex 相关数据。</p>
          </div>
          <div class="mt-5 grid grid-cols-2 gap-3">
            <div class="settings-metric">
              <div>OpenAI 账号</div>
              <strong>{{ sysInfo.accounts?.openai || 0 }}</strong>
            </div>
            <div class="settings-metric">
              <div>Codex 池账号</div>
              <strong>{{ sysInfo.accounts?.codex_pool || 0 }}</strong>
            </div>
          </div>
        </article>
      </div>

      <article class="card p-5 space-y-5">
        <div>
          <h2 class="text-lg font-semibold text-white">系统开关与连接</h2>
          <p class="mt-1 text-sm text-gray-500">保留 EasyLLM 本机网络和 SQLite 路径配置。</p>
        </div>

        <div class="grid gap-4 md:grid-cols-2">
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
              <div class="mt-1 text-xs text-gray-500">上游请求统一经过本机配置的 HTTP 代理。</div>
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
          <div class="md:col-span-2">
            <label class="label">SQLite 数据库路径</label>
            <input
              v-model="database.sqlite_path"
              class="input"
              placeholder="默认: 系统应用数据目录/EasyLLM/data/easyllm.db"
            />
          </div>
        </div>

        <div class="flex justify-end gap-2">
          <button class="btn btn-secondary" @click="saveSwitches">保存开关</button>
          <button class="btn btn-secondary" @click="saveProxy">保存代理</button>
          <button class="btn btn-primary" @click="saveDatabase">保存 SQLite 路径</button>
        </div>
      </article>

      <article class="card p-5 space-y-5">
        <div>
          <h2 class="text-lg font-semibold text-white">配额全局检测</h2>
          <p class="mt-1 text-sm text-gray-500">批量查询 OAuth 账号剩余额度的超时和并发控制。</p>
        </div>

        <div class="grid gap-4 md:grid-cols-2">
          <div>
            <label class="label">全局检测超时（秒）</label>
            <input v-model.number="quotaCheck.timeout_seconds" type="number" min="10" max="600" class="input" />
          </div>
          <div>
            <label class="label">并发账号数</label>
            <input v-model.number="quotaCheck.concurrency" type="number" min="1" max="50" class="input" />
          </div>
        </div>

        <div v-if="quotaCheckResult" class="rounded-2xl border border-gray-800 bg-gray-950/60 px-4 py-3 text-sm text-gray-300">
          {{ quotaCheckResult }}
        </div>

        <div class="flex justify-end gap-2">
          <button class="btn btn-secondary" :disabled="quotaCheckSaving || quotaCheckRunning" @click="saveQuotaCheck">
            {{ quotaCheckSaving ? '保存中...' : '保存检测设置' }}
          </button>
          <button class="btn btn-primary" :disabled="quotaCheckRunning" @click="runQuotaCheck">
            {{ quotaCheckRunning ? '检测中...' : `${quotaCheck.timeout_seconds || 60}s 全局检测` }}
          </button>
        </div>
      </article>
    </section>

    <section v-else class="card p-6 space-y-6">
      <div class="flex flex-col gap-2 sm:flex-row sm:items-center sm:justify-between border-b border-gray-800 pb-4">
        <div>
          <h2 class="text-lg font-semibold text-white">访问安全与密码保护</h2>
          <p class="mt-1 text-sm text-gray-400">
            EasyLLM 默认免密访问。开启密码保护后，访问 Web 控制台与客户端将需要输入访问密码。
          </p>
        </div>
        <div class="flex items-center gap-2 shrink-0">
          <span
            class="px-2.5 py-1 text-xs font-medium rounded-full border flex items-center gap-1.5"
            :class="authEnabled ? 'bg-emerald-500/15 text-emerald-400 border-emerald-500/30' : 'bg-gray-800 text-gray-400 border-gray-700'"
          >
            <span class="h-1.5 w-1.5 rounded-full" :class="authEnabled ? 'bg-emerald-400' : 'bg-gray-500'"></span>
            {{ authEnabled ? '已开启密码保护' : '未开启 (默认免密)' }}
          </span>
        </div>
      </div>

      <!-- State 1: Auth is currently DISABLED (Default) -->
      <div v-if="!authEnabled" class="space-y-4">
        <div class="rounded-2xl border border-gray-800 bg-gray-950/60 p-4 text-sm text-gray-300 space-y-2">
          <div class="font-medium text-white flex items-center gap-2">
            <span>🛡️ 免密访问模式</span>
          </div>
          <p class="text-xs text-gray-400 leading-relaxed">
            当前处于免密访问状态，您和任何访问本服务的人无需登录即可直接管理渠道与配置。
            若您的 EasyLLM 暴露在局域网或公网，建议开启密码保护以保障凭据安全。
          </p>
        </div>

        <div class="max-w-xl space-y-4 pt-2">
          <h3 class="text-sm font-semibold text-white">开启访问密码</h3>
          <div>
            <label class="label">设置新密码 *</label>
            <input v-model="pwForm.newPassword" type="password" class="input" placeholder="输入访问密码" />
          </div>
          <div>
            <label class="label">确认密码 *</label>
            <input v-model="pwForm.confirmPassword" type="password" class="input" placeholder="再次输入确认密码" />
          </div>
          <div v-if="pwError" class="rounded-2xl border border-red-700 bg-red-900/20 px-4 py-3 text-sm text-red-300">
            {{ pwError }}
          </div>

          <div class="flex flex-wrap items-center gap-3 pt-2">
            <button class="btn btn-primary" :disabled="pwSaving" @click="enablePasswordAuth">
              {{ pwSaving ? '开启中...' : '开启密码保护' }}
            </button>
            <button
              v-if="passwordSet"
              class="btn btn-secondary text-xs"
              :disabled="pwSaving"
              @click="enableWithExistingPassword"
              title="使用此前设置过的密码直接启用"
            >
              使用已有密码直接启用
            </button>
          </div>
        </div>
      </div>

      <!-- State 2: Auth is currently ENABLED -->
      <div v-else class="space-y-6">
        <div class="rounded-2xl border border-emerald-500/20 bg-emerald-500/5 p-4 text-sm text-emerald-300">
          <div class="font-medium text-emerald-200 flex items-center gap-2">
            <span>🔒 密码保护已生效</span>
          </div>
          <p class="text-xs text-emerald-400/80 mt-1 leading-relaxed">
            已开启访问控制。未携带有效身份令牌的访问将被重定向到登录页面。
          </p>
        </div>

        <!-- Section A: Change Password -->
        <div class="max-w-xl space-y-4">
          <h3 class="text-sm font-semibold text-white">修改访问密码</h3>
          <div>
            <label class="label">当前密码 *</label>
            <input v-model="pwForm.oldPassword" type="password" class="input" placeholder="输入当前密码" />
          </div>
          <div>
            <label class="label">新密码 *</label>
            <input v-model="pwForm.newPassword" type="password" class="input" placeholder="输入新密码" />
          </div>
          <div>
            <label class="label">确认新密码 *</label>
            <input v-model="pwForm.confirmPassword" type="password" class="input" placeholder="再次输入新密码" />
          </div>
          <div v-if="pwError" class="rounded-2xl border border-red-700 bg-red-900/20 px-4 py-3 text-sm text-red-300">
            {{ pwError }}
          </div>
          <div class="flex justify-start">
            <button class="btn btn-primary" :disabled="pwSaving" @click="savePassword">
              {{ pwSaving ? '保存中...' : '修改密码' }}
            </button>
          </div>
        </div>

        <!-- Section B: Disable Password (turn off) -->
        <div class="border-t border-gray-800 pt-6 max-w-xl space-y-4">
          <div>
            <h3 class="text-sm font-semibold text-rose-400">关闭密码保护</h3>
            <p class="text-xs text-gray-500 mt-0.5">
              关闭后将恢复为免密模式，访问 EasyLLM 将不再要求输入密码。
            </p>
          </div>
          <div>
            <label class="label">输入当前密码以确认关闭 *</label>
            <input v-model="disableForm.password" type="password" class="input" placeholder="输入当前密码" />
          </div>
          <div v-if="disableError" class="rounded-2xl border border-red-700 bg-red-900/20 px-4 py-3 text-sm text-red-300">
            {{ disableError }}
          </div>
          <div>
            <button
              class="btn btn-danger"
              :disabled="pwDisabling"
              @click="disablePasswordAuth"
            >
              {{ pwDisabling ? '关闭中...' : '确认关闭密码保护 (恢复免密)' }}
            </button>
          </div>
        </div>
      </div>
    </section>
  </div>
</template>

<script setup>
import { computed, inject, onMounted, ref } from 'vue'
import { authAPI, settingsAPI } from '@/api'
import { ACCENT_THEMES, THEME_MODES } from '@/config/theme'
import { useAppearance } from '@/composables/useAppearance'
import { setAuthStatusCache } from '@/lib/auth'

const notify = inject('notify')
const { themeMode, accentTheme, setThemeMode, setAccentTheme } = useAppearance()
const themeModes = THEME_MODES
const accentThemes = ACCENT_THEMES

const tabs = [
  { id: 'appearance', label: '外观' },
  { id: 'runtime', label: '运行状态' },
  { id: 'security', label: '安全' },
]

const activeTab = ref('appearance')
const switches = ref({ ip_blacklist_enabled: false, proxy_enabled: false })
const proxy = ref({ enabled: false, host: '', port: 0, username: '', password: '' })
const database = ref({ type: 'sqlite', sqlite_path: '' })
const quotaCheck = ref({ timeout_seconds: 60, concurrency: 10 })
const sysInfo = ref({})
const authEnabled = ref(false)
const passwordSet = ref(false)
const quotaCheckSaving = ref(false)
const quotaCheckRunning = ref(false)
const quotaCheckResult = ref('')

const pwForm = ref({ oldPassword: '', newPassword: '', confirmPassword: '' })
const disableForm = ref({ password: '' })
const pwError = ref('')
const disableError = ref('')
const pwSaving = ref(false)
const pwDisabling = ref(false)

const appearance = computed(() => ({
  mode: themeMode.value,
  accent: accentTheme.value,
}))

onMounted(loadSettings)

async function loadSettings() {
  try {
    const [switchData, proxyData, databaseData, quotaCheckData, sysData, authData] = await Promise.all([
      settingsAPI.getSwitches(),
      settingsAPI.getProxy(),
      settingsAPI.getDatabase(),
      settingsAPI.getQuotaCheck(),
      settingsAPI.systemInfo(),
      authAPI.check(),
    ])
    switches.value = switchData
    proxy.value = { ...proxy.value, ...proxyData }
    database.value = { ...database.value, ...databaseData }
    quotaCheck.value = { ...quotaCheck.value, ...quotaCheckData }
    sysInfo.value = sysData
    authEnabled.value = !!authData.auth_enabled
    passwordSet.value = !!authData.password_set
    setAuthStatusCache(authData)
  } catch (error) {
    notify?.(error.message || '加载设置失败', 'error')
  }
}

function setThemeModePreference(mode) {
  setThemeMode(mode)
}

function setAccentThemePreference(accent) {
  setAccentTheme(accent)
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
      sqlite_path: database.value.sqlite_path,
    })
    notify?.('SQLite 路径已保存，重启后生效', 'success')
  } catch (error) {
    notify?.(error.message || '保存失败', 'error')
  }
}

async function saveQuotaCheck(notifySuccess = true) {
  quotaCheckSaving.value = true
  try {
    const data = await settingsAPI.updateQuotaCheck({
      timeout_seconds: Number(quotaCheck.value.timeout_seconds) || 60,
      concurrency: Number(quotaCheck.value.concurrency) || 10,
    })
    quotaCheck.value = { ...quotaCheck.value, ...data }
    if (notifySuccess) notify?.('配额检测设置已保存', 'success')
  } catch (error) {
    notify?.(error.message || '保存失败', 'error')
    throw error
  } finally {
    quotaCheckSaving.value = false
  }
}

async function runQuotaCheck() {
  quotaCheckRunning.value = true
  quotaCheckResult.value = ''
  try {
    await saveQuotaCheck(false)
    const startedAt = Date.now()
    const data = await settingsAPI.runQuotaCheck()
    const results = Array.isArray(data?.results) ? data.results : []
    const ok = results.filter(r => r.success && !r.is_forbidden).length
    const forbidden = results.filter(r => r.success && r.is_forbidden).length
    const failed = results.filter(r => !r.success).length
    const elapsed = Math.max(1, Math.round((Date.now() - startedAt) / 1000))
    quotaCheckResult.value = `完成 ${results.length} 个账号，用时 ${elapsed}s：${ok} 个可用，${forbidden} 个禁用/停用，${failed} 个失败`
    notify?.('全局配额检测完成', failed > 0 && ok === 0 ? 'error' : 'success')
  } catch (error) {
    quotaCheckResult.value = error.message || '检测失败'
    notify?.(quotaCheckResult.value, 'error')
  } finally {
    quotaCheckRunning.value = false
  }
}

async function enablePasswordAuth() {
  pwError.value = ''
  if (!pwForm.value.newPassword) {
    pwError.value = '请输入访问密码'
    return
  }
  if (pwForm.value.newPassword !== pwForm.value.confirmPassword) {
    pwError.value = '两次输入的密码不一致'
    return
  }

  pwSaving.value = true
  try {
    const res = await authAPI.enable({ password: pwForm.value.newPassword })
    if (res?.token) {
      localStorage.setItem('easyllm_token', res.token)
    }
    authEnabled.value = true
    passwordSet.value = true
    setAuthStatusCache({ auth_enabled: true, password_set: true })
    pwForm.value = { oldPassword: '', newPassword: '', confirmPassword: '' }
    notify?.('访问密码保护已开启', 'success')
  } catch (err) {
    pwError.value = err.message || '开启密码保护失败'
  } finally {
    pwSaving.value = false
  }
}

async function enableWithExistingPassword() {
  pwSaving.value = true
  pwError.value = ''
  try {
    const res = await authAPI.enable({})
    if (res?.token) {
      localStorage.setItem('easyllm_token', res.token)
    }
    authEnabled.value = true
    passwordSet.value = true
    setAuthStatusCache({ auth_enabled: true, password_set: true })
    notify?.('访问密码保护已开启', 'success')
  } catch (err) {
    pwError.value = err.message || '开启密码保护失败'
  } finally {
    pwSaving.value = false
  }
}

async function savePassword() {
  pwError.value = ''
  if (!pwForm.value.oldPassword) {
    pwError.value = '请输入当前密码'
    return
  }
  if (!pwForm.value.newPassword) {
    pwError.value = '请输入新密码'
    return
  }
  if (pwForm.value.newPassword !== pwForm.value.confirmPassword) {
    pwError.value = '两次输入的密码不一致'
    return
  }

  pwSaving.value = true
  try {
    await authAPI.changePassword(pwForm.value.oldPassword, pwForm.value.newPassword)
    pwForm.value = { oldPassword: '', newPassword: '', confirmPassword: '' }
    notify?.('密码已成功修改', 'success')
  } catch (error) {
    pwError.value = error.message || '密码保存失败'
  } finally {
    pwSaving.value = false
  }
}

async function disablePasswordAuth() {
  disableError.value = ''
  if (!disableForm.value.password) {
    disableError.value = '请输入当前密码以确认关闭'
    return
  }

  pwDisabling.value = true
  try {
    await authAPI.disable({ password: disableForm.value.password })
    authEnabled.value = false
    setAuthStatusCache({ auth_enabled: false })
    disableForm.value = { password: '' }
    notify?.('访问密码保护已关闭（恢复免密访问模式）', 'success')
  } catch (err) {
    disableError.value = err.message || '关闭密码保护失败'
  } finally {
    pwDisabling.value = false
  }
}
</script>

<style scoped>
.theme-segmented {
  display: inline-flex;
  border: 1px solid var(--app-border);
  border-radius: 0.75rem;
  background: var(--app-control-bg);
  padding: 0.25rem;
}

.theme-segmented-option {
  border-radius: 0.5rem;
  padding: 0.5rem 1rem;
  color: var(--app-text-muted);
  font-size: 0.875rem;
  transition: color 0.2s ease, background 0.2s ease;
}

.theme-segmented-option-active {
  background: var(--app-control-active-bg);
  color: var(--app-text);
}

.accent-theme-grid {
  display: grid;
  gap: 0.5rem;
  grid-template-columns: repeat(auto-fit, minmax(9rem, 1fr));
}

.accent-theme-option {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  border: 1px solid var(--app-border);
  border-radius: 0.75rem;
  background: var(--app-control-bg);
  padding: 0.5rem 0.75rem;
  color: var(--app-text-secondary);
  font-size: 0.875rem;
  transition: color 0.2s ease, border-color 0.2s ease, background 0.2s ease;
}

.accent-theme-option-active {
  border-color: var(--app-accent);
  color: var(--app-text);
}

.accent-theme-dot {
  display: inline-block;
  width: 0.75rem;
  height: 0.75rem;
  border-radius: 9999px;
  background: var(--accent-swatch);
}

.settings-stat {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 1rem;
  border: 1px solid var(--app-border);
  border-radius: 1rem;
  background: var(--app-control-bg);
  padding: 0.75rem 1rem;
}

.settings-stat dd {
  color: var(--app-text);
}

.settings-metric {
  border: 1px solid var(--app-border);
  border-radius: 1rem;
  background: var(--app-control-bg);
  padding: 1rem;
}

.settings-metric div {
  color: var(--app-text-faint);
  font-size: 0.75rem;
}

.settings-metric strong {
  display: block;
  margin-top: 0.5rem;
  color: var(--app-text);
  font-size: 1.5rem;
  font-weight: 600;
}
</style>
