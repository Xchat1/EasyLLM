import { createRouter, createWebHistory } from 'vue-router'
import { defaultHomePath, syncMacAppFromRoute } from '@/lib/runtime'

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
    path: '/openai',
    redirect: '/codex',
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
]

const router = createRouter({
  history: createWebHistory(),
  routes,
})

router.beforeEach(async (to, from, next) => {
  syncMacAppFromRoute(to)
  next()
})

export default router
