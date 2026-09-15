import type { Domain, SearchTemplate } from './types'

const PREFIX = 'kaizengo.search.templates.'

function key(model: string) {
  return PREFIX + model.trim()
}

function readAll(model: string): SearchTemplate[] {
  try {
    const raw = localStorage.getItem(key(model))
    if (!raw) return []
    const parsed = JSON.parse(raw) as SearchTemplate[]
    return Array.isArray(parsed) ? parsed : []
  } catch {
    return []
  }
}

function writeAll(model: string, items: SearchTemplate[]) {
  try {
    localStorage.setItem(key(model), JSON.stringify(items))
  } catch {
    /* ignore quota */
  }
}

export function listSearchTemplates(model: string): SearchTemplate[] {
  return readAll(model).sort((a, b) => a.name.localeCompare(b.name))
}

export function saveSearchTemplate(input: {
  model: string
  name: string
  q: string
  searchIn: string[]
  domain: Domain
  groupBy: string[]
  id?: string
}): SearchTemplate {
  const now = new Date().toISOString()
  const items = readAll(input.model)
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
    createdAt: existing >= 0 ? items[existing].createdAt : now,
    updatedAt: now,
  }
  if (existing >= 0) items[existing] = next
  else items.push(next)
  writeAll(input.model, items)
  return next
}

export function deleteSearchTemplate(model: string, id: string) {
  writeAll(
    model,
    readAll(model).filter((t) => t.id !== id),
  )
}

export function renameSearchTemplate(model: string, id: string, name: string) {
  const items = readAll(model)
  const i = items.findIndex((t) => t.id === id)
  if (i < 0) return
  items[i] = { ...items[i], name: name.trim() || items[i].name, updatedAt: new Date().toISOString() }
  writeAll(model, items)
}
