import { fetchOrganizations, type Organization } from './graphql'
import { listModelRecords } from '@kaizengo/sdk-svelte/ui'

type MeResponse = { me: { roles: string[] } }

async function gqlFetch<T>(query: string, variables?: Record<string, unknown>): Promise<T> {
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

let ready = $state(false)
let loading = $state(true)
let error = $state('')
let orgs = $state<Organization[]>([])
let selectedOrg = $state<Organization | null>(null)
let isAdmin = $state(false)
let userCount = $state(0)
let unitCount = $state(0)
let initPromise: Promise<void> | null = null

export async function refreshStats(_orgId: string) {
  try {
    const [users, units] = await Promise.all([
      listModelRecords('identity', 'user', ['id']),
      listModelRecords('identity', 'org_unit', ['id']),
    ])
    userCount = users.length
    unitCount = units.length
  } catch {
    /* views show their own errors */
  }
}

export async function initIdentity() {
  if (ready) return
  if (initPromise) return initPromise

  initPromise = (async () => {
    loading = true
    error = ''
    try {
      const [orgData, meData] = await Promise.all([
        fetchOrganizations(),
        gqlFetch<MeResponse>('query { me { roles } }'),
      ])
      orgs = orgData.organizations
      selectedOrg = orgs[0] ?? null
      isAdmin = meData.me.roles.includes('admin')
      if (selectedOrg) await refreshStats(selectedOrg.id)
      ready = true
    } catch (e) {
      error = e instanceof Error ? e.message : String(e)
    } finally {
      loading = false
    }
  })()

  return initPromise
}

export function selectOrg(orgId: string) {
  selectedOrg = orgs.find((o) => o.id === orgId) ?? null
  if (selectedOrg) void refreshStats(selectedOrg.id)
}

export function updateSelectedOrg(org: Organization) {
  selectedOrg = org
  orgs = orgs.map((o) => (o.id === org.id ? org : o))
}

export function identityState() {
  return {
    get ready() {
      return ready
    },
    get loading() {
      return loading
    },
    get error() {
      return error
    },
    get orgs() {
      return orgs
    },
    get selectedOrg() {
      return selectedOrg
    },
    get isAdmin() {
      return isAdmin
    },
    get userCount() {
      return userCount
    },
    get unitCount() {
      return unitCount
    },
    set error(v: string) {
      error = v
    },
    onStats(counts: { users?: number; units?: number }) {
      if (counts.users !== undefined) userCount = counts.users
      if (counts.units !== undefined) unitCount = counts.units
    },
  }
}
