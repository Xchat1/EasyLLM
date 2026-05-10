# EasyLLM

轻量级 AI 多平台账号管理与代理工具，Go + Vue 3 全栈，全 Web 界面操作。

[![GitHub](https://img.shields.io/badge/GitHub-EasyLLM-blue?logo=github)](https://github.com/libaxuan/EasyLLM)

## 界面预览

![EasyLLM 总览](./总览.png)

## 功能特性

**多平台账号管理**
- **OpenAI / Codex** — OAuth 账号管理、API Key 配置、Token 刷新、配额查询、Codex CLI 一键切换
- **Antigravity** — 账号管理与激活

**Codex 代理池**
- OpenAI 兼容 API（`/v1/responses`），多账号自动负载均衡
- 支持 round_robin / random / least_used 三种策略
- WebSocket 代理支持（Codex CLI wss 连接）
- 不保留 API 调用日志，避免敏感提示词和响应内容落盘
- API Key 鉴权保护

**系统能力**
- 全 Web 操作界面，暗色主题
- SQLite / PostgreSQL 双数据库支持
- HTTP 代理转发、IP 黑名单
- Docker 一键部署
- 内置使用文档

## 快速开始

更多维护说明见：

- [开发说明](./docs/DEVELOPMENT.md)
- [项目结构](./docs/PROJECT_STRUCTURE.md)

### 方式一：直接运行

**前置要求：** Go 1.22+、Node.js 18+、gcc（SQLite CGO 依赖）

```bash
git clone https://github.com/libaxuan/EasyLLM.git
cd EasyLLM

# 1. 构建前端
cd web && npm install && npm run build && cd ..

# 2. 配置（可选）
cp .env.example .env

# 3. 编译并运行
CGO_ENABLED=1 go build -o easyllm .
./easyllm
```

访问 http://localhost:8022

### 方式二：Docker 部署

```bash
git clone https://github.com/libaxuan/EasyLLM.git
cd EasyLLM
docker compose up -d
```

访问 http://localhost:8022

### 方式三：macOS App

保留现有 Web 项目和 Go 后端，同时可以封装成原生 macOS App：

```bash
./scripts/build-macos-app.sh
open build/macos/EasyLLM.app
```

App 会内置后端二进制和 `web/dist`，运行数据保存在：

```text
~/Library/Application Support/EasyLLM/
```

### 方式四：一键启动脚本（推荐）

自动检测并释放被占用的端口，无需手动杀进程：

**Mac / Linux**
```bash
# 开发模式（go run，无需预编译）
./scripts/start.sh

# 先编译前端+后端再运行
./scripts/start.sh --build

# 直接运行已编译的二进制（需提前 --build）
./scripts/start.sh --prod
```

**Windows (PowerShell)**
```powershell
.\scripts\start.ps1          # 开发模式
.\scripts\start.ps1 --build  # 编译后运行
.\scripts\start.ps1 --prod   # 运行二进制
```

**Windows (CMD)**
```bat
scripts\start.bat
scripts\start.bat --build
scripts\start.bat --prod
```

兼容旧入口：根目录 `./start.sh` 仍可用，但内部已统一转发到 `./scripts/start.sh`。

**若提示「端口被 ghost socket 占用」**（常见于 Mac 上 LVSecurityAgent 等代理曾占用 8022）：
- **不重启**：执行一次 `sudo ./scripts/setup-pf-8022-redirect.sh`，将 8022 转发到 8026；然后 `SERVER_PORT=8026 ./scripts/start.sh` 启动，访问 http://localhost:8022 即可。
- **重启 Mac**：内核会释放 ghost socket，之后直接 `./scripts/start.sh` 使用 8022。

## 配置

复制 `.env.example` 为 `.env` 并按需修改：

| 变量 | 默认值 | 说明 |
|------|--------|------|
| `SERVER_PORT` | `8022` | 服务端口 |
| `SERVER_HOST` | `0.0.0.0` | 监听地址 |
| `DB_TYPE` | `sqlite` | 数据库类型（sqlite / postgres） |
| `DB_SQLITE_PATH` | `./data/easyllm.db` | SQLite 文件路径 |
| `DB_DSN` | - | PostgreSQL 连接字符串 |
| `DATA_DIR` | `./data` | 数据目录 |
| `SECRET_KEY` | - | 应用密钥（生产环境务必修改） |
| `DEBUG` | `false` | 调试模式 |
| `PROXY_ENABLED` | `false` | HTTP 代理开关 |
| `PROXY_HOST` | - | 代理主机 |
| `PROXY_PORT` | - | 代理端口 |

## Git 推送前隐私保护

仓库内置了版本化 `pre-push` 钩子，用来拦截常见敏感文件和疑似密钥内容，避免把 `.env`、token JSON、私钥或真实 API Key 推到远端。

首次在本地启用：

```bash
git config core.hooksPath .githooks
```

已默认忽略的高风险本地文件包括 `.env`、`auth/*.json`、`cred.json`、`big_token.json`。如果某个敏感文件已经被 Git 跟踪，还需要执行 `git rm --cached <file>`，否则历史提交里仍可能包含它。

## Codex CLI 接入

**OAuth 账号（推荐）：** 在 Web 界面添加 OAuth 账号后点击"切换"，自动写入 `~/.codex/auth.json` 并注入 `chatgpt_base_url`。

**代理池模式：** 启用多个账号的代理开关后，在 `~/.codex/config.toml` 中配置：

```toml
chatgpt_base_url = "http://localhost:8022"
```

Codex CLI 的所有请求将自动通过 EasyLLM 轮询池中的账号。

![Codex CLI 接入效果](./codex.png)

## API 参考

### 代理端点（OpenAI 兼容）

```
POST /v1/responses              — Codex Responses API（流式）
GET  /v1/models                 — 获取模型列表
GET  /pool/status               — 代理池状态
```

### cURL 示例

```bash
# 发送请求（通过代理池）
curl http://localhost:8022/v1/responses \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer YOUR_API_KEY" \
  -d '{"model":"gpt-5.4","input":"hello","stream":true}'

# 查看代理池状态
curl http://localhost:8022/pool/status
```

### 管理 API

```
GET  /api/v1/openai/accounts              — OpenAI 账号列表
POST /api/v1/openai/import/refresh-tokens — 批量导入 refresh_token
POST /api/v1/openai/import/scan-dir       — 扫描目录导入 token 文件
POST /api/v1/openai/accounts/fetch-quotas — 批量查询配额
GET  /api/v1/health                       — 健康检查（鉴权路由）
GET  /api/health                          — 兼容健康检查（Docker healthcheck）
GET  /api/v1/system/info                  — 系统信息
```

## 技术栈

| 层 | 技术 |
|---|---|
| 后端 | Go 1.25、Gin、GORM |
| 前端 | Vue 3、Vite 6、Tailwind CSS |
| 数据库 | SQLite / PostgreSQL |
| 部署 | Docker、Docker Compose |

## 项目结构

```text
EasyLLM/
├── cmd/                        # 辅助命令
├── docs/                       # 维护文档
├── main.go                     # 入口
├── config/                     # 配置加载
├── internal/
│   ├── models/                 # 数据模型
│   ├── storage/                # 数据存储层
│   ├── handlers/               # HTTP 路由处理
│   ├── platforms/              # 平台业务逻辑
│   │   └── openai/             # OpenAI OAuth & 配额
│   ├── proxy/                  # Codex 代理 & WebSocket
│   └── server/                 # HTTP 服务器
├── web/                        # Vue 3 前端
│   ├── src/
│   │   ├── views/              # 页面组件
│   │   ├── api/                # API 封装
│   │   └── router/             # 路由
│   └── dist/                   # 构建产物
├── scripts/                    # 启动/构建/系统脚本
├── start.sh                    # 兼容入口（转发到 scripts/start.sh）
├── Dockerfile
└── .env.example
```

更完整说明见 [docs/PROJECT_STRUCTURE.md](./docs/PROJECT_STRUCTURE.md)。

## License

MIT

## Links

- **GitHub:** https://github.com/libaxuan/EasyLLM
