async function gql<T>(query: string, variables?: Record<string, unknown>): Promise<T> {
  const res = await fetch('/graphql', {
    method: 'POST',
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

export function fetchCounter() {
  return gql<{ counter: number }>('query { counter }')
}

export function addCounter(by: number) {
  return gql<{ addCounter: number }>('mutation ($by: Int!) { addCounter(by: $by) }', { by })
}

export function resetCounter() {
  return gql<{ resetCounter: number }>('mutation { resetCounter }')
}
