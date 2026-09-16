import type { GroupBucket } from '@/lib/model-client'

export type GroupColumn = {
  id: string
  label: string
  count?: number
}

/** Distinct top-level columns from composite group buckets (fixes nested duplicate lanes). */
export function topLevelGroupColumns(
  columnField: string,
  groups: GroupBucket[],
  items: Record<string, unknown>[],
): GroupColumn[] {
  if (!columnField) return [{ id: 'all', label: 'All' }]

  const order: string[] = []
  const counts = new Map<string, number>()
  for (const bucket of groups) {
    const id = bucket.values[0] ?? ''
    if (!counts.has(id)) order.push(id)
    counts.set(id, (counts.get(id) ?? 0) + bucket.count)
  }
  if (order.length) {
    return order.map((id) => ({
      id,
      label: id || '(empty)',
      count: counts.get(id),
    }))
  }

  const seen = new Set<string>()
  const fallback: string[] = []
  for (const item of items) {
    const id = String(item[columnField] ?? '')
    if (!seen.has(id)) {
      seen.add(id)
      fallback.push(id)
    }
  }
  return fallback.map((id) => ({ id, label: id || '(empty)' }))
}

export type GroupedTableRow<T extends Record<string, unknown>> =
  | {
      kind: 'header'
      key: string
      level: number
      field: string
      fieldLabel: string
      value: string
      count: number
    }
  | { kind: 'row'; key: string; level: number; data: T }

export function buildGroupedTableRows<T extends Record<string, unknown>>(
  items: T[],
  groupBy: string[],
  labelOf: (field: string) => string,
  keyOf: (row: T, index: number) => string,
): GroupedTableRow<T>[] {
  if (!groupBy.length) {
    return items.map((data, i) => ({ kind: 'row', key: keyOf(data, i), level: 0, data }))
  }

  const out: GroupedTableRow<T>[] = []

  function walk(rows: T[], fields: string[], level: number, prefix: string) {
    if (!fields.length) {
      rows.forEach((data, i) => {
        out.push({ kind: 'row', key: `${prefix}-row-${keyOf(data, i)}`, level, data })
      })
      return
    }
    const [field, ...rest] = fields
    const buckets = new Map<string, T[]>()
    const order: string[] = []
    for (const row of rows) {
      const val = String(row[field] ?? '')
      if (!buckets.has(val)) {
        buckets.set(val, [])
        order.push(val)
      }
      buckets.get(val)!.push(row)
    }
    for (const val of order) {
      const groupRows = buckets.get(val)!
      const groupKey = `${prefix}-${field}-${val}`
      out.push({
        kind: 'header',
        key: groupKey,
        level,
        field,
        fieldLabel: labelOf(field),
        value: val || '(empty)',
        count: groupRows.length,
      })
      walk(groupRows, rest, level + 1, groupKey)
    }
  }

  walk(items, groupBy, 0, 'g')
  return out
}

export function itemsInColumn<T extends Record<string, unknown>>(
  items: T[],
  columnField: string,
  columnId: string,
): T[] {
  if (columnId === 'all') return items
  return items.filter((item) => String(item[columnField] ?? '') === columnId)
}

export type NestedColumnGroup<T> = {
  id: string
  label: string
  items: T[]
}

export function nestedGroupsInColumn<T extends Record<string, unknown>>(
  items: T[],
  nestedField: string,
): NestedColumnGroup<T>[] {
  const buckets = new Map<string, T[]>()
  const order: string[] = []
  for (const item of items) {
    const id = String(item[nestedField] ?? '')
    if (!buckets.has(id)) {
      buckets.set(id, [])
      order.push(id)
    }
    buckets.get(id)!.push(item)
  }
  return order.map((id) => ({
    id,
    label: id || '(empty)',
    items: buckets.get(id)!,
  }))
}
