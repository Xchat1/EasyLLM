import openAIIcon from '@/assets/brand/openai.svg'
import cursorIcon from '@/assets/brand/cursor.svg'

export const codexRoutes = [
  {
    id: 'codex',
    label: 'Codex',
    icon: '🤖',
    iconSrc: openAIIcon,
    iconBg: '#ffffff',
    route: '/codex',
    description: '管理 OpenAI OAuth 账号及 Codex API 配置。',
    category: 'workspace',
    managementMode: 'legacy',
    heroClass: 'from-sky-500/20 via-blue-400/10 to-gray-950',
    supports: { instances: false, wakeup: false, quota: true },
  },
  {
    id: 'antigravity',
    label: 'Antigravity',
    icon: '🚀',
    iconBg: '#1e1b4b',
    route: '/antigravity',
    description: '管理 Google OAuth 账号、配额监控与 Antigravity IDE 切号。',
    category: 'workspace',
    managementMode: 'generic',
    heroClass: 'from-orange-500/20 via-amber-400/10 to-gray-950',
    supports: { instances: false, wakeup: true, quota: true },
  },
  {
    id: 'cursor',
    label: 'Cursor',
    icon: '⚡',
    iconSrc: cursorIcon,
    iconBg: '#18181b',
    route: '/cursor',
    description: '管理 Cursor 账号凭据、配额监控与 Cursor IDE 一键切号。',
    category: 'workspace',
    managementMode: 'generic',
    heroClass: 'from-purple-500/20 via-indigo-400/10 to-gray-950',
    supports: { instances: false, wakeup: true, quota: true },
  },
]

export const codexRouteMap = Object.fromEntries(codexRoutes.map((item) => [item.id, item]))

// codexModeRoutes — Codex 对接模式入口，显示在 Codex 分区账号列表之下
export const codexModeRoutes = [
  { path: '/relay', icon: '🔗', label: 'Relay' },
]

export const systemRoutes = [
  { path: '/dashboard', icon: '📊', label: '总览' },
  { path: '/docs', icon: '📖', label: '文档' },
  { path: '/settings', icon: '⚙️', label: '设置' },
]

export function getCodexRouteMeta(id) {
  return codexRouteMap[id] || null
}
