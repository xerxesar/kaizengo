import {
  deleteListViewPreset,
  fetchDefaultListViewPreset,
  fetchListViewPresets,
  saveListViewPreset,
  type ListViewPreset,
} from '@/lib/list-presets-client'
import { decodeDomain, encodeDomain, type Domain, type SearchTemplate } from './types'

const PREFIX = 'kaizengo.search.templates.'

function localKey(model: string) {
  return PREFIX + model.trim()
}

function readLocal(model: string): SearchTemplate[] {
  try {
    const raw = localStorage.getItem(localKey(model))
    if (!raw) return []
    const parsed = JSON.parse(raw) as SearchTemplate[]
    return Array.isArray(parsed) ? parsed : []
  } catch {
    return []
  }
}

function writeLocal(model: string, items: SearchTemplate[]) {
  try {
    localStorage.setItem(localKey(model), JSON.stringify(items))
  } catch {
    /* ignore quota */
  }
}

function fromServer(p: ListViewPreset): SearchTemplate {
  let domain: Domain = []
  if (p.domain?.trim()) {
    try {
      domain = decodeDomain(p.domain)
    } catch {
      domain = []
    }
  }
  return {
    id: p.id,
    name: p.name,
    model: p.model,
    q: p.q?.trim() ?? '',
    searchIn: p.searchIn ?? [],
    domain,
    groupBy: p.groupBy ?? [],
    shared: p.shared,
    isDefault: p.isDefault,
    predefined: p.predefined,
    pageSize: p.pageSize && p.pageSize > 0 ? p.pageSize : undefined,
    ownerId: p.ownerId,
    createdAt: p.createdAt,
    updatedAt: p.updatedAt,
  }
}

function mergeTemplates(server: SearchTemplate[], local: SearchTemplate[]): SearchTemplate[] {
  const byId = new Map<string, SearchTemplate>()
  for (const item of local) byId.set(item.id, item)
  for (const item of server) byId.set(item.id, item)
  return [...byId.values()].sort((a, b) => a.name.localeCompare(b.name))
}

export async function listSearchTemplates(model: string): Promise<SearchTemplate[]> {
  try {
    const server = (await fetchListViewPresets(model)).map(fromServer)
    return mergeTemplates(server, readLocal(model))
  } catch {
    return readLocal(model).sort((a, b) => a.name.localeCompare(b.name))
  }
}

export function listSearchTemplatesSync(model: string): SearchTemplate[] {
  return readLocal(model).sort((a, b) => a.name.localeCompare(b.name))
}

export async function saveSearchTemplate(input: {
  model: string
  name: string
  q: string
  searchIn: string[]
  domain: Domain
  groupBy: string[]
  id?: string
  shared?: boolean
  isDefault?: boolean
  predefined?: boolean
  pageSize?: number
}): Promise<SearchTemplate> {
  try {
    const saved = await saveListViewPreset({
      id: input.id,
      model: input.model,
      name: input.name,
      q: input.q,
      searchIn: input.searchIn,
      domain: encodeDomain(input.domain),
      groupBy: input.groupBy,
      shared: input.shared,
      isDefault: input.isDefault,
      predefined: input.predefined,
      pageSize: input.pageSize,
    })
    return fromServer(saved)
  } catch {
    const now = new Date().toISOString()
    const items = readLocal(input.model)
    const id = input.id?.trim() || crypto.randomUUID()
    const existing = items.findIndex((t) => t.id === id)
    const next: SearchTemplate = {
      id,
      name: input.name.trim() || 'Untitled',
      model: input.model,
      q: input.q,
      searchIn: [...input.searchIn],
      domain: [...input.domain],
      groupBy: [...input.groupBy],
      shared: input.shared,
      isDefault: input.isDefault,
      predefined: input.predefined,
      pageSize: input.pageSize,
      createdAt: existing >= 0 ? items[existing].createdAt : now,
      updatedAt: now,
    }
    if (input.isDefault) {
      for (const item of items) {
        if (item.id !== id) item.isDefault = false
      }
    }
    if (existing >= 0) items[existing] = next
    else items.push(next)
    writeLocal(input.model, items)
    return next
  }
}

export async function deleteSearchTemplate(model: string, id: string) {
  try {
    await deleteListViewPreset(id)
  } catch {
    /* fall through to local */
  }
  writeLocal(
    model,
    readLocal(model).filter((t) => t.id !== id),
  )
}

export async function fetchDefaultSearchTemplate(model: string): Promise<SearchTemplate | null> {
  try {
    const preset = await fetchDefaultListViewPreset(model)
    if (preset) return fromServer(preset)
  } catch {
    /* fall through to local */
  }
  return readLocal(model).find((t) => t.isDefault) ?? null
}

export function predefinedTemplates(templates: SearchTemplate[]) {
  return templates.filter((t) => t.predefined)
}
