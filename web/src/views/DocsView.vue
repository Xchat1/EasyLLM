<template>
  <div class="docs-page p-5 sm:p-6 w-full max-w-none space-y-6">
    <!-- Header -->
    <div class="flex flex-col gap-4 lg:flex-row lg:items-center lg:justify-between">
      <div class="flex items-center gap-3.5 min-w-0">
        <span class="inline-flex h-11 w-11 shrink-0 items-center justify-center rounded-2xl bg-gradient-to-br from-blue-500/20 to-indigo-600/30 border border-blue-500/30 text-2xl shadow-lg shadow-blue-950/40">
          📖
        </span>
        <div class="min-w-0">
          <h1 class="text-2xl font-bold text-white tracking-tight">EasyLLM 开发者手册</h1>
          <p class="text-gray-400 text-xs sm:text-sm mt-0.5">
            掌握 OpenAI / Codex、Google Antigravity 与 Cursor 三大渠道账号调度、配额水位监控与本机代理环境
          </p>
        </div>
      </div>

      <div class="flex flex-wrap items-center gap-2 shrink-0">
        <router-link to="/" class="btn btn-secondary btn-sm">📊 仪表盘</router-link>
        <router-link to="/codex" class="btn btn-secondary btn-sm">🤖 Codex</router-link>
        <router-link to="/antigravity" class="btn btn-secondary btn-sm">🚀 Antigravity</router-link>
        <router-link to="/cursor" class="btn btn-secondary btn-sm">⚡ Cursor</router-link>
      </div>
    </div>

    <!-- Quick Navigation Bar -->
    <div class="rounded-2xl border border-slate-800 bg-slate-900 p-5 shadow-xl">
      <div class="flex items-center justify-between mb-3.5">
        <div class="text-xs font-semibold text-slate-400 uppercase tracking-wider flex items-center gap-2">
          <span class="w-2 h-2 rounded-full bg-blue-500"></span>
          快速导航
        </div>
        <span class="text-xs text-slate-500">点击直达对应章节</span>
      </div>
      <div class="grid grid-cols-2 sm:grid-cols-5 lg:grid-cols-10 gap-2">
        <button
          v-for="section in sections"
          :key="section.id"
          @click="scrollTo(section.id)"
          class="flex flex-col items-center justify-center p-2.5 rounded-xl bg-slate-800/60 hover:bg-blue-500/20 text-slate-200 hover:text-white transition-all duration-200 border border-slate-700/60 hover:border-blue-500/50 group shadow-sm"
        >
          <span class="text-xl mb-1 transform group-hover:scale-110 transition-transform">{{ section.icon }}</span>
          <span class="text-xs font-medium truncate max-w-full">{{ section.label }}</span>
        </button>
      </div>
    </div>

    <div class="space-y-6">

      <!-- Section 1: Overview -->
      <div id="sec-overview" class="rounded-2xl border border-slate-800 bg-slate-900 p-5 sm:p-6 shadow-xl doc-section">
        <h2 class="text-xl font-bold text-white mb-4 flex items-center gap-2.5">
          <span class="text-2xl">🎯</span> 产品简介
        </h2>
        <div class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-4 mb-5">
          <div class="bg-slate-800/60 border border-blue-500/30 rounded-xl p-4 flex flex-col justify-between">
            <div>
              <div class="text-2xl mb-2">🤖</div>
              <div class="text-sm font-bold text-white mb-1">Codex / OpenAI</div>
              <div class="text-xs text-slate-300 leading-relaxed text-left">
                OAuth 登录、API Key、Token 自动刷新、Responses API 协议转换与配额水位监控
              </div>
            </div>
          </div>
          <div class="bg-slate-800/60 border border-amber-500/30 rounded-xl p-4 flex flex-col justify-between">
            <div>
              <div class="text-2xl mb-2">🚀</div>
              <div class="text-sm font-bold text-white mb-1">Google Antigravity</div>
              <div class="text-xs text-slate-300 leading-relaxed text-left">
                Claude 3.7 & Gemini 2.5 额度监控、与 agy CLI / IDE 凭证同步及唤醒测试
              </div>
            </div>
          </div>
          <div class="bg-slate-800/60 border border-purple-500/30 rounded-xl p-4 flex flex-col justify-between">
            <div>
              <div class="text-2xl mb-2">⚡</div>
              <div class="text-sm font-bold text-white mb-1">Cursor 聚合</div>
              <div class="text-xs text-slate-300 leading-relaxed text-left">
                多账号凭证管理、Fast Requests 配额水位与计费重置周期统一看板
              </div>
            </div>
          </div>
          <div class="bg-slate-800/60 border border-emerald-500/30 rounded-xl p-4 flex flex-col justify-between">
            <div>
              <div class="text-2xl mb-2">🔒</div>
              <div class="text-sm font-bold text-white mb-1">本地安全代理池</div>
              <div class="text-xs text-slate-300 leading-relaxed text-left">
                本地 SQLite 存储、API Key 鉴权、Local-First 架构，敏感密钥不出本机
              </div>
            </div>
          </div>
        </div>
        <div class="flex flex-wrap items-center gap-2 pt-3 border-t border-slate-800">
          <span class="text-xs text-slate-400 mr-1">技术栈与架构:</span>
          <span class="px-2.5 py-1 bg-slate-800 border border-slate-700 rounded-md text-[11px] font-medium text-slate-300">Go 1.25</span>
          <span class="px-2.5 py-1 bg-slate-800 border border-slate-700 rounded-md text-[11px] font-medium text-slate-300">Vue 3</span>
          <span class="px-2.5 py-1 bg-slate-800 border border-slate-700 rounded-md text-[11px] font-medium text-slate-300">Tailwind CSS</span>
          <span class="px-2.5 py-1 bg-slate-800 border border-slate-700 rounded-md text-[11px] font-medium text-slate-300">本地 SQLite</span>
          <span class="px-2.5 py-1 bg-emerald-950/40 border border-emerald-700/40 rounded-md text-[11px] font-medium text-emerald-300">Local-First</span>
          <span class="px-2.5 py-1 bg-blue-950/40 border border-blue-700/40 rounded-md text-[11px] font-medium text-blue-300">Native macOS .app</span>
        </div>
      </div>

      <!-- Section 2: Codex -->
      <div id="sec-codex" class="rounded-2xl border border-slate-800 bg-slate-900 p-5 sm:p-6 shadow-xl doc-section">
        <h2 class="text-xl font-bold text-white mb-2 flex items-center gap-2.5">
          <span class="text-2xl">🖥️</span> Codex 接入
        </h2>
        <p class="text-sm text-slate-300 mb-6">将 EasyLLM 作为 Codex 客户端 / Codex CLI 的代理，实现多账号轮询、本机配置注入和账号集合调度。</p>

        <!-- Supported Models -->
        <div class="mb-6 p-4 rounded-xl border border-slate-800 bg-slate-950">
          <div class="flex items-center justify-between mb-3">
            <h3 class="text-sm font-semibold text-white flex items-center gap-2">
              <span>✨</span> 支持的 OpenAI / Codex 模型
            </h3>
            <span class="text-xs text-blue-400 font-mono">GPT-6 & GPT-5.x</span>
          </div>
          <div class="grid grid-cols-1 sm:grid-cols-2 md:grid-cols-3 gap-2.5">
            <div class="p-3 rounded-lg bg-slate-900 border border-slate-800 hover:border-slate-700 transition-colors">
              <div class="flex items-center justify-between mb-1">
                <span class="font-medium text-white text-xs">GPT-6 Astra</span>
                <span class="text-[10px] px-1.5 py-0.5 rounded bg-purple-500/20 text-purple-300 font-mono">gpt-6-astra</span>
              </div>
              <p class="text-[11px] text-slate-300">新一代旗舰模型，顶尖代码推理与综合能力</p>
            </div>
            <div class="p-3 rounded-lg bg-slate-900 border border-slate-800 hover:border-slate-700 transition-colors">
              <div class="flex items-center justify-between mb-1">
                <span class="font-medium text-white text-xs">GPT-6 Luna</span>
                <span class="text-[10px] px-1.5 py-0.5 rounded bg-indigo-500/20 text-indigo-300 font-mono">gpt-6-luna</span>
              </div>
              <p class="text-[11px] text-slate-300">新一代轻量高效模型，极快速度与低延迟响应</p>
            </div>
            <div class="p-3 rounded-lg bg-slate-900 border border-slate-800 hover:border-slate-700 transition-colors">
              <div class="flex items-center justify-between mb-1">
                <span class="font-medium text-white text-xs">GPT-5.6 Sol</span>
                <span class="text-[10px] px-1.5 py-0.5 rounded bg-amber-500/20 text-amber-300 font-mono">gpt-5.6-sol</span>
              </div>
              <p class="text-[11px] text-slate-300">GPT-5.6 旗舰全功能模型，支持深度推理模式</p>
            </div>
            <div class="p-3 rounded-lg bg-slate-900 border border-slate-800 hover:border-slate-700 transition-colors">
              <div class="flex items-center justify-between mb-1">
                <span class="font-medium text-white text-xs">GPT-5.6 Terra</span>
                <span class="text-[10px] px-1.5 py-0.5 rounded bg-emerald-500/20 text-emerald-300 font-mono">gpt-5.6-terra</span>
              </div>
              <p class="text-[11px] text-slate-300">GPT-5.6 平衡型主力模型，兼顾速度与质量</p>
            </div>
            <div class="p-3 rounded-lg bg-slate-900 border border-slate-800 hover:border-slate-700 transition-colors">
              <div class="flex items-center justify-between mb-1">
                <span class="font-medium text-white text-xs">GPT-5.6 Luna</span>
                <span class="text-[10px] px-1.5 py-0.5 rounded bg-cyan-500/20 text-cyan-300 font-mono">gpt-5.6-luna</span>
              </div>
              <p class="text-[11px] text-slate-300">GPT-5.6 极速响应轻量模型，日常快速交互</p>
            </div>
          </div>
        </div>

        <div class="space-y-4">
          <!-- Method 1 -->
          <div class="p-4.5 bg-slate-800/80 border border-blue-500/40 rounded-xl">
            <div class="flex items-center gap-2 mb-2">
              <span class="px-2 py-0.5 bg-blue-500/20 text-blue-400 text-xs font-semibold rounded">推荐</span>
              <h3 class="text-sm font-semibold text-white">方式一：本地 API 服务调度</h3>
            </div>
            <p class="text-xs text-slate-300 mb-3">在 OpenAI / Codex 页面导入 OAuth 账号后，打开"服务配置"，点击"启动并注入 Codex"，EasyLLM 会写入本机 Codex 配置并使用账号集合调度请求。</p>
            <div class="doc-code">
              <div class="doc-code-header">
                <span>自动配置的 ~/.codex/config.toml</span>
                <button @click="copyCurl('codex-oauth')" class="doc-code-copy">复制</button>
              </div>
              <pre>model_provider = "easyllm"
