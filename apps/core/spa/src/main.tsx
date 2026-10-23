import { createRoot } from 'react-dom/client'
import { initTheme, setI18nLocale, syncDocumentLocale } from '@/lib'
import '@/styles/index.css'
import '@/styles/app.css'
import App from './App'

initTheme('carbon')
void syncDocumentLocale().then(({ locale }) => setI18nLocale(locale))

const root = document.getElementById('app')
if (root) {
  createRoot(root).render(<App />)
}

if (import.meta.hot) {
  import.meta.hot.on('kaizengo:app-updated', () => {
    window.dispatchEvent(new Event('kaizengo:app-updated'))
  })
}
