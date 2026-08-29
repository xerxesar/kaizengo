import type { MenuItem } from './types'
import { flattenMenuItems, firstMenuLeaf } from './menu'

const APP_BASE = '/app'

/** Path segment used for a leaf menu item (`route`, else `id`). */
export function menuRouteOf(item: Pick<MenuItem, 'route' | 'id'>): string {
  const route = (item.route || item.id || '').trim()
  return route.replace(/^\/+|\/+$/g, '')
}

/** `/app/{app}` */
export function appPath(app: string): string {
  const name = app.trim().replace(/^\/+|\/+$/g, '')
  return name ? `${APP_BASE}/${name}` : `${APP_BASE}/`
}

/** `/app/{app}/{page}` */
export function menuPagePath(app: string, page: string): string {
  const base = appPath(app).replace(/\/$/, '')
  const seg = page.trim().replace(/^\/+|\/+$/g, '')
  return seg ? `${base}/${seg}` : base
}

export function menuItemHref(app: string, item: MenuItem): string {
  return menuPagePath(app, menuRouteOf(item))
}

/** First path segment after `/app/` (the mounted app). */
export function currentAppRoute(pathname = window.location.pathname): string {
  const path = pathname.replace(/\/+$/, '')
  const m = path.match(/^\/app\/([^/]+)/)
  return m?.[1] ?? ''
}

/** Remaining path after `/app/{app}/` (in-app page). */
export function currentMenuPage(pathname = window.location.pathname): string {
  const path = pathname.replace(/\/+$/, '')
  const m = path.match(/^\/app\/[^/]+\/(.+)$/)
  return m?.[1] ?? ''
}

export function findMenuByRoute(items: MenuItem[], page: string): MenuItem | null {
  const want = page.trim().replace(/^\/+|\/+$/g, '')
  if (!want) return null
  for (const leaf of flattenMenuItems(items)) {
    if (menuRouteOf(leaf) === want) return leaf
    if (leaf.id === want) return leaf
    if (leaf.view && leaf.view === want) return leaf
  }
  return null
}

/** Resolve active leaf from URL, or first leaf when page is empty. */
export function resolveMenuSelection(items: MenuItem[], page = currentMenuPage()): MenuItem | null {
  if (!items.length) return null
  if (page) {
    return findMenuByRoute(items, page) ?? firstMenuLeaf(items) ?? null
  }
  return firstMenuLeaf(items) ?? null
}

/**
 * Navigate within the shell without a full reload.
 * Updates history and notifies listeners (shell + LayoutMenu).
 */
export function navigateApp(path: string, replace = false): void {
  const url = path.startsWith('/') ? path : `${APP_BASE}/${path}`
  const current = window.location.pathname + window.location.search + window.location.hash
  if (url === current || url === window.location.pathname) return
  if (replace) history.replaceState({}, '', url)
  else history.pushState({}, '', url)
  window.dispatchEvent(new PopStateEvent('popstate'))
  window.dispatchEvent(new CustomEvent('kaizengo:location'))
}
