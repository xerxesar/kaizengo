import { mount } from 'svelte'
import { initTheme, setI18nLocale, syncDocumentLocale } from '@kaizengo/sdk-svelte/ui'
import '@kaizengo/sdk-svelte/ui/styles.css'
import App from './App.svelte'
import './app.css'

initTheme('carbon')
void syncDocumentLocale().then(({ locale }) => setI18nLocale(locale))

mount(App, { target: document.getElementById('app')! })

if (import.meta.hot) {
  import.meta.hot.on('kaizengo:app-updated', () => {
    window.dispatchEvent(new Event('kaizengo:app-updated'))
  })
}
