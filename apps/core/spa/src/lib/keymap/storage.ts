import type { KeymapOverrides } from './types'

const STORAGE_KEY = 'kg-keymap'

export function getKeymapOverrides(): KeymapOverrides {
  try {
    const raw = localStorage.getItem(STORAGE_KEY)
    if (!raw) return {}
    const parsed = JSON.parse(raw) as unknown
    if (!parsed || typeof parsed !== 'object') return {}
    const out: KeymapOverrides = {}
    for (const [id, keys] of Object.entries(parsed as Record<string, unknown>)) {
      if (typeof keys === 'string' && keys.trim()) out[id] = keys.trim()
    }
    return out
  } catch {
    return {}
  }
}

function save(overrides: KeymapOverrides) {
  try {
    localStorage.setItem(STORAGE_KEY, JSON.stringify(overrides))
  } catch {
    /* private browsing */
  }
  window.dispatchEvent(new CustomEvent('kaizengo:keymap', { detail: { overrides } }))
}

export function setKeymapOverride(id: string, keys: string) {
  const next = { ...getKeymapOverrides(), [id]: keys.trim() }
  save(next)
}

export function clearKeymapOverride(id: string) {
  const next = { ...getKeymapOverrides() }
  delete next[id]
  save(next)
}

export function clearAllKeymapOverrides() {
  save({})
}
