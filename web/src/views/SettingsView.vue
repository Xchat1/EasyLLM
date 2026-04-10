<template>
  <div class="p-6 max-w-3xl">
    <h1 class="text-2xl font-bold text-white mb-6">⚙️ 设置</h1>

    <!-- Tabs -->
    <div class="flex gap-2 mb-6 border-b border-gray-700 pb-4">
      <button v-for="tab in tabs" :key="tab.id" @click="activeTab = tab.id"
        :class="activeTab === tab.id ? 'btn btn-primary btn-sm' : 'btn btn-secondary btn-sm'">
        {{ tab.label }}
      </button>
    </div>

    <!-- Switches (开关管理) -->
    <div v-if="activeTab === 'switches'" class="space-y-4">
      <!-- 总览开关卡片 -->
      <div class="card p-4">
        <h3 class="font-medium text-white mb-4">功能开关</h3>
        <p class="text-gray-500 text-xs mb-4">管理系统功能的开关状态，默认仅开启日志。</p>
        <div class="space-y-3">
          <!-- 日志开关 -->
          <div class="flex items-center justify-between bg-gray-800/60 rounded-lg px-4 py-3">
            <div>
              <div class="text-sm text-white font-medium">日志记录</div>
              <div class="text-xs text-gray-500 mt-0.5">开启后记录所有 HTTP 请求日志</div>
            </div>
            <label class="relative inline-flex items-center cursor-pointer">
              <input v-model="switches.log_enabled" type="checkbox" class="sr-only peer" @change="saveSwitches" />
              <div class="w-11 h-6 bg-gray-600 rounded-full peer peer-checked:bg-blue-500 after:content-[''] after:absolute after:top-[2px] after:left-[2px] after:bg-white after:rounded-full after:h-5 after:w-5 after:transition-all peer-checked:after:translate-x-full"></div>
            </label>
          </div>

          <!-- IP 黑名单开关 -->
          <div class="flex items-center justify-between bg-gray-800/60 rounded-lg px-4 py-3">
            <div>
              <div class="text-sm text-white font-medium">IP 黑名单</div>
              <div class="text-xs text-gray-500 mt-0.5">开启后黑名单内的 IP 将被禁止调用 API</div>
            </div>
            <label class="relative inline-flex items-center cursor-pointer">
              <input v-model="switches.ip_blacklist_enabled" type="checkbox" class="sr-only peer" @change="saveSwitches" />
              <div class="w-11 h-6 bg-gray-600 rounded-full peer peer-checked:bg-blue-500 after:content-[''] after:absolute after:top-[2px] after:left-[2px] after:bg-white after:rounded-full after:h-5 after:w-5 after:transition-all peer-checked:after:translate-x-full"></div>
            </label>
          </div>

          <!-- 代理开关 -->
          <div class="flex items-center justify-between bg-gray-800/60 rounded-lg px-4 py-3">
            <div>
              <div class="text-sm text-white font-medium">HTTP 代理</div>
              <div class="text-xs text-gray-500 mt-0.5">开启后所有 API 请求将通过配置的代理转发</div>
            </div>
            <label class="relative inline-flex items-center cursor-pointer">
              <input v-model="switches.proxy_enabled" type="checkbox" class="sr-only peer" @change="saveSwitches" />
              <div class="w-11 h-6 bg-gray-600 rounded-full peer peer-checked:bg-blue-500 after:content-[''] after:absolute after:top-[2px] after:left-[2px] after:bg-white after:rounded-full after:h-5 after:w-5 after:transition-all peer-checked:after:translate-x-full"></div>
            </label>
          </div>
        </div>
      </div>

      <!-- IP 黑名单管理 -->
      <div v-if="switches.ip_blacklist_enabled" class="card p-4">
        <h3 class="font-medium text-white mb-4">IP 黑名单管理</h3>
        <div class="space-y-3">
          <div class="space-y-2">
            <textarea v-model="newBlacklistIP" class="input w-full text-sm font-mono" rows="3" placeholder="支持一次添加多个 IP，每行一个或用逗号/空格分隔&#10;例: 192.168.1.100, 10.0.0.1&#10;或每行一个 IP" @keydown.enter.ctrl="addBlacklistIP"></textarea>
            <div class="flex items-center justify-between">
              <span class="text-xs text-gray-600">Ctrl+Enter 快速添加</span>
              <button @click="addBlacklistIP" class="btn btn-primary btn-sm">添加</button>
            </div>
          </div>
          <div v-if="blacklist.ips.length === 0" class="text-gray-500 text-sm text-center py-4">
            暂无黑名单 IP
          </div>
          <div v-else class="space-y-1.5">
            <div v-for="(ip, idx) in blacklist.ips" :key="idx"
              class="flex items-center justify-between bg-gray-800/60 rounded-lg px-3 py-2">
              <code class="text-red-400 text-sm font-mono">{{ ip }}</code>
              <button @click="removeBlacklistIP(idx)" class="text-gray-500 hover:text-red-400 text-xs">移除</button>
            </div>
          </div>
        </div>
      </div>

      <!-- 代理配置（仅在代理开关开启时展示） -->
      <div v-if="switches.proxy_enabled" class="card p-4">
        <h3 class="font-medium text-white mb-4">代理配置</h3>
        <div class="space-y-3">
          <div class="grid grid-cols-2 gap-3">
            <div><label class="label">代理主机</label><input v-model="proxy.host" class="input" placeholder="127.0.0.1" /></div>
            <div><label class="label">端口</label><input v-model.number="proxy.port" type="number" class="input" placeholder="7890" /></div>
          </div>
          <div class="grid grid-cols-2 gap-3">
            <div><label class="label">用户名 (可选)</label><input v-model="proxy.username" class="input" /></div>
            <div><label class="label">密码 (可选)</label><input v-model="proxy.password" type="password" class="input" /></div>
          </div>
        </div>
        <div class="flex justify-end mt-4">
          <button @click="saveProxy" class="btn btn-primary">保存代理设置</button>
        </div>
      </div>
    </div>

    <!-- System Info -->
    <div v-if="activeTab === 'system'" class="space-y-4">
      <!-- Runtime Info -->
      <div class="card p-4">
        <h3 class="font-medium text-white mb-3">项目信息</h3>
        <div class="grid grid-cols-2 gap-x-6 gap-y-2 text-sm">
          <div class="flex justify-between"><span class="text-gray-400">版本</span><span class="text-blue-400 font-mono">v{{ sysInfo.version }}</span></div>
          <div class="flex justify-between"><span class="text-gray-400">GitHub</span><a v-if="sysInfo.git_repo" :href="sysInfo.git_repo" target="_blank" class="text-blue-400 hover:text-blue-300 font-mono text-xs truncate max-w-[200px]">{{ sysInfo.git_repo.replace('https://github.com/', '') }}</a><span v-else class="text-gray-500">-</span></div>
          <div class="flex justify-between"><span class="text-gray-400">Go 版本</span><span class="text-gray-200 font-mono">{{ sysInfo.go_version || '-' }}</span></div>
          <div class="flex justify-between"><span class="text-gray-400">操作系统</span><span class="text-gray-200">{{ sysInfo.os || '-' }} / {{ sysInfo.arch || '-' }}</span></div>
          <div class="flex justify-between"><span class="text-gray-400">主机名</span><span class="text-gray-200">{{ sysInfo.hostname || '-' }}</span></div>
          <div class="flex justify-between"><span class="text-gray-400">进程 PID</span><span class="text-gray-200 font-mono">{{ sysInfo.pid || '-' }}</span></div>
          <div class="flex justify-between"><span class="text-gray-400">运行时间</span><span class="text-green-400">{{ sysInfo.uptime || '-' }}</span></div>
          <div class="flex justify-between"><span class="text-gray-400">Goroutines</span><span class="text-gray-200 font-mono">{{ sysInfo.goroutines || '-' }}</span></div>
          <div class="flex justify-between"><span class="text-gray-400">内存占用</span><span class="text-gray-200 font-mono">{{ sysInfo.memory_alloc_mb || '-' }} MB</span></div>
          <div class="flex justify-between"><span class="text-gray-400">系统内存</span><span class="text-gray-200 font-mono">{{ sysInfo.memory_sys_mb || '-' }} MB</span></div>
          <div class="flex justify-between"><span class="text-gray-400">GC 次数</span><span class="text-gray-200 font-mono">{{ sysInfo.memory_gc_cycles ?? '-' }}</span></div>
        </div>
      </div>



      <!-- Account Counts -->
      <div v-if="sysInfo.accounts" class="card p-4">
        <h3 class="font-medium text-white mb-3">账号统计</h3>
        <div class="grid grid-cols-3 sm:grid-cols-4 gap-3">
          <div v-for="(count, platform) in accountCountItems" :key="platform"
            class="bg-gray-800/60 rounded-lg p-3 text-center">
            <div class="text-2xl font-bold text-white">{{ count }}</div>
            <div class="text-xs text-gray-400 mt-1">{{ platform }}</div>
          </div>
        </div>
      </div>
      <div class="card p-4">
        <h3 class="font-medium text-white mb-4">API 端点参考</h3>
        <div class="space-y-4">
          <!-- OpenAI / Codex Proxy -->
          <div>
            <p class="text-xs font-semibold text-gray-500 uppercase tracking-wider mb-2">OpenAI / Codex 代理</p>
            <div class="space-y-1.5">
              <div v-for="ep in allEndpoints.codex" :key="ep.path"
                class="flex items-center justify-between bg-gray-800/60 rounded-lg px-3 py-2 gap-3">
                <div class="flex items-center gap-2 min-w-0">
                  <span class="shrink-0 text-xs font-bold font-mono px-1.5 py-0.5 rounded"
                    :class="ep.method === 'GET' ? 'bg-green-500/20 text-green-400' : 'bg-orange-500/20 text-orange-400'">
                    {{ ep.method }}
                  </span>
                  <code class="text-blue-400 text-xs font-mono truncate">http://localhost:{{ sysInfo.server_port }}{{ ep.path }}</code>
                </div>
                <div class="flex items-center gap-2 shrink-0">
                  <span class="text-gray-600 text-xs hidden md:inline">{{ ep.desc }}</span>
                  <button @click="copy(`http://localhost:${sysInfo.server_port}${ep.path}`)" class="btn btn-secondary btn-xs">复制</button>
                </div>
              </div>
            </div>
          </div>
          <!-- Augment Import -->
          <div>
            <p class="text-xs font-semibold text-gray-500 uppercase tracking-wider mb-2">Augment 导入接口</p>
            <div class="space-y-1.5">
              <div v-for="ep in allEndpoints.augment" :key="ep.path"
                class="flex items-center justify-between bg-gray-800/60 rounded-lg px-3 py-2 gap-3">
                <div class="flex items-center gap-2 min-w-0">
                  <span class="shrink-0 text-xs font-bold font-mono px-1.5 py-0.5 rounded bg-orange-500/20 text-orange-400">POST</span>
                  <code class="text-blue-400 text-xs font-mono truncate">http://localhost:{{ sysInfo.server_port }}{{ ep.path }}</code>
                </div>
                <div class="flex items-center gap-2 shrink-0">
                  <span class="text-gray-600 text-xs hidden md:inline">{{ ep.desc }}</span>
                  <button @click="copy(`http://localhost:${sysInfo.server_port}${ep.path}`)" class="btn btn-secondary btn-xs">复制</button>
                </div>
              </div>
            </div>
          </div>
          <!-- System -->
          <div>
            <p class="text-xs font-semibold text-gray-500 uppercase tracking-wider mb-2">系统</p>
            <div class="space-y-1.5">
              <div v-for="ep in allEndpoints.system" :key="ep.path"
                class="flex items-center justify-between bg-gray-800/60 rounded-lg px-3 py-2 gap-3">
                <div class="flex items-center gap-2 min-w-0">
                  <span class="shrink-0 text-xs font-bold font-mono px-1.5 py-0.5 rounded bg-green-500/20 text-green-400">GET</span>
                  <code class="text-blue-400 text-xs font-mono truncate">http://localhost:{{ sysInfo.server_port }}{{ ep.path }}</code>
                </div>
                <div class="flex items-center gap-2 shrink-0">
                  <span class="text-gray-600 text-xs hidden md:inline">{{ ep.desc }}</span>
                  <button @click="copy(`http://localhost:${sysInfo.server_port}${ep.path}`)" class="btn btn-secondary btn-xs">复制</button>
                </div>
              </div>
            </div>
          </div>
        </div>
      </div>
    </div>

    <!-- Database Settings -->
    <div v-if="activeTab === 'database'" class="space-y-4">
      <div class="card p-4">
        <h3 class="font-medium text-white mb-4">数据库配置</h3>
        <div class="space-y-3">
          <div>
            <label class="label">数据库类型</label>
            <select v-model="database.type" class="input">
              <option value="sqlite">SQLite (本地文件, 推荐)</option>
              <option value="postgres">PostgreSQL</option>
            </select>
          </div>
          <div v-if="database.type === 'sqlite'">
            <label class="label">SQLite 文件路径</label>
            <input v-model="database.sqlite_path" class="input" placeholder="./data/easyllm.db" />
          </div>
          <div v-if="database.type === 'postgres'">
            <label class="label">PostgreSQL DSN</label>
            <input v-model="database.dsn" class="input" placeholder="host=localhost user=postgres password=xxx dbname=easyllm port=5432 sslmode=disable" />
          </div>
          <div class="bg-yellow-900/30 border border-yellow-700 rounded-lg p-3 text-sm text-yellow-300">
            ⚠️ 修改数据库配置后需要重启服务器才能生效
          </div>
        </div>
        <div class="flex justify-end mt-4">
          <button @click="saveDatabase" class="btn btn-primary">保存数据库配置</button>
        </div>
      </div>
    </div>

    <!-- Security Settings -->
    <div v-if="activeTab === 'security'" class="space-y-4">
      <div class="card p-4">
        <h3 class="font-medium text-white mb-4">访问密码</h3>
        <p class="text-gray-500 text-xs mb-4">设置访问密码后，所有操作需先登录。修改后当前会话保持有效。</p>
        <div class="space-y-3 max-w-md">
          <div v-if="hasPassword">
            <label class="label">当前密码</label>
            <input v-model="pwForm.oldPassword" type="password" class="input" placeholder="输入当前密码" />
          </div>
          <div>
            <label class="label">{{ hasPassword ? '新密码' : '设置密码' }}</label>
            <input v-model="pwForm.newPassword" type="password" class="input" placeholder="至少 4 位" />
          </div>
          <div>
            <label class="label">确认{{ hasPassword ? '新' : '' }}密码</label>
            <input v-model="pwForm.confirmPassword" type="password" class="input" placeholder="再次输入" />
          </div>
          <div v-if="pwError" class="text-red-400 text-sm bg-red-900/30 rounded-lg px-3 py-2">{{ pwError }}</div>
        </div>
        <div class="flex justify-end mt-4">
          <button @click="savePassword" class="btn btn-primary" :disabled="pwSaving">
            {{ pwSaving ? '保存中...' : hasPassword ? '修改密码' : '设置密码' }}
          </button>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, computed, onMounted, inject } from 'vue'
