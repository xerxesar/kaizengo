import { createContext, useContext } from 'react'
import type { KFormContext } from './kform-types'

export const KFormCtx = createContext<KFormContext | null>(null)

export function getKFormContext(): KFormContext {
  const ctx = useContext(KFormCtx)
  if (!ctx) throw new Error('getKFormContext() requires <KForm>')
  return ctx
}

export function useKForm(): KFormContext {
  return getKFormContext()
}
