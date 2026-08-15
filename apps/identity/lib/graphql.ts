async function gql<T>(query: string, variables?: Record<string, unknown>): Promise<T> {
  const res = await fetch('/graphql', {
    method: 'POST',
    credentials: 'include',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ query, variables }),
  })
  if (!res.ok) throw new Error(`GraphQL HTTP ${res.status}`)
  const body = await res.json()
  if (body.errors?.length) {
    throw new Error(body.errors.map((e: { message: string }) => e.message).join(', '))
  }
  return body.data as T
}

export type Organization = {
  id: string
  name: string
  slug: string
  createdAt: string
}

export type OrgUnitNode = {
  id: string
  orgId: string
  parentId: string | null
  type: string
  name: string
  createdAt: string
  children: OrgUnitNode[]
}

export function fetchOrganizations() {
  return gql<{ identityOrganizations: Organization[] }>(`
    query {
      identityOrganizations { id name slug createdAt }
    }
  `).then((d) => ({ organizations: d.identityOrganizations }))
}

export function fetchOrgTree(_orgId?: string) {
  return gql<{ identityOrgUnits: Array<Omit<OrgUnitNode, 'children'> & { parentId?: string | null }> }>(
    `query { identityOrgUnits { id orgId parentId type name createdAt } }`,
  ).then((d) => ({ orgTree: nestUnits(d.identityOrgUnits) }))
}

function nestUnits(
  units: Array<Omit<OrgUnitNode, 'children'> & { parentId?: string | null }>,
): OrgUnitNode[] {
  const byParent = new Map<string, typeof units>()
  const roots: typeof units = []
  for (const u of units) {
    const parent = (u.parentId ?? '').trim()
    if (!parent) {
      roots.push(u)
      continue
    }
    const list = byParent.get(parent) ?? []
    list.push(u)
    byParent.set(parent, list)
  }
  const build = (u: (typeof units)[number]): OrgUnitNode => ({
    ...u,
    parentId: (u.parentId ?? '').trim() || null,
    children: (byParent.get(u.id) ?? []).map(build),
  })
  return roots.map(build)
}

export function createOrgUnit(orgId: string, type: string, name: string, parentId?: string) {
  return gql<{ createIdentityOrgUnit: { id: string; name: string; type: string } }>(
    `mutation ($name: String!, $type: String!, $parentId: String) {
      createIdentityOrgUnit(name: $name, type: $type, parentId: $parentId) {
        id name type
      }
    }`,
    { name, type, parentId: parentId ?? null },
  )
}

export const ORG_UNIT_TYPES = [
  { value: 'business_unit', key: 'identity.unit_type.business_unit' },
  { value: 'department', key: 'identity.unit_type.department' },
  { value: 'location', key: 'identity.unit_type.location' },
  { value: 'team', key: 'identity.unit_type.team' },
  { value: 'position', key: 'identity.unit_type.position' },
] as const

const UNIT_TYPE_KEYS: Record<string, string> = Object.fromEntries(
  ORG_UNIT_TYPES.map((u) => [u.value, u.key]),
)

export function flattenTree(nodes: OrgUnitNode[], depth = 0): Array<OrgUnitNode & { depth: number }> {
  const out: Array<OrgUnitNode & { depth: number }> = []
  for (const n of nodes) {
    out.push({ ...n, depth })
    out.push(...flattenTree(n.children ?? [], depth + 1))
  }
  return out
}

export function formatUnitType(type: string, t: (key: string) => string) {
  const key = UNIT_TYPE_KEYS[type]
  return key ? t(key) : type.replace(/_/g, ' ').replace(/\b\w/g, (c) => c.toUpperCase())
}
