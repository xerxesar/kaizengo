import { useMemo } from 'react'
import type { ModelRecord } from '@/lib/model-client'
import type { Column } from '@/lib/types'
import { cn } from '@/lib/utils'
import { buildPivot, formatMeasure, type MeasureKind } from './aggregate'
import { KQueryShell } from './KQueryShell'
import {
  useKQuery,
  type KQueryPaginationConfig,
  type KQuerySearchConfig,
} from './useKQuery'
import { usePivotUrlState } from './url-state'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'

export type KPivotMeasure = MeasureKind
export type KPivotPaginationConfig = KQueryPaginationConfig
export type KPivotSearchConfig = KQuerySearchConfig

export type KPivotFieldOption = { key: string; label: string }

export type KPivotViewProps<T extends Record<string, unknown> = ModelRecord> = {
  rows: T[]
  rowFields: string[]
  colFields?: string[]
  measureField?: string
  measure?: KPivotMeasure
  /** Available fields for the settings toolbar. */
  fields?: KPivotFieldOption[]
  /** Show row/col/measure controls when fields are provided (default true). */
  toolbar?: boolean
  /** Sync pivot settings to URL (default true). */
  url?: boolean
  fieldLabel?: (field: string) => string
  emptyMessage?: string
  className?: string
}

const MEASURE_OPTIONS: Array<{ value: KPivotMeasure; label: string }> = [
  { value: 'count', label: 'Count' },
  { value: 'sum', label: 'Sum' },
  { value: 'avg', label: 'Average' },
]

const NONE = '__none__'

function cellKey(rowKey: string, colKey: string): string {
  return `${rowKey}\0${colKey}`
}

function normalizeMeasure(raw?: string | null): KPivotMeasure | undefined {
  if (raw === 'count' || raw === 'sum' || raw === 'avg') return raw
  return undefined
}