import { settingsAPI, authAPI } from '@/api'

const notify = inject('notify')
const activeTab = ref('switches')

const tabs = [
  { id: 'switches', label: '功能开关' },
  { id: 'system', label: '系统信息' },
  { id: 'database', label: '数据库' },
  { id: 'security', label: '安全' },
]

const sysInfo = ref({ version: '-', db_type: '-', server_port: 8022, proxy_enabled: false, log_enabled: true, ip_blacklist_enabled: false })

const platformLabels = {
  openai: 'OpenAI',
  augment: 'Augment',
  cursor: 'Cursor',
  windsurf: 'Windsurf',
  antigravity: 'Antigravity',
  claude: 'Claude',
  codex_pool: 'Codex Pool',
}

const accountCountItems = computed(() => {
  if (!sysInfo.value.accounts) return {}
  const result = {}
  for (const [key, count] of Object.entries(sysInfo.value.accounts)) {
    result[platformLabels[key] || key] = count
  }
  return result
})

const switches = ref({ log_enabled: true, ip_blacklist_enabled: false, proxy_enabled: false })
const blacklist = ref({ ips: [] })
const newBlacklistIP = ref('')

const allEndpoints = {
  codex: [
    { method: 'POST', path: '/v1/chat/completions', desc: 'OpenAI 兼容聊天补全' },
    { method: 'GET',  path: '/v1/models',           desc: '获取可用模型列表' },
    { method: 'GET',  path: '/pool/status',         desc: '代理账号池状态' },
  ],
  augment: [
    { path: '/api/import/session',  desc: '导入单个 Augment Session' },
    { path: '/api/import/sessions', desc: '批量导入 Augment Sessions' },
  ],
  system: [
    { path: '/api/health', desc: '服务健康检查' },
  ],
}
const proxy = ref({ enabled: false, host: '', port: 7890, username: '', password: '' })
const database = ref({ type: 'sqlite', dsn: '', sqlite_path: './data/easyllm.db' })

