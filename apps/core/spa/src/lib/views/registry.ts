import type { Component } from 'svelte'
import SearchBar from '@kaizengo/sdk-svelte/search/SearchBar.svelte'
import { registerViewComponent } from '@kaizengo/sdk-svelte/ui'

type ViewModule = { default: Component }

/** All app view files under apps/<app>/views/<Name>.svelte */
const appViewModules = import.meta.glob<ViewModule>('../../../../../*/views/*.svelte', {
  eager: true,
})

const viewByKey = new Map<string, Component>()
const viewByComponent = new Map<string, Component>()

for (const [path, mod] of Object.entries(appViewModules)) {
  const match = path.match(/(?:^|\/)apps\/([^/]+)\/views\/([^/]+)\.svelte$/) ??
    path.match(/\/([^/]+)\/views\/([^/]+)\.svelte$/)
  if (!match) continue
  const [, app, name] = match
  viewByKey.set(`${app}.${name}`, mod.default)
}

/** Cross-app component exports (from app.yaml exports.components). */
const componentExports: Record<string, Component> = {
  'platform.SearchBar': SearchBar,
}

registerViewComponent('platform.SearchBar', SearchBar)

function registerComponentExport(id: string, component: Component) {
  componentExports[id] = component
  viewByComponent.set(id, component)
  registerViewComponent(id, component)
}

// typesense.SearchSettings lives in apps/typesense/views/
const searchSettings = viewByKey.get('typesense.SearchSettings')
if (searchSettings) {
  registerComponentExport('typesense.SearchSettings', searchSettings)
}

export type ViewResolveContext = {
  app: string
  view?: string
  component?: string
}

/** Resolve a menu leaf to a Svelte component. */
export function resolveView(ctx: ViewResolveContext): Component | null {
  const component = ctx.component?.trim()
  if (component) {
    const byExport = componentExports[component] ?? viewByComponent.get(component)
    if (byExport) return byExport
  }

  const view = ctx.view?.trim()
  const app = ctx.app.trim()
  if (view && app) {
    const key = `${app}.${view}`
    const direct = viewByKey.get(key)
    if (direct) return direct
  }

  // Apps without menus: fall back to Index view.
  if (app && !view) {
    return viewByKey.get(`${app}.Index`) ?? null
  }

  return null
}

/** Resolve an exported component id (view slots, cross-app menus). */
export function resolveComponent(componentId: string): Component | null {
  const id = componentId.trim()
  return componentExports[id] ?? viewByComponent.get(id) ?? null
}

export function listViewKeys(): string[] {
  return [...viewByKey.keys()]
}
