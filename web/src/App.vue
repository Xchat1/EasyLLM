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
            <div class="px-2 pb-2 text-xs font-semibold uppercase tracking-wider text-gray-500">Codex</div>
            <div class="space-y-1">
              <router-link
                v-for="item in codexNavRoutes"
                :key="item.route"
                :to="item.route"
                class="nav-item"
                :class="{ 'nav-item-active': $route.path === item.route }"
              >
                <CodexIcon :item="item" size="xs" />
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
              <span class="text-gray-500">Local API</span>
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

      <div v-if="notification.show" class="fixed right-4 top-4 z-[120] w-[calc(100vw-2rem)] max-w-sm">
        <div
          class="operation-toast"
          :class="notificationClass"
        >
          <span class="operation-toast-mark" aria-hidden="true">{{ notificationMark }}</span>
          <span class="min-w-0 break-words">{{ notification.message }}</span>
        </div>
      </div>

      <div
        v-if="operationConfirm.show"
        class="fixed inset-0 z-[110] flex items-center justify-center bg-black/60 p-4"
        @click.self="cancelOperationConfirm"
      >
        <div class="w-full max-w-md rounded-2xl border border-gray-700 bg-gray-900 shadow-2xl">
          <div class="flex items-start gap-4 border-b border-gray-700 p-6">
            <div class="operation-confirm-mark" :class="operationConfirmMarkClass" aria-hidden="true">
              {{ operationConfirmMark }}
            </div>
            <div class="min-w-0 flex-1">
              <h2 class="text-lg font-semibold text-white">{{ operationConfirm.title }}</h2>
              <p class="mt-2 text-sm leading-6 text-gray-300">{{ operationConfirm.message }}</p>
              <p v-if="operationConfirm.details" class="mt-3 text-xs leading-5 text-gray-500">
                {{ operationConfirm.details }}
              </p>
            </div>
          </div>
          <div class="flex items-center justify-end gap-2 p-6">
            <button type="button" class="btn btn-secondary" @click="cancelOperationConfirm">
              {{ operationConfirm.cancelText }}
            </button>
            <button type="button" :class="operationConfirmButtonClass" @click="approveOperationConfirm">
              {{ operationConfirm.confirmText }}
            </button>
          </div>
        </div>
      </div>

      <router-view />
    </main>
  </div>
</template>

