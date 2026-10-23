import type { KeymapCatalog } from './types'

export async function fetchKeymap(): Promise<KeymapCatalog> {
  const res = await fetch('/api/keymap', { credentials: 'include' })
  if (!res.ok) throw new Error(`Failed to load keymap (${res.status})`)
  return res.json() as Promise<KeymapCatalog>
}
