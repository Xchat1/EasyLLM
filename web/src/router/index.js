import { createRouter, createWebHistory } from 'vue-router'

import { cockpitPlatforms } from '@/lib/platforms'

const genericPlatformRoutes = cockpitPlatforms
  .filter((platform) => platform.managementMode === 'generic')
  .map((platform) => ({
    path: platform.route,
    name: platform.id,
    component: () => import('@/views/PlatformWorkspaceView.vue'),
    props: { platformId: platform.id },
    meta: { title: platform.label, icon: platform.icon },
  }))

const routes = [
  {
    path: '/',
    redirect: '/dashboard',
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
  ...genericPlatformRoutes,
  {
    path: '/instances',
    name: 'instances',
    component: () => import('@/views/InstancesView.vue'),
    meta: { title: '实例', icon: '🪟' },
  },
  {
    path: '/wakeup',
    name: 'wakeup',
    component: () => import('@/views/WakeupView.vue'),
    meta: { title: '唤醒', icon: '⏰' },
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
  if (to.meta.public) {
    next()
    return
  }

  const token = localStorage.getItem('easyllm_token')
  if (!token) {
    next('/login')
    return
  }
  next()
})

export default router
