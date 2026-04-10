#!/usr/bin/env bash
# 一键用 pf 把 8022 转发到 8026，解决 ghost socket 占用 8022 无法绑定的问题（无需重启）
# 必须用 sudo 运行:  sudo ./scripts/setup-pf-8022-redirect.sh
# 之后：EasyLLM 监听 8026，访问 http://127.0.0.1:8022 即访问 8026

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ANCHOR_SRC="${SCRIPT_DIR}/pf-easyllm.anchor"
ANCHOR_DEST="/etc/pf.anchors/easyllm"
PF_CONF="/etc/pf.conf"

if [[ $EUID -ne 0 ]]; then
  echo "请使用 sudo 运行此脚本: sudo $0"
  exit 1
fi

echo "安装 pf 转发规则: 127.0.0.1:8022 -> 127.0.0.1:8026"

# 1. 安装 anchor 文件
cp -f "${ANCHOR_SRC}" "${ANCHOR_DEST}"
chmod 644 "${ANCHOR_DEST}"
echo "  - 已写入 ${ANCHOR_DEST}"

# 2. 在 pf.conf 中追加 easyllm anchor（若尚未存在）
if ! grep -q 'rdr-anchor "easyllm"' "${PF_CONF}"; then
  echo '' >> "${PF_CONF}"
  echo '# EasyLLM: 8022 -> 8026 (ghost socket 规避)' >> "${PF_CONF}"
  echo 'rdr-anchor "easyllm"' >> "${PF_CONF}"
  echo 'load anchor "easyllm" from "/etc/pf.anchors/easyllm"' >> "${PF_CONF}"
  echo "  - 已在 ${PF_CONF} 末尾添加 easyllm anchor"
else
  echo "  - ${PF_CONF} 中已存在 easyllm anchor，跳过"
fi

# 3. 重新加载 pf
pfctl -f "${PF_CONF}" 2>/dev/null || true
pfctl -e 2>/dev/null || true
echo "  - 已重新加载 pf"

echo ""
echo "完成。请将 EasyLLM 运行在 8026 端口（如 SERVER_PORT=8026 scripts/start.sh），"
echo "然后通过 http://127.0.0.1:8022 或 http://localhost:8022 访问（流量会转发到 8026）。"
