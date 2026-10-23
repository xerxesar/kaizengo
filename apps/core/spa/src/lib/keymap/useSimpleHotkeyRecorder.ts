import { useEffect, useRef, useState } from 'react'
import { formatHotkey, normalizeHotkey } from '@zag-js/hotkeys'

export type RecordedHotkey = {
  value: string
  display: string
}

type RecorderState = {
  recording: boolean
  value: RecordedHotkey | null
}

const MODIFIER_KEYS = new Set(['Control', 'Alt', 'Shift', 'Meta'])

function chordFromEvent(event: KeyboardEvent): string {
  const parts: string[] = []
  if (event.ctrlKey) parts.push('Control')
  if (event.altKey) parts.push('Alt')
  if (event.shiftKey) parts.push('Shift')
  if (event.metaKey) parts.push('Meta')
  if (!MODIFIER_KEYS.has(event.key) && event.key !== 'Dead') {
    parts.push(event.key.length === 1 ? event.key.toUpperCase() : event.key)
  }
  if (parts.length === 0) return ''
  return normalizeHotkey(parts.join('+'))
}

/** Lightweight hotkey capture that wins over the global keymap store. */
export function useSimpleHotkeyRecorder() {
  const [state, setState] = useState<RecorderState>({ recording: false, value: null })
  const listening = useRef(false)

  function onKeyDown(event: KeyboardEvent) {
    if (!listening.current) return
    event.preventDefault()
    event.stopImmediatePropagation()

    if (event.key === 'Escape') {
      cancel()
      return
    }
    if (MODIFIER_KEYS.has(event.key)) return

    const value = chordFromEvent(event)
    if (!value) return
    setState({
      recording: true,
      value: { value, display: formatHotkey(value) },
    })
  }

  function attach() {
    if (listening.current) return
    listening.current = true
    window.addEventListener('keydown', onKeyDown, true)
  }

  function detach() {
    if (!listening.current) return
    listening.current = false
    window.removeEventListener('keydown', onKeyDown, true)
  }

  function start() {
    setState({ recording: true, value: null })
    attach()
  }

  function stop() {
    detach()
    setState((prev) => ({ recording: false, value: prev.value }))
  }

  function cancel() {
    detach()
    setState({ recording: false, value: null })
  }

  function clear() {
    detach()
    setState({ recording: false, value: null })
  }

  useEffect(() => () => detach(), [])

  return { state, start, stop, cancel, clear }
}
