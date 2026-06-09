# EasyLLM macOS App

`macos/` 目录包含 EasyLLM 的原生 macOS 启动器。它不替换 Go 后端或 Vue 前端，而是在 App Bundle 中内置后端二进制和 `web/dist`，启动后通过 `WKWebView` 打开本地服务。

## 打包

```bash
./scripts/build-macos-app.sh
```

Release zip：

```bash
./scripts/build-macos-app.sh --package --version 2.0.0
```

构建产物：

```text
build/macos/EasyLLM.app
build/release/EasyLLM-2.0.0-macos-<arch>.zip
```

启动：

```bash
open build/macos/EasyLLM.app
```

## 打包流程

脚本会执行：

1. 构建前端 `web/dist`。
2. 编译 Go 后端为 `EasyLLM.app/Contents/Resources/easyllm`。
3. 拷贝 `web/dist` 到 `EasyLLM.app/Contents/Resources/web/dist`。
4. 使用 `macos/MakeAppIcon.swift` 从 `web/src/assets/brand/easyllm-app-icon.png` 生成 `.icns`。
5. 编译 `macos/EasyLLMApp.swift` 为 App 主程序。
6. 如系统存在 `codesign`，执行 ad-hoc 签名。
7. 传入 `--package` 时，用 `ditto` 生成保留 App Bundle 元数据的 zip。

## 运行数据

App 运行数据存放在：

```text
~/Library/Application Support/EasyLLM/
```

普通浏览器方式启动的本地后端在未显式配置 `DATA_DIR` / `DB_SQLITE_PATH` 时，也会默认使用同一套数据目录，确保浏览器访问和 macOS App 访问看到相同账号数据。

常见文件：

```text
data/easyllm.db
easyllm.log
secret.key
```

## 客户端行为

- App 启动时会自动寻找可用端口，默认从 `8022` 开始。
- 后端仅监听 `127.0.0.1`。
- App 启动时会带上 `mac_app=1` 标记，前端默认进入 Codex 管理页。
- WebView 支持系统文件选择面板，导入 JSON 或选择目录时会弹出 macOS 原生面板。

## 相关文件

```text
macos/EasyLLMApp.swift     # App 启动器
macos/MakeAppIcon.swift    # 图标生成
scripts/build-macos-app.sh # 打包脚本
```
