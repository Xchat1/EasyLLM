<template>
  <div class="p-6 max-w-4xl">
    <h1 class="text-2xl font-bold text-white mb-6">📖 使用文档</h1>

    <!-- Quick nav -->
    <div class="flex flex-wrap gap-2 mb-6">
      <button v-for="section in sections" :key="section.id"
        @click="scrollTo(section.id)"
        class="btn btn-secondary btn-sm">
        {{ section.icon }} {{ section.label }}
      </button>
    </div>

    <div class="space-y-6">

      <!-- Codex CLI -->
      <div id="sec-codex" class="card p-5">
        <h2 class="text-lg font-semibold text-white mb-1">Codex CLI 接入</h2>
        <p class="text-sm text-gray-400 mb-4">将 EasyLLM 作为 Codex CLI 的代理，实现多账号轮询、请求日志记录。</p>

        <h3 class="text-sm font-semibold text-blue-400 mb-2">方式一：OAuth 账号（推荐）</h3>
        <p class="text-xs text-gray-400 mb-2">在 OpenAI / Codex 页面添加 OAuth 账号后，点击"切换"按钮即可自动写入 <code class="code">~/.codex/auth.json</code>。EasyLLM 会自动注入 <code class="code">chatgpt_base_url</code>，让 Codex CLI 的请求经过本地代理。</p>
        <div class="doc-code mb-4">
          <div class="doc-code-header">自动配置的 ~/.codex/config.toml</div>
          <pre>chatgpt_base_url = "http://localhost:{{ port }}"</pre>
        </div>

        <h3 class="text-sm font-semibold text-blue-400 mb-2">方式二：API Key 账号</h3>
        <p class="text-xs text-gray-400 mb-2">在 OpenAI / Codex 页面的"API 配置"标签添加第三方 Provider（如 OpenRouter、DeepSeek 等），点击"切换"后自动写入 <code class="code">~/.codex/config.toml</code>。</p>
        <div class="doc-code mb-4">
          <div class="doc-code-header">示例：配置 OpenRouter</div>
          <pre>model_provider = "openrouter"
model = "deepseek/deepseek-chat"

