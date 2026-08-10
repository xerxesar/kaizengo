import { mount, unmount, type Component } from 'svelte'
import Counter from './Counter.svelte'

type Mounted = Record<string, unknown>

let instance: Mounted | null = null
let cssLink: HTMLLinkElement | null = null

function ensureCSS() {
  if (document.querySelector('link[data-app-css="counter"]')) return
  cssLink = document.createElement('link')
  cssLink.rel = 'stylesheet'
  cssLink.href = '/app-assets/counter/spa.css'
  cssLink.dataset.appCss = 'counter'
  document.head.appendChild(cssLink)
}

export default {
  async mount(el: HTMLElement) {
    ensureCSS()
    instance = mount(Counter as Component, { target: el }) as Mounted
  },
  unmount() {
    if (instance) {
      unmount(instance)
      instance = null
    }
  },
}
