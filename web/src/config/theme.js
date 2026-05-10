export const THEME_STORAGE_KEYS = {
  mode: 'easyllm_theme_mode',
  accent: 'easyllm_theme_accent',
}

export const THEME_MODES = [
  { id: 'system', label: '跟随系统', shortLabel: '系统' },
  { id: 'light', label: '白天模式', shortLabel: '白天' },
  { id: 'dark', label: '黑夜模式', shortLabel: '黑夜' },
]

export const ACCENT_THEMES = [
  { id: 'blue', label: 'Apple 蓝', swatch: '#0A84FF' },
  { id: 'purple', label: '紫色', swatch: '#AF52DE' },
  { id: 'green', label: '绿色', swatch: '#34C759' },
  { id: 'orange', label: '橙色', swatch: '#FF9F0A' },
  { id: 'graphite', label: '石墨', swatch: '#8E8E93' },
]

export const DEFAULT_APPEARANCE = {
  mode: 'system',
  accent: 'blue',
}

export function normalizeThemeMode(value) {
  return THEME_MODES.some((mode) => mode.id === value) ? value : DEFAULT_APPEARANCE.mode
}

export function normalizeAccentTheme(value) {
  return ACCENT_THEMES.some((theme) => theme.id === value) ? value : DEFAULT_APPEARANCE.accent
}

export function getThemeModeLabel(value) {
  return THEME_MODES.find((mode) => mode.id === value)?.label || THEME_MODES[0].label
}

export function getThemeModeShortLabel(value) {
  return THEME_MODES.find((mode) => mode.id === value)?.shortLabel || THEME_MODES[0].shortLabel
}

export function getAccentThemeLabel(value) {
  return ACCENT_THEMES.find((theme) => theme.id === value)?.label || ACCENT_THEMES[0].label
}
