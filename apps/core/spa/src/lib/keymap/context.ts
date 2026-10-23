import { createContext, useContext } from 'react'
import type { KeymapContextValue } from './types'

export const KeymapContext = createContext<KeymapContextValue | null>(null)

export function useKeymap(): KeymapContextValue {
  const ctx = useContext(KeymapContext)
  if (!ctx) throw new Error('useKeymap must be used within KeymapProvider')
  return ctx
}