/** Presentational pivot table — no fetching. */
export function KPivotView<T extends Record<string, unknown>>(props: KPivotViewProps<T>) {
  const syncUrl = props.url !== false
  const pivotUrl = usePivotUrlState({ url: syncUrl })
  const labelOf = props.fieldLabel ?? ((field: string) => field)
  const showToolbar = props.toolbar !== false && Boolean(props.fields?.length)

  const rowFields =
    pivotUrl.hasUrlRows ? pivotUrl.state.rowFields : props.rowFields.filter(Boolean)
  const colFields = pivotUrl.hasUrlRows
    ? pivotUrl.state.colFields
    : (props.colFields?.filter(Boolean) ?? [])
  const measureField = pivotUrl.hasUrlRows
    ? pivotUrl.state.measureField.trim() || undefined
    : props.measureField || undefined
  const measure =
    (pivotUrl.hasUrlRows
      ? normalizeMeasure(pivotUrl.state.measure)
      : undefined) ??
    props.measure ??
    (measureField ? 'sum' : 'count')

  const pivot = useMemo(
    () =>
      buildPivot(props.rows as Record<string, unknown>[], rowFields, colFields, {
        measureField,
        measure,
      }),
    [props.rows, rowFields, colFields, measureField, measure],
  )

  const fieldOptions = props.fields ?? []
  const primaryRow = rowFields[0] ?? ''
  const primaryCol = colFields[0] ?? ''

  function writePivot(next: {
    rowFields?: string[]
    colFields?: string[]
    measureField?: string
    measure?: string
  }) {
    // Once the user edits, seed the full pivot URL so empty cols/measure stick.
    pivotUrl.setAll({
      rowFields: next.rowFields ?? rowFields,
      colFields: next.colFields ?? colFields,
      measureField:
        next.measureField !== undefined ? next.measureField : (measureField ?? ''),
      measure: next.measure !== undefined ? next.measure : measure,
    })
  }

  function setPrimaryRow(next: string) {
    writePivot({ rowFields: next ? [next] : [] })
  }

  function setPrimaryCol(next: string) {
    writePivot({ colFields: next && next !== NONE ? [next] : [] })
  }

  function setMeasureField(next: string) {
    const field = next === NONE ? '' : next
    writePivot({
      measureField: field,
      measure: !field ? 'count' : measure,
    })
  }

  function setMeasure(next: KPivotMeasure) {
    writePivot({ measure: next })
  }

  const toolbar = showToolbar ? (
    <div className="flex flex-wrap items-end gap-3">
      <PivotSelect
        label="Rows"
        value={primaryRow}
        options={fieldOptions.map((f) => ({ value: f.key, label: f.label }))}
        onChange={setPrimaryRow}
        placeholder="Row field"
      />
      <PivotSelect
        label="Columns"
        value={primaryCol || NONE}
        options={[
          { value: NONE, label: 'None' },
          ...fieldOptions.map((f) => ({ value: f.key, label: f.label })),
        ]}
        onChange={setPrimaryCol}
        placeholder="Column field"
      />
      <PivotSelect
        label="Measure field"
        value={measureField || NONE}
        options={[
          { value: NONE, label: 'None (count)' },
          ...fieldOptions.map((f) => ({ value: f.key, label: f.label })),
        ]}
        onChange={setMeasureField}
        placeholder="Measure field"
      />
      <PivotSelect
        label="Measure"
        value={measure}
        options={MEASURE_OPTIONS}
        onChange={(v) => setMeasure(v as KPivotMeasure)}
        placeholder="Measure"
      />
    </div>
  ) : null

  if (!props.rows.length) {
    return (
      <div className={cn('flex flex-col gap-3', props.className)}>
        {toolbar}
        <div className="rounded border border-dashed border-[var(--kg-border)] bg-[var(--kg-surface)] px-6 py-10 text-center">
          <p className="text-base font-medium text-[var(--kg-text)]">
            {props.emptyMessage ?? 'No records found'}
          </p>
        </div>
      </div>
    )
  }

  if (!rowFields.length) {
    return (
      <div className={cn('flex flex-col gap-3', props.className)}>
        {toolbar}
        <div className="rounded border border-dashed border-[var(--kg-border)] bg-[var(--kg-surface)] px-6 py-10 text-center">
          <p className="text-base font-medium text-[var(--kg-text)]">Pivot requires rowFields</p>
        </div>
      </div>
    )
  }

  const showRowTotal = colFields.length > 0
  const showColTotal = colFields.length > 0

  return (
    <div className={cn('flex flex-col gap-3', props.className)}>
      {toolbar}
      <div className="overflow-x-auto border border-[var(--kg-border)] bg-[var(--kg-surface)]">
        <table className="kg-table w-full min-w-[20rem] border-collapse text-sm">
          <thead>
            <tr className="border-b border-[var(--kg-border)] bg-[var(--kg-surface-muted,var(--kg-field-hover))]">
              <th className="px-4 py-3 text-left font-semibold text-[var(--kg-text)]">
                {rowFields.map(labelOf).join(' / ')}
              </th>
              {pivot.colOrder.map((colKey) => (
                <th
                  key={colKey}
                  className="px-4 py-3 text-right font-semibold text-[var(--kg-text)]"
                >
                  {pivot.colLabels.get(colKey) ?? colKey}
                </th>
              ))}
              {showRowTotal ? (
                <th className="px-4 py-3 text-right font-semibold text-[var(--kg-text)]">Total</th>
              ) : null}
            </tr>
          </thead>
          <tbody>
            {pivot.rowOrder.map((rowKey) => (
              <tr key={rowKey} className="border-b border-[var(--kg-border)]">
                <td className="px-4 py-3 font-medium text-[var(--kg-text)]">
                  {pivot.rowLabels.get(rowKey) ?? rowKey}
                </td>
                {pivot.colOrder.map((colKey) => (
                  <td
                    key={colKey}
                    className="px-4 py-3 text-right tabular-nums text-[var(--kg-text)]"
                  >
                    {formatMeasure(
                      pivot.cells.get(cellKey(rowKey, colKey)) ?? 0,
                      measure,
                    )}
                  </td>
                ))}
                {showRowTotal ? (
                  <td className="px-4 py-3 text-right tabular-nums font-medium text-[var(--kg-text)]">
                    {formatMeasure(pivot.rowTotals.get(rowKey) ?? 0, measure)}
                  </td>
                ) : null}
              </tr>
            ))}
            {showColTotal ? (
              <tr className="border-t border-[var(--kg-border)] bg-[var(--kg-surface-muted,var(--kg-field-hover))]">
                <td className="px-4 py-3 font-semibold text-[var(--kg-text)]">Total</td>
                {pivot.colOrder.map((colKey) => (
                  <td
                    key={colKey}
                    className="px-4 py-3 text-right tabular-nums font-semibold text-[var(--kg-text)]"
                  >
                    {formatMeasure(pivot.colTotals.get(colKey) ?? 0, measure)}
                  </td>
                ))}
                <td className="px-4 py-3 text-right tabular-nums font-semibold text-[var(--kg-text)]">
                  {formatMeasure(pivot.grandTotal, measure)}
                </td>
              </tr>
            ) : null}
          </tbody>
        </table>
      </div>
    </div>
  )
}

