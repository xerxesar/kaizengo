import { render } from 'solid-js/web'
import { initTheme, setI18nLocale, syncDocumentLocale } from '@kaizengo/sdk-solid/ui'
import '@kaizengo/sdk-solid/ui/styles.css'
import App from './App'
import './app.css'

initTheme('carbon')
void syncDocumentLocale().then(({ locale }) => setI18nLocale(locale))

const root = document.getElementById('app')
if (root) {
  render(() => <App />, root)
}

if (import.meta.hot) {
  import.meta.hot.on('kaizengo:app-updated', () => {
    window.dispatchEvent(new Event('kaizengo:app-updated'))
  })
}
