import { print, type DocumentNode } from 'graphql'

type GraphQLResponse<T> = {
  data?: T
  errors?: { message: string }[]
}

export async function execute<TResult, TVariables extends object = Record<string, never>>(
  document: DocumentNode,
  variables?: TVariables,
): Promise<TResult> {
  const res = await fetch('/graphql', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({
      query: print(document),
      variables: variables ?? {},
    }),
  })

  if (!res.ok) {
    throw new Error(`GraphQL HTTP ${res.status}`)
  }

  const body = (await res.json()) as GraphQLResponse<TResult>
  if (body.errors?.length) {
    throw new Error(body.errors.map((e) => e.message).join(', '))
  }
  if (!body.data) {
    throw new Error('GraphQL response missing data')
  }
  return body.data
}
