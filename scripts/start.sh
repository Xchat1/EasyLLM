#!/usr/bin/env bash
# EasyLLM 快速启动脚本 (macOS / Linux)
# 用法:
#   ./scripts/start.sh           # go run 模式（开发）
#   ./scripts/start.sh --build   # 先构建前端和后端再运行
#   ./scripts/start.sh --prod    # 运行已编译的二进制

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT_DIR="$(cd "${SCRIPT_DIR}/.." && pwd)"
cd "${ROOT_DIR}"

MODE="dev"
for arg in "$@"; do
  case "$arg" in
    --build) MODE="build" ;;
    --prod) MODE="prod" ;;
  esac
done

PORT="${SERVER_PORT:-8022}"
if [[ -f ".env" ]]; then
  ENV_PORT="$(awk -F= '/^[[:space:]]*SERVER_PORT[[:space:]]*=/{print $2}' .env | tail -n 1 | tr -d '[:space:]')"
  if [[ -n "${ENV_PORT}" ]]; then
    PORT="${ENV_PORT}"
  fi
fi

print_info() {
  local active_port="${SERVER_PORT:-$PORT}"
  echo
  echo "========================================"
  echo "  EasyLLM 快速启动脚本"
  echo "  模式  : ${MODE}"
  echo "  端口  : ${active_port}"
  echo "  目录  : ${ROOT_DIR}"
  echo "========================================"
  echo "  Web UI : http://localhost:${active_port}"
  echo "  API    : http://localhost:${active_port}/api/v1"
  echo "  Pool   : http://localhost:${active_port}/pool/status"
  echo "========================================"
  echo
  echo "按 Ctrl+C 停止服务"
  echo
}

kill_port() {
  local port="$1"
  local pids

  pids="$(lsof -nP -tiTCP:"${port}" -sTCP:LISTEN 2>/dev/null || true)"
  if [[ -z "${pids}" ]]; then
    echo "✓ 端口 ${port} 空闲"
    return
  fi

  echo "⚠ 端口 ${port} 被占用 (PID: ${pids})，尝试优雅停止..."
  kill ${pids} 2>/dev/null || true
  sleep 1

  pids="$(lsof -nP -tiTCP:"${port}" -sTCP:LISTEN 2>/dev/null || true)"
  if [[ -n "${pids}" ]]; then
    echo "⚠ 端口 ${port} 仍被占用，强制终止 PID: ${pids}"
    kill -9 ${pids} 2>/dev/null || true
    sleep 1
  fi

  echo "✓ 端口 ${port} 已检查/释放"
}

load_env() {
  if [[ ! -f ".env" ]]; then
    echo "→ 未找到 .env，使用默认配置（可 cp .env.example .env 自定义）"
    return
  fi

  echo "→ 加载 .env"
  set -a
  # shellcheck disable=SC1091
  . ".env"
  set +a
}

build_frontend() {
  echo
  echo "→ 构建前端..."
  cd "${ROOT_DIR}/web"
  npm install --legacy-peer-deps --silent
  npm run build
  cd "${ROOT_DIR}"
  echo "✓ 前端构建完成"
}

build_backend() {
  echo
  echo "→ 编译 Go 后端..."
  CGO_ENABLED=1 go build -ldflags="-w -s" -o easyllm .
  echo "✓ 编译完成 -> ./easyllm"
}

kill_port "${PORT}"
load_env

case "${MODE}" in
  dev)
    echo
    echo "→ 以 go run 方式启动（开发模式）"
    print_info
    exec go run main.go
    ;;
  build)
    build_frontend
    build_backend
    echo
    echo "→ 启动编译后的二进制"
    print_info
    exec ./easyllm
    ;;
  prod)
    if [[ ! -x "./easyllm" ]]; then
      echo "✗ 未找到二进制文件，请先运行: ./scripts/start.sh --build"
      exit 1
    fi
    echo
    echo "→ 启动本地二进制"
    print_info
    exec ./easyllm
    ;;
  *)
    echo "✗ 未知模式: ${MODE}"
    exit 1
    ;;
esac
