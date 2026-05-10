import antigravityIcon from '@/assets/platforms/antigravity.png'
import geminiIcon from '@/assets/platforms/gemini.svg'
import githubCopilotIcon from '@/assets/platforms/github-copilot.svg'
import kiroIcon from '@/assets/platforms/kiro.svg'
import openAIIcon from '@/assets/platforms/openai.svg'

export const cockpitPlatforms = [
  {
    id: 'codex',
    label: 'Codex',
    icon: '🤖',
    iconSrc: openAIIcon,
    iconBg: '#ffffff',
    route: '/codex',
    description: '沿用 EasyLLM 现有 OpenAI / Codex 代理池和账号切换能力。',
    category: 'workspace',
    managementMode: 'legacy',
    heroClass: 'from-sky-500/20 via-blue-400/10 to-gray-950',
    supports: { instances: true, wakeup: true, quota: true },
  },
  {
    id: 'antigravity',
    label: 'Antigravity',
    icon: '🚀',
    iconSrc: antigravityIcon,
    iconBg: '#050505',
    iconTight: true,
    route: '/antigravity',
    description: '多账号、实例与唤醒任务的核心工作台。',
    category: 'workspace',
    managementMode: 'generic',
    heroClass: 'from-orange-500/20 via-amber-400/10 to-gray-950',
    supports: { instances: true, wakeup: true, quota: true },
  },
  {
    id: 'github-copilot',
    label: 'GitHub Copilot',
    icon: '🐙',
    iconSrc: githubCopilotIcon,
    iconBg: '#ffffff',
    route: '/github-copilot',
    description: '统一管理 Copilot 账号、实例和额度备注。',
    category: 'ide',
    managementMode: 'generic',
    heroClass: 'from-violet-500/20 via-indigo-400/10 to-gray-950',
    supports: { instances: true, wakeup: false, quota: true },
  },
  {
    id: 'kiro',
    label: 'Kiro',
    icon: '🪐',
    iconSrc: kiroIcon,
    iconBg: '#9046ff',
    route: '/kiro',
    description: '管理账号、路径与实例编排。',
    category: 'ide',
    managementMode: 'generic',
    heroClass: 'from-fuchsia-500/20 via-pink-400/10 to-gray-950',
    supports: { instances: true, wakeup: false, quota: true },
  },
  {
    id: 'gemini',
    label: 'Gemini CLI',
    icon: '✨',
    iconSrc: geminiIcon,
    iconBg: '#ffffff',
    route: '/gemini',
    description: '管理 Gemini CLI 账号、路径和刷新策略。',
    category: 'cli',
    managementMode: 'generic',
    heroClass: 'from-yellow-500/20 via-lime-300/10 to-gray-950',
    supports: { instances: false, wakeup: false, quota: true },
  },
]

export const cockpitPlatformMap = Object.fromEntries(cockpitPlatforms.map((item) => [item.id, item]))

export const cockpitSystemRoutes = [
  { path: '/dashboard', icon: '📊', label: '总览' },
  { path: '/docs', icon: '📖', label: '文档' },
  { path: '/settings', icon: '⚙️', label: '设置' },
]

export function getPlatformMeta(id) {
  return cockpitPlatformMap[id] || null
}
