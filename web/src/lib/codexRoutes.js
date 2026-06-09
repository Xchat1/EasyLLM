import openAIIcon from '@/assets/brand/openai.svg'

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
]

export const codexRouteMap = Object.fromEntries(codexRoutes.map((item) => [item.id, item]))

export const systemRoutes = [
  { path: '/dashboard', icon: '📊', label: '总览' },
  { path: '/docs', icon: '📖', label: '文档' },
  { path: '/settings', icon: '⚙️', label: '设置' },
]

export function getCodexRouteMeta(id) {
  return codexRouteMap[id] || null
}
