export type MeasureKind = 'count' | 'sum' | 'avg'

export type ChartDatum = {
  label: string
  value: number
  series?: string
}

function cellKey(rowKey: string, colKey: string): string {
  return `${rowKey}\0${colKey}`
}

function numericValue(value: unknown): number | null {
  if (value == null || value === '') return null
  const n = Number(value)
  return Number.isFinite(n) ? n : null
}

function measureValue(
  values: unknown[],
  measure: MeasureKind,
  measureField?: string,
): number {
  if (measure === 'count') return values.length
  const nums = values
    .map((row) => {
      const rec = row as Record<string, unknown>
      return numericValue(measureField ? rec[measureField] : row)
    })
    .filter((n): n is number => n != null)
  if (!nums.length) return 0
  const sum = nums.reduce((a, b) => a + b, 0)
  if (measure === 'sum') return sum
  return sum / nums.length
}

export function aggregateChart(
  rows: Record<string, unknown>[],
  xField: string,
  options?: {
    yField?: string
    measure?: MeasureKind
    seriesField?: string
  },
): ChartDatum[] {
  const measure = options?.measure ?? (options?.yField ? 'sum' : 'count')
  const measureField = options?.yField
  const seriesField = options?.seriesField?.trim()

  const buckets = new Map<string, Record<string, unknown>[]>()
  const order: string[] = []

  for (const row of rows) {
    const x = String(row[xField] ?? '')
    const series = seriesField ? String(row[seriesField] ?? '') : ''
    const key = seriesField ? `${x}\0${series}` : x
    if (!buckets.has(key)) {
      buckets.set(key, [])
      order.push(key)
    }
    buckets.get(key)!.push(row)
  }

  return order.map((key) => {
    const groupRows = buckets.get(key)!
    const [label, series] = seriesField ? key.split('\0') : [key, undefined]
    return {
      label: label || '(empty)',
      series: series || undefined,
      value: measureValue(groupRows, measure, measureField),
    }
  })
}

export type PivotData = {
  rowOrder: string[]
  colOrder: string[]
  rowLabels: Map<string, string>
  colLabels: Map<string, string>
  cells: Map<string, number>
  rowTotals: Map<string, number>
  colTotals: Map<string, number>
  grandTotal: number
}

function compositeKey(row: Record<string, unknown>, fields: string[]): string {
  return fields.map((f) => String(row[f] ?? '')).join('\0')
}

function compositeLabel(row: Record<string, unknown>, fields: string[]): string {
  return fields.map((f) => String(row[f] ?? '') || '(empty)').join(' / ')
}

export function buildPivot(
  rows: Record<string, unknown>[],
  rowFields: string[],
  colFields: string[],
  options?: {
    measureField?: string
    measure?: MeasureKind
  },
): PivotData {
  const measure = options?.measure ?? (options?.measureField ? 'sum' : 'count')
  const measureField = options?.measureField

  const rowOrder: string[] = []
  const colOrder: string[] = []
  const rowLabels = new Map<string, string>()
  const colLabels = new Map<string, string>()
  const groups = new Map<string, Record<string, unknown>[]>()

  for (const row of rows) {
    const rowKey = rowFields.length ? compositeKey(row, rowFields) : '(all)'
    const colKey = colFields.length ? compositeKey(row, colFields) : '(total)'
    if (!rowLabels.has(rowKey)) {
      rowLabels.set(
        rowKey,
        rowFields.length ? compositeLabel(row, rowFields) : 'All',
      )
      rowOrder.push(rowKey)
    }
    if (!colLabels.has(colKey)) {
      colLabels.set(
        colKey,
        colFields.length ? compositeLabel(row, colFields) : 'Total',
      )
      colOrder.push(colKey)
    }
    const key = cellKey(rowKey, colKey)
    if (!groups.has(key)) groups.set(key, [])
    groups.get(key)!.push(row)
  }

  const cells = new Map<string, number>()
  const rowTotals = new Map<string, number>()
  const colTotals = new Map<string, number>()
  let grandTotal = 0

  for (const rowKey of rowOrder) {
    for (const colKey of colOrder) {
      const key = cellKey(rowKey, colKey)
      const groupRows = groups.get(key) ?? []
      const value = measureValue(groupRows, measure, measureField)
      cells.set(key, value)
      rowTotals.set(rowKey, (rowTotals.get(rowKey) ?? 0) + value)
      colTotals.set(colKey, (colTotals.get(colKey) ?? 0) + value)
      grandTotal += value
    }
  }

  return {
    rowOrder,
    colOrder,
    rowLabels,
    colLabels,
    cells,
    rowTotals,
    colTotals,
    grandTotal,
  }
}

export function formatMeasure(value: number, measure: MeasureKind): string {
  if (measure === 'count') return String(Math.round(value))
  if (Number.isInteger(value)) return String(value)
  return value.toLocaleString(undefined, { maximumFractionDigits: 2 })
}
