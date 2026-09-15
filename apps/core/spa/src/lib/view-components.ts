import type { ComponentType } from 'react'

const registry = new Map<string, ComponentType>()

export function registerViewComponent(id: string, component: ComponentType) {
  registry.set(id, component)
}

export function resolveViewComponent(id: string): ComponentType | null {
  return registry.get(id) ?? null
}
