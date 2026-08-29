import type { Component } from 'solid-js'
import { SearchBar } from '@kaizengo/sdk-solid/search'
import { registerViewComponent } from '@kaizengo/sdk-solid/ui'

type ViewModule = { default: Component }

/** App pages (`*.page.tsx`) under apps/<app>/views/. */
const appViewModules = import.meta.glob<ViewModule>('../../../../../*/views/**/*.page.tsx', {
  eager: true,
})

const viewByKey = new Map<string, Component>()
const viewByComponent = new Map<string, Component>()

for (const [path, mod] of Object.entries(appViewModules)) {
  const match =
    path.match(/(?:^|\/)apps\/([^/]+)\/views\/(.+)\.page\.tsx$/) ??
    path.match(/\/([^/]+)\/views\/(.+)\.page\.tsx$/)
  if (!match) continue
  const [, app, rest] = match
  const name = rest.split('/').pop()
  if (!name) continue
  viewByKey.set(`${app}.${name}`, mod.default)
}

/** Cross-app component exports (from app.yaml exports.components). */
const componentExports: Record<string, Component> = {}

function registerComponentExport(id: string, component: Component) {
  componentExports[id] = component
  viewByComponent.set(id, component)
  registerViewComponent(id, component)
}

const permissionsAccess = viewByKey.get('permissions.Access')
if (permissionsAccess) {
  registerComponentExport('permissions.Access', permissionsAccess)
}

const permissionsRoles = viewByKey.get('permissions.Roles')
if (permissionsRoles) {
  registerComponentExport('permissions.Roles', permissionsRoles)
}

const typesenseSearchSettings = viewByKey.get('typesense.SearchSettings')
if (typesenseSearchSettings) {
  registerComponentExport('typesense.SearchSettings', typesenseSearchSettings)
}

registerComponentExport('platform.SearchBar', SearchBar)

export type ViewResolveContext = {
  app: string
  view?: string
  component?: string
}

/** Resolve a menu leaf to a Solid component. */
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

  if (app && !view) {
    return viewByKey.get(`${app}.Index`) ?? null
  }

  return null
}

export function resolveComponent(componentId: string): Component | null {
  const id = componentId.trim()
  return componentExports[id] ?? viewByComponent.get(id) ?? null
}

export function listViewKeys(): string[] {
  return [...viewByKey.keys()]
}
