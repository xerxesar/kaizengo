import type { ComponentType, ReactNode } from 'react'

export type FieldWidgetProps = {
  value: unknown
  record?: Record<string, unknown>
  field: { key: string; label?: string; type?: string; values?: string[] }
  onChange?: (value: unknown) => void
  readOnly?: boolean
}

const builtins = new Set(['input', 'textarea', 'checkbox', 'select', 'relation'])
const widgets = new Map<string, ComponentType<FieldWidgetProps>>()

/** Register a compiled React control for a spec `widget:` id. */
export function registerFieldWidget(id: string, component: ComponentType<FieldWidgetProps>) {
  const name = id.trim()
  if (!name) return
  widgets.set(name, component)
}

/** Resolve a custom widget. Built-ins and unknown ids return null so the default control is used. */
export function resolveFieldWidget(id?: string | null): ComponentType<FieldWidgetProps> | null {
  const name = id?.trim()
  if (!name || builtins.has(name)) return null
  return widgets.get(name) ?? null
}

export function FieldWidgetView(props: FieldWidgetProps & { widget?: string | null; fallback: ReactNode }) {
  const Widget = resolveFieldWidget(props.widget)
  if (!Widget) return props.fallback
  return <Widget {...props} />
}