function copy(text) {
  navigator.clipboard.writeText(text).then(() => notify('已复制', 'success'))
}

async function loadAll() {
  try {
    sysInfo.value = await settingsAPI.systemInfo()
    const switchData = await settingsAPI.getSwitches()
    switches.value = { ...switches.value, ...switchData }
    const blData = await settingsAPI.getIPBlacklist()
    blacklist.value = { ips: blData.ips || [] }
    const proxyData = await settingsAPI.getProxy()
    proxy.value = { ...proxy.value, ...proxyData }
    const dbData = await settingsAPI.getDatabase()
    database.value = { ...database.value, ...dbData }
  } catch (e) {
    notify(e.message, 'error')
  }
}

async function saveSwitches() {
  try {
    await settingsAPI.updateSwitches(switches.value)
    sysInfo.value = await settingsAPI.systemInfo()
    notify('开关设置已更新', 'success')
  } catch (e) { notify(e.message, 'error') }
}

function addBlacklistIP() {
  const raw = newBlacklistIP.value.trim()
  if (!raw) return
  const ips = raw.split(/[\n,;\s]+/).map(s => s.trim()).filter(Boolean)
  if (!ips.length) return
  let added = 0, skipped = 0
  for (const ip of ips) {
    if (blacklist.value.ips.includes(ip)) { skipped++; continue }
    blacklist.value.ips.push(ip)
    added++
  }
  newBlacklistIP.value = ''
  if (added > 0) {
    saveBlacklist()
    notify(`已添加 ${added} 个 IP` + (skipped ? `，跳过 ${skipped} 个重复` : ''), 'success')
  } else {
    notify(`${skipped} 个 IP 均已存在`, 'error')
  }
}