model = "gpt-5.6-sol"

[model_providers.easyllm]
name = "EasyLLM API Service"
base_url = "http://localhost:{{ port }}/backend-api/codex"
wire_api = "responses"
requires_openai_auth = true
supports_websockets = false</pre>
            </div>
          </div>

          <!-- Method 2 -->
          <div class="p-4.5 bg-slate-800/60 border border-slate-700/60 hover:border-slate-600 transition-colors rounded-xl">
            <h3 class="text-sm font-semibold text-white mb-2">方式二：OAuth 单账号切换</h3>
            <p class="text-xs text-slate-300 mb-3">在 OpenAI / Codex 页面添加 OAuth 账号后，点击账号卡片里的"切换"按钮即可写入 <code class="code">~/.codex/auth.json</code>。</p>
            <div class="doc-code">
              <div class="doc-code-header">
                <span>自动配置的 ~/.codex/config.toml</span>
                <button @click="copyCurl('codex-pool')" class="doc-code-copy">复制</button>
              </div>
              <pre>chatgpt_base_url = "http://localhost:{{ port }}"</pre>
            </div>
          </div>

          <!-- Method 3 -->
          <div class="p-4.5 bg-slate-800/60 border border-slate-700/60 hover:border-slate-600 transition-colors rounded-xl">
            <h3 class="text-sm font-semibold text-white mb-2">方式三：API Key 账号</h3>
            <p class="text-xs text-slate-300 mb-3">在 OpenAI / Codex 页面的「API 账号」标签添加自定义 provider、模型、Base URL 与 API Key，点击切换后写入 <code class="code">~/.codex/config.toml</code>。</p>
            <div class="doc-code">
              <div class="doc-code-header">
                <span>示例：自定义 API 账号</span>
                <button @click="copyCurl('api-account')" class="doc-code-copy">复制</button>
              </div>
              <pre>model_provider = "my-provider"
