import { useEffect, useMemo, useState, type ReactNode } from 'react'
import { createPortal } from 'react-dom'
import { currentAppRoute } from '@/lib/menu-route'
import { t } from '@/lib/i18n-context'
import { fetchKeymap } from './client'
import { runKeymapAction } from './actions'
import { KeymapContext } from './context'
import { KeymapHints } from './KeymapHints'
import { useFormatHotkey, useHotkeys, useHotkeyStore } from './hotkeys'
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

export function KeymapProvider({ children }: { children?: ReactNode }) {
  const [catalog, setCatalog] = useState<KeymapCatalog | null>(null)
  const [overrides, setOverrides] = useState(getKeymapOverrides())
  const [hintsVisible, setHintsVisible] = useState(false)
  const [recordingPaused, setRecordingPaused] = useState(false)

  const formatHotkey = useFormatHotkey()
  useHotkeyStore({ store: keymapHotkeyStore })

  const bindings = useMemo(() => {
    if (!catalog) return [] as KeymapBinding[]
    const disabled = new Set(catalog.disable)
    return catalog.bindings.filter((b) => !disabled.has(b.id))
  }, [catalog])

  function effectiveKeys(id: string, fallback: string) {
    return overrides[id] ?? fallback
  }

  function reloadOverrides() {
    setOverrides(getKeymapOverrides())
  }

  useEffect(() => {
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

    return () => {
      if (hideTimer !== undefined) window.clearTimeout(hideTimer)
      window.removeEventListener('kaizengo:keymap', onKeymap)
      window.removeEventListener('kaizengo:keymap-recording', onRecording)
      window.removeEventListener('keydown', onKeyDown, true)
      window.removeEventListener('keyup', onKeyUp, true)
      document.removeEventListener('visibilitychange', hideOnHide)
    }
  }, [])

  useHotkeys({
    store: keymapHotkeyStore,
    commands: bindings.map((binding) => ({
      id: binding.id,
      hotkey: effectiveKeys(binding.id, binding.keys),
      label: labelFor(binding),
      category: binding.app,
      scopes: ['*'],
      enabled: () =>
        !recordingPaused && scopeActive(binding.scope, currentAppRoute(), binding.app),
      options: {
        preventDefault: true,
        enableOnFormTags: binding.inForm ? true : false,
      },
      action: () => runKeymapAction(binding.action),
    })),
  })

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
      {children}
      {hintsVisible && bindings.length
        ? createPortal(
            <KeymapHints
              bindings={bindings}
              effectiveKeys={effectiveKeys}
              formatHotkey={formatHotkey}
            />,
            document.body,
          )
        : null}
    </KeymapContext.Provider>
  )
}

export { labelFor as keymapLabelFor }
