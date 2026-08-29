import { createContext, useContext } from 'solid-js'
import type { KFormContext as KFormContextValue } from './kform-types'

export const KFormCtx = createContext<KFormContextValue>()

export function getKFormContext(): KFormContextValue {
  const ctx = useContext(KFormCtx)
  if (!ctx) {
    throw new Error('getKFormContext() requires <KForm>')
  }
  return ctx
}
