import { type ParentProps } from 'solid-js'
import { I18nContext, type I18nScope } from './i18n-context'
import { bundledTranslator, t } from './i18n-runtime'

type Props = ParentProps & {
  app?: string
}

export function I18nProvider(props: Props) {
  const scope: I18nScope = {
    app: props.app ?? '',
    get translator() {
      return bundledTranslator()
    },
    t,
  }
  return <I18nContext.Provider value={scope}>{props.children}</I18nContext.Provider>
}
