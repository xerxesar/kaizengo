import { createContext, useContext } from 'react'
import type { Translator } from './i18n'
import { bundledTranslator, t as translate } from './i18n-runtime'

export type I18nScope = {
  app: string
  translator: Translator
  t: (key: string, ...args: Array<string | number>) => string
}

export const I18nContext = createContext<I18nScope | null>(null)

export function getI18n(): I18nScope {
  const ctx = useContext(I18nContext)
  if (ctx) return ctx
  return {
    app: '',
    get translator() {
      return bundledTranslator()
    },
    t: translate,
  }
}

export function useI18n(): I18nScope {
  return getI18n()
}

export function t(key: string, ...args: Array<string | number>): string {
  return translate(key, ...args)
}
