<template>
  <div v-if="$route.path === '/login'" class="h-screen">
    <router-view />
  </div>
  <div v-else class="flex h-screen overflow-hidden bg-gray-950">
    <!-- Sidebar -->
    <aside class="w-56 flex-shrink-0 bg-gray-900 border-r border-gray-800 flex flex-col">
      <!-- Logo -->
      <div class="p-4 border-b border-gray-800">
        <div class="flex items-center gap-2">
          <div class="w-8 h-8 bg-blue-600 rounded-lg flex items-center justify-center font-bold text-white text-sm">EL</div>
          <div>
            <div class="font-bold text-white text-sm">EasyLLM</div>
            <div class="text-xs text-gray-500">v2.0.0</div>
          </div>
        </div>
      </div>

      <!-- Navigation -->
      <nav class="flex-1 overflow-y-auto py-2">
        <div class="px-3 mb-1">
          <p class="text-xs font-semibold text-gray-500 uppercase tracking-wider px-2 py-1">平台</p>
        </div>
        <router-link
          v-for="item in platformRoutes"
          :key="item.path"
          :to="item.path"
          class="nav-item"
          :class="{ 'nav-item-active': $route.path === item.path }"
        >
          <span class="text-base">{{ item.icon }}</span>
          <span>{{ item.label }}</span>
        </router-link>

        <div class="px-3 mt-3 mb-1">
          <p class="text-xs font-semibold text-gray-500 uppercase tracking-wider px-2 py-1">系统</p>
        </div>
        <router-link
          v-for="item in systemRoutes"
          :key="item.path"
          :to="item.path"
          class="nav-item"
          :class="{ 'nav-item-active': $route.path === item.path }"
        >
          <span class="text-base">{{ item.icon }}</span>
          <span>{{ item.label }}</span>
        </router-link>
      </nav>

      <!-- Bottom: Server Status + Logout -->
      <div class="p-3 border-t border-gray-800 space-y-2">
        <div class="flex items-center justify-between text-xs">
          <span class="text-gray-500">API Server</span>
          <div class="flex items-center gap-1">
            <div class="w-2 h-2 rounded-full" :class="serverRunning ? 'bg-green-400' : 'bg-gray-600'"></div>
            <span class="text-gray-400">:{{ serverPort }}</span>
          </div>
        </div>
        <button v-if="isLoggedIn" @click="handleLogout"
          class="w-full flex items-center justify-center gap-1.5 px-3 py-1.5 text-xs text-gray-400 hover:text-red-400 hover:bg-gray-800 rounded-lg transition-colors">
          <span>🚪</span><span>退出登录</span>
        </button>
      </div>
    </aside>

    <!-- Main Content -->
    <main class="flex-1 overflow-y-auto">
      <!-- Global notification -->
      <div v-if="notification.show" class="fixed top-4 right-4 z-50 max-w-sm">
        <div
          class="px-4 py-3 rounded-lg shadow-lg flex items-center gap-3 text-sm"
          :class="notification.type === 'error' ? 'bg-red-800 text-red-100' :
                  notification.type === 'success' ? 'bg-green-800 text-green-100' :
                  'bg-blue-800 text-blue-100'"
        >
          <span>{{ notification.type === 'error' ? '❌' : notification.type === 'success' ? '✅' : 'ℹ️' }}</span>
          <span>{{ notification.message }}</span>
        </div>
      </div>

      <router-view v-slot="{ Component, route }">
        <keep-alive :exclude="['DashboardView']">
          <component :is="Component" :key="route.path" />
        </keep-alive>
      </router-view>
    </main>
  </div>
</template>

<script setup>
import { ref, computed, onMounted, onUnmounted, provide } from 'vue'
import { useRouter } from 'vue-router'
import { settingsAPI, authAPI } from './api'

const router = useRouter()

const platformRoutes = [
  { path: '/openai', icon: '🤖', label: 'OpenAI / Codex' },
  { path: '/augment', icon: '🔑', label: 'Augment' },
  { path: '/cursor', icon: '💻', label: 'Cursor' },
  { path: '/windsurf', icon: '🏄', label: 'Windsurf' },
  { path: '/antigravity', icon: '🚀', label: 'Antigravity' },
  { path: '/claude', icon: '🧠', label: 'Claude' },
]

const systemRoutes = [
  { path: '/dashboard', icon: '📊', label: '看板' },
  { path: '/docs', icon: '📖', label: '文档' },
  { path: '/settings', icon: '⚙️', label: '设置' },
]

const serverRunning = ref(false)
const serverPort = ref(8022)

const notification = ref({ show: false, message: '', type: 'info' })
let notificationTimer = null
let statusInterval = null

function showNotification(message, type = 'info') {
  if (notificationTimer) clearTimeout(notificationTimer)
  notification.value = { show: true, message, type }
  notificationTimer = setTimeout(() => {
    notification.value.show = false
  }, 3000)
}

provide('notify', showNotification)

const isLoggedIn = computed(() => !!localStorage.getItem('easyllm_token'))

async function handleLogout() {
  try { await authAPI.logout() } catch { /* ignore */ }
  localStorage.removeItem('easyllm_token')
  router.push('/login')
}

async function checkServerStatus() {
  try {
    const data = await settingsAPI.apiServerStatus()
    serverRunning.value = data.running
    if (data.port) serverPort.value = data.port
  } catch {
    serverRunning.value = false
  }
}

onMounted(() => {
  checkServerStatus()
  statusInterval = setInterval(checkServerStatus, 30000)
})

onUnmounted(() => {
  if (statusInterval) clearInterval(statusInterval)
  if (notificationTimer) clearTimeout(notificationTimer)
})
</script>

<style>
.nav-item {
  @apply flex items-center gap-2.5 px-4 py-2.5 mx-2 rounded-lg text-sm text-gray-400
         hover:text-white hover:bg-gray-800 transition-colors cursor-pointer;
}
.nav-item-active {
  @apply bg-blue-600 text-white hover:bg-blue-500;
}
</style>
