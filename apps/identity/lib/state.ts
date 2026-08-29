import { createSignal } from 'solid-js'
import { listModelRecords } from '@kaizengo/sdk-solid/ui'
import { fetchOrganizations, type Organization } from './graphql'

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

const [ready, setReady] = createSignal(false)
const [loading, setLoading] = createSignal(true)
const [error, setError] = createSignal('')
const [orgs, setOrgs] = createSignal<Organization[]>([])
const [selectedOrg, setSelectedOrg] = createSignal<Organization | null>(null)
const [isAdmin, setIsAdmin] = createSignal(false)
const [userCount, setUserCount] = createSignal(0)
const [unitCount, setUnitCount] = createSignal(0)

let initPromise: Promise<void> | null = null

export async function refreshStats(_orgId: string) {
  try {
    const [users, units] = await Promise.all([
      listModelRecords('identity', 'user', ['id']),
      listModelRecords('identity', 'org_unit', ['id']),
    ])
    setUserCount(users.length)
    setUnitCount(units.length)
  } catch {
    /* views show their own errors */
  }
}

export async function initIdentity() {
  if (ready()) return
  if (initPromise) return initPromise

  initPromise = (async () => {
    setLoading(true)
    setError('')
    try {
      const [orgData, meData] = await Promise.all([
        fetchOrganizations(),
        gqlFetch<MeResponse>('query { me { roles } }'),
      ])
      const list = orgData.organizations
      setOrgs(list)
      setSelectedOrg(list[0] ?? null)
      setIsAdmin(meData.me.roles.includes('admin'))
      if (list[0]) await refreshStats(list[0].id)
      setReady(true)
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e))
    } finally {
      setLoading(false)
    }
  })()

  return initPromise
}

export function selectOrg(orgId: string) {
  const org = orgs().find((o) => o.id === orgId) ?? null
  setSelectedOrg(org)
  if (org) void refreshStats(org.id)
}

export function updateSelectedOrg(org: Organization) {
  setSelectedOrg(org)
  setOrgs((prev) => prev.map((o) => (o.id === org.id ? org : o)))
}

export function identityState() {
  return {
    get ready() {
      return ready()
    },
    get loading() {
      return loading()
    },
    get error() {
      return error()
    },
    get orgs() {
      return orgs()
    },
    get selectedOrg() {
      return selectedOrg()
    },
    get isAdmin() {
      return isAdmin()
    },
    get userCount() {
      return userCount()
    },
    get unitCount() {
      return unitCount()
    },
    set error(v: string) {
      setError(v)
    },
    onStats(counts: { users?: number; units?: number }) {
      if (counts.users !== undefined) setUserCount(counts.users)
      if (counts.units !== undefined) setUnitCount(counts.units)
    },
  }
}
