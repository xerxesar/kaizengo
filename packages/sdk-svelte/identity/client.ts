export type IdentityUser = {
  id: string
  orgId?: string
  email: string
  name: string
  status: string
}

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

/** fetchUsers loads org members via engine GraphQL (principal org scope). */
export async function fetchUsers(_orgId?: string): Promise<IdentityUser[]> {
  const data = await gql<{ identityUsers: IdentityUser[] }>(
    `query {
      identityUsers { id orgId email name status }
    }`,
  )
  return data.identityUsers
}

/** fetchActiveUsers returns active org members only. */
export async function fetchActiveUsers(orgId?: string): Promise<IdentityUser[]> {
  const users = await fetchUsers(orgId)
  return users.filter((u) => u.status === 'active')
}
