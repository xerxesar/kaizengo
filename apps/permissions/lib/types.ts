import type { ModelRecord } from '@kaizengo/sdk-solid/ui'

export type Role = ModelRecord & { name?: string; label?: string; active?: boolean; description?: string }
export type UserRole = ModelRecord & { userId?: string; roleId?: string }
export type User = ModelRecord & { name?: string; email?: string }
export type AclEntry = ModelRecord & {
  name?: string
  roleId?: string
  authorId?: string
  effect?: string
  kind?: string
  resource?: string
  actions?: string
  fields?: string
  domain?: string
  priority?: number
  active?: boolean
}

const CALL_STYLE_KINDS = new Set(['query', 'command', 'view'])

/** Mirrors packages/sdk-go/acl InferKind for the Access UI. */
export function inferResourceKind(resource: string): string {
  const parts = resource.trim().split('.')
  if (parts.length === 1) return 'app'
  switch (parts[1]) {
    case 'view':
    case 'query':
    case 'command':
    case 'mutation':
    case 'event':
    case 'api':
      return parts[1]
    case 'menu':
    case 'nav':
      // Legacy ids — not cataloged; implied from views.
      return parts[1]
    default:
      return parts.length === 2 ? 'model' : 'api'
  }
}

export function isCallStyleKind(kind: string): boolean {
  return CALL_STYLE_KINDS.has(kind)
}

/** Tailwind classes for kind badges (query blue, command green, view gray). */
export function kindBadgeClass(kind: string): string {
  switch (kind) {
    case 'query':
      return 'bg-sky-100 text-sky-800'
    case 'command':
      return 'bg-emerald-100 text-emerald-800'
    case 'view':
      return 'bg-zinc-200 text-zinc-700'
    default:
      return 'bg-zinc-100 text-zinc-600'
  }
}
