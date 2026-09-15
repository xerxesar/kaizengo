import { createContext, useContext, type ReactNode } from 'react'
import type { MenuItem } from './types'

export type LayoutSlot = ReactNode

export type MenuState = {
  app: string
  ready: boolean
  error: string
  active: string
  selected: MenuItem | null
  hasMenus: boolean
}

export type LayoutRegistry = {
  setActions: (slot: LayoutSlot | null) => void
  setAlerts: (slot: LayoutSlot | null) => void
  setNav: (slot: LayoutSlot | null) => void
  /** @deprecated use setNav */
  setTabs: (slot: LayoutSlot | null) => void
  setMain: (slot: LayoutSlot | null) => void
  setMenuState: (state: MenuState) => void
}

export const LayoutContext = createContext<LayoutRegistry | null>(null)
export const MenuContext = createContext<MenuState | null>(null)

export function getLayoutRegistry(): LayoutRegistry {
  const ctx = useContext(LayoutContext)
  if (!ctx) throw new Error('getLayoutRegistry() requires <Layout>')
  return ctx
}

export function useLayoutRegistry(): LayoutRegistry {
  return getLayoutRegistry()
}

export function getMenuContext(): MenuState {
  const ctx = useContext(MenuContext)
  if (!ctx) throw new Error('getMenuContext() requires <Layout> with <LayoutMenu>')
  return ctx
}

export function useMenuContext(): MenuState {
  return getMenuContext()
}

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
