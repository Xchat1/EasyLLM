# EasyLLM

EasyLLM 是一个轻量级 AI 编码账号管理与本地代理对接工具（支持 OpenAI / Codex、Antigravity、Cursor），后端使用 Go，前端使用 Vue 3。它把账号导入、OAuth 与 Token 刷新、配额查询、本地 CLI 配置切换和 OpenAI 兼容代理集中到一个本机界面里，不提供公网部署服务。

[![GitHub](https://img.shields.io/badge/GitHub-EasyLLM-blue?logo=github)](https://github.com/Xchat1/EasyLLM)
[![License](https://img.shields.io/badge/License-Apache--2.0-blue.svg)](./LICENSE)

## 核心能力

- **多平台凭据管理**：统一管理 OpenAI / Codex（OAuth & API Key）、Antigravity 与 Cursor 账号凭据与本地 CLI 环境。
- **Antigravity 集成**：支持 Antigravity OAuth 登录与 Refresh Token 刷新、配额/模型额度查询与激活、一键切换本地 CLI 凭据与唤醒状态。
- **Cursor 集成**：支持 Cursor 账号/Token 录入与刷新、使用量与额度查询、账号轮换切换。
- **Codex CLI 一键切换**：自动原子写入 `~/.codex/auth.json` 与本机代理配置，支持凭据防损坏与异常恢复。
- **批量导入与恢复**：批量导入 Token、CPA、refresh token 列表和 EasyLLM 备份文件，适合多账号迁移与恢复。
- **本地 API 代理池**：内置 OpenAI 兼容代理（`/v1/responses`、`/v1/chat/completions`、`/v1/models` 等），支持 `auto`、`round_robin`、`quota_high_first` 等调度策略，自动根据额度与健康度轮换。
- **Relay 模式**：Codex CLI 通过 EasyLLM 对接任意 OpenAI 兼容上游（DeepSeek、Mistral、OpenRouter、Kimi、Qwen 等），支持多渠道轮询、模型映射和协议转换。
- **高可用与长流式保障**：流式 SSE 接口零死限断开，结合 `http.Flusher` 兼容 gzip 分块实时传输；长任务、大文件补全不断流。
- **安全与防误删防护**：
  - `purge401` 与 `filtersub2api` 仅剔除明确失效凭据，网络抖动、429 限流和 503 故障一律保留，关键清理具备交互式清单二次确认。
  - 本地代理严格锚定 `localhost`，阻断外部 Host Header 投毒攻击。
  - 访问认证与密码重置防接管保护，本地 SQLite 数据库（`0600`）与数据目录（`0700`）权限隔离。
- **Dashboard & 监控**：展示本地代理池与 Relay 的最近调用、状态码、模型、耗时和 Token 用量；Relay 实时日志与会话历史统计。
- **跨平台分发**：支持脚本启动、手动构建、Windows zip 和 macOS App（原生菜单栏与窗口外壳）打包分发。

## 项目优势

- 轻量低占用：macOS M4 空闲状态实测，EasyLLM App 外壳约 78 MB RSS，后端 `easyllm` 约 32 MB RSS，合计约 110 MB，CPU 约 0.0%。实际占用会随账号数量、导入任务和并发请求变化。
- 本机优先：脚本模式默认监听 `127.0.0.1:8022`，macOS App 默认从 `8022` 起自动选择可用本机端口；账号、Token、配置和 SQLite 数据都留在本机，不依赖公网托管服务。
- Codex 友好：围绕 Codex CLI 的账号池、切换、代理注入和配置修复设计，减少手动编辑 `auth.json` / `config.toml` 的成本。
- OpenAI 兼容：本地代理保持 OpenAI 风格接口，现有脚本、工具和客户端可以直接指向 EasyLLM。
- 批量维护效率高：账号导入、配额刷新、Token 刷新、导出备份和恢复集中在一个界面里完成。
- 安全边界清晰：默认不记录代理请求内容，敏感运行产物已加入忽略规则，并提供 pre-push 隐私检查脚本。
- 开源可分发：采用 Apache-2.0 许可，仓库结构精简到 OpenAI / Codex 主线，方便二次开发和本地审计。

## 文档入口

- [使用指南](./docs/USAGE.md)：账号导入、Codex CLI 接入、代理池、Relay、API 示例。
- [Relay 集成说明](./docs/CODEX_RELAY_INTEGRATION.md)：多上游配置、协议转换、模型映射、调用示例。
- [开发说明](./docs/DEVELOPMENT.md)：本地环境、常用命令、测试与构建。
- [项目结构](./docs/PROJECT_STRUCTURE.md)：源码目录、路由结构、运行产物和维护约定。
- [macOS App](./macos/README.md)：原生 App 打包与运行数据位置。

## 主要使用入口

- 「Codex 管理」：导入 OAuth / API Key / CPA / 备份文件，刷新 Token 与配额，切换或注入 Codex 配置。
- 「Antigravity 管理」：管理 Antigravity OAuth 账号、模型额度监控、Token 唤醒与 CLI 配置一键切换。
- 「Cursor 管理」：录入与管理 Cursor 凭证、实时查询使用额度与账号状态。
- 「服务配置」：启动 Codex 本地 API 服务，选择代理池账号、路由策略、端口和本机 API Key。
- 「Relay 配置」：维护第三方上游渠道、模型映射、工具拒绝列表、会话历史限制和 Codex 上下文参数。
- 「Dashboard」：查看本地代理池、Relay 请求统计、最近调用记录和运行状态。
- 「配置」：维护登录、IP 黑名单、出站代理、数据库位置和全局配额检测参数。

## 快速开始

### Release 包

从 [GitHub Releases](https://github.com/Xchat1/EasyLLM/releases) 下载对应系统的压缩包：

| 系统 / 设备 | 下载文件 | 启动方式 |
| --- | --- | --- |
| Windows 10/11 64 位 | `EasyLLM-*-windows-amd64.zip` | 解压后运行 `start-easyllm.bat` |
| Mac Apple Silicon（M1/M2/M3/M4） | `EasyLLM-*-macos-arm64.zip` | 解压后运行 `EasyLLM.app` |
| Mac Intel 芯片 | `EasyLLM-*-macos-amd64.zip` | 解压后运行 `EasyLLM.app` |

不确定 Mac 芯片类型时，点击系统左上角 Apple 菜单 →「关于本机」，查看“芯片”或“处理器”。

默认访问：

```text
http://localhost:8022
```

### 一键脚本

```bash
git clone https://github.com/Xchat1/EasyLLM.git
cd EasyLLM
cp .env.example .env
./scripts/start.sh --build
```

访问：

```text
http://localhost:8022
```

根目录 `./start.sh` 仍可用，它会转发到 `./scripts/start.sh`。

### 手动构建

```bash
cd web
npm install
npm run build
cd ..

CGO_ENABLED=1 go build -o easyllm .
./easyllm
```

### macOS App

```bash
./scripts/build-macos-app.sh
open build/macos/EasyLLM.app
```

生成 macOS Release zip：

```bash
./scripts/build-macos-app.sh --package --version 2.0.2
./scripts/check-release-archives.sh build/release/EasyLLM-2.0.2-macos-*.zip
```

Windows Release zip 由 Windows / PowerShell 环境执行：

```powershell
.\scripts\package-windows.ps1 -Version 2.0.2 -Arch amd64
```

发布包生成后建议执行仓库内置隐私扫描脚本，确认 zip 中没有 `.env`、数据库、Token JSON、日志或本地助手配置等私有文件。

## 基础配置

复制 `.env.example` 为 `.env` 后按需修改：

| 变量 | 默认值 | 说明 |
| --- | --- | --- |
| `SERVER_PORT` | `8022` | HTTP 服务端口 |
| `SERVER_HOST` | `127.0.0.1` | 监听地址 |
| `DB_SQLITE_PATH` | `DATA_DIR/easyllm.db` | SQLite 文件路径；留空时和 macOS App 共用同一套本地数据 |
| `DATA_DIR` | 系统应用配置目录下的 `EasyLLM/data` | 本地数据目录；macOS 为 `~/Library/Application Support/EasyLLM/data` |
| `SECRET_KEY` | 空 | JWT/会话密钥，建议设置长随机值以便重启后保持登录 |
| `DEFAULT_PASSWORD` | 空 | 可选；留空时首次访问 Web UI 创建登录密码；如设置需至少 8 位 |
| `PROXY_ENABLED` | `false` | 出站 HTTP 代理开关 |
| `PROXY_HOST` | 空 | 出站代理主机 |
| `PROXY_PORT` | `7890` | 出站代理端口 |
| `PROXY_USERNAME` | 空 | 出站代理用户名 |
| `PROXY_PASSWORD` | 空 | 出站代理密码 |

## 隐私与安全

- 不要提交 `.env`、`data/`、`auth/`、`exports/`、`backups/`、Token/CPA JSON、EasyLLM 导出备份、私钥、API Key、数据库文件、日志、`build/`、`web/dist/` 或本地助手目录。
- 建议启用仓库内置 pre-push 钩子：

```bash
git config core.hooksPath .githooks
```

- 发布包上传前执行：

```bash
./scripts/check-release-archives.sh build/release/*.zip
```

- EasyLLM 面向本机使用，脚本模式默认监听 `127.0.0.1:8022`；不要把包含账号 Token 的本地服务对公网开放。

## 技术栈

| 层 | 技术 |
| --- | --- |
| 后端 | Go、Gin、GORM |
| 前端 | Vue 3、Vite、Tailwind CSS |
| 数据库 | SQLite |
| 运行 | 本地脚本、手动构建、Windows zip、macOS App |

## License

EasyLLM is licensed under the [Apache License 2.0](./LICENSE).
