import type { KChartConfig, KChartMeasure, KChartType } from './chart-config'
import { DEFAULT_CHART_TYPES } from './chart-config'

const PREFIX = 'kaizengo.chart.presets.'

export type SavedChartPreset = KChartConfig & {
  id: string
  label: string
  model: string
  shared?: boolean
  isDefault?: boolean
  ownerId?: string
  updatedAt: string
}

export type SaveChartPresetInput = {
  model: string
  label: string
  xField: string
  yField?: string
  seriesField?: string
  measure?: KChartMeasure
  type?: KChartType
  types?: KChartType[]
  id?: string
  shared?: boolean
  isDefault?: boolean
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
  id model name ownerId shared isDefault type types xField yField seriesField measure createdAt updatedAt
`

type ServerPreset = {
  id: string
  model: string
  name: string
  ownerId: string
  shared: boolean
  isDefault: boolean
  type: string
  types?: string[] | null
  xField: string
  yField?: string | null
  seriesField?: string | null
  measure: string
  createdAt: string
  updatedAt: string
}

function fromServer(p: ServerPreset): SavedChartPreset {
  return {
    id: p.id,
    label: p.name,
    model: p.model,
    shared: p.shared,
    isDefault: p.isDefault,
    ownerId: p.ownerId,
    type: (p.type as KChartType) || 'bar',
    types: (p.types?.filter(Boolean) as KChartType[] | undefined) ?? DEFAULT_CHART_TYPES,
    xField: p.xField,
    yField: p.yField?.trim() || undefined,
    seriesField: p.seriesField?.trim() || undefined,
    measure: (p.measure as KChartMeasure) || 'count',
    updatedAt: p.updatedAt,
  }
}

function localKey(model: string) {
  return PREFIX + model.trim()
}

function readLocal(model: string): SavedChartPreset[] {
  try {
    const raw = localStorage.getItem(localKey(model))
    if (!raw) return []
    const parsed = JSON.parse(raw) as SavedChartPreset[]
    return Array.isArray(parsed) ? parsed : []
  } catch {
    return []
  }
}

function writeLocal(model: string, items: SavedChartPreset[]) {
  try {
    localStorage.setItem(localKey(model), JSON.stringify(items))
  } catch {
    /* ignore quota */
  }
}

function mergePresets(server: SavedChartPreset[], local: SavedChartPreset[]): SavedChartPreset[] {
  const byId = new Map<string, SavedChartPreset>()
  for (const item of local) byId.set(item.id, item)
  for (const item of server) byId.set(item.id, item)
  return [...byId.values()].sort((a, b) => a.label.localeCompare(b.label))
}

/** Sync local-only list (offline / initial paint). */
export function listChartPresetsSync(model: string): SavedChartPreset[] {
  if (!model.trim()) return []
  return readLocal(model).sort((a, b) => a.label.localeCompare(b.label))
}

export async function listChartPresets(model: string): Promise<SavedChartPreset[]> {
  if (!model.trim()) return []
  try {
    const data = await gql<{ chartViewPresets: ServerPreset[] }>(`query($model: String!) {
      chartViewPresets(model: $model) { ${presetFields} }
    }`, { model })
    return mergePresets((data.chartViewPresets ?? []).map(fromServer), readLocal(model))
  } catch {
    return listChartPresetsSync(model)
  }
}

export async function fetchDefaultChartPreset(model: string): Promise<SavedChartPreset | null> {
  if (!model.trim()) return null
  try {
    const data = await gql<{ defaultChartViewPreset: ServerPreset | null }>(`query($model: String!) {
      defaultChartViewPreset(model: $model) { ${presetFields} }
    }`, { model })
    if (data.defaultChartViewPreset) return fromServer(data.defaultChartViewPreset)
  } catch {
    /* fall through to local */
  }
  return readLocal(model).find((p) => p.isDefault) ?? null
}

function isServerPresetId(id?: string): boolean {
  if (!id || id.startsWith('user:')) return false
  return /^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$/i.test(id)
}

export async function saveChartPreset(input: SaveChartPresetInput): Promise<SavedChartPreset> {
  const model = input.model.trim()
  const label = input.label.trim()
  if (!model) throw new Error('model is required')
  if (!label) throw new Error('name is required')
  if (!input.xField.trim()) throw new Error('xField is required')

  try {
    const data = await gql<{ saveChartViewPreset: ServerPreset }>(`mutation($input: SaveChartViewPresetInput!) {
      saveChartViewPreset(input: $input) { ${presetFields} }
    }`, {
      input: {
        id: isServerPresetId(input.id) ? input.id : undefined,
        model,
        name: label,
        shared: input.shared ?? false,
        isDefault: input.isDefault ?? false,
        type: input.type ?? 'bar',
        types: input.types?.length ? input.types : DEFAULT_CHART_TYPES,
        xField: input.xField.trim(),
        yField: input.yField?.trim() || undefined,
        seriesField: input.seriesField?.trim() || undefined,
        measure: input.measure ?? 'count',
      },
    })
    const saved = fromServer(data.saveChartViewPreset)
    if (input.id?.startsWith('user:')) {
      writeLocal(model, readLocal(model).filter((p) => p.id !== input.id))
    }
    return saved
  } catch {
    return saveChartPresetLocal(input)
  }
}

function saveChartPresetLocal(input: SaveChartPresetInput): SavedChartPreset {
  const model = input.model.trim()
  const items = readLocal(model)
  const id = input.id?.trim() || `user:${crypto.randomUUID()}`
  if (input.isDefault) {
    for (const item of items) item.isDefault = item.id === id
  }
  const next: SavedChartPreset = {
    id,
    label: input.label.trim(),
    model,
    shared: input.shared ?? false,
    isDefault: input.isDefault ?? false,
    xField: input.xField.trim(),
    yField: input.yField?.trim() || undefined,
    seriesField: input.seriesField?.trim() || undefined,
    measure: input.measure,
    type: input.type ?? 'bar',
    types: input.types?.length ? input.types : DEFAULT_CHART_TYPES,
    updatedAt: new Date().toISOString(),
  }
  const idx = items.findIndex((p) => p.id === id)
  if (idx >= 0) items[idx] = next
  else items.push(next)
  writeLocal(model, items)
  return next
}

export async function deleteChartPreset(model: string, id: string): Promise<void> {
  if (!id.startsWith('user:')) {
    try {
      await gql(`mutation($id: String!) { deleteChartViewPreset(id: $id) }`, { id })
    } catch {
      /* continue to clear local */
    }
  }
  writeLocal(model, readLocal(model).filter((p) => p.id !== id))
}

export function savedToChartConfig(p: SavedChartPreset): KChartConfig {
  return {
    id: p.id,
    label: p.label,
    type: p.type,
    types: p.types,
    xField: p.xField,
    yField: p.yField,
    seriesField: p.seriesField,
    measure: p.measure,
    shared: p.shared,
    isDefault: p.isDefault,
  }
}
