#!/usr/bin/env bash
# Build EasyLLM as a native macOS .app wrapper around the existing Go + Web UI.

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT_DIR="$(cd "${SCRIPT_DIR}/.." && pwd)"
APP_NAME="EasyLLM"
BUILD_DIR="${ROOT_DIR}/build/macos"
RELEASE_DIR="${ROOT_DIR}/build/release"
APP_DIR="${BUILD_DIR}/${APP_NAME}.app"
CONTENTS_DIR="${APP_DIR}/Contents"
MACOS_DIR="${CONTENTS_DIR}/MacOS"
RESOURCES_DIR="${CONTENTS_DIR}/Resources"
VERSION="${EASYLLM_VERSION:-2.0.0}"
PACKAGE_RELEASE=0
PACKAGE_ARCH="${EASYLLM_PACKAGE_ARCH:-}"
export GOCACHE="${GOCACHE:-${BUILD_DIR}/go-cache}"
SWIFT_MODULE_CACHE="${BUILD_DIR}/swift-module-cache"
export CLANG_MODULE_CACHE_PATH="${BUILD_DIR}/clang-module-cache"
ICONSET_DIR="${BUILD_DIR}/${APP_NAME}.iconset"
ICON_BUILDER="${BUILD_DIR}/make-app-icon"
APP_ICON_SOURCE="${ROOT_DIR}/web/src/assets/brand/easyllm-app-icon.png"

while [[ $# -gt 0 ]]; do
  case "$1" in
    --package)
      PACKAGE_RELEASE=1
      ;;
    --version)
      VERSION="${2:?缺少 --version 参数}"
      shift
      ;;
    --arch-label)
      PACKAGE_ARCH="${2:?缺少 --arch-label 参数}"
      shift
      ;;
    *)
      echo "未知参数: $1"
      exit 1
      ;;
  esac
  shift
done

if [[ -z "${PACKAGE_ARCH}" ]]; then
  case "$(uname -m)" in
    x86_64) PACKAGE_ARCH="amd64" ;;
    arm64) PACKAGE_ARCH="arm64" ;;
    *) PACKAGE_ARCH="$(uname -m)" ;;
  esac
fi

require_command() {
  if ! command -v "$1" >/dev/null 2>&1; then
    echo "✗ 缺少依赖: $1"
    exit 1
  fi
}

require_command go
require_command npm
require_command swiftc

cd "${ROOT_DIR}"

echo "=== Building EasyLLM macOS App ==="
echo "Version: ${VERSION}"
echo "Arch:    ${PACKAGE_ARCH}"

echo "→ 构建前端 web/dist"
if [[ ! -d "${ROOT_DIR}/web/node_modules" ]]; then
  (cd "${ROOT_DIR}/web" && npm install --legacy-peer-deps)
fi
(cd "${ROOT_DIR}/web" && npm run build)

echo "→ 准备 App Bundle"
rm -rf "${APP_DIR}"
mkdir -p "${MACOS_DIR}" "${RESOURCES_DIR}/web" "${GOCACHE}" "${SWIFT_MODULE_CACHE}" "${CLANG_MODULE_CACHE_PATH}"

echo "→ 编译 Go 后端"
CGO_ENABLED=1 go build -trimpath -ldflags="-w -s" -o "${RESOURCES_DIR}/easyllm" .
chmod +x "${RESOURCES_DIR}/easyllm"

echo "→ 拷贝前端资源"
cp -R "${ROOT_DIR}/web/dist" "${RESOURCES_DIR}/web/dist"

echo "→ 生成 macOS 图标"
rm -rf "${ICONSET_DIR}"
swiftc \
  -O \
  -module-cache-path "${SWIFT_MODULE_CACHE}" \
  -framework AppKit \
"${ROOT_DIR}/macos/MakeAppIcon.swift" \
  -o "${ICON_BUILDER}"
"${ICON_BUILDER}" "${APP_ICON_SOURCE}" "${ICONSET_DIR}" "${RESOURCES_DIR}/${APP_NAME}.icns"

echo "→ 编译 macOS 启动器"
swiftc \
  -O \
  -module-cache-path "${SWIFT_MODULE_CACHE}" \
  -framework AppKit \
  -framework WebKit \
  "${ROOT_DIR}/macos/EasyLLMApp.swift" \
  -o "${MACOS_DIR}/${APP_NAME}"

cat > "${CONTENTS_DIR}/Info.plist" <<PLIST
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
  <key>CFBundleDevelopmentRegion</key>
  <string>zh_CN</string>
  <key>CFBundleDisplayName</key>
  <string>EasyLLM</string>
  <key>CFBundleExecutable</key>
  <string>EasyLLM</string>
  <key>CFBundleIdentifier</key>
  <string>com.easyllm.desktop</string>
  <key>CFBundleIconFile</key>
  <string>EasyLLM</string>
  <key>CFBundleInfoDictionaryVersion</key>
  <string>6.0</string>
  <key>CFBundleName</key>
  <string>EasyLLM</string>
  <key>CFBundlePackageType</key>
  <string>APPL</string>
  <key>CFBundleShortVersionString</key>
  <string>${VERSION}</string>
  <key>CFBundleVersion</key>
  <string>${VERSION}</string>
  <key>LSApplicationCategoryType</key>
  <string>public.app-category.developer-tools</string>
  <key>LSMinimumSystemVersion</key>
  <string>12.0</string>
  <key>NSAppTransportSecurity</key>
  <dict>
    <key>NSAllowsArbitraryLoads</key>
    <false/>
    <key>NSAllowsLocalNetworking</key>
    <true/>
  </dict>
  <key>NSHighResolutionCapable</key>
  <true/>
</dict>
</plist>
PLIST

chmod +x "${MACOS_DIR}/${APP_NAME}"

if command -v codesign >/dev/null 2>&1; then
  echo "→ Ad-hoc 签名"
  codesign --force --deep --sign - "${APP_DIR}" >/dev/null
fi

echo
echo "=== macOS App Build Complete ==="
echo "App: ${APP_DIR}"
echo "Run: open \"${APP_DIR}\""

if [[ "${PACKAGE_RELEASE}" == "1" ]]; then
  PACKAGE_NAME="${APP_NAME}-${VERSION}-macos-${PACKAGE_ARCH}"
  ZIP_PATH="${RELEASE_DIR}/${PACKAGE_NAME}.zip"
  mkdir -p "${RELEASE_DIR}"
  rm -f "${ZIP_PATH}"
  echo "→ 生成 release zip"
  ditto -c -k --sequesterRsrc --keepParent "${APP_DIR}" "${ZIP_PATH}"
  echo "Package: ${ZIP_PATH}"
fi
