export type KeymapScope = 'global' | 'app' | 'view'

export type KeymapBinding = {
  id: string
  app: string
  action: string
  keys: string
  label?: string
  labelKey?: string
  scope: KeymapScope
  hint: boolean
  inForm?: boolean
}

export type KeymapCatalog = {
  disable: string[]
  bindings: KeymapBinding[]
}

/** DOM attribute for element-bound shortcuts (action: element:<id>). */
export const KEYMAP_ID_ATTR = 'data-keymap-id'

export type KeymapOverrides = Record<string, string>

export type KeymapContextValue = {
  catalog: () => KeymapCatalog | null
  bindings: () => KeymapBinding[]
  effectiveKeys: (id: string, fallback: string) => string
  overrides: () => KeymapOverrides
  setOverride: (id: string, keys: string) => void
  clearOverride: (id: string) => void
  clearAllOverrides: () => void
  hintsVisible: () => boolean
}
