import type { MenuItem } from './types'

/** Context core passes when mounting an app SPA as a view host. */
export type SpaMountContext = {
  /** App whose URL/menus are active (e.g. `settings`). */
  hostApp: string
  /** Bundle being mounted (`sourceApp` or `hostApp`). */
  app: string
  /** View name from the selected menu leaf. */
  view?: string
  /** Optional exported component id (cross-app contributions). */
  component?: string
  /** URL page segment. */
  page?: string
}

/**
 * Which app bundle should render this menu leaf.
 * Contributed items use `sourceApp`; local items use the host app.
 */
export function contentAppForMenu(item: MenuItem | null | undefined, hostApp: string): string {
  const fromItem = item?.sourceApp?.trim()
  if (fromItem) return fromItem
  return hostApp.trim()
}

/** Default ESM URL for an app SPA bundle. */
export function appModuleUrl(app: string): string {
  const name = app.trim()
  if (!name) return ''
  return `/app-assets/${name}/spa.js`
}