function PivotSelect(props: {
  label: string
  value: string
  options: Array<{ value: string; label: string }>
  onChange: (value: string) => void
  placeholder?: string
}) {
  return (
    <div className="flex min-w-[9rem] flex-col gap-1">
      <span className="text-xxs font-medium uppercase tracking-wide text-[var(--kg-text-muted)]">
        {props.label}
      </span>
      <Select value={props.value || undefined} onValueChange={props.onChange}>
        <SelectTrigger className="h-8" aria-label={props.label}>
          <SelectValue placeholder={props.placeholder} />
        </SelectTrigger>
        <SelectContent>
          {props.options.map((opt) => (
            <SelectItem key={opt.value} value={opt.value}>
              {opt.label}
            </SelectItem>
          ))}
        </SelectContent>
      </Select>
    </div>
  )
}

type QueryProps = {
  query?: string
  model?: string
  rowFields?: string[]
  colFields?: string[]
  measureField?: string
  measure?: KPivotMeasure
  paginated?: boolean | KPivotPaginationConfig
  searchable?: boolean | KPivotSearchConfig
  /** Sync pivot settings to URL (default true). */
  url?: boolean
  emptyMessage?: string
  onerror?: (message: string) => void
  className?: string
  /** @deprecated use className */
  class?: string
  refreshToken?: number
}

type ItemsProps<T extends Record<string, unknown>> = {
  rows: T[]
  rowFields: string[]
  colFields?: string[]
  measureField?: string
  measure?: KPivotMeasure
  fields?: KPivotFieldOption[]
  fieldLabel?: (field: string) => string
  /** Sync pivot settings to URL (default true). */
  url?: boolean
  emptyMessage?: string
  className?: string
  /** @deprecated use className */
  class?: string
  query?: undefined
  model?: undefined
}

export type KPivotProps<T extends Record<string, unknown> = ModelRecord> =
  | QueryProps
  | ItemsProps<T>

function isQueryMode(props: { query?: string; model?: string }): props is QueryProps {
  return Boolean(props.query?.trim() || props.model?.trim())
}

function defaultPivotFields(
  columns: Column<ModelRecord>[],
  groupBy: string[],
  rowFields?: string[],
  colFields?: string[],
): { rowFields: string[]; colFields: string[] } {
  const rows = rowFields?.filter(Boolean) ?? []
  const cols = colFields?.filter(Boolean) ?? []
  if (rows.length) {
    return { rowFields: rows, colFields: cols }
  }
  if (groupBy[0]?.trim()) {
    return {
      rowFields: [groupBy[0].trim()],
      colFields: groupBy[1]?.trim() ? [groupBy[1].trim()] : cols,
    }
  }
  const skip = new Set(['id', 'updatedAt', 'createdAt'])
  const col = columns.find((c) => !skip.has(c.key))
  return { rowFields: col ? [col.key] : [], colFields: cols }
}

/**
 * Pivot table. Prefer `query`/`model` (fetches via useKQuery).
 * Pass `rows` + `rowFields` for a presentational escape hatch.
 */
export function KPivot<T extends Record<string, unknown> = ModelRecord>(
  props: KPivotProps<T>,
) {
  if (isQueryMode(props)) {
    return <KPivotQuery {...(props as QueryProps)} />
  }
  const itemsProps = props as ItemsProps<T>
  return (
    <KPivotView
      rows={itemsProps.rows}
      rowFields={itemsProps.rowFields}
      colFields={itemsProps.colFields}
      measureField={itemsProps.measureField}
      measure={itemsProps.measure}
      fields={itemsProps.fields}
      fieldLabel={itemsProps.fieldLabel}
      emptyMessage={itemsProps.emptyMessage}
      className={itemsProps.className ?? itemsProps.class}
      url={itemsProps.url}
    />
  )
}

function KPivotQuery(props: QueryProps) {
  const kq = useKQuery({
    query: props.query,
    model: props.model,
    paginated: props.paginated,
    searchable: props.searchable,
    refreshToken: props.refreshToken,
    onerror: props.onerror,
  })

  const { rowFields, colFields } = defaultPivotFields(
    kq.columns,
    kq.groupByFields,
    props.rowFields,
    props.colFields,
  )

  return (
    <KQueryShell
      query={kq}
      emptyMessage={props.emptyMessage}
      className={props.className ?? props.class}
    >
      <KPivotView
        rows={kq.items}
        rowFields={rowFields}
        colFields={colFields}
        measureField={props.measureField}
        measure={props.measure}
        fields={kq.searchFields.map((f) => ({ key: f.key, label: f.label }))}
        fieldLabel={(field) =>
          kq.searchFields.find((f) => f.key === field)?.label ?? field
        }
        url={props.url}
      />
    </KQueryShell>
  )
}
