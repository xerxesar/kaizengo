export type SearchFieldOption = {
  name: string
  type: string
  selected: boolean
}

export type SearchModelConfig = {
  app: string
  model: string
  collection: string
  enabled: boolean
  source: string
  documentCount: number
  fields: SearchFieldOption[]
}

export type SearchConfig = {
  backend: string
  connected: boolean
  models: SearchModelConfig[]
}

export type ReindexResult = {
  indexed: number
  searchConfig: SearchConfig
}

const searchConfigFields = `
  backend connected
  models {
    app model collection enabled source documentCount
    fields { name type selected }
  }
`

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

export function fetchSearchConfig() {
  return gql<{ searchConfig: SearchConfig }>(`query { searchConfig { ${searchConfigFields} } }`)
}

export function updateSearchModelConfig(input: {
  app: string
  model: string
  enabled: boolean
  fields: string[]
}) {
  return gql<{ updateSearchModelConfig: SearchConfig }>(
    `mutation ($app: String!, $model: String!, $enabled: Boolean!, $fields: [String!]!) {
      updateSearchModelConfig(app: $app, model: $model, enabled: $enabled, fields: $fields) {
        ${searchConfigFields}
      }
    }`,
    input,
  )
}

export function reindexSearchModel(input: { app: string; model: string; field?: string }) {
  return gql<{ reindexSearchModel: ReindexResult }>(
    `mutation ($app: String!, $model: String!, $field: String) {
      reindexSearchModel(app: $app, model: $model, field: $field) {
        indexed
        searchConfig { ${searchConfigFields} }
      }
    }`,
    input,
  )
}
