import { authAPI } from '@/api'

let authStatusCache = null
let authStatusPromise = null

export async function fetchAuthStatus(forceRefresh = false) {
  if (authStatusCache !== null && !forceRefresh) {
    return authStatusCache
  }
  if (authStatusPromise && !forceRefresh) {
    return authStatusPromise
  }
  authStatusPromise = authAPI.check()
    .then(data => {
      authStatusCache = data
      authStatusPromise = null
      return data
    })
    .catch(() => {
      authStatusPromise = null
      return { auth_enabled: false, password_set: false }
    })
  return authStatusPromise
}

export function setAuthStatusCache(data) {
  authStatusCache = { ...(authStatusCache || {}), ...data }
}

export function getCachedAuthStatus() {
  return authStatusCache
}
