# 项目结构

EasyLLM 采用 Go 后端 + Vue 3 前端的单仓结构，开发入口和部署入口统一放在仓库根目录与 `scripts/`、`docs/` 下。

## 目录总览

```text
EasyLLM/
├── cmd/                        # 辅助命令行工具
│   ├── importrefresh/
│   └── importsub2api/
├── config/                     # 配置加载与运行时配置对象
├── docs/                       # 项目文档
│   ├── DEVELOPMENT.md
│   └── PROJECT_STRUCTURE.md
├── internal/
│   ├── handlers/               # HTTP Handler 与接口编排
│   ├── models/                 # 数据模型与响应结构
│   ├── platforms/              # 平台侧能力（OAuth、配额等）
│   ├── proxy/                  # 代理池、转发、WebSocket
│   ├── server/                 # Gin 服务装配与路由注册
│   └── storage/                # 持久化与数据访问层
├── scripts/                    # 启动、构建、平台辅助脚本
│   ├── build.sh
│   ├── setup-pf-8022-redirect.sh
│   ├── start.bat
│   ├── start.ps1
│   └── start.sh
├── web/                        # Vue 3 前端
│   ├── public/
│   ├── src/
│   │   ├── api/
│   │   ├── lib/
│   │   ├── router/
│   │   └── views/
│   └── dist/                   # 前端构建产物（生成文件）
├── Dockerfile
├── docker-compose.yml
├── main.go                     # 应用入口
├── start.sh                    # 兼容入口，转发到 scripts/start.sh
└── README.md
```

## 约定

- `internal/` 中按职责分层，避免跨层直接耦合。
- `cmd/` 只放独立执行的辅助命令，不混入主服务逻辑。
- `scripts/` 放“可执行流程”，`docs/` 放“可阅读规范”。
- `web/dist/`、`data/`、认证文件和本地二进制均属于运行产物，不作为源码结构的一部分。
