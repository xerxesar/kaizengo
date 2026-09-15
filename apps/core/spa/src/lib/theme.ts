export type ThemeId = 'carbon' | 'carbon-dark' | 'kaizen'
export type ThemeMode = 'light' | 'dark'

export type ThemeDef = {
  id: ThemeId
  label: string
  mode: ThemeMode
}

export const THEMES: ThemeDef[] = [
  { id: 'carbon', label: 'Carbon (Light)', mode: 'light' },
  { id: 'carbon-dark', label: 'Carbon (Dark)', mode: 'dark' },
  { id: 'kaizen', label: 'KaizenGo Brand', mode: 'light' },
]

const STORAGE_KEY = 'kg-theme'
const VALID: ThemeId[] = THEMES.map((t) => t.id)

function isThemeId(v: string | undefined): v is ThemeId {
  return VALID.includes(v as ThemeId)
}

export function getThemeDef(id: ThemeId = getTheme()): ThemeDef {
  return THEMES.find((t) => t.id === id) ?? THEMES[0]
}

export function getThemeMode(id: ThemeId = getTheme()): ThemeMode {
  return getThemeDef(id).mode
}

export function themeIconHref(mode: ThemeMode = getThemeMode()): string {
  return mode === 'dark' ? '/static/icon-mono-dark.png' : '/static/icon-mono.png'
}

function applyFavicon(mode: ThemeMode) {
  const href = themeIconHref(mode)
  document.querySelectorAll('link[rel="icon"]').forEach((el) => el.remove())
  const link = document.createElement('link')
  link.rel = 'icon'
  link.type = 'image/png'
  link.href = href
  document.head.appendChild(link)
}

/** Apply a theme globally (html[data-kg-theme]). */
export function setTheme(id: ThemeId) {
  const mode = getThemeMode(id)
  document.documentElement.dataset.kgTheme = id
  document.documentElement.dataset.kgThemeMode = mode
  applyFavicon(mode)
  try {
    localStorage.setItem(STORAGE_KEY, id)
  } catch {
    /* private browsing */
  }
  window.dispatchEvent(new CustomEvent('kaizengo:theme', { detail: { theme: id } }))
}

export function getTheme(): ThemeId {
  const fromDom = document.documentElement.dataset.kgTheme
  if (isThemeId(fromDom)) return fromDom
  try {
    const saved = localStorage.getItem(STORAGE_KEY)
    if (isThemeId(saved ?? undefined)) return saved as ThemeId
  } catch {
    /* ignore */
  }
  return 'carbon'
}

/** Call once at app startup. Defaults to IBM Carbon light theme. */
export function initTheme(preferred: ThemeId = 'carbon') {
  const theme = getTheme() || preferred
  setTheme(theme)
}
