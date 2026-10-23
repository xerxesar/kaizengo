import { useSyncExternalStore } from 'react'
import { listModelRecords } from '@/lib'
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

type IdentitySnapshot = {
  ready: boolean
  loading: boolean
  error: string
  orgs: Organization[]
  selectedOrg: Organization | null
  isAdmin: boolean
  userCount: number
  unitCount: number
}

let snapshot: IdentitySnapshot = {
  ready: false,
  loading: true,
  error: '',
  orgs: [],
  selectedOrg: null,
  isAdmin: false,
  userCount: 0,
  unitCount: 0,
}

const listeners = new Set<() => void>()

function emit() {
  for (const listener of listeners) listener()
}

function setSnapshot( partial: Partial<IdentitySnapshot>) {
  snapshot = { ...snapshot, ...partial }
  emit()
}

function subscribe(listener: () => void) {
  listeners.add(listener)
  return () => listeners.delete(listener)
}

export async function refreshStats(_orgId: string) {
  try {
    const [users, units] = await Promise.all([
      listModelRecords('identity', 'user', ['id']),
      listModelRecords('identity', 'org_unit', ['id']),
    ])
    setSnapshot({ userCount: users.length, unitCount: units.length })
  } catch {
    /* views show their own errors */
  }
}

let initPromise: Promise<void> | null = null

export async function initIdentity() {
  if (snapshot.ready) return
  if (initPromise) return initPromise

  initPromise = (async () => {
    setSnapshot({ loading: true, error: '' })
    try {
      const [orgData, meData] = await Promise.all([
        fetchOrganizations(),
        gqlFetch<MeResponse>('query { me { roles } }'),
      ])
      const list = orgData.organizations
      setSnapshot({
        orgs: list,
        selectedOrg: list[0] ?? null,
        isAdmin: meData.me.roles.includes('admin'),
        ready: true,
      })
      if (list[0]) await refreshStats(list[0].id)
    } catch (e) {
      setSnapshot({ error: e instanceof Error ? e.message : String(e) })
    } finally {
      setSnapshot({ loading: false })
    }
  })()

  return initPromise
}

export function selectOrg(orgId: string) {
  const org = snapshot.orgs.find((o) => o.id === orgId) ?? null
  setSnapshot({ selectedOrg: org })
  if (org) void refreshStats(org.id)
}

export function updateSelectedOrg(org: Organization) {
  setSnapshot({
    selectedOrg: org,
    orgs: snapshot.orgs.map((o) => (o.id === org.id ? org : o)),
  })
}

export function identityState() {
  return {
    get ready() {
      return snapshot.ready
    },
    get loading() {
      return snapshot.loading
    },
    get error() {
      return snapshot.error
    },
    get orgs() {
      return snapshot.orgs
    },
    get selectedOrg() {
      return snapshot.selectedOrg
    },
    get isAdmin() {
      return snapshot.isAdmin
    },
    get userCount() {
      return snapshot.userCount
    },
    get unitCount() {
      return snapshot.unitCount
    },
    set error(v: string) {
      setSnapshot({ error: v })
    },
    onStats(counts: { users?: number; units?: number }) {
      setSnapshot({
        ...(counts.users !== undefined ? { userCount: counts.users } : {}),
        ...(counts.units !== undefined ? { unitCount: counts.units } : {}),
      })
    },
  }
}

/** Subscribe to identity store for React re-renders. */
export function useIdentityState() {
  return useSyncExternalStore(subscribe, () => snapshot, () => snapshot)
}
