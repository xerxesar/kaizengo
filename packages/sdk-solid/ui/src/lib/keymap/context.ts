import { createContext, useContext } from 'solid-js'
import type { KeymapContextValue } from './types'

export const KeymapContext = createContext<KeymapContextValue>()

export function useKeymap(): KeymapContextValue {
  const ctx = useContext(KeymapContext)
  if (!ctx) throw new Error('useKeymap must be used within KeymapProvider')
  return ctx
}