model = "gpt-5.6-sol"

[model_providers.my-provider]
name = "my-provider"
base_url = "https://api.example.com/v1"
wire_api = "responses"</pre>
            </div>
          </div>

          <!-- Method 4 -->
          <div class="p-4.5 bg-slate-800/60 border border-slate-700/60 hover:border-slate-600 transition-colors rounded-xl">
            <h3 class="text-sm font-semibold text-white mb-2">方式四：Relay 第三方上游</h3>
            <p class="text-xs text-slate-300 mb-3">在 <router-link to="/relay" class="text-blue-400 hover:underline">Relay 配置</router-link> 页填写 DeepSeek、Mistral、OpenRouter、Kimi、Qwen、MiMo 等 OpenAI 兼容上游，点击「启动并注入 Codex」。EasyLLM 负责 Responses API → Chat Completions 协议转换，Codex 客户端 / Codex CLI 无需直连上游。</p>
            <div class="doc-code">
              <div class="doc-code-header">
                <span>自动配置的 ~/.codex/config.toml</span>
                <button @click="copyCurl('codex-relay')" class="doc-code-copy">复制</button>
              </div>
              <pre>model_provider = "relay"
model = "gpt-5.6-sol"

[model_providers.relay]
name = "EasyLLM Relay"
base_url = "http://localhost:{{ port }}/v1"
wire_api = "responses"
requires_openai_auth = false</pre>
            </div>
          </div>

          <!-- Method 5 -->
          <div class="p-4.5 bg-slate-800/60 border border-slate-700/60 hover:border-slate-600 transition-colors rounded-xl">
            <h3 class="text-sm font-semibold text-white mb-2">方式五：代理池多账号均衡模式</h3>
            <p class="text-xs text-slate-300 mb-3">启用多个 OAuth 账号的"代理开关"，请求将按配置策略调度到账号池。</p>
            <div class="doc-code">
              <div class="doc-code-header">
                <span>~/.codex/config.toml</span>
                <button @click="copyCurl('codex-pool')" class="doc-code-copy">复制</button>
              </div>
              <pre>chatgpt_base_url = "http://localhost:{{ port }}"</pre>
            </div>
          </div>
        </div>
      </div>

      <!-- Section 3: Antigravity -->
      <div id="sec-antigravity" class="rounded-2xl border border-slate-800 bg-slate-900 p-5 sm:p-6 shadow-xl doc-section">
        <h2 class="text-xl font-bold text-white mb-2 flex items-center gap-2.5">
          <span class="text-2xl">🚀</span> Google Antigravity 接入
        </h2>
        <p class="text-sm text-slate-300 mb-5">
          管理 Google Antigravity 账号，监控 Claude 3.7 Sonnet / Gemini 2.5 Pro 与 Flash 配额水位，并与本机 Antigravity IDE / agy CLI 状态自动同步。
        </p>

        <div class="space-y-4">
          <div class="p-4.5 bg-slate-800/80 border border-amber-500/40 rounded-xl">
            <div class="flex items-center gap-2 mb-2">
              <span class="px-2 py-0.5 bg-amber-500/20 text-amber-400 text-xs font-semibold rounded">快捷集成</span>
              <h3 class="text-sm font-semibold text-white">Google OAuth 授权登录与状态同步</h3>
            </div>
            <p class="text-xs text-slate-300 mb-3">
              在 Antigravity 渠道页面点击右上角「OAuth 登录」，完成 Google 授权后账号自动入库并获取当前生效模型配额。点击「设为当前」可直接将激活凭证同步到本机 ~/.antigravity/ 及 agy CLI 环境。
            </p>
            <div class="doc-code">
              <div class="doc-code-header">
                <span>agy CLI 命令行验证</span>
                <button @click="copyCurl('antigravity-agy')" class="doc-code-copy">复制</button>
              </div>
              <pre># 查看当前激活账号与配额水位
