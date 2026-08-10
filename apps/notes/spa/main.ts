import { mount, unmount, type Component } from 'svelte'
import App from './App.svelte'

type Mounted = Record<string, unknown>

let instance: Mounted | null = null

function ensureCSS() {
  if (document.querySelector('link[data-app-css="notes"]')) return
  const cssLink = document.createElement('link')
  cssLink.rel = 'stylesheet'
  cssLink.href = '/app-assets/notes/spa.css'
  cssLink.dataset.appCss = 'notes'
  document.head.appendChild(cssLink)
}

export default {
  async mount(el: HTMLElement) {
    ensureCSS()
    instance = mount(App as Component, { target: el }) as Mounted
  },
  unmount() {
    if (instance) {
      unmount(instance)
      instance = null
    }
  },
}
