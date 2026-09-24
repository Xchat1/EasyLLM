import { createRouter, createWebHistory } from 'vue-router'
import { defaultHomePath, syncMacAppFromRoute } from '@/lib/runtime'
import { fetchAuthStatus } from '@/lib/auth'

const routes = [
  {
    path: '/',
    redirect: () => defaultHomePath(),
  },
  {
    path: '/login',
    name: 'login',
    component: () => import('@/views/LoginView.vue'),
    meta: { title: '登录', public: true },
  },
  {
    path: '/dashboard',
    name: 'dashboard',
    component: () => import('@/views/DashboardView.vue'),
    meta: { title: '总览', icon: '📊' },
  },
  {
    path: '/codex',
    name: 'codex',
    component: () => import('@/views/OpenAIView.vue'),
    meta: { title: 'Codex', icon: '🤖' },
  },
  {
    path: '/antigravity',
    name: 'antigravity',
    component: () => import('@/views/AntigravityView.vue'),
    meta: { title: 'Antigravity', icon: '🚀' },
  },
  {
    path: '/cursor',
    name: 'cursor',
    component: () => import('@/views/CursorView.vue'),
    meta: { title: 'Cursor', icon: '⚡' },
  },
  {
    path: '/openai',
    redirect: '/codex',
  },
  {
    path: '/ag',
    redirect: '/antigravity',
  },
  {
    path: '/docs',
    name: 'docs',
    component: () => import('@/views/DocsView.vue'),
    meta: { title: '文档', icon: '📖' },
  },
  {
    path: '/settings',
    name: 'settings',
    component: () => import('@/views/SettingsView.vue'),
    meta: { title: '设置', icon: '⚙️' },
  },
  {
    path: '/relay',
    name: 'relay',
    component: () => import('@/views/RelayConfigView.vue'),
    meta: { title: 'Relay', icon: '🔗' },
  },
]

const router = createRouter({
  history: createWebHistory(),
  routes,
})

router.beforeEach(async (to, from, next) => {
  syncMacAppFromRoute(to)

  const authStatus = await fetchAuthStatus()
  const authEnabled = !!authStatus?.auth_enabled

  if (to.path === '/login') {
    if (!authEnabled) {
      next(defaultHomePath())
      return
    }
    next()
    return
  }

  if (to.meta.public) {
    next()
    return
  }

  if (authEnabled) {
    const token = localStorage.getItem('easyllm_token')
    if (!token) {
      next('/login')
      return
    }
  }
  next()
})

export default router
