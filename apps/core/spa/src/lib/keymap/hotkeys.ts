import { useEffect, useMemo, useRef, useState } from 'react'
import {
  createHotkeyStore,
  formatHotkey,
  normalizeHotkey,
  type HotkeyStore,
} from '@zag-js/hotkeys'

export type UseFormatHotkeyReturn = (
  hotkey: string,
  options?: Parameters<typeof formatHotkey>[1],
) => string

export type UseHotkeysCommand = {
  id?: string
  hotkey: string
  label?: string
  description?: string
  category?: string
  keywords?: string[]
  scopes?: string[]
  options?: Record<string, unknown>
  enabled?: boolean | (() => boolean)
  action: (event: KeyboardEvent) => void
}

export type UseHotkeysProps = {
  id?: string
  store?: HotkeyStore
  commands: UseHotkeysCommand[]
}

export type UseHotkeyStoreProps = {
  store?: HotkeyStore
}

function detectPlatform(): 'mac' | 'linux' | 'windows' {
  if (typeof navigator === 'undefined') return 'windows'
  const platform = navigator.platform || ''
  if (/Mac|iPhone|iPad|iPod/i.test(platform)) return 'mac'
  if (/Linux|CrOS/i.test(platform)) return 'linux'
  return 'windows'
}

export function usePlatform() {
  const [platform, setPlatform] = useState<'mac' | 'linux' | 'windows'>('windows')
  useEffect(() => setPlatform(detectPlatform()), [])
  return platform
}

export function useFormatHotkey(): UseFormatHotkeyReturn {
  const platform = usePlatform()
  return (hotkey, options = {}) => formatHotkey(hotkey, { platform, ...options })
}

let defaultHotkeyStore: HotkeyStore | undefined
const initialized = new WeakSet<HotkeyStore>()

function getDefaultHotkeyStore() {
  defaultHotkeyStore ??= createHotkeyStore()
  return defaultHotkeyStore
}

export function useHotkeyStore(props: UseHotkeyStoreProps = {}): HotkeyStore {
  const store = props.store ?? getDefaultHotkeyStore()
  useEffect(() => {
    if (initialized.has(store)) return
    initialized.add(store)
    store.init({ target: document })
  }, [store])
  return store
}

function shallowEqualRegistration(
  a: { hotkey: string; scopes?: string[]; label?: string; options?: unknown },
  b: { hotkey: string; scopes?: string[]; label?: string; options?: unknown },
) {
  return (
    a.hotkey === b.hotkey &&
    a.label === b.label &&
    JSON.stringify(a.scopes ?? []) === JSON.stringify(b.scopes ?? []) &&
    JSON.stringify(a.options ?? {}) === JSON.stringify(b.options ?? {})
  )
}

let hotkeyInstanceCounter = 0

export function useHotkeys(props: UseHotkeysProps | (() => UseHotkeysProps)) {
  const resolved = typeof props === 'function' ? props() : props
  const store = useHotkeyStore({ store: resolved.store })
  const platform = usePlatform()
  const instanceId = useRef(resolved.id ?? `hk-${++hotkeyInstanceCounter}`).current
  const commandsRef = useRef(resolved.commands)
  commandsRef.current = resolved.commands
  const registered = useRef(
    new Map<string, { hotkey: string; scopes?: string[]; label?: string; options?: unknown }>(),
  )

  const resolveId = (command: UseHotkeysCommand, index: number) =>
    command.id ?? `${instanceId}:${index}`

  useEffect(() => {
    const current = commandsRef.current
    const nextIds = new Set(current.map(resolveId))

    for (const id of [...registered.current.keys()]) {
      if (nextIds.has(id)) continue
      store.unregister(id)
      registered.current.delete(id)
    }

    current.forEach((command, index) => {
      const id = resolveId(command, index)
      const next = {
        hotkey: normalizeHotkey(command.hotkey, platform),
        scopes: command.scopes,
        label: command.label,
        options: command.options,
      }
      const previous = registered.current.get(id)
      if (previous && shallowEqualRegistration(previous, next)) return
      if (previous) store.unregister(id)

      store.register({
        ...command,
        id,
        hotkey: next.hotkey,
        action: (event: KeyboardEvent) => {
          const cmd = commandsRef.current.find((item, i) => resolveId(item, i) === id)
          cmd?.action(event)
        },
        enabled: () => {
          const cmd = commandsRef.current.find((item, i) => resolveId(item, i) === id)
          const enabled = cmd?.enabled
          if (enabled === undefined) return true
          return typeof enabled === 'function' ? enabled() : enabled
        },
      })
      registered.current.set(id, next)
    })
  }, [store, platform, resolved.commands])

  useEffect(
    () => () => {
      for (const id of registered.current.keys()) store.unregister(id)
      registered.current.clear()
    },
    [store],
  )
}

export { createHotkeyStore, formatHotkey, normalizeHotkey }
export type { HotkeyStore } from '@zag-js/hotkeys'
