import type { ReactNode } from 'react'
import { I18nContext, type I18nScope } from './i18n-context'
import { bundledTranslator, t } from './i18n-runtime'

type Props = {
  app?: string
  children?: ReactNode
}

export function I18nProvider({ app = '', children }: Props) {
  const scope: I18nScope = {
    app,
    get translator() {
      return bundledTranslator()
    },
    t,
  }
  return <I18nContext.Provider value={scope}>{children}</I18nContext.Provider>
}
