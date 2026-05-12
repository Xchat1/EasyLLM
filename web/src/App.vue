<template>
  <div v-if="$route.path === '/login'" class="h-screen">
    <router-view />
  </div>
  <div v-else class="app-shell flex h-screen overflow-hidden">
    <aside
      v-show="!sidebarCollapsed"
      class="app-sidebar"
      :class="{ 'app-sidebar-resizing': sidebarResizing }"
      :style="{ width: `${sidebarWidth}px` }"
    >
      <div class="flex h-full flex-col">
        <div class="sidebar-brand border-b border-gray-800 px-5 pb-4">
          <div class="flex min-w-0 items-center justify-between gap-3">
            <div class="flex min-w-0 items-center gap-3">
              <img
                :src="logoUrl"
                alt=""
                class="h-10 w-10 shrink-0 rounded-2xl shadow-lg shadow-cyan-950/40"
              />
              <div class="min-w-0">
                <div class="truncate font-semibold text-white">EasyLLM</div>
            <div class="truncate text-xs text-gray-500">EasyAI Coding</div>
              </div>
            </div>
            <button
              type="button"
              class="sidebar-icon-btn"
              title="隐藏侧边栏"
              aria-label="隐藏侧边栏"
              @click="collapseSidebar"
            >
              ‹
            </button>
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
                <PlatformIcon :platform="item" size="xs" />
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
          <button
            type="button"
            class="theme-toggle-btn"
            :title="appearanceButtonTitle"
            @click="cycleThemeMode"
          >
            <span class="theme-toggle-mark">{{ resolvedThemeMode === 'dark' ? '夜' : '昼' }}</span>
            <span class="min-w-0 text-left">
              <span class="block text-sm font-medium text-white">外观</span>
              <span class="block truncate text-xs text-gray-500">{{ themeModeShortLabel }} · {{ accentThemeLabel }}</span>
            </span>
          </button>
          <div class="rounded-2xl border border-gray-800 bg-gray-950/60 px-3 py-3 text-xs">
            <div class="flex items-center justify-between">
              <span class="text-gray-500">API Server</span>
              <div class="flex items-center gap-2">
                <span class="h-2 w-2 rounded-full" :class="serverRunning ? 'bg-emerald-400' : 'bg-gray-600'" />
                <span class="text-gray-300">:{{ serverPort }}</span>
              </div>
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
      <button
        type="button"
        class="sidebar-resize-handle"
        title="拖动调整侧边栏宽度"
        aria-label="拖动调整侧边栏宽度"
        @pointerdown="startSidebarResize"
        @dblclick="resetSidebarWidth"
      />
    </aside>

    <main class="relative min-w-0 flex-1 overflow-y-auto">
      <button
        v-if="sidebarCollapsed"
        type="button"
        class="sidebar-reopen-btn"
        title="显示侧边栏"
        aria-label="显示侧边栏"
        @click="expandSidebar"
      >
        ›
      </button>

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
import { computed, onMounted, onUnmounted, provide, ref, watch } from 'vue'
import { useRouter } from 'vue-router'
import PlatformIcon from '@/components/PlatformIcon.vue'
import { cockpitPlatforms, cockpitSystemRoutes } from '@/lib/platforms'
import logoUrl from '@/assets/brand/easyllm-app-icon.png'
import { useAppearance } from '@/composables/useAppearance'
import { authAPI, settingsAPI } from './api'

const router = useRouter()

const platformRoutes = cockpitPlatforms
const systemRoutes = cockpitSystemRoutes

const serverRunning = ref(false)
const serverPort = ref(8022)
const SIDEBAR_MIN_WIDTH = 220
const SIDEBAR_MAX_WIDTH = 420
const SIDEBAR_LEGACY_DEFAULT_WIDTH = 288
const SIDEBAR_DEFAULT_WIDTH = 245
const sidebarWidth = ref(loadSidebarWidth())
const sidebarCollapsed = ref(localStorage.getItem('easyllm_sidebar_collapsed') === '1')
const sidebarResizing = ref(false)
let sidebarResizeStartX = 0
let sidebarResizeStartWidth = SIDEBAR_DEFAULT_WIDTH

const {
  resolvedThemeMode,
  themeModeShortLabel,
  accentThemeLabel,
  cycleThemeMode,
} = useAppearance()
const appearanceButtonTitle = computed(() => `切换外观：${themeModeShortLabel.value} · ${accentThemeLabel.value}`)

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

watch(sidebarWidth, (value) => {
  localStorage.setItem('easyllm_sidebar_width', String(value))
})

watch(sidebarCollapsed, (value) => {
  localStorage.setItem('easyllm_sidebar_collapsed', value ? '1' : '0')
})

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

function loadSidebarWidth() {
  const stored = Number(localStorage.getItem('easyllm_sidebar_width'))
  if (!Number.isFinite(stored) || stored <= 0) return SIDEBAR_DEFAULT_WIDTH
  if (stored === SIDEBAR_LEGACY_DEFAULT_WIDTH) return SIDEBAR_DEFAULT_WIDTH
  return clampSidebarWidth(stored)
}

