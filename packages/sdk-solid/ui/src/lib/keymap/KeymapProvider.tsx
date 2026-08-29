import { createMemo, createSignal, onCleanup, onMount, Show, type ParentProps } from 'solid-js'
import { Portal } from 'solid-js/web'
import { useHotkeys, useFormatHotkey, useHotkeyStore } from '@ark-ui/solid'
import { currentAppRoute } from '../menu-route'
import { t } from '../i18n-context'
import { fetchKeymap } from './client'
import { runKeymapAction } from './actions'
import { KeymapContext } from './context'
import { KeymapHints } from './KeymapHints'
import { keymapHotkeyStore } from './store'
import {
  clearAllKeymapOverrides,
  clearKeymapOverride,
  getKeymapOverrides,
  setKeymapOverride,
} from './storage'
import type { KeymapBinding, KeymapCatalog, KeymapScope } from './types'

function scopeActive(scope: KeymapScope, route: string, app: string): boolean {
  switch (scope) {
    case 'global':
      return true
    case 'app':
    case 'view':
      return route.length > 0 && route === app
    default:
      return true
  }
}

function isEditableTarget(target: EventTarget | null): boolean {
  if (!(target instanceof HTMLElement)) return false
  return !!target.closest('input, textarea, select, [contenteditable=true]')
}

function labelFor(binding: KeymapBinding): string {
  if (binding.labelKey) {
    const translated = t(binding.labelKey)
    if (translated !== binding.labelKey) return translated
  }
  return binding.label || binding.id
}

function isAltKey(event: KeyboardEvent): boolean {
  return event.key === 'Alt' || event.key === 'AltGraph'
}

function altActive(event: KeyboardEvent): boolean {
  if (isEditableTarget(event.target)) return false
  return isAltKey(event) || event.altKey || event.getModifierState('Alt')
}

export function KeymapProvider(props: ParentProps) {
  const [catalog, setCatalog] = createSignal<KeymapCatalog | null>(null)
  const [overrides, setOverrides] = createSignal(getKeymapOverrides())
  const [hintsVisible, setHintsVisible] = createSignal(false)
  const [recordingPaused, setRecordingPaused] = createSignal(false)

  const formatHotkey = useFormatHotkey()
  useHotkeyStore({ store: keymapHotkeyStore })

  const bindings = createMemo(() => {
    const cat = catalog()
    if (!cat) return [] as KeymapBinding[]
    const disabled = new Set(cat.disable)
    return cat.bindings.filter((b) => !disabled.has(b.id))
  })

  function effectiveKeys(id: string, fallback: string) {
    return overrides()[id] ?? fallback
  }

  function reloadOverrides() {
    setOverrides(getKeymapOverrides())
  }

  onMount(() => {
    let hideTimer: number | undefined

    const onKeymap = () => reloadOverrides()
    const onRecording = (e: Event) => {
      setRecordingPaused((e as CustomEvent<{ active?: boolean }>).detail?.active ?? false)
    }
    const onKeyDown = (e: KeyboardEvent) => {
      if (hideTimer !== undefined) {
        window.clearTimeout(hideTimer)
        hideTimer = undefined
      }
      if (altActive(e)) setHintsVisible(true)
    }
    const onKeyUp = (e: KeyboardEvent) => {
      if (isAltKey(e) || !e.getModifierState('Alt')) {
        hideTimer = window.setTimeout(() => {
          hideTimer = undefined
          setHintsVisible(false)
        }, 150)
      }
    }
    const hideOnHide = () => {
      if (document.hidden) setHintsVisible(false)
    }

    window.addEventListener('kaizengo:keymap', onKeymap)
    window.addEventListener('kaizengo:keymap-recording', onRecording)
    window.addEventListener('keydown', onKeyDown, true)
    window.addEventListener('keyup', onKeyUp, true)
    document.addEventListener('visibilitychange', hideOnHide)

    void fetchKeymap()
      .then(setCatalog)
      .catch(() => setCatalog({ disable: [], bindings: [] }))

    onCleanup(() => {
      if (hideTimer !== undefined) window.clearTimeout(hideTimer)
      window.removeEventListener('kaizengo:keymap', onKeymap)
      window.removeEventListener('kaizengo:keymap-recording', onRecording)
      window.removeEventListener('keydown', onKeyDown, true)
      window.removeEventListener('keyup', onKeyUp, true)
      document.removeEventListener('visibilitychange', hideOnHide)
    })
  })

  useHotkeys(() => ({
    store: keymapHotkeyStore,
    commands: bindings().map((binding) => ({
      id: binding.id,
      hotkey: effectiveKeys(binding.id, binding.keys),
      label: labelFor(binding),
      category: binding.app,
      scopes: ['*'],
      enabled: () =>
        !recordingPaused() &&
        scopeActive(binding.scope, currentAppRoute(), binding.app),
      options: {
        preventDefault: true,
        enableOnFormTags: binding.inForm ? true : false,
      },
      action: () => runKeymapAction(binding.action),
    })),
  }))

  const ctx = {
    catalog,
    bindings,
    effectiveKeys,
    overrides,
    hintsVisible,
    setOverride: (id: string, keys: string) => {
      setKeymapOverride(id, keys)
      reloadOverrides()
    },
    clearOverride: (id: string) => {
      clearKeymapOverride(id)
      reloadOverrides()
    },
    clearAllOverrides: () => {
      clearAllKeymapOverrides()
      reloadOverrides()
    },
  }

  return (
    <KeymapContext.Provider value={ctx}>
      {props.children}
      <Show when={hintsVisible() && bindings().length}>
        <Portal mount={document.body}>
          <KeymapHints bindings={bindings} effectiveKeys={effectiveKeys} formatHotkey={formatHotkey} />
        </Portal>
      </Show>
    </KeymapContext.Provider>
  )
}

export { labelFor as keymapLabelFor }
