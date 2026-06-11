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
          <input v-model="pwForm.newPassword" type="password" class="input" placeholder="输入密码" />
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
import { authAPI, settingsAPI } from '@/api'
import { ACCENT_THEMES, THEME_MODES } from '@/config/theme'
import { useAppearance } from '@/composables/useAppearance'

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
const sysInfo = ref({})
const passwordSet = ref(false)

const pwForm = ref({ oldPassword: '', newPassword: '', confirmPassword: '' })
const pwError = ref('')
const pwSaving = ref(false)

const appearance = computed(() => ({
  mode: themeMode.value,
  accent: accentTheme.value,
}))

onMounted(loadSettings)

async function loadSettings() {
  try {
    const [switchData, proxyData, databaseData, sysData, authData] = await Promise.all([
      settingsAPI.getSwitches(),
      settingsAPI.getProxy(),
      settingsAPI.getDatabase(),
      settingsAPI.systemInfo(),
      authAPI.check(),
    ])
    switches.value = switchData
    proxy.value = { ...proxy.value, ...proxyData }
    database.value = { ...database.value, ...databaseData }
    sysInfo.value = sysData
    passwordSet.value = !!authData.password_set
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

async function savePassword() {
  pwError.value = ''
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
    notify?.('密码已更新', 'success')
  } catch (error) {
    pwError.value = error.message || '密码保存失败'
  } finally {
    pwSaving.value = false
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