agy auth status

# 运行连通性测试
agy doctor</pre>
            </div>
          </div>

          <div class="grid grid-cols-1 md:grid-cols-2 gap-4">
            <div class="p-4 rounded-xl border border-slate-700/60 bg-slate-800/60">
              <div class="text-sm font-semibold text-white mb-1.5 flex items-center gap-2">
                <span>⚡</span> 唤醒与连通性测试
              </div>
              <p class="text-xs text-slate-300 leading-relaxed">
                在账号卡片点击唤醒图标（小铃铛），EasyLLM 将向 Antigravity 网关发送轻量探测请求，验证 Access Token 有效性并实时刷新模型限额桶。
              </p>
            </div>
            <div class="p-4 rounded-xl border border-slate-700/60 bg-slate-800/60">
              <div class="text-sm font-semibold text-white mb-1.5 flex items-center gap-2">
                <span>🛡️</span> 隐私保护模式
              </div>
              <p class="text-xs text-slate-300 leading-relaxed">
                支持一键开启隐私模式，所有卡片及列表中的 Google 邮箱地址、头像和账号唯一标识将自动打码遮蔽，适合录屏和分享展示。
              </p>
            </div>
          </div>
        </div>
      </div>

      <!-- Section 4: Cursor -->
      <div id="sec-cursor" class="rounded-2xl border border-slate-800 bg-slate-900 p-5 sm:p-6 shadow-xl doc-section">
        <h2 class="text-xl font-bold text-white mb-2 flex items-center gap-2.5">
          <span class="text-2xl">⚡</span> Cursor 接入
        </h2>
        <p class="text-sm text-slate-300 mb-5">
          聚合管理多个 Cursor 账号凭据，监控 Fast Requests / Pro 额度水位与月度账单周期。
        </p>

        <div class="grid grid-cols-1 md:grid-cols-3 gap-4">
          <div class="p-4 rounded-xl border border-slate-700/60 bg-slate-800/60 flex flex-col justify-between">
            <div>
              <div class="text-base font-semibold text-white mb-1.5">🔑 凭证管理</div>
              <p class="text-xs text-slate-300 leading-relaxed">
                支持导入 Token、Cookie 与账号会话，多账号状态与会员等级一目了然。
              </p>
            </div>
          </div>
          <div class="p-4 rounded-xl border border-slate-700/60 bg-slate-800/60 flex flex-col justify-between">
            <div>
              <div class="text-base font-semibold text-white mb-1.5">📊 额度水位监控</div>
              <p class="text-xs text-slate-300 leading-relaxed">
                清晰呈现当前账号的已用 / 总额度进度条，超出限额或即将耗尽时触发告警提示。
              </p>
            </div>
          </div>
          <div class="p-4 rounded-xl border border-slate-700/60 bg-slate-800/60 flex flex-col justify-between">
            <div>
              <div class="text-base font-semibold text-white mb-1.5">🔄 统一看板聚合</div>
              <p class="text-xs text-slate-300 leading-relaxed">
                在首页仪表盘中与 Codex、Antigravity 并列展示，随时对比三大渠道健康状态。
              </p>
            </div>
          </div>
        </div>
      </div>

      <!-- Section 5: cURL -->
      <div id="sec-curl" class="rounded-2xl border border-slate-800 bg-slate-900 p-5 sm:p-6 shadow-xl doc-section">
        <h2 class="text-xl font-semibold text-white mb-4 flex items-center gap-2">
          <span class="text-2xl">📡</span> cURL 调用示例
        </h2>
        <p class="text-sm text-slate-300 mb-5">通过代理池或 Relay 的 OpenAI 兼容接口发送请求。Relay 最近调用记录只保存时间、渠道、模型和 Token 用量，不保存提示词或响应正文。</p>

        <div class="space-y-4">
          <div>
            <h3 class="text-sm font-semibold text-white mb-2">Responses（流式）</h3>
            <div class="doc-code">
              <div class="doc-code-header">
                <span>bash</span>
                <button @click="copyCurl('responses')" class="doc-code-copy">复制</button>
              </div>
              <pre>curl http://localhost:{{ port }}/v1/responses \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer YOUR_API_KEY" \
  -d '{
    "model": "gpt-5.6-sol",
    "input": "写一个快速排序算法",
    "stream": true
  }'</pre>
            </div>
          </div>

          <div>
            <h3 class="text-sm font-semibold text-white mb-2">获取可用模型列表</h3>
            <div class="doc-code">
              <div class="doc-code-header">
                <span>bash</span>
                <button @click="copyCurl('models')" class="doc-code-copy">复制</button>
              </div>
              <pre>curl http://localhost:{{ port }}/v1/models \
  -H "Authorization: Bearer YOUR_API_KEY"</pre>
            </div>
          </div>

          <div>
            <h3 class="text-sm font-semibold text-white mb-2">查看代理池状态</h3>
            <div class="doc-code">
              <div class="doc-code-header">
                <span>bash</span>
                <button @click="copyCurl('pool')" class="doc-code-copy">复制</button>
              </div>
              <pre>curl http://localhost:{{ port }}/pool/status</pre>
            </div>
          </div>
        </div>
      </div>

      <!-- Section 6: Python -->
      <div id="sec-python" class="rounded-2xl border border-slate-800 bg-slate-900 p-5 sm:p-6 shadow-xl doc-section">
        <h2 class="text-xl font-semibold text-white mb-4 flex items-center gap-2">
          <span class="text-2xl">🐍</span> Python 调用
        </h2>
        <p class="text-sm text-slate-300 mb-4">使用 OpenAI Python SDK 通过 EasyLLM 的 OpenAI 兼容接口发送请求。</p>

        <div class="doc-code">
          <div class="doc-code-header">
            <span>python</span>
            <button @click="copyCurl('python')" class="doc-code-copy">复制</button>
          </div>
          <pre>from openai import OpenAI

