<template>
  <div class="docs-page p-4 sm:p-6 w-full max-w-7xl mx-auto space-y-6">
    <!-- Header -->
    <div class="mb-8">
      <h1 class="text-3xl font-bold text-white mb-2">📖 使用文档</h1>
      <p class="text-gray-400">快速上手 EasyLLM，管理你的 AI 开发工具与账号</p>
    </div>

    <!-- Quick nav -->
    <div class="card docs-card p-4">
      <div class="text-xs font-semibold text-gray-500 uppercase tracking-wider mb-3">快速导航</div>
      <div class="grid docs-nav-grid gap-2">
        <button v-for="section in sections" :key="section.id"
          @click="scrollTo(section.id)"
          class="flex items-center gap-2 px-3 py-2 bg-gray-800/50 hover:bg-gray-700/50 rounded-lg text-sm text-gray-300 hover:text-white transition-all duration-200 border border-gray-700/50 hover:border-blue-500/50">
          <span class="text-lg">{{ section.icon }}</span>
          <span class="truncate">{{ section.label }}</span>
        </button>
      </div>
    </div>

    <div class="space-y-6">

      <!-- Overview -->
      <div id="sec-overview" class="card doc-section p-4 sm:p-5">
        <h2 class="text-xl font-semibold text-white mb-4 flex items-center gap-2">
          <span class="text-2xl">🎯</span> 产品简介
        </h2>
        <div class="grid grid-cols-1 md:grid-cols-3 gap-4 mb-4">
          <div class="bg-gradient-to-br from-blue-500/10 to-purple-500/10 border border-blue-500/20 rounded-lg p-4">
            <div class="text-2xl mb-2">🤖</div>
            <div class="text-sm font-semibold text-white mb-1">多平台账号管理</div>
            <div class="text-xs text-gray-400">OpenAI / Codex、Cursor、Antigravity 统一Web界面管理</div>
          </div>
          <div class="bg-gradient-to-br from-green-500/10 to-teal-500/10 border border-green-500/20 rounded-lg p-4">
            <div class="text-2xl mb-2">⚡</div>
            <div class="text-sm font-semibold text-white mb-1">智能代理池</div>
            <div class="text-xs text-gray-400">多账号自动负载均衡，支持轮询/随机/最少使用策略</div>
          </div>
          <div class="bg-gradient-to-br from-orange-500/10 to-red-500/10 border border-orange-500/20 rounded-lg p-4">
            <div class="text-2xl mb-2">🔒</div>
            <div class="text-sm font-semibold text-white mb-1">安全可靠</div>
            <div class="text-xs text-gray-400">API Key 鉴权、IP 黑名单、请求日志监控</div>
          </div>
        </div>
        <div class="flex flex-wrap gap-2 text-xs">
          <span class="px-2 py-1 bg-gray-800 rounded text-gray-400">Go 1.25</span>
          <span class="px-2 py-1 bg-gray-800 rounded text-gray-400">Vue 3</span>
          <span class="px-2 py-1 bg-gray-800 rounded text-gray-400">SQLite / PostgreSQL</span>
          <span class="px-2 py-1 bg-gray-800 rounded text-gray-400">Docker 支持</span>
        </div>
      </div>

      <!-- Codex CLI -->
      <div id="sec-codex" class="card doc-section p-4 sm:p-5">
        <h2 class="text-xl font-semibold text-white mb-4 flex items-center gap-2">
          <span class="text-2xl">🖥️</span> Codex CLI 接入
        </h2>
        <p class="text-sm text-gray-400 mb-5">将 EasyLLM 作为 Codex CLI 的代理，实现多账号轮询、请求日志记录和本机配置注入。</p>

        <!-- Method 1 -->
        <div class="mb-5 p-4 bg-blue-500/5 border border-blue-500/20 rounded-lg">
          <div class="flex items-center gap-2 mb-2">
            <span class="px-2 py-0.5 bg-blue-500/20 text-blue-400 text-xs font-semibold rounded">推荐</span>
            <h3 class="text-sm font-semibold text-white">方式一：本地 API 服务</h3>
          </div>
          <p class="text-xs text-gray-400 mb-3">在 OpenAI / Codex 页面导入 OAuth 账号后，打开"服务配置"，点击"启动并注入 Codex"，EasyLLM 会写入本机 Codex 配置并使用账号集合调度请求。</p>
          <div class="doc-code">
            <div class="doc-code-header">自动配置的 ~/.codex/config.toml</div>
            <pre>model_provider = "easyllm"
