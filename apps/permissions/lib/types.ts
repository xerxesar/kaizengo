import type { ModelRecord } from '@kaizengo/sdk-solid/ui'

export type Role = ModelRecord & { name?: string; label?: string; active?: boolean; description?: string }
export type UserRole = ModelRecord & { userId?: string; roleId?: string }
export type User = ModelRecord & { name?: string; email?: string }
export type AclEntry = ModelRecord & {
  name?: string
  roleId?: string
  authorId?: string
  effect?: string
  resource?: string
  actions?: string
  fields?: string
  domain?: string
  priority?: number
  active?: boolean
}
