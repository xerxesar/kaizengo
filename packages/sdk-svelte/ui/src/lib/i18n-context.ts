import { getContext } from 'svelte'
import type { Translator } from './i18n'
import { bundledTranslator, t as translate } from './i18n-runtime.svelte'

export const I18N_CTX = Symbol('kg-i18n')

export type I18nScope = {
  app: string
  translator: Translator
  /** Translate a namespaced catalog key (`hellospec.ping`, `identity.users.title`, …). */
  t: (key: string, ...args: Array<string | number>) => string
}

/** I18n scope for the current component tree — call once during component init. */
export function getI18n(): I18nScope {
  return (
    getContext<I18nScope>(I18N_CTX) ?? {
      app: '',
      get translator() {
        return bundledTranslator()
      },
      t: translate,
    }
  )
}

/** Translate a namespaced key; safe in templates and `$derived` (reads the compiled catalog). */
export function t(key: string, ...args: Array<string | number>): string {
  return translate(key, ...args)
}
