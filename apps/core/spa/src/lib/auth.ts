export type AuthUser = {
  id: string
  orgId: string
  email: string
  name: string
  roles: string[]
}

export async function fetchMe(): Promise<AuthUser | null> {
  const res = await fetch('/auth/me', { credentials: 'include' })
  if (res.status === 401) return null
  if (!res.ok) throw new Error(`Auth check failed (${res.status})`)
  return res.json() as Promise<AuthUser>
}

export async function logout(): Promise<void> {
  await fetch('/auth/logout', { method: 'POST', credentials: 'include' })
}

export function gqlFetch<T>(query: string, variables?: Record<string, unknown>): Promise<T> {
  return fetch('/graphql', {
    method: 'POST',
    credentials: 'include',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ query, variables }),
  }).then(async (res) => {
    if (!res.ok) throw new Error(`GraphQL HTTP ${res.status}`)
    const body = await res.json()
    if (body.errors?.length) {
      throw new Error(body.errors.map((e: { message: string }) => e.message).join(', '))
    }
    return body.data as T
  })
}
