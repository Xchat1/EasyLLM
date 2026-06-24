# Codex Relay 集成说明

EasyLLM 内置了 Codex Relay 协议转换层，让 Codex CLI 可以通过本机服务对接任意 OpenAI 兼容的上游提供商，不再依赖 chatgpt.com。

## 架构概览

```
Codex CLI
    │  Responses API (POST /v1/responses)
    ▼
EasyLLM Relay (localhost:8022)
    │  协议转换 + 会话管理 + 轮询调度
    ▼
上游 Chat Completions API
(DeepSeek / Kimi / Qwen / OpenRouter / 任意 OpenAI 兼容服务)
```

## 核心能力

- **协议转换**：Responses API ↔ Chat Completions API 完整双向转换
- **SSE 流式**：上游流式 delta → Responses API 事件序列，实时推送给 Codex CLI
- **会话历史**：`previous_response_id` 机制自动拼接历史 messages，支持多轮对话
- **多上游轮询**：可配置多个上游渠道，按 round-robin 策略自动分流
- **模型映射**：全局模型名映射（如 `gpt-5.4` → `deepseek-chat`）
- **工具过滤**：通过工具拒绝列表屏蔽不被上游支持的工具类型
- **MiMo 支持**：针对 MiMo 思考模型做了专项适配（reasoning_content 往返）

## 端点

| 方法 | 路径 | 说明 |
|------|------|------|
| `POST` | `/v1/responses` | Relay 主入口，协议转换并转发 |
| `GET` | `/v1/models` | 代理上游模型列表 |

Relay 端点与现有 OpenAI 兼容代理共用 `/v1` 根路径，优先匹配后直接处理，无需额外配置路由前缀。

## 配置

### Web UI 配置（推荐）

访问侧边栏 **Codex → Relay**，在「上游渠道」区域添加渠道：

| 字段 | 说明 |
|------|------|
| 名称 | 渠道标识，仅用于界面显示 |
| 上游 URL | 如 `https://api.openai.com/v1` |
| API Key | 上游服务的鉴权 Key |
| 认证头 | 默认 `Authorization`，部分服务需改为 `api-key` |
| 认证前缀 | 默认 `Bearer `，部分服务留空 |

全局配置：

| 字段 | 默认值 | 说明 |
|------|--------|------|
| 默认模型 | 空 | 无映射匹配时的兜底模型 |
| 模型映射 | `{}` | JSON 格式，如 `{"gpt-5.4":"deepseek-chat"}` |
| 工具拒绝列表 | 空 | 逗号分隔，如 `web_search,image_generation` |
| 最大会话数 | 256 | 超出后 LRU 淘汰最旧会话 |
| 最大历史字节 | 512MB | 单次会话历史的字节上限 |
| 会话 TTL | 168h（7天）| 超时自动清理 |

### 注入 Codex CLI 配置

配置好上游渠道后，点击「启动并注入 Codex」按钮，EasyLLM 自动写入：

**`~/.codex/config.toml`**

```toml
model_provider = "relay"
model = "your-default-model"

[model_providers.relay]
name = "EasyLLM Relay"
base_url = "http://localhost:8022/v1"
wire_api = "responses"
requires_openai_auth = false
supports_websockets = false
```

**`~/.codex/auth.json`**

移除 `OPENAI_API_KEY`（认证由 EasyLLM 侧持有，Codex CLI 无需携带）。

### 手动配置

如需手动编辑 `~/.codex/config.toml`：

```toml
model_provider = "relay"
model = "your-model-name"

[model_providers.relay]
name = "EasyLLM Relay"
base_url = "http://localhost:8022/v1"
wire_api = "responses"
requires_openai_auth = false
supports_websockets = false
```

## 多上游轮询

添加多个渠道后，EasyLLM 按 round-robin 策略依次选择启用的上游：

```
请求 1 → 渠道 A
请求 2 → 渠道 B
请求 3 → 渠道 A
...
```

Codex CLI 始终连接本地 `http://localhost:8022/v1`，无感知上游切换。

## 模型映射

在「全局模型映射」中配置 JSON：

```json
{
  "gpt-5.4": "deepseek-chat",
  "gpt-5.5": "deepseek-reasoner",
  "o3": "qwen-max"
}
```

Codex CLI 请求 `gpt-5.4` 时，EasyLLM 实际向上游发送 `deepseek-chat`。

## 调用示例

```bash
# 非流式
curl -X POST http://localhost:8022/v1/responses \
  -H "Content-Type: application/json" \
  -d '{"model":"gpt-5.4","input":"你好","stream":false}'

# 流式
curl -X POST http://localhost:8022/v1/responses \
  -H "Content-Type: application/json" \
  -d '{"model":"gpt-5.4","input":"你好","stream":true}'
```

## 相关模块

| 文件 | 职责 |
|------|------|
| `internal/proxy/relay_handler.go` | HTTP 入口、路由分发、配置 CRUD |
| `internal/proxy/relay_translate.go` | Responses ↔ Chat Completions 协议转换 |
| `internal/proxy/relay_stream.go` | SSE 流式转换 |
| `internal/proxy/relay_session.go` | 会话历史管理（previous_response_id） |
| `internal/proxy/relay_types.go` | 数据结构定义 |
| `internal/proxy/relay_config.go` | 配置持久化（settings 表） |
| `internal/proxy/relay_client.go` | 上游 HTTP Client |
| `internal/proxy/relay_log.go` | Relay 请求日志 |
| `internal/proxy/relay_usage.go` | Token 用量统计 |
| `internal/proxy/relay_mimo.go` | MiMo 思考模型专项适配 |
| `web/src/views/RelayConfigView.vue` | Relay 配置页面 |
