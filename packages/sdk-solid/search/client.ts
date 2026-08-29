export type SearchHit = {
  id: string
  collection: string
  title: string
  snippet?: string
  score?: number
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

export async function searchQuery(
  q: string,
  collections?: string[],
  limit = 20,
): Promise<SearchHit[]> {
  const data = await gql<{ search: SearchHit[] }>(
    `query ($q: String!, $collections: [String!], $limit: Int) {
      search(q: $q, collections: $collections, limit: $limit) {
        id collection title snippet score
      }
    }`,
    { q, collections, limit },
  )
  return data.search
}

export async function searchBackend(): Promise<string> {
  const data = await gql<{ searchBackend: string }>('query { searchBackend }')
  return data.searchBackend
}
