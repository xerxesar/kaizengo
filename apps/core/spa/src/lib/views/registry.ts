import type { ComponentType } from 'react'
import { SearchBar } from '@/components/shell/SearchBar'
import { registerViewComponent } from '@/lib/view-components'

type ViewModule = { default: ComponentType }

const appViewModules = import.meta.glob<ViewModule>('@apps/*/views/**/*.page.tsx', {
  eager: true,
})

const viewByKey = new Map<string, ComponentType>()
const viewByComponent = new Map<string, ComponentType>()

const HOST_APP = 'core'

function parseAppViewPath(path: string): { app: string; name: string } | null {
  const p = path.replace(/\\/g, '/')

  const fromApps = p.match(/(?:^|\/)apps\/([^/]+)\/views\/(.+)\.page\.tsx$/)
  if (fromApps && fromApps[1] !== '..') {
    const name = fromApps[2].split('/').pop()
    if (name) return { app: fromApps[1], name }
  }

  const fromAlias = p.match(/(?:^|\/)@apps\/([^/]+)\/views\/(.+)\.page\.tsx$/)
  if (fromAlias) {
    const name = fromAlias[2].split('/').pop()
    if (name) return { app: fromAlias[1], name }
  }

  const fromShort = p.match(/^(?:\.\.\/)+views\/(.+)\.page\.tsx$/)
  if (fromShort) {
    const name = fromShort[1].split('/').pop()
    if (name) return { app: HOST_APP, name }
  }

  const fromNamed = p.match(/\/((?!\.\.)[^/]+)\/views\/(.+)\.page\.tsx$/)
  if (fromNamed) {
    const name = fromNamed[2].split('/').pop()
    if (name) return { app: fromNamed[1], name }
  }

  return null
}

for (const [path, mod] of Object.entries(appViewModules)) {
  if (!mod?.default) continue
  const parsed = parseAppViewPath(path)
  if (!parsed) continue
  viewByKey.set(`${parsed.app}.${parsed.name}`, mod.default)
}

const componentExports: Record<string, ComponentType> = {}

function registerComponentExport(id: string, component: ComponentType) {
  componentExports[id] = component
  viewByComponent.set(id, component)
  registerViewComponent(id, component)
}

const permissionsAccess = viewByKey.get('permissions.Access')
if (permissionsAccess) registerComponentExport('permissions.Access', permissionsAccess)

const permissionsRoles = viewByKey.get('permissions.Roles')
if (permissionsRoles) registerComponentExport('permissions.Roles', permissionsRoles)

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

export function resolveView(ctx: ViewResolveContext): ComponentType | null {
  const component = ctx.component?.trim()
  if (component) {
    const byExport = componentExports[component] ?? viewByComponent.get(component)
    if (byExport) return byExport
  }

  const view = ctx.view?.trim()
  const app = ctx.app.trim()
  if (view && app) {
    const direct = viewByKey.get(`${app}.${view}`)
    if (direct) return direct
  }

  if (app && !view) {
    return viewByKey.get(`${app}.Index`) ?? null
  }

  return null
}

export function resolveComponent(componentId: string): ComponentType | null {
  const id = componentId.trim()
  return componentExports[id] ?? viewByComponent.get(id) ?? null
}

export function listViewKeys(): string[] {
  return [...viewByKey.keys()]
}