model = "gpt-5-codex"

[model_providers.easyllm]
name = "EasyLLM API Service"
base_url = "http://localhost:{{ port }}/v1"
wire_api = "responses"
requires_openai_auth = true</pre>
            <button @click="copyCurl('codex-oauth')" class="doc-code-copy">复制</button>
          </div>
        </div>

        <!-- Method 2 -->
        <div class="mb-5">
          <h3 class="text-sm font-semibold text-white mb-2">方式二：OAuth 单账号切换</h3>
          <p class="text-xs text-gray-400 mb-3">在 OpenAI / Codex 页面添加 OAuth 账号后，点击账号卡片里的"切换"按钮即可写入 <code class="code">~/.codex/auth.json</code>。</p>
          <div class="doc-code">
            <div class="doc-code-header">自动配置的 ~/.codex/config.toml</div>
            <pre>chatgpt_base_url = "http://localhost:{{ port }}"</pre>
            <button @click="copyCurl('codex-pool')" class="doc-code-copy">复制</button>
          </div>
        </div>

        <!-- Method 3 -->
        <div class="mb-5">
          <h3 class="text-sm font-semibold text-white mb-2">方式三：API Key 账号</h3>
          <p class="text-xs text-gray-400 mb-3">在 OpenAI / Codex 页面的"API 配置"标签添加第三方 Provider（如 OpenRouter、DeepSeek 等）。</p>
          <div class="doc-code">
            <div class="doc-code-header">示例：配置 OpenRouter</div>
            <pre>model_provider = "openrouter"
model = "deepseek/deepseek-chat"

[model_providers.openrouter]
name = "openrouter"
base_url = "https://openrouter.ai/api/v1"
wire_api = "chat"</pre>
            <button @click="copyCurl('openrouter')" class="doc-code-copy">复制</button>
          </div>
        </div>

        <!-- Method 4 -->
        <div>
          <h3 class="text-sm font-semibold text-white mb-2">方式四：代理池模式</h3>
          <p class="text-xs text-gray-400 mb-3">启用多个 OAuth 账号的"代理开关"，请求将按配置策略调度到账号池。</p>
          <div class="doc-code">
            <div class="doc-code-header">~/.codex/config.toml</div>
            <pre>chatgpt_base_url = "http://localhost:{{ port }}"</pre>
            <button @click="copyCurl('codex-pool')" class="doc-code-copy">复制</button>
          </div>
        </div>
      </div>

      <!-- cURL -->
      <div id="sec-curl" class="card doc-section p-4 sm:p-5">
        <h2 class="text-xl font-semibold text-white mb-4 flex items-center gap-2">
          <span class="text-2xl">📡</span> cURL 调用示例
        </h2>
        <p class="text-sm text-gray-400 mb-5">通过代理池的 OpenAI 兼容接口发送请求。</p>

        <div class="space-y-4">
          <div>
            <h3 class="text-sm font-semibold text-white mb-2">Chat Completions（流式）</h3>
            <div class="doc-code">
              <div class="doc-code-header">bash</div>
              <pre>curl http://localhost:{{ port }}/v1/responses \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer YOUR_API_KEY" \
  -d '{
    "model": "gpt-5.4",
    "input": "写一个快速排序算法",
    "stream": true
  }'</pre>
              <button @click="copyCurl('responses')" class="doc-code-copy">复制</button>
            </div>
          </div>

          <div>
            <h3 class="text-sm font-semibold text-white mb-2">获取可用模型列表</h3>
            <div class="doc-code">
              <div class="doc-code-header">bash</div>
              <pre>curl http://localhost:{{ port }}/v1/models \
  -H "Authorization: Bearer YOUR_API_KEY"</pre>
              <button @click="copyCurl('models')" class="doc-code-copy">复制</button>
            </div>
          </div>

          <div>
            <h3 class="text-sm font-semibold text-white mb-2">查看代理池状态</h3>
            <div class="doc-code">
              <div class="doc-code-header">bash</div>
              <pre>curl http://localhost:{{ port }}/pool/status</pre>
              <button @click="copyCurl('pool')" class="doc-code-copy">复制</button>
            </div>
          </div>
        </div>
      </div>

      <!-- Python -->
      <div id="sec-python" class="card doc-section p-4 sm:p-5">
        <h2 class="text-xl font-semibold text-white mb-4 flex items-center gap-2">
          <span class="text-2xl">🐍</span> Python 调用
        </h2>
        <p class="text-sm text-gray-400 mb-4">使用 OpenAI Python SDK 通过 EasyLLM 代理池发送请求。</p>

        <div class="doc-code">
          <div class="doc-code-header">python</div>
          <pre>from openai import OpenAI