[model_providers.openrouter]
name = "openrouter"
base_url = "https://openrouter.ai/api/v1"
wire_api = "chat"</pre>
        </div>

        <h3 class="text-sm font-semibold text-blue-400 mb-2">方式三：代理池模式</h3>
        <p class="text-xs text-gray-400 mb-2">启用多个 OAuth 账号的"代理开关"，然后在 Codex CLI 中将 <code class="code">chatgpt_base_url</code> 指向 EasyLLM，请求将自动轮询池中账号。</p>
        <div class="doc-code">
          <div class="doc-code-header">~/.codex/config.toml</div>
          <pre>chatgpt_base_url = "http://localhost:{{ port }}"</pre>
        </div>
      </div>

      <!-- curl -->
      <div id="sec-curl" class="card p-5">
        <h2 class="text-lg font-semibold text-white mb-1">cURL 调用示例</h2>
        <p class="text-sm text-gray-400 mb-4">通过代理池的 OpenAI 兼容接口发送请求。</p>

        <h3 class="text-sm font-semibold text-blue-400 mb-2">Chat Completions（流式）</h3>
        <div class="doc-code mb-4">
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

        <h3 class="text-sm font-semibold text-blue-400 mb-2">获取可用模型</h3>
        <div class="doc-code mb-4">
          <div class="doc-code-header">bash</div>
          <pre>curl http://localhost:{{ port }}/v1/models \
  -H "Authorization: Bearer YOUR_API_KEY"</pre>
          <button @click="copyCurl('models')" class="doc-code-copy">复制</button>
        </div>

        <h3 class="text-sm font-semibold text-blue-400 mb-2">查看代理池状态</h3>
        <div class="doc-code">
          <div class="doc-code-header">bash</div>
          <pre>curl http://localhost:{{ port }}/pool/status</pre>
          <button @click="copyCurl('pool')" class="doc-code-copy">复制</button>
        </div>
      </div>

      <!-- Python -->
      <div id="sec-python" class="card p-5">
        <h2 class="text-lg font-semibold text-white mb-1">Python 调用</h2>
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

      <!-- Augment Import -->
      <div id="sec-augment" class="card p-5">
        <h2 class="text-lg font-semibold text-white mb-1">Augment Session 导入</h2>
        <p class="text-sm text-gray-400 mb-4">通过 API 批量导入 Augment 的 auth session。</p>

        <h3 class="text-sm font-semibold text-blue-400 mb-2">导入单个 Session</h3>
        <div class="doc-code mb-4">
          <div class="doc-code-header">bash</div>
          <pre>curl -X POST http://localhost:{{ port }}/api/import/session \
  -H "Content-Type: application/json" \
  -d '{"session": "YOUR_AUTH_SESSION_STRING"}'</pre>
          <button @click="copyCurl('augment-single')" class="doc-code-copy">复制</button>
        </div>

        <h3 class="text-sm font-semibold text-blue-400 mb-2">批量导入 Sessions</h3>
        <div class="doc-code">
          <div class="doc-code-header">bash</div>
          <pre>curl -X POST http://localhost:{{ port }}/api/import/sessions \
  -H "Content-Type: application/json" \
  -d '{"sessions": ["SESSION_1", "SESSION_2", "SESSION_3"]}'</pre>
          <button @click="copyCurl('augment-batch')" class="doc-code-copy">复制</button>
        </div>
      </div>

      <!-- OpenAI Token Import -->
      <div id="sec-import" class="card p-5">
        <h2 class="text-lg font-semibold text-white mb-1">OpenAI 账号批量导入</h2>
        <p class="text-sm text-gray-400 mb-4">通过 refresh_token 批量导入 OpenAI OAuth 账号。</p>

        <div class="doc-code mb-4">
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

        <h3 class="text-sm font-semibold text-blue-400 mb-2">通过扫描目录导入</h3>
        <p class="text-xs text-gray-400 mb-2">将 token JSON 文件放在 <code class="code">./auth/</code> 目录下，然后调用扫描接口自动导入。</p>
        <div class="doc-code">
          <div class="doc-code-header">bash</div>
          <pre>curl -X POST http://localhost:{{ port }}/api/v1/openai/import/scan-dir \
  -H "Content-Type: application/json" \
  -d '{"dir": "./auth"}'</pre>
          <button @click="copyCurl('openai-scan')" class="doc-code-copy">复制</button>
        </div>
      </div>

      <!-- API Key Auth -->
      <div id="sec-auth" class="card p-5">
        <h2 class="text-lg font-semibold text-white mb-1">代理池鉴权</h2>
        <p class="text-sm text-gray-400 mb-4">保护你的代理池端点，防止未授权访问。</p>

        <div class="space-y-3 text-sm text-gray-300">
          <div class="flex gap-2">
            <span class="text-blue-400 font-bold shrink-0">1.</span>
            <span>在 OpenAI / Codex 页面 → 服务配置 → 设置一个 API Key</span>
          </div>
          <div class="flex gap-2">
            <span class="text-blue-400 font-bold shrink-0">2.</span>
            <span>所有 <code class="code">/v1/*</code> 请求都需要携带 <code class="code">Authorization: Bearer YOUR_KEY</code></span>
          </div>
          <div class="flex gap-2">
            <span class="text-blue-400 font-bold shrink-0">3.</span>
            <span>本地 Codex CLI 通过已知的 managed token 认证（passthrough 模式），无需额外配置</span>
          </div>
        </div>

        <div class="doc-code mt-4">
          <div class="doc-code-header">支持的负载均衡策略</div>
          <pre>round_robin  — 轮询（默认），均匀分配请求到各账号
random       — 随机选择一个账号
least_used   — 选择请求次数最少的账号</pre>
        </div>
      </div>

      <!-- Docker -->
      <div id="sec-docker" class="card p-5">
        <h2 class="text-lg font-semibold text-white mb-1">Docker 部署</h2>
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
      <div id="sec-faq" class="card p-5">
        <h2 class="text-lg font-semibold text-white mb-3">常见问题</h2>
        <div class="space-y-4">
          <div v-for="faq in faqs" :key="faq.q">
            <div class="text-sm font-medium text-white">{{ faq.q }}</div>
            <div class="text-sm text-gray-400 mt-1">{{ faq.a }}</div>
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
  { id: 'sec-codex', icon: '🖥️', label: 'Codex CLI' },
  { id: 'sec-curl', icon: '📡', label: 'cURL' },
  { id: 'sec-python', icon: '🐍', label: 'Python' },
  { id: 'sec-augment', icon: '🔑', label: 'Augment' },
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
]

const curlSnippets = {
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
  'augment-single': `curl -X POST http://localhost:PORT/api/import/session \\
  -H "Content-Type: application/json" \\
  -d '{"session": "YOUR_AUTH_SESSION_STRING"}'`,
  'augment-batch': `curl -X POST http://localhost:PORT/api/import/sessions \\
  -H "Content-Type: application/json" \\
  -d '{"sessions": ["SESSION_1", "SESSION_2", "SESSION_3"]}'`,
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
.code {
  @apply bg-gray-800 text-blue-400 px-1.5 py-0.5 rounded text-xs font-mono;
}
.doc-code {
  @apply bg-gray-950 border border-gray-800 rounded-lg overflow-hidden relative;
}
.doc-code-header {
  @apply bg-gray-900 px-3 py-1.5 text-xs text-gray-500 border-b border-gray-800 font-mono;
}
.doc-code pre {
  @apply px-4 py-3 text-xs text-gray-300 font-mono overflow-x-auto whitespace-pre leading-relaxed;
}
.doc-code-copy {
  @apply absolute top-1.5 right-2 text-xs text-gray-500 hover:text-white bg-gray-800 hover:bg-gray-700
         px-2 py-0.5 rounded transition-colors;
}
</style>