function clampSidebarWidth(width) {
  return Math.min(SIDEBAR_MAX_WIDTH, Math.max(SIDEBAR_MIN_WIDTH, Math.round(width)))
}

function collapseSidebar() {
  sidebarCollapsed.value = true
}

function expandSidebar() {
  sidebarCollapsed.value = false
}

function resetSidebarWidth() {
  sidebarWidth.value = SIDEBAR_DEFAULT_WIDTH
}

function startSidebarResize(event) {
  if (sidebarCollapsed.value) return
  sidebarResizing.value = true
  sidebarResizeStartX = event.clientX
  sidebarResizeStartWidth = sidebarWidth.value
  document.body.style.userSelect = 'none'
  document.body.style.cursor = 'col-resize'
  window.addEventListener('pointermove', resizeSidebar)
  window.addEventListener('pointerup', stopSidebarResize)
  window.addEventListener('pointercancel', stopSidebarResize)
  event.preventDefault()
}

function resizeSidebar(event) {
  if (!sidebarResizing.value) return
  sidebarWidth.value = clampSidebarWidth(sidebarResizeStartWidth + event.clientX - sidebarResizeStartX)
}

function stopSidebarResize() {
  if (!sidebarResizing.value) return
  sidebarResizing.value = false
  document.body.style.userSelect = ''
  document.body.style.cursor = ''
  window.removeEventListener('pointermove', resizeSidebar)
  window.removeEventListener('pointerup', stopSidebarResize)
  window.removeEventListener('pointercancel', stopSidebarResize)
}

onMounted(() => {
  checkServerStatus()
  statusInterval = setInterval(checkServerStatus, 30000)
})

onUnmounted(() => {
  if (statusInterval) clearInterval(statusInterval)
  if (notificationTimer) clearTimeout(notificationTimer)
  stopSidebarResize()
})
</script>

<style>
.app-shell {
  background:
    radial-gradient(circle at 18% 0%, var(--app-bg-glow), transparent 32rem),
    var(--app-bg);
  color: var(--app-text);
}

.app-sidebar {
  @apply relative flex-shrink-0 border-r;
  background: var(--app-sidebar-bg);
  border-color: var(--app-border);
  min-width: 220px;
  max-width: 420px;
}

.app-sidebar-resizing {
  @apply select-none;
}

.sidebar-brand {
  padding-top: 2.75rem;
}

.sidebar-icon-btn {
  @apply flex h-7 w-7 shrink-0 items-center justify-center rounded-lg border text-lg leading-none transition-colors;
  background: var(--app-control-bg);
  border-color: var(--app-border);
  color: var(--app-text-muted);
}

.sidebar-icon-btn:hover {
  border-color: var(--app-accent-soft);
  color: var(--app-text);
}

.sidebar-reopen-btn {
  @apply fixed left-3 top-3 z-40 flex h-8 w-8 items-center justify-center rounded-lg border text-xl leading-none shadow-xl transition-colors;
  background: var(--app-sidebar-bg);
  border-color: var(--app-border);
  color: var(--app-text-secondary);
  box-shadow: var(--app-shadow-lg);
}

.sidebar-reopen-btn:hover {
  border-color: var(--app-accent-soft);
  color: var(--app-text);
}

.sidebar-resize-handle {
  @apply absolute right-0 top-0 h-full w-2 cursor-col-resize border-0 bg-transparent transition-colors;
  transform: translateX(50%);
}

.sidebar-resize-handle:hover,
.app-sidebar-resizing .sidebar-resize-handle {
  background: var(--app-accent-soft);
}

.theme-toggle-btn {
  @apply flex w-full items-center gap-3 rounded-2xl border px-3 py-2.5 transition-colors;
  background: var(--app-control-bg);
  border-color: var(--app-border);
}

.theme-toggle-btn:hover {
  border-color: var(--app-accent-soft);
  background: var(--app-control-hover-bg);
}

.theme-toggle-mark {
  @apply flex h-8 w-8 shrink-0 items-center justify-center rounded-xl text-xs font-semibold text-white;
  background: linear-gradient(135deg, var(--app-accent), var(--app-accent-strong));
  box-shadow: 0 8px 22px var(--app-accent-shadow);
}

.nav-item {
  @apply flex items-center gap-3 rounded-xl px-3 py-2.5 text-sm transition-colors;
  color: var(--app-text-secondary);
}

.nav-item:hover {
  background: var(--app-control-hover-bg);
  color: var(--app-text);
}

.nav-item-active {
  color: #fff;
  background: linear-gradient(135deg, var(--app-accent), var(--app-accent-strong));
  box-shadow: 0 14px 34px var(--app-accent-shadow);
}

.nav-item-active:hover {
  color: #fff;
  background: linear-gradient(135deg, var(--app-accent), var(--app-accent-strong));
}
</style>
