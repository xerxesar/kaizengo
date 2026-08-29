import type { Component } from 'solid-js'

const registry = new Map<string, Component>()

/** Register a component id for cross-app view slot resolution. */
export function registerViewComponent(id: string, component: Component): void {
  const key = id.trim()
  if (!key) return
  registry.set(key, component)
}

/** Resolve an exported component id (view slots, cross-app menus). */
export function resolveViewComponent(id: string): Component | null {
  return registry.get(id.trim()) ?? null
}
