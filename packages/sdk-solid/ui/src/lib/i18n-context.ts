import { createContext, useContext } from 'solid-js'
import type { Translator } from './i18n'
import { bundledTranslator, t as translate } from './i18n-runtime'

export type I18nScope = {
  app: string
  translator: Translator
  /** Translate a namespaced catalog key (`hellospec.ping`, `identity.users.title`, …). */
  t: (key: string, ...args: Array<string | number>) => string
}

export const I18nContext = createContext<I18nScope>()

/** I18n scope for the current component tree. */
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

/** Translate a namespaced key using the compiled catalog. */
export function t(key: string, ...args: Array<string | number>): string {
  return translate(key, ...args)
}
