import { useEffect, useMemo, useState, type CSSProperties } from 'react'
import { elementIdFromAction } from './actions'
import { t } from '../i18n-context'
import { KEYMAP_ID_ATTR } from './types'
import type { KeymapBinding } from './types'
import './keymap.css'

type Props = {
  bindings: KeymapBinding[]
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

function hintStyle(hint: HintTarget): CSSProperties {
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

function collectHints(
  bindings: KeymapBinding[],
  effectiveKeys: Props['effectiveKeys'],
): {
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
  const [tick, setTick] = useState(0)

  useEffect(() => {
    const bump = () => setTick((n) => n + 1)
    window.addEventListener('scroll', bump, true)
    window.addEventListener('resize', bump)
    const id = window.setInterval(bump, 250)
    return () => {
      window.removeEventListener('scroll', bump, true)
      window.removeEventListener('resize', bump)
      window.clearInterval(id)
    }
  }, [])

  const hints = useMemo(() => {
    void tick
    return collectHints(props.bindings, props.effectiveKeys)
  }, [tick, props.bindings, props.effectiveKeys])

  return (
    <div className="kg-keymap-hints" aria-hidden="true">
      {hints.elements.map((hint) => (
        <div key={hint.id} className="kg-keymap-hint kg-keymap-hint--element" style={hintStyle(hint)}>
          <kbd>{props.formatHotkey(hint.keys)}</kbd>
        </div>
      ))}
    </div>
  )
}
