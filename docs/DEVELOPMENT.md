# 开发说明

## 本地环境

- Go 1.22+
- Node.js 18+
- `gcc` 或等价 C 编译工具链（SQLite CGO 依赖）

## 常用命令

### 开发启动

```bash
./scripts/start.sh
```

### 构建后启动

```bash
./scripts/start.sh --build
```

### 运行已有二进制

```bash
./scripts/start.sh --prod
```

### 后端测试

```bash
go test ./...
```

### 前端构建校验

```bash
cd web
npm run build
```

## 开发流程建议

1. 修改后端接口或数据结构时，优先补上对应测试。
2. 修改前端路由、页面文案或 API 调用后，至少执行一次 `npm run build`。
3. 结构性调整优先同步更新 `README.md` 和 `docs/PROJECT_STRUCTURE.md`，避免文档落后于代码。
4. 若 8022 端口被系统 ghost socket 占用，可使用 `sudo ./scripts/setup-pf-8022-redirect.sh` 后改用 `SERVER_PORT=8026 ./scripts/start.sh`。
