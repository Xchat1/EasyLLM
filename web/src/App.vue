<template>
  <div v-if="$route.path === '/login'" class="h-screen">
    <router-view />
  </div>
  <div v-else class="flex h-screen overflow-hidden bg-gray-950">
    <aside class="w-72 flex-shrink-0 border-r border-gray-800 bg-gray-900/95">
      <div class="flex h-full flex-col">
        <div class="border-b border-gray-800 px-5 py-4">
          <div class="flex items-center gap-3">
            <div class="flex h-10 w-10 items-center justify-center rounded-2xl bg-gradient-to-br from-sky-500 to-cyan-400 font-bold text-white shadow-lg shadow-sky-900/40">
              EL
            </div>
            <div>
              <div class="font-semibold text-white">EasyLLM</div>
              <div class="text-xs text-gray-500">多平台账号与实例管理台</div>
            </div>
          </div>
        </div>

        <nav class="flex-1 space-y-5 overflow-y-auto px-3 py-4">
          <section>
            <div class="px-2 pb-2 text-xs font-semibold uppercase tracking-wider text-gray-500">平台</div>
            <div class="space-y-1">
              <router-link
                v-for="item in platformRoutes"
                :key="item.route"
                :to="item.route"
                class="nav-item"
                :class="{ 'nav-item-active': $route.path === item.route }"
              >
                <span class="text-base">{{ item.icon }}</span>
                <span class="truncate">{{ item.label }}</span>
              </router-link>
            </div>
          </section>

          <section>
            <div class="px-2 pb-2 text-xs font-semibold uppercase tracking-wider text-gray-500">系统</div>
            <div class="space-y-1">
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
            </div>
          </section>
        </nav>

        <div class="border-t border-gray-800 p-3 space-y-3">
          <div class="rounded-2xl border border-gray-800 bg-gray-950/60 px-3 py-3 text-xs">
            <div class="flex items-center justify-between">
              <span class="text-gray-500">API Server</span>
              <div class="flex items-center gap-2">
                <span class="h-2 w-2 rounded-full" :class="serverRunning ? 'bg-emerald-400' : 'bg-gray-600'" />
                <span class="text-gray-300">:{{ serverPort }}</span>
              </div>
            </div>
            <div class="mt-2 text-gray-500">
              {{ serverRunning ? '服务运行正常，可访问总览和代理接口。' : '服务状态未知，正在等待下一次轮询。' }}
            </div>
          </div>
          <button
            v-if="isLoggedIn"
            @click="handleLogout"
            class="w-full rounded-xl border border-gray-800 bg-gray-950/60 px-3 py-2 text-sm text-gray-300 transition-colors hover:border-red-500/40 hover:text-red-300"
          >
            退出登录
          </button>
        </div>
      </div>
    </aside>

    <main class="relative min-w-0 flex-1 overflow-y-auto">
      <div v-if="notification.show" class="fixed right-4 top-4 z-50 max-w-sm">
        <div
          class="flex items-center gap-3 rounded-2xl border px-4 py-3 text-sm shadow-2xl"
          :class="notification.type === 'error'
            ? 'border-red-700 bg-red-900/90 text-red-100'
            : notification.type === 'success'
              ? 'border-emerald-700 bg-emerald-900/90 text-emerald-100'
              : 'border-sky-700 bg-sky-900/90 text-sky-100'"
        >
          <span>{{ notification.type === 'error' ? '❌' : notification.type === 'success' ? '✅' : 'ℹ️' }}</span>
          <span>{{ notification.message }}</span>
        </div>
      </div>

      <router-view />
    </main>
  </div>
</template>

<script setup>
import { computed, onMounted, onUnmounted, provide, ref } from 'vue'
import { useRouter } from 'vue-router'
import { cockpitPlatforms, cockpitSystemRoutes } from '@/lib/platforms'
import { authAPI, settingsAPI } from './api'

const router = useRouter()

const platformRoutes = cockpitPlatforms
const systemRoutes = cockpitSystemRoutes

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
  }, 3200)
}

provide('notify', showNotification)

const isLoggedIn = computed(() => !!localStorage.getItem('easyllm_token'))

async function handleLogout() {
  try {
    await authAPI.logout()
  } catch {
    // ignore
  }
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
  @apply flex items-center gap-3 rounded-xl px-3 py-2.5 text-sm text-gray-400 transition-colors hover:bg-gray-800 hover:text-white;
}

.nav-item-active {
  @apply bg-sky-600 text-white shadow-lg shadow-sky-950/40 hover:bg-sky-500;
}
</style>
