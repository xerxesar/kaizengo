import { navigateApp } from '../menu-route'
import { KEYMAP_ID_ATTR } from './types'
import { runCustomAction } from './registry'

export function runKeymapAction(action: string): void {
  if (action.startsWith('element:')) {
    const id = action.slice('element:'.length)
    if (!id) return
    const el = document.querySelector(`[${KEYMAP_ID_ATTR}="${CSS.escape(id)}"]`) as HTMLElement | null
    if (!el || el.hasAttribute('disabled') || el.getAttribute('aria-disabled') === 'true') return
    el.click()
    return
  }

  if (action.startsWith('custom:')) {
    runCustomAction(action.slice('custom:'.length))
    return
  }

  if (action.startsWith('nav.')) {
    const route = action.slice('nav.'.length).replace(/^\/+/, '')
    navigateApp(route ? `/app/${route}` : '/app/')
    return
  }

  switch (action) {
    case 'shell.toggleApps':
      window.dispatchEvent(new CustomEvent('kaizengo:shell.toggleApps'))
      break
    case 'shell.goHome':
      navigateApp('/app/')
      break
    case 'shell.signOut':
      window.dispatchEvent(new CustomEvent('kaizengo:shell.signOut'))
      break
    default:
      break
  }
}

export function elementIdFromAction(action: string): string | null {
  if (!action.startsWith('element:')) return null
  const id = action.slice('element:'.length)
  return id || null
}
