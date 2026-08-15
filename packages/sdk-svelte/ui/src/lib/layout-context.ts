import { getContext } from 'svelte'
import type { MenuItem } from './types'

export const LAYOUT_CTX = Symbol('kg-layout')
export const MENU_CTX = Symbol('kg-menu')

export type MenuContext = {
  /** Inferred app name (`settings`, `hellospec`, …). */
  app: () => string
  /** Menu tree finished loading (even if empty). */
  ready: () => boolean
  /** Load / GraphQL error, if any. */
  error: () => string
  /** Active leaf menu id. */
  active: () => string
  /** Active leaf menu item. */
  selected: () => MenuItem | null
  /** True when the app exposes menus. */
  hasMenus: () => boolean
}

export type MenuState = {
  app: string
  /** Menu query finished (may be empty for apps without menus). */
  ready: boolean
  error: string
  active: string
  selected: MenuItem | null
  /** True when the app has at least one menu leaf. */
  hasMenus: boolean
}

export type LayoutRegistry = {
  setActions: (snippet: import('svelte').Snippet | null) => void
  setAlerts: (snippet: import('svelte').Snippet | null) => void
  /** Full-width top nav (menubar / tabs). */
  setNav: (snippet: import('svelte').Snippet | null) => void
  /** @deprecated use setNav */
  setTabs: (snippet: import('svelte').Snippet | null) => void
  setMain: (snippet: import('svelte').Snippet | null) => void
  /** LayoutMenu writes selection/load state here for MenuOutlet. */
  setMenuState: (state: MenuState) => void
}

export function getLayoutRegistry(): LayoutRegistry {
  return getContext(LAYOUT_CTX)
}

export function getMenuContext(): MenuContext {
  const ctx = getContext<MenuContext | undefined>(MENU_CTX)
  if (!ctx) {
    throw new Error('getMenuContext() requires <Layout> with <LayoutMenu>')
  }
  return ctx
}

/** Resolve current app id from mount host or `/app/{name}` route. */
export function inferAppName(el?: HTMLElement | null): string {
  const host = el?.closest?.('[data-kg-app]') as HTMLElement | null
  const fromHost = host?.dataset?.kgApp?.trim()
  if (fromHost) return fromHost

  const path = window.location.pathname.replace(/\/+$/, '')
  const m = path.match(/\/app\/([^/]+)/)
  if (m?.[1]) return m[1]

  const asset = document.querySelector('link[data-app-css], script[data-app-js]') as HTMLElement | null
  const fromAsset = asset?.getAttribute('data-app-css') || asset?.getAttribute('data-app-js')
  if (fromAsset) return fromAsset

  return ''
}
