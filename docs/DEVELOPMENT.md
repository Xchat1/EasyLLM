# 开发说明

## 环境要求

- Go 1.25+
- Node.js 18+
- `gcc` 或等价 C 编译工具链（SQLite CGO 依赖）
- macOS App 打包需要 `swiftc`（可选）
- Windows Release 打包需要 PowerShell、Node.js、Go 和可用的 CGO 编译工具链

## 常用命令

### 启动

```bash
# 开发模式：go run + 当前 web/dist
./scripts/start.sh

# 构建前端和后端后启动
./scripts/start.sh --build

# 运行已有本地二进制
./scripts/start.sh --prod
```

Windows：

```powershell
.\scripts\start.ps1
.\scripts\start.ps1 -build
.\scripts\start.ps1 -prod
```

```bat
scripts\start.bat
scripts\start.bat --build
scripts\start.bat --prod
```

### 前端

```bash
cd web
npm install
npm run dev
npm run build
```

`npm run build` 会依次执行：

1. `npm run normalize:structure`
2. `npm run audit:theme`
3. `vite build`

前端开发服务器默认监听 `127.0.0.1:5180`，用于开发 Vue 页面和热更新。它会把 `/api`、`/v1`、`/pool` 等请求代理到 Go 后端，默认目标是 `http://localhost:8022`。

Go 后端默认监听 `127.0.0.1:8022`。后端除了提供 API 和 Relay 接口，也会把 `web/dist` 挂到 `/`，所以直接访问 `http://localhost:8022` 也会看到 Web UI。开发时如果同时打开 `5180` 和 `8022`，两个页面看起来可能一样；区别是 `5180` 使用当前前端源码，`8022` 使用上一次构建生成的 `web/dist`。

### 后端

```bash
go test ./...
go vet ./...
CGO_ENABLED=1 go build -o easyllm .
```

### macOS App

```bash
./scripts/build-macos-app.sh
open build/macos/EasyLLM.app
```

生成 macOS Release zip：

```bash
./scripts/build-macos-app.sh --package --version 2.0.0
./scripts/check-release-archives.sh build/release/EasyLLM-2.0.0-macos-*.zip
```

### Windows Release

在 Windows / PowerShell 环境执行：

```powershell
.\scripts\package-windows.ps1 -Version 2.0.0 -Arch amd64
```

产物：

```text
build/release/EasyLLM-2.0.0-windows-amd64.zip
```

## 本地配置

```bash
cp .env.example .env
```

开发时常用变量：

```env
SERVER_PORT=8022
SERVER_HOST=127.0.0.1
DB_SQLITE_PATH=
DATA_DIR=
SECRET_KEY=replace-with-a-long-random-secret
DEFAULT_PASSWORD=
```

`DEFAULT_PASSWORD` 留空时，首次访问 Web UI 会引导创建登录密码；如设置需至少 8 位。不要在示例、文档或提交记录中保留固定默认密码。

## 端口占用处理

脚本模式默认使用 `SERVER_PORT=8022`。如果端口被占用，可临时换端口启动：

```bash
SERVER_PORT=8026 ./scripts/start.sh
```

访问：

```text
http://localhost:8026
```

## 隐私保护

启用仓库内置 Git hook：

```bash
git config core.hooksPath .githooks
```

不要提交：

- `.env`
- `data/`
- `auth/`
- `exports/`
- `backups/`
- `build/`
- `web/dist/`
- `logs/`
- Token/CPA JSON
- EasyLLM 导出备份
- API Key、私钥、数据库文件
- `.claude/`、`.codex/`、`.agents/` 等本地助手配置

发布包生成后执行：

```bash
./scripts/check-release-archives.sh build/release/*.zip
```

这个脚本会检查 zip 中是否包含 `.env`、Token/CPA JSON、数据库、日志、本地助手目录或疑似密钥。

## 变更建议

- 后端接口、模型或存储逻辑变更后，补充或更新 Go 测试。
- 前端页面、路由、API 调用或样式变更后，至少运行 `npm run build`。
- 改动导入格式、代理池、Codex 配置写入等高风险流程时，优先补充端到端的手动验证步骤。
- 新增用户可见功能时，同步更新 `README.md` 和 `docs/USAGE.md`。
- 新增目录、脚本或命令时，同步更新 `docs/PROJECT_STRUCTURE.md`。

## 发布前检查

```bash
go test ./...
go vet ./...
cd web && npm run build
```

如需发布 macOS App：

```bash
./scripts/build-macos-app.sh --package --version 2.0.0
./scripts/check-release-archives.sh build/release/EasyLLM-2.0.0-macos-*.zip
```

如需发布 Windows zip：

```powershell
.\scripts\package-windows.ps1 -Version 2.0.0 -Arch amd64
```

GitHub Release 自动打包：

```bash
git tag -a v2.0.0 -m "v2.0.0"
git push origin v2.0.0
```

`.github/workflows/release.yml` 会构建并上传：

- `EasyLLM-*-windows-amd64.zip`
- `EasyLLM-*-macos-arm64.zip`
- `EasyLLM-*-macos-amd64.zip`
- `SHA256SUMS.txt`