client = OpenAI(
    base_url="http://localhost:{{ port }}/v1",
    api_key="YOUR_API_KEY",  # 在服务配置中设置的 API Key
)

response = client.responses.create(
    model="gpt-5.4",
    input="用 Python 写一个 HTTP 服务器",
)

print(response.output_text)</pre>
          <button @click="copyCurl('python')" class="doc-code-copy">复制</button>
        </div>
      </div>

      <!-- Quota -->
      <div id="sec-quota" class="card doc-section p-4 sm:p-5">
        <h2 class="text-xl font-semibold text-white mb-4 flex items-center gap-2">
          <span class="text-2xl">📊</span> 配额查询
        </h2>
        <p class="text-sm text-gray-400 mb-4">查看 5 小时和 7 天配额使用情况（支持图表与趋势分析）。</p>

        <div class="grid grid-cols-1 md:grid-cols-2 gap-4 mb-4">
          <div class="bg-gray-800/50 rounded-lg p-4 border border-gray-700/50">
            <div class="text-sm font-semibold text-white mb-2">⏱️ 5 小时配额</div>
            <div class="text-xs text-gray-400">短期会话限制，快速重置</div>
          </div>
          <div class="bg-gray-800/50 rounded-lg p-4 border border-gray-700/50">
            <div class="text-sm font-semibold text-white mb-2">📅 7 天配额</div>
            <div class="text-xs text-gray-400">长期总量限制，周重置周期</div>
          </div>
        </div>
        <div class="text-xs text-gray-500">
          💡 在 OpenAI 账号列表中点击"刷新配额"即可获取最新的配额数据
        </div>
      </div>

      <!-- Import -->
      <div id="sec-import" class="card doc-section p-4 sm:p-5">
        <h2 class="text-xl font-semibold text-white mb-4 flex items-center gap-2">
          <span class="text-2xl">📦</span> 批量导入
        </h2>
        <p class="text-sm text-gray-400 mb-4">支持 token 文件、扫描目录、refresh_token、Sub2API、cockpit-tools 导出文件，以及 EasyLLM 备份文件。</p>

        <div class="space-y-4">
          <div>
            <h3 class="text-sm font-semibold text-white mb-2">通过 refresh_token 导入</h3>
            <div class="doc-code">
              <div class="doc-code-header">bash</div>
              <pre>curl -X POST http://localhost:{{ port }}/api/v1/openai/import/refresh-tokens \
  -H "Content-Type: application/json" \
  -d '{
    "refresh_tokens": [
      "REFRESH_TOKEN_1",
      "REFRESH_TOKEN_2"
    ]
  }'</pre>
              <button @click="copyCurl('openai-import')" class="doc-code-copy">复制</button>
            </div>
          </div>

          <div>
            <h3 class="text-sm font-semibold text-white mb-2">扫描目录导入</h3>
            <p class="text-xs text-gray-400 mb-2">将 token JSON 文件放在 <code class="code">./auth/</code> 目录下，然后调用扫描接口自动导入。</p>
            <div class="doc-code">
              <div class="doc-code-header">bash</div>
              <pre>curl -X POST http://localhost:{{ port }}/api/v1/openai/import/scan-dir \
  -H "Content-Type: application/json" \
  -d '{"dir": "./auth"}'</pre>
              <button @click="copyCurl('openai-scan')" class="doc-code-copy">复制</button>
            </div>
          </div>

          <div>
            <h3 class="text-sm font-semibold text-white mb-2">备份恢复</h3>
            <p class="text-xs text-gray-400 mb-2">通过"一键导出所有最新数据"生成的备份可在"批量导入 → 从备份导入"恢复账号；如果备份包含本地 API 服务配置，也会一并恢复账号集合、端口和调度策略。</p>
            <div class="doc-code">
              <div class="doc-code-header">备份文件结构</div>
              <pre>{
  "oauth_accounts": [],
  "api_accounts": [],
  "local_access": {
    "enabled": true,
    "port": 8022,
    "routing_strategy": "auto",
    "account_ids": []
  }
}</pre>
            </div>
          </div>
        </div>
      </div>

      <!-- Auth -->
      <div id="sec-auth" class="card doc-section p-4 sm:p-5">
        <h2 class="text-xl font-semibold text-white mb-4 flex items-center gap-2">
          <span class="text-2xl">🔒</span> 代理池鉴权
        </h2>
        <p class="text-sm text-gray-400 mb-4">保护你的代理池端点，防止未授权访问。</p>

        <div class="space-y-3 mb-4">
          <div class="flex gap-3 items-start p-3 bg-gray-800/30 rounded-lg">
            <span class="flex-shrink-0 w-6 h-6 bg-blue-500/20 text-blue-400 text-xs font-bold rounded-full flex items-center justify-center">1</span>
            <span class="text-sm text-gray-300">在 OpenAI / Codex 页面 → 服务配置 → 设置一个 API Key</span>
          </div>
          <div class="flex gap-3 items-start p-3 bg-gray-800/30 rounded-lg">
            <span class="flex-shrink-0 w-6 h-6 bg-blue-500/20 text-blue-400 text-xs font-bold rounded-full flex items-center justify-center">2</span>
            <span class="text-sm text-gray-300">所有 <code class="code">/v1/*</code> 请求都需要携带 <code class="code">Authorization: Bearer YOUR_KEY</code></span>
          </div>
          <div class="flex gap-3 items-start p-3 bg-gray-800/30 rounded-lg">
            <span class="flex-shrink-0 w-6 h-6 bg-blue-500/20 text-blue-400 text-xs font-bold rounded-full flex items-center justify-center">3</span>
            <span class="text-sm text-gray-300">本地 Codex CLI 通过已知的 managed token 认证（passthrough 模式），无需额外配置</span>
          </div>
        </div>

        <div class="doc-code">
          <div class="doc-code-header">支持的负载均衡策略</div>
          <pre>auto               — 综合订阅计划和剩余额度选择
quota_high_first   — 优先使用剩余额度高的账号
quota_low_first    — 优先消耗剩余额度低的账号
plan_high_first    — 优先使用高订阅账号
plan_low_first     — 优先使用低订阅账号
expiry_soon_first  — 优先使用更早到期的账号
round_robin        — 轮询
random             — 随机
least_used         — 选择请求次数最少的账号</pre>
        </div>
      </div>

      <!-- Docker -->
      <div id="sec-docker" class="card doc-section p-4 sm:p-5">
        <h2 class="text-xl font-semibold text-white mb-4 flex items-center gap-2">
          <span class="text-2xl">🐳</span> Docker 部署
        </h2>
        <p class="text-sm text-gray-400 mb-4">使用 Docker 快速部署 EasyLLM。</p>

        <div class="doc-code mb-4">
          <div class="doc-code-header">docker-compose.yml</div>
          <pre>services:
  easyllm:
    build: .
    ports:
      - "8022:8022"
    volumes:
      - ./data:/app/data
    environment:
      - SERVER_PORT=8022
      - DB_TYPE=sqlite
    restart: unless-stopped</pre>
          <button @click="copyCurl('docker')" class="doc-code-copy">复制</button>
        </div>

        <div class="doc-code">
          <div class="doc-code-header">启动命令</div>
          <pre>docker compose up -d</pre>
        </div>
      </div>

      <!-- FAQ -->
      <div id="sec-faq" class="card doc-section p-4 sm:p-5">
        <h2 class="text-xl font-semibold text-white mb-4 flex items-center gap-2">
          <span class="text-2xl">❓</span> 常见问题
        </h2>
        <div class="space-y-4">
          <div v-for="(faq, index) in faqs" :key="index" class="p-4 bg-gray-800/30 rounded-lg border border-gray-700/50">
            <div class="text-sm font-semibold text-white mb-2 flex items-start gap-2">
              <span class="text-blue-400 mt-0.5">Q:</span>
              <span>{{ faq.q }}</span>
            </div>
            <div class="text-sm text-gray-400 ml-5">{{ faq.a }}</div>
          </div>
        </div>
      </div>

    </div>
  </div>
</template>

<script setup>
import { ref, inject, onMounted } from 'vue'
import { settingsAPI } from '@/api'

const notify = inject('notify')
const port = ref(8022)

onMounted(async () => {
  try {
    const data = await settingsAPI.apiServerStatus()
    if (data.port) port.value = data.port
  } catch {}
})

const sections = [
  { id: 'sec-overview', icon: '🎯', label: '简介' },
  { id: 'sec-codex', icon: '🖥️', label: 'Codex CLI' },
  { id: 'sec-curl', icon: '📡', label: 'cURL' },
  { id: 'sec-python', icon: '🐍', label: 'Python' },
  { id: 'sec-quota', icon: '📊', label: '配额查询' },
  { id: 'sec-import', icon: '📦', label: '批量导入' },
  { id: 'sec-auth', icon: '🔒', label: '鉴权' },
  { id: 'sec-docker', icon: '🐳', label: 'Docker' },
  { id: 'sec-faq', icon: '❓', label: 'FAQ' },
]

const faqs = [
  { q: 'Codex CLI 报 "Token data is not available." 怎么办？', a: '确保 auth.json 中 last_refresh 在顶层而非 tokens 内部。在 EasyLLM 中重新点击"切换"即可自动修复。' },
  { q: '代理池请求返回 401 Unauthorized', a: '检查是否在服务配置中设置了 API Key。如果设置了，所有 /v1/* 请求都需要在 Header 中携带 Authorization: Bearer YOUR_KEY。' },
  { q: 'Token 过期了怎么办？', a: '在 OpenAI 账号列表中点击"刷新 Token"按钮，或使用"刷新全部"一键刷新所有 OAuth 账号。' },
  { q: '配额查询显示 Forbidden', a: '该账号可能没有 Codex 访问权限（需要 ChatGPT Plus/Pro 订阅），或 Token 已失效。' },
  { q: '如何更改数据库？', a: '在设置 → 数据库页面切换为 PostgreSQL 并填写 DSN，保存后重启服务即可。' },
  { q: '如何在公网暴露服务？', a: '建议在前面加 Nginx 反向代理并启用 HTTPS。同时务必设置代理池 API Key 和 IP 黑名单来保护端点。' },
  { q: '5h 和 7d 配额是什么意思？', a: '5h 是短期会话限制，7d 是长期总量限制。在 OpenAI 页面点击"刷新配额"可查看最新使用情况。' },
]

const curlSnippets = {
  'codex-oauth': `model_provider = "easyllm"
model = "gpt-5-codex"

[model_providers.easyllm]
name = "EasyLLM API Service"
base_url = "http://localhost:PORT/v1"
wire_api = "responses"
requires_openai_auth = true`,
  openrouter: `model_provider = "openrouter"
model = "deepseek/deepseek-chat"

[model_providers.openrouter]
name = "openrouter"
base_url = "https://openrouter.ai/api/v1"
wire_api = "chat"`,
  'codex-pool': `chatgpt_base_url = "http://localhost:PORT"`,
  responses: `curl http://localhost:PORT/v1/responses \\
  -H "Content-Type: application/json" \\
  -H "Authorization: Bearer YOUR_API_KEY" \\
  -d '{
    "model": "gpt-5.4",
    "input": "写一个快速排序算法",
    "stream": true
  }'`,
  models: `curl http://localhost:PORT/v1/models \\
  -H "Authorization: Bearer YOUR_API_KEY"`,
  pool: `curl http://localhost:PORT/pool/status`,
  python: `from openai import OpenAI

client = OpenAI(
    base_url="http://localhost:PORT/v1",
    api_key="YOUR_API_KEY",
)

response = client.responses.create(
    model="gpt-5.4",
    input="用 Python 写一个 HTTP 服务器",
)

print(response.output_text)`,
  'openai-import': `curl -X POST http://localhost:PORT/api/v1/openai/import/refresh-tokens \\
  -H "Content-Type: application/json" \\
  -d '{
    "refresh_tokens": [
      "REFRESH_TOKEN_1",
      "REFRESH_TOKEN_2"
    ]
  }'`,
  'openai-scan': `curl -X POST http://localhost:PORT/api/v1/openai/import/scan-dir \\
  -H "Content-Type: application/json" \\
  -d '{"dir": "./auth"}'`,
  docker: `services:
  easyllm:
    build: .
    ports:
      - "8022:8022"
    volumes:
      - ./data:/app/data
    environment:
      - SERVER_PORT=8022
      - DB_TYPE=sqlite
    restart: unless-stopped`,
}

function copyCurl(key) {
  const text = (curlSnippets[key] || '').replace(/PORT/g, port.value)
  navigator.clipboard.writeText(text).then(() => notify('已复制', 'success'))
}

function scrollTo(id) {
  document.getElementById(id)?.scrollIntoView({ behavior: 'smooth', block: 'start' })
}
</script>

<style scoped>
.docs-page {
  min-width: 0;
}
.docs-card,
.doc-section {
  min-width: 0;
}
.doc-section {
  scroll-margin-top: 24px;
}
.docs-nav-grid {
  grid-template-columns: repeat(auto-fit, minmax(112px, 1fr));
}
.code {
  @apply bg-gray-800 text-blue-400 px-1.5 py-0.5 rounded text-xs font-mono;
}
.doc-code {
  @apply bg-gray-950 border border-gray-800 rounded-lg overflow-hidden relative max-w-full;
  min-width: 0;
}
.doc-code-header {
  @apply bg-gray-900 pl-3 pr-16 py-1.5 text-xs text-gray-500 border-b border-gray-800 font-mono truncate;
}
.doc-code pre {
  @apply px-4 py-3 text-xs text-gray-300 font-mono overflow-x-auto whitespace-pre leading-relaxed;
  max-width: 100%;
  min-width: 0;
}
.doc-code-copy {
  @apply absolute top-1.5 right-2 text-xs text-gray-500 hover:text-white bg-gray-800 hover:bg-gray-700
         px-2 py-0.5 rounded transition-colors;
}
@media (max-width: 640px) {
  .docs-nav-grid {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }
  .doc-code pre {
    @apply px-3 text-[11px];
  }
}
</style>