function removeBlacklistIP(idx) {
  blacklist.value.ips.splice(idx, 1)
  saveBlacklist()
}

async function saveBlacklist() {
  try {
    await settingsAPI.updateIPBlacklist({ ips: blacklist.value.ips })
    notify('IP 黑名单已更新', 'success')
  } catch (e) { notify(e.message, 'error') }
}

async function saveProxy() {
  try {
    await settingsAPI.updateProxy({ ...proxy.value, enabled: switches.value.proxy_enabled })
    notify('代理设置已保存', 'success')
  } catch (e) { notify(e.message, 'error') }
}

async function saveDatabase() {
  try {
    await settingsAPI.updateDatabase(database.value)
    notify('数据库设置已保存，重启后生效', 'success')
  } catch (e) { notify(e.message, 'error') }
}

const hasPassword = ref(false)
const pwForm = ref({ oldPassword: '', newPassword: '', confirmPassword: '' })
const pwError = ref('')
const pwSaving = ref(false)

async function checkPasswordStatus() {
  try {
    const data = await authAPI.check()
    hasPassword.value = data.password_set
  } catch { /* ignore */ }
}

async function savePassword() {
  pwError.value = ''
  if (!pwForm.value.newPassword || pwForm.value.newPassword.length < 4) {
    pwError.value = '密码至少 4 位'
    return
  }
  if (pwForm.value.newPassword !== pwForm.value.confirmPassword) {
    pwError.value = '两次密码不一致'
    return
  }

  pwSaving.value = true
  try {
    if (hasPassword.value) {
      if (!pwForm.value.oldPassword) {
        pwError.value = '请输入当前密码'
        pwSaving.value = false
        return
      }
      await authAPI.changePassword(pwForm.value.oldPassword, pwForm.value.newPassword)
      notify('密码已修改', 'success')
    } else {
      const data = await authAPI.setup(pwForm.value.newPassword)
      if (data.token) localStorage.setItem('easyllm_token', data.token)
      notify('密码已设置，已自动登录', 'success')
    }
    hasPassword.value = true
    pwForm.value = { oldPassword: '', newPassword: '', confirmPassword: '' }
  } catch (e) {
    pwError.value = e.message || '操作失败'
  } finally {
    pwSaving.value = false
  }
}

onMounted(() => {
  loadAll()
  checkPasswordStatus()
})
</script>
