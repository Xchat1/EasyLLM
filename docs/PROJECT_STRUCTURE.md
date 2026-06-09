# 项目结构

EasyLLM 是 Go 后端、Vue 3 前端和可选 macOS App 启动器组成的单仓项目。源码、文档、脚本和运行产物分层放置，便于本地运行和桌面端打包。

## 目录总览

```text
EasyLLM/
├── .github/workflows/           # GitHub Actions 自动化
│   └── release.yml              # Windows / macOS Release 打包发布
├── config/                     # 环境变量与运行配置
├── docs/                       # 项目文档
│   ├── DEVELOPMENT.md          # 开发说明
│   ├── PROJECT_STRUCTURE.md    # 项目结构
│   └── USAGE.md                # 使用指南
├── internal/
│   ├── handlers/               # HTTP Handler 与业务编排
│   ├── models/                 # 数据模型与响应结构
│   ├── openai/                 # OAuth、配额、本地 Codex 配置写入
│   ├── proxy/                  # /v1/* 转发、代理池、WebSocket
│   ├── server/                 # Gin 服务装配与路由注册
│   └── storage/                # 数据库访问层
├── macos/                      # macOS App 启动器与图标生成脚本
├── scripts/                    # 启动、构建、系统辅助脚本
│   ├── build-macos-app.sh
│   ├── build.sh
│   ├── package-windows.ps1
│   ├── setup-pf-8022-redirect.sh
│   ├── start.bat
│   ├── start.ps1
│   └── start.sh
├── web/                        # Vue 3 前端
│   ├── public/
│   ├── src/
│   │   ├── api/                # API 封装
│   │   ├── assets/             # 图片、图标等静态资源
│   │   ├── components/         # 通用组件
│   │   ├── composables/        # Vue 组合式逻辑
│   │   ├── config/             # 前端配置
│   │   ├── lib/                # 前端纯逻辑工具
│   │   ├── router/             # 路由
│   │   └── views/              # 页面
│   └── dist/                   # 前端构建产物
├── go.mod
├── LICENSE                     # Apache-2.0
├── main.go                     # 应用入口
├── README.md                   # 项目入口文档
└── start.sh                    # 兼容入口，转发到 scripts/start.sh
```

## 分层约定

- `internal/handlers` 负责 HTTP 请求解析、响应组装和跨模块编排。
- `internal/storage` 只处理持久化，不直接承载 HTTP 语义。
- `internal/proxy` 负责代理池、请求转发、WebSocket 和账号轮换。
- `internal/openai` 放 OpenAI / Codex 相关的 OAuth、配额、配置写入等能力。
- `scripts/` 放可执行流程，`docs/` 放可阅读维护文档。
- `.github/workflows/release.yml` 在推送 `v*` 标签或手动触发时生成 Windows / macOS Release 包。

## 运行产物

以下内容属于本地运行产物或敏感数据，不应提交到 Git：

- `.env`
- `data/`
- `auth/`
- `exports/`
- `backups/`
- `web/dist/`
- `build/`
- `build/release/`
- `.claude/`、`.codex/`、`.agents/` 等本地助手配置
- Token JSON、CPA JSON、EasyLLM 备份、数据库文件、API Key、私钥

如果敏感文件已经被 Git 跟踪，需要先执行：

```bash
git rm --cached <file>
```

## 文档维护

- 新增用户可见能力时，同步更新 `README.md` 和 `docs/USAGE.md`。
- 新增目录、脚本或辅助命令时，同步更新本文件。
- 修改开发流程、测试命令或构建要求时，同步更新 `docs/DEVELOPMENT.md`。
