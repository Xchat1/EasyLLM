const MAC_APP_STORAGE_KEY = 'easyllm_mac_app'

/** 从路由 query 同步 macOS App 标记（与 EasyLLMApp.swift 的 mac_app=1 一致） */
export function syncMacAppFromRoute(route) {
  if (route?.query?.mac_app === '1') {
    sessionStorage.setItem(MAC_APP_STORAGE_KEY, '1')
    return true
  }
  return isMacApp()
}

export function isMacApp() {
  return sessionStorage.getItem(MAC_APP_STORAGE_KEY) === '1'
}

/** Web 默认进总览；macOS App 默认进 Codex 管理（与客户端启动 URL 一致） */
export function defaultHomePath() {
  return isMacApp() ? '/codex' : '/dashboard'
}

/** 当前页面的 EasyLLM 服务根地址（与后端 buildCodexAPIServiceBaseURL 对齐） */
export function localServiceOrigin() {
  const { protocol, hostname, port } = window.location
  const effectivePort = port || '8022'
  return `${protocol}//${hostname}:${effectivePort}`
}

export function localServiceAPIBaseURL() {
  return `${localServiceOrigin()}/v1`
}
