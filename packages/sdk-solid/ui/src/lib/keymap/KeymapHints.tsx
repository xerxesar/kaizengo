import { createMemo, createSignal, For, onCleanup, onMount, Show, type Accessor } from 'solid-js'
import { elementIdFromAction } from './actions'
import { t } from '../i18n-context'
import { KEYMAP_ID_ATTR } from './types'
import type { KeymapBinding } from './types'
import './keymap.css'

type Props = {
  bindings: Accessor<KeymapBinding[]> | KeymapBinding[]
  effectiveKeys: (id: string, fallback: string) => string
  formatHotkey: (hotkey: string) => string
}

type HintTarget = {
  id: string
  keys: string
  label: string
  rect: DOMRect
  placement: 'above' | 'below'
}

type GlobalHint = {
  id: string
  keys: string
  label: string
}

function readBindings(bindings: Props['bindings']): KeymapBinding[] {
  return typeof bindings === 'function' ? bindings() : bindings
}

function labelFor(binding: KeymapBinding): string {
  if (binding.labelKey) {
    const translated = t(binding.labelKey)
    if (translated !== binding.labelKey) return translated
  }
  return binding.label || binding.id
}

function hintPlacement(rect: DOMRect): 'above' | 'below' {
  const badgeH = 24
  const pad = 6
  const spaceAbove = rect.top
  const spaceBelow = window.innerHeight - rect.bottom
  if (spaceAbove < badgeH + pad && spaceBelow >= badgeH + pad) return 'below'
  if (spaceBelow < badgeH + pad && spaceAbove >= badgeH + pad) return 'above'
  return spaceBelow >= spaceAbove ? 'below' : 'above'
}

function hintStyle(hint: HintTarget): Record<string, string> {
  const pad = 6
  if (hint.placement === 'below') {
    return {
      top: `${hint.rect.bottom + pad}px`,
      left: `${hint.rect.right}px`,
    }
  }
  return {
    top: `${hint.rect.top - pad}px`,
    left: `${hint.rect.right}px`,
    transform: 'translate(-100%, -100%)',
  }
}

function collectHints(bindings: KeymapBinding[], effectiveKeys: Props['effectiveKeys']): {
  elements: HintTarget[]
  global: GlobalHint[]
} {
  const byElementId = new Map<string, { keys: string; label: string }>()
  const global: GlobalHint[] = []

  for (const binding of bindings) {
    if (!binding.hint) continue
    const keys = effectiveKeys(binding.id, binding.keys)
    const label = labelFor(binding)
    const elementId = elementIdFromAction(binding.action)
    if (elementId) {
      byElementId.set(elementId, { keys, label })
      global.push({ id: binding.id, keys, label })
      continue
    }
    global.push({ id: binding.id, keys, label })
  }

  const elements: HintTarget[] = []
  const seen = new Set<string>()

  for (const el of document.querySelectorAll(`[${KEYMAP_ID_ATTR}]`)) {
    const id = el.getAttribute(KEYMAP_ID_ATTR)
    if (!id || seen.has(id)) continue
    const meta = byElementId.get(id)
    if (!meta) continue
    const rect = el.getBoundingClientRect()
    if (rect.width === 0 && rect.height === 0) continue
    seen.add(id)
    elements.push({
      id,
      keys: meta.keys,
      label: meta.label,
      rect,
      placement: hintPlacement(rect),
    })
  }

  return { elements, global }
}

export function KeymapHints(props: Props) {
  const [tick, setTick] = createSignal(0)

  onMount(() => {
    const bump = () => setTick((n) => n + 1)
    window.addEventListener('scroll', bump, true)
    window.addEventListener('resize', bump)
    const id = window.setInterval(bump, 250)
    onCleanup(() => {
      window.removeEventListener('scroll', bump, true)
      window.removeEventListener('resize', bump)
      window.clearInterval(id)
    })
  })

  const hints = createMemo(() => {
    tick()
    return collectHints(readBindings(props.bindings), props.effectiveKeys)
  })

  return (
    <div class="kg-keymap-hints" aria-hidden="true">
      <For each={hints().elements}>
        {(hint) => (
          <div class="kg-keymap-hint kg-keymap-hint--element" style={hintStyle(hint)}>
            <kbd>{props.formatHotkey(hint.keys)}</kbd>
          </div>
        )}
      </For>
    </div>
  )
}