client = OpenAI(
    base_url="http://localhost:{{ port }}/v1",
    api_key="YOUR_API_KEY",  # 在服务配置中设置的 API Key
)

response = client.responses.create(
    model="gpt-5.6-sol",
    input="用 Python 写一个命令行参数解析示例",
)

print(response.output_text)</pre>
        </div>
      </div>

      <!-- Section 7: Quota -->
      <div id="sec-quota" class="rounded-2xl border border-slate-800 bg-slate-900 p-5 sm:p-6 shadow-xl doc-section">
        <h2 class="text-xl font-semibold text-white mb-4 flex items-center gap-2">
          <span class="text-2xl">📊</span> 配额查询机制
        </h2>
        <p class="text-sm text-slate-300 mb-4">集中查看短期会话和长期周期配额使用情况（支持图表与趋势分析）。</p>

        <div class="grid grid-cols-1 md:grid-cols-2 gap-4 mb-4">
          <div class="bg-slate-800/60 rounded-xl p-4 border border-slate-700/60 hover:border-slate-600 transition-colors">
            <div class="text-sm font-semibold text-white mb-2 flex items-center gap-2">
              <span>⏱️</span> 5 小时短期配额
            </div>
            <div class="text-xs text-slate-300 leading-relaxed">
              短期会话突发速率限制，重置周期短，用于防高频滥用并保护账号稳定性。
            </div>
          </div>
          <div class="bg-slate-800/60 rounded-xl p-4 border border-slate-700/60 hover:border-slate-600 transition-colors">
            <div class="text-sm font-semibold text-white mb-2 flex items-center gap-2">
              <span>📅</span> 7 天周度总量配额
            </div>
            <div class="text-xs text-slate-300 leading-relaxed">
              周期性累计消耗上限，按周滚动刷新，EasyLLM 会在总览看板实时标注重置倒计时。
            </div>
          </div>
        </div>
        <div class="p-3 rounded-lg bg-blue-500/15 border border-blue-500/30 text-xs text-blue-300 flex items-center gap-2">
          <span>💡</span> 在账号列表点击"刷新配额"或右上角"全局刷新"即可获取最新的官方配额数据。
        </div>
      </div>

      <!-- Section 8: Import -->
      <div id="sec-import" class="rounded-2xl border border-slate-800 bg-slate-900 p-5 sm:p-6 shadow-xl doc-section">
        <h2 class="text-xl font-semibold text-white mb-4 flex items-center gap-2">
          <span class="text-2xl">📦</span> 批量导入
        </h2>
        <p class="text-sm text-slate-300 mb-4">支持 token 文件、自适应导入（自动识别 Sub2API 等格式）、refresh_token、CPA JSON，以及 EasyLLM 备份文件。</p>

        <div class="space-y-4">
          <div>
            <h3 class="text-sm font-semibold text-white mb-2">通过 refresh_token 导入</h3>
            <div class="doc-code">
              <div class="doc-code-header">
                <span>bash</span>
                <button @click="copyCurl('openai-import')" class="doc-code-copy">复制</button>
              </div>
              <pre>curl -X POST http://localhost:{{ port }}/api/v1/openai/import/refresh-tokens \
  -H "Content-Type: application/json" \
  -d '{
    "refresh_tokens": [
      "REFRESH_TOKEN_1",
      "REFRESH_TOKEN_2"
    ]
  }'</pre>
            </div>
          </div>

          <div>
            <h3 class="text-sm font-semibold text-white mb-2">自适应导入（上传 JSON）</h3>
            <p class="text-xs text-slate-300 mb-2">在 Web 端选择单个或多个 JSON 文件即可自动识别并导入；支持 Sub2API 当前 v1 与旧版账号导出，也可用 API 上传 multipart 文件。</p>
            <div class="doc-code">
              <div class="doc-code-header">
                <span>bash</span>
                <button @click="copyCurl('openai-auto')" class="doc-code-copy">复制</button>
              </div>
              <pre>curl -X POST http://localhost:{{ port }}/api/v1/openai/import/auto-files \
  -H "Authorization: Bearer TOKEN" \
  -F "files=@account1.json" \
  -F "files=@account2-cpa.json"</pre>
            </div>
          </div>

          <div>
            <h3 class="text-sm font-semibold text-white mb-2">备份恢复</h3>
            <p class="text-xs text-slate-300 mb-2">通过"一键导出所有最新数据"生成的备份可在"批量导入 → 从备份导入"恢复账号；如果备份包含本地 API 服务配置，也会一并恢复账号集合、端口和调度策略。</p>
            <div class="doc-code">
              <div class="doc-code-header">
                <span>备份文件结构</span>
              </div>
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

      <!-- Section 9: Auth -->
      <div id="sec-auth" class="rounded-2xl border border-slate-800 bg-slate-900 p-5 sm:p-6 shadow-xl doc-section">
        <h2 class="text-xl font-semibold text-white mb-4 flex items-center gap-2">
          <span class="text-2xl">🔒</span> 代理池鉴权与负载策略
        </h2>
        <p class="text-sm text-slate-300 mb-4">保护你的代理池端点，防止未授权访问，并精确调度多个工作账号。</p>

        <div class="space-y-3 mb-5">
          <div class="flex gap-3 items-start p-3.5 bg-slate-800/60 rounded-xl border border-slate-700/60">
            <span class="flex-shrink-0 w-6 h-6 bg-blue-500/20 text-blue-400 text-xs font-bold rounded-full flex items-center justify-center">1</span>
            <span class="text-sm text-slate-300">在 OpenAI / Codex 页面 → 服务配置 → 设置专属 API Key</span>
          </div>
          <div class="flex gap-3 items-start p-3.5 bg-slate-800/60 rounded-xl border border-slate-700/60">
            <span class="flex-shrink-0 w-6 h-6 bg-blue-500/20 text-blue-400 text-xs font-bold rounded-full flex items-center justify-center">2</span>
            <span class="text-sm text-slate-300">所有 <code class="code">/v1/*</code> 请求都需要携带 <code class="code">Authorization: Bearer YOUR_KEY</code></span>
          </div>
          <div class="flex gap-3 items-start p-3.5 bg-slate-800/60 rounded-xl border border-slate-700/60">
            <span class="flex-shrink-0 w-6 h-6 bg-blue-500/20 text-blue-400 text-xs font-bold rounded-full flex items-center justify-center">3</span>
            <span class="text-sm text-slate-300">本地 Codex 客户端或 Codex CLI 通过已知的 managed token 认证（passthrough 模式），无需额外配置</span>
          </div>
        </div>

        <div class="doc-code">
          <div class="doc-code-header">
            <span>支持的智能负载均衡策略</span>
          </div>
          <pre>auto               — 综合订阅计划、响应延迟和剩余额度自适应调度（推荐）
quota_high_first   — 优先使用剩余配额高的账号
quota_low_first    — 优先消耗剩余配额低即将用尽的账号
plan_high_first    — 优先使用高订阅等级账号（Plus / Pro）
plan_low_first     — 优先使用低订阅等级账号
expiry_soon_first  — 优先使用更早到期重置的账号
round_robin        — 严格顺序轮询
random             — 随机选择可用账号
least_used         — 选择历史请求次数最少的账号</pre>
        </div>
      </div>

      <!-- Section 10: FAQ -->
      <div id="sec-faq" class="rounded-2xl border border-slate-800 bg-slate-900 p-5 sm:p-6 shadow-xl doc-section">
        <h2 class="text-xl font-semibold text-white mb-4 flex items-center gap-2">
          <span class="text-2xl">❓</span> 常见问题与排错
        </h2>
        <div class="space-y-3.5">
          <div v-for="(faq, index) in faqs" :key="index" class="p-4 bg-slate-800/60 rounded-xl border border-slate-700/60 hover:border-slate-600 transition-colors">
            <div class="text-sm font-semibold text-white mb-2 flex items-start gap-2">
              <span class="text-blue-400 font-mono font-bold mt-0.5">Q:</span>
              <span>{{ faq.q }}</span>
            </div>
            <div class="text-sm text-slate-300 ml-5 leading-relaxed">{{ faq.a }}</div>
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
  { id: 'sec-codex', icon: '🖥️', label: 'Codex' },
  { id: 'sec-antigravity', icon: '🚀', label: 'Antigravity' },
  { id: 'sec-cursor', icon: '⚡', label: 'Cursor' },
  { id: 'sec-curl', icon: '📡', label: 'cURL' },
  { id: 'sec-python', icon: '🐍', label: 'Python' },
  { id: 'sec-quota', icon: '📊', label: '配额机制' },
  { id: 'sec-import', icon: '📦', label: '批量导入' },
  { id: 'sec-auth', icon: '🔒', label: '鉴权策略' },
  { id: 'sec-faq', icon: '❓', label: 'FAQ' },
]

const faqs = [
  { q: 'Codex 客户端 / CLI 报 "Token data is not available." 怎么办？', a: '确保 auth.json 中 last_refresh 在顶层而非 tokens 内部。在 EasyLLM 中重新点击"切换"即可自动修复。' },
  { q: '代理池请求返回 401 Unauthorized', a: '若返回 {"detail":"Unauthorized"} 且带 cf-ray，通常是代理池里 OAuth access_token 已失效。EasyLLM 会自动轮换账号并尝试 refresh_token 刷新；若全部账号 refresh 失败，请重新 OAuth 登录或导入带有效 refresh_token 的 CPA/Token 文件。若返回 {"error":{"message":"Invalid API key"}} 则是本地 proxy_api_key 与 Codex 配置不一致。' },
  { q: 'Token 过期了怎么办？', a: '在 OpenAI 账号列表中点击"刷新 Token"按钮，或使用"刷新所有 Token"一键刷新所有 OAuth 账号；刷新完成后会自动触发账号导出。' },
  { q: '配额查询显示 Forbidden', a: '该账号可能没有 Codex 访问权限（需要 ChatGPT Plus/Pro 订阅），或 Token 已失效。' },
  { q: '如何更改数据库位置？', a: '在设置 → 运行状态中修改 SQLite 数据库路径，保存后重启 EasyLLM 即可。' },
  { q: 'EasyLLM 是否适合对公网开放？', a: '不适合。EasyLLM 面向本机 Codex / OpenAI / Antigravity / Cursor 编码对接，后端默认只监听 127.0.0.1 本地回环。' },
  { q: '哪些文件不能提交或打进发布包？', a: '不要提交 .env、data/、auth/、exports/、backups/、Token/CPA JSON、数据库、日志、build/、web/dist/ 或 .codex/.agents 等本地助手目录。发布 zip 前请运行 scripts/check-release-archives.sh。' },
  { q: '5h 和 7d 配额是什么意思？', a: '5h 是短期会话限制，7d 是长期总量限制。在 OpenAI 页面点击"刷新配额"可查看最新使用情况。' },
  { q: 'Google Antigravity 渠道如何与 agy 命令行联动？', a: '在 Antigravity 渠道管理页面点击"设为当前"，EasyLLM 会自动将当前账号的 Token 写入本地配置，随后在终端直接执行 agy 命令即可使用该账号的额度。' },
]

const curlSnippets = {
  'codex-oauth': `model_provider = "easyllm"
model = "gpt-5.6-sol"

[model_providers.easyllm]
name = "EasyLLM API Service"
base_url = "http://localhost:PORT/backend-api/codex"
wire_api = "responses"
requires_openai_auth = true
supports_websockets = false`,
  'api-account': `model_provider = "my-provider"
model = "gpt-5.6-sol"

[model_providers.my-provider]
name = "my-provider"
base_url = "https://api.example.com/v1"
wire_api = "responses"`,
  'codex-pool': `chatgpt_base_url = "http://localhost:PORT"`,
  'codex-relay': `model_provider = "relay"
model = "gpt-5.6-sol"

[model_providers.relay]
name = "EasyLLM Relay"
base_url = "http://localhost:PORT/v1"
wire_api = "responses"
requires_openai_auth = false`,
  responses: `curl http://localhost:PORT/v1/responses \\
  -H "Content-Type: application/json" \\
  -H "Authorization: Bearer YOUR_API_KEY" \\
  -d '{
    "model": "gpt-5.6-sol",
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
    model="gpt-5.6-sol",
    input="用 Python 写一个命令行参数解析示例",
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
  'openai-auto': `curl -X POST http://localhost:PORT/api/v1/openai/import/auto-files \\
  -H "Authorization: Bearer TOKEN" \\
  -F "files=@account1.json" \\
  -F "files=@account2-cpa.json"`,
  'antigravity-agy': `agy auth status
agy doctor`,
}

async function copyCurl(key) {
  const text = (curlSnippets[key] || '').replace(/PORT/g, port.value)
  try {
    if (navigator.clipboard?.writeText) {
      await navigator.clipboard.writeText(text)
    } else {
      fallbackCopy(text)
    }
    notify?.('已复制', 'success')
  } catch {
    try {
      fallbackCopy(text)
      notify?.('已复制', 'success')
    } catch {
      notify?.('复制失败，请手动选择代码内容', 'error')
    }
  }
}

function fallbackCopy(text) {
  const textarea = document.createElement('textarea')
  textarea.value = text
  textarea.setAttribute('readonly', '')
  textarea.style.position = 'fixed'
  textarea.style.left = '-9999px'
  document.body.appendChild(textarea)
  textarea.select()
  const ok = document.execCommand('copy')
  document.body.removeChild(textarea)
  if (!ok) throw new Error('copy failed')
}

function scrollTo(id) {
  document.getElementById(id)?.scrollIntoView({ behavior: 'smooth', block: 'start' })
}
</script>

<style scoped>
.docs-page {
  min-width: 0;
}
.doc-section {
  scroll-margin-top: 24px;
}
.code {
  padding: 0.125rem 0.375rem;
  border-radius: 0.375rem;
  font-size: 0.75rem;
  font-family: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, monospace;
  background: rgba(59, 130, 246, 0.15);
  color: #60a5fa;
  border: 1px solid rgba(59, 130, 246, 0.25);
}
.doc-code {
  border-radius: 0.75rem;
  overflow: hidden;
  position: relative;
  max-width: 100%;
  background: #080c14;
  border: 1px solid rgba(51, 65, 85, 0.7);
  box-shadow: 0 4px 20px -2px rgba(0, 0, 0, 0.5);
  min-width: 0;
}
.doc-code-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 0.5rem 1rem;
  font-size: 0.75rem;
  font-family: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, monospace;
  background: rgba(15, 23, 42, 0.9);
  border-bottom: 1px solid rgba(51, 65, 85, 0.5);
  color: #94a3b8;
}
.doc-code pre {
  padding: 0.875rem 1rem;
  font-size: 0.75rem;
  font-family: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, monospace;
  overflow-x: auto;
  white-space: pre;
  line-height: 1.6;
  color: #e2e8f0;
  max-width: 100%;
  min-width: 0;
}
.doc-code-copy {
  display: inline-flex;
  align-items: center;
  gap: 0.25rem;
  font-size: 0.75rem;
  padding: 0.2rem 0.625rem;
  border-radius: 0.375rem;
  transition: all 0.15s ease;
  background: rgba(30, 41, 59, 0.8);
  border: 1px solid rgba(71, 85, 105, 0.5);
  color: #cbd5e1;
  cursor: pointer;
}
.doc-code-copy:hover {
  background: rgba(37, 99, 235, 0.8);
  border-color: rgba(59, 130, 246, 0.8);
  color: #ffffff;
}
</style>