<script setup>
import { computed, onMounted, onUnmounted, provide, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import CodexIcon from '@/components/CodexIcon.vue'
import { codexRoutes, systemRoutes as appSystemRoutes } from '@/lib/codexRoutes'
import logoUrl from '@/assets/brand/easyllm-app-icon.png'
import { useAppearance } from '@/composables/useAppearance'
import { authAPI, settingsAPI } from './api'
import { isMacApp, syncMacAppFromRoute } from '@/lib/runtime'

const router = useRouter()
const route = useRoute()

const codexNavRoutes = codexRoutes
const systemRoutes = appSystemRoutes

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
const operationConfirm = ref(createEmptyOperationConfirm())
let notificationTimer = null
let operationConfirmResolver = null
let statusInterval = null

function showNotification(message, type = 'info') {
  if (notificationTimer) clearTimeout(notificationTimer)
  notification.value = { show: true, message, type: normalizeNotificationType(type) }
  notificationTimer = setTimeout(() => {
    notification.value.show = false
  }, 3200)
}

provide('notify', showNotification)
provide('confirmOperation', confirmOperation)

const notificationClass = computed(() => {
  if (notification.value.type === 'error') return 'operation-toast-error'
  if (notification.value.type === 'success') return 'operation-toast-success'
  if (notification.value.type === 'warning') return 'operation-toast-warning'
  return 'operation-toast-info'
})

const notificationMark = computed(() => {
  if (notification.value.type === 'error') return '!'
  if (notification.value.type === 'success') return '✓'
  if (notification.value.type === 'warning') return '!'
  return 'i'
})

const operationConfirmMark = computed(() => ['danger', 'warning'].includes(operationConfirm.value.tone) ? '!' : 'i')
const operationConfirmMarkClass = computed(() => {
  if (operationConfirm.value.tone === 'danger') return 'operation-confirm-mark-danger'
  if (operationConfirm.value.tone === 'warning') return 'operation-confirm-mark-warning'
  return 'operation-confirm-mark-info'
})
const operationConfirmButtonClass = computed(() => operationConfirm.value.tone === 'danger' ? 'btn btn-danger' : 'btn btn-primary')

function normalizeNotificationType(type) {
  return ['error', 'success', 'warning', 'info'].includes(type) ? type : 'info'
}

function createEmptyOperationConfirm() {
  return {
    show: false,
    title: '确认操作',
    message: '',
    details: '',
    confirmText: '确认',
    cancelText: '取消',
    tone: 'info',
  }
}

function confirmOperation(options = {}) {
  if (operationConfirmResolver) operationConfirmResolver(false)
  operationConfirm.value = {
    ...createEmptyOperationConfirm(),
    ...options,
    show: true,
  }
  return new Promise((resolve) => {
    operationConfirmResolver = resolve
  })
}

function resolveOperationConfirm(confirmed) {
  if (!operationConfirm.value.show) return
  const resolver = operationConfirmResolver
  operationConfirmResolver = null
  operationConfirm.value = createEmptyOperationConfirm()
  resolver?.(confirmed)
}

function cancelOperationConfirm() {
  resolveOperationConfirm(false)
}

function approveOperationConfirm() {
  resolveOperationConfirm(true)
}

function handleAppKeydown(event) {
  if (event.key === 'Escape' && operationConfirm.value.show) {
    cancelOperationConfirm()
  }
}

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

onMounted(async () => {
  window.addEventListener('keydown', handleAppKeydown)
  syncMacAppFromRoute(route)
  if (!isMacApp()) {
    try {
      const info = await settingsAPI.systemInfo()
      if (info?.mac_app) {
        sessionStorage.setItem('easyllm_mac_app', '1')
      }
    } catch {
      // ignore
    }
  }
  checkServerStatus()
  statusInterval = setInterval(checkServerStatus, 30000)
})

onUnmounted(() => {
  if (statusInterval) clearInterval(statusInterval)
  if (notificationTimer) clearTimeout(notificationTimer)
  window.removeEventListener('keydown', handleAppKeydown)
  if (operationConfirmResolver) operationConfirmResolver(false)
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

.operation-toast {
  @apply flex items-start gap-3 rounded-2xl border px-4 py-3 text-sm leading-5 shadow-2xl backdrop-blur;
}

.operation-toast-success {
  @apply border-emerald-600/40 bg-emerald-950/90 text-emerald-100;
}

.operation-toast-error {
  @apply border-red-600/40 bg-red-950/90 text-red-100;
}

.operation-toast-warning {
  @apply border-amber-500/40 bg-amber-950/90 text-amber-100;
}

.operation-toast-info {
  @apply border-sky-600/40 bg-sky-950/90 text-sky-100;
}

.operation-toast-mark,
.operation-confirm-mark {
  @apply inline-flex shrink-0 items-center justify-center rounded-full font-semibold;
}

.operation-toast-mark {
  @apply mt-0.5 h-5 w-5 text-xs;
  background: rgba(255, 255, 255, 0.14);
}

.operation-confirm-mark {
  @apply h-10 w-10 border text-base;
}

.operation-confirm-mark-danger {
  @apply border-red-500/30 bg-red-500/10 text-red-300;
}

.operation-confirm-mark-warning {
  @apply border-amber-500/30 bg-amber-500/10 text-amber-200;
}

.operation-confirm-mark-info {
  @apply border-sky-500/30 bg-sky-500/10 text-sky-200;
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
