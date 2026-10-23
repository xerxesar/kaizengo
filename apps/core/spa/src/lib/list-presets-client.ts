import { encodeDomain, type Domain } from '@/components/k/search/types'

export type ListViewPreset = {
  id: string
  model: string
  name: string
  ownerId: string
  shared: boolean
  isDefault: boolean
  predefined: boolean
  pageSize?: number | null
  q?: string | null
  searchIn?: string[] | null
  domain?: string | null
  groupBy?: string[] | null
  createdAt: string
  updatedAt: string
}

export type SaveListViewPresetInput = {
  id?: string
  model: string
  name: string
  shared?: boolean
  isDefault?: boolean
  predefined?: boolean
  pageSize?: number
  q?: string
  searchIn?: string[]
  domain?: Domain | string
  groupBy?: string[]
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

const presetFields = `
  id model name ownerId shared isDefault predefined pageSize q searchIn domain groupBy createdAt updatedAt
`

export async function fetchListViewPresets(model: string): Promise<ListViewPreset[]> {
  const data = await gql<{ listViewPresets: ListViewPreset[] }>(`query($model: String!) {
    listViewPresets(model: $model) { ${presetFields} }
  }`, { model })
  return data.listViewPresets ?? []
}

export async function fetchDefaultListViewPreset(model: string): Promise<ListViewPreset | null> {
  const data = await gql<{ defaultListViewPreset: ListViewPreset | null }>(`query($model: String!) {
    defaultListViewPreset(model: $model) { ${presetFields} }
  }`, { model })
  return data.defaultListViewPreset ?? null
}

export async function saveListViewPreset(input: SaveListViewPresetInput): Promise<ListViewPreset> {
  const domain =
    typeof input.domain === 'string'
      ? input.domain
      : encodeDomain(input.domain ?? [])
  const data = await gql<{ saveListViewPreset: ListViewPreset }>(`mutation($input: SaveListViewPresetInput!) {
    saveListViewPreset(input: $input) { ${presetFields} }
  }`, {
    input: {
      ...input,
      domain,
    },
  })
  return data.saveListViewPreset
}

export async function deleteListViewPreset(id: string): Promise<void> {
  await gql(`mutation($id: String!) { deleteListViewPreset(id: $id) }`, { id })
}

export function presetToFilterPreset(p: ListViewPreset) {
  return {
    id: `saved:${p.id}`,
    label: p.name,
    domain: p.domain,
    q: p.q,
    searchIn: p.searchIn,
    groupBy: p.groupBy,
  }
}
