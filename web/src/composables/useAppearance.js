import { computed, ref } from 'vue'
import {
  DEFAULT_APPEARANCE,
  THEME_MODES,
  THEME_STORAGE_KEYS,
  getAccentThemeLabel,
  getThemeModeLabel,
  getThemeModeShortLabel,
  normalizeAccentTheme,
  normalizeThemeMode,
} from '@/config/theme'

const themeMode = ref(readStoredValue(THEME_STORAGE_KEYS.mode, DEFAULT_APPEARANCE.mode))
const accentTheme = ref(readStoredValue(THEME_STORAGE_KEYS.accent, DEFAULT_APPEARANCE.accent))
const resolvedThemeMode = ref('dark')
let mediaQuery = null
let initialized = false

const themeModeLabel = computed(() => getThemeModeLabel(themeMode.value))
const themeModeShortLabel = computed(() => getThemeModeShortLabel(themeMode.value))
const accentThemeLabel = computed(() => getAccentThemeLabel(accentTheme.value))

export function initAppearance() {
  if (typeof window === 'undefined' || initialized) return
  initialized = true
  mediaQuery = window.matchMedia('(prefers-color-scheme: dark)')
  const onSystemThemeChange = () => applyAppearance()

  if (mediaQuery.addEventListener) {
    mediaQuery.addEventListener('change', onSystemThemeChange)
  } else if (mediaQuery.addListener) {
    mediaQuery.addListener(onSystemThemeChange)
  }

  applyAppearance()
}

export function useAppearance() {
  initAppearance()

  return {
    themeMode,
    accentTheme,
    resolvedThemeMode,
    themeModeLabel,
    themeModeShortLabel,
    accentThemeLabel,
    setAppearance,
    setThemeMode,
    setAccentTheme,
    cycleThemeMode,
  }
}

function setAppearance({ mode, theme, accent, accentTheme: nextAccent } = {}) {
  if (mode !== undefined || theme !== undefined) {
    themeMode.value = normalizeThemeMode(mode ?? theme)
  }
  if (accent !== undefined || nextAccent !== undefined) {
    accentTheme.value = normalizeAccentTheme(accent ?? nextAccent)
  }
  applyAppearance()
}

function setThemeMode(value) {
  themeMode.value = normalizeThemeMode(value)
  applyAppearance()
}

function setAccentTheme(value) {
  accentTheme.value = normalizeAccentTheme(value)
  applyAppearance()
}

function cycleThemeMode() {
  const order = THEME_MODES.map((mode) => mode.id)
  const currentIndex = order.indexOf(themeMode.value)
  setThemeMode(order[(currentIndex + 1) % order.length] || DEFAULT_APPEARANCE.mode)
}

function applyAppearance() {
  if (typeof document === 'undefined') return

  const normalizedMode = normalizeThemeMode(themeMode.value)
  const normalizedAccent = normalizeAccentTheme(accentTheme.value)
  const resolved = normalizedMode === 'system'
    ? (mediaQuery?.matches ? 'dark' : 'light')
    : normalizedMode

  themeMode.value = normalizedMode
  accentTheme.value = normalizedAccent
  resolvedThemeMode.value = resolved

  const root = document.documentElement
  root.classList.toggle('dark', resolved === 'dark')
  root.dataset.themePreference = normalizedMode
  root.dataset.themeMode = resolved
  root.dataset.themeAccent = normalizedAccent
  root.style.colorScheme = resolved

  localStorage.setItem(THEME_STORAGE_KEYS.mode, normalizedMode)
  localStorage.setItem(THEME_STORAGE_KEYS.accent, normalizedAccent)
}

function readStoredValue(key, fallback) {
  if (typeof localStorage === 'undefined') return fallback
  return localStorage.getItem(key) || fallback
}
