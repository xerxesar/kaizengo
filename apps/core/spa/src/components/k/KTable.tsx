import { useMemo, type ReactNode } from 'react'
import { Button } from '@/components/ui/button'
import type { ModelRecord } from '@/lib/model-client'
import type { Column } from '@/lib/types'
import { cn } from '@/lib/utils'
import { buildGroupedTableRows } from './group-by'
import { KQueryShell } from './KQueryShell'
import {
  useKQuery,
  type KQueryPaginationConfig,
  type KQuerySearchConfig,
} from './useKQuery'

export type KTablePaginationConfig = KQueryPaginationConfig
export type KTableSearchConfig = KQuerySearchConfig

export type KTableViewProps<T extends Record<string, unknown> = ModelRecord> = {
  columns: Column<T>[]
  rows: T[]
  /** Nested group fields (from KSearch groupBy order). */
  groupBy?: string[]
  fieldLabel?: (field: string) => string
  emptyMessage?: string
  actions?: (row: T) => ReactNode
  keyField?: string
  keyOf?: (row: T) => string
  className?: string
}

const GROUP_HEADER_TONES = [
  'bg-[var(--kg-surface-muted,var(--kg-field-hover))]',
  'bg-[var(--kg-surface)]',
] as const

/** Presentational table — no fetching. */
export function KTableView<T extends Record<string, unknown>>(props: KTableViewProps<T>) {
  const groupBy = props.groupBy?.filter(Boolean) ?? []
  const labelOf = props.fieldLabel ?? ((field: string) => field)

  function rowKey(row: T, index: number): string {
    return props.keyOf?.(row) ?? String(row[props.keyField ?? 'id'] ?? index)
  }

  const groupedRows = useMemo(
    () => buildGroupedTableRows(props.rows, groupBy, labelOf, rowKey),
    [props.rows, groupBy, labelOf, props.keyField, props.keyOf],
  )

  if (!props.rows.length) {
    return (
      <div className="rounded border border-dashed border-[var(--kg-border)] bg-[var(--kg-surface)] px-6 py-10 text-center">
        <p className="text-base font-medium text-[var(--kg-text)]">
          {props.emptyMessage ?? 'No records found'}
        </p>
      </div>
    )
  }

  function cellText(row: T, col: Column<T>): string {
    if (col.render) return col.render(row)
    const val = row[col.key]
    return val == null ? '' : String(val)
  }

  const colSpan = props.columns.length + (props.actions ? 1 : 0)

  return (
    <div
      className={cn(
        'overflow-x-auto border-y border-[var(--kg-border)] bg-[var(--kg-surface)]',
        props.className,
      )}
    >
      <table className="kg-table w-full border-collapse text-sm">
        <thead>
          <tr className="border-b border-[var(--kg-border)] bg-[var(--kg-surface-muted,var(--kg-field-hover))]">
            {props.columns.map((col) => (
              <th
                key={col.key}
                className="px-4 py-3 text-left font-semibold text-[var(--kg-text)]"
                style={{ width: col.width, textAlign: col.align ?? 'left' }}
              >
                {col.label}
              </th>
            ))}
            {props.actions ? (
              <th className="kg-table-actions-col px-4 py-3 text-left">Actions</th>
            ) : null}
          </tr>
        </thead>
        <tbody>
          {groupedRows.map((entry) => {
            if (entry.kind === 'header') {
              const tone = GROUP_HEADER_TONES[entry.level % GROUP_HEADER_TONES.length]
              const indent = 16 + entry.level * 28
              return (
                <tr key={entry.key} className={cn('border-b border-[var(--kg-border)]', tone)}>
                  <td colSpan={colSpan} className="py-2.5 pr-4" style={{ paddingLeft: indent }}>
                    <div className="flex items-baseline gap-2">
                      <span
                        className="inline-block w-1 shrink-0 self-stretch rounded-full bg-[var(--kg-border-strong)]"
                        aria-hidden
                        style={{ minHeight: '1em' }}
                      />
                      <span>
                        <span className="text-xs font-semibold uppercase tracking-wide text-[var(--kg-text-muted)]">
                          {entry.fieldLabel}
                        </span>
                        <span className="mx-1.5 text-[var(--kg-text-muted)]">·</span>
                        <span className="font-semibold text-[var(--kg-text)]">{entry.value}</span>
                        <span className="ml-2 text-xs tabular-nums text-[var(--kg-text-muted)]">
                          ({entry.count})
                        </span>
                      </span>
                    </div>
                  </td>
                </tr>
              )
            }
            const row = entry.data
            return (
              <tr key={entry.key} className="border-b border-[var(--kg-border)]">
                {props.columns.map((col, colIndex) => (
                  <td
                    key={col.key}
                    className={cn('px-4 py-3', col.mono && 'font-mono text-xs')}
                    style={{
                      textAlign: col.align ?? 'left',
                      paddingLeft: colIndex === 0 ? 16 + entry.level * 28 : undefined,
                    }}
                  >
                    {col.cell ? col.cell(row) : cellText(row, col)}
                  </td>
                ))}
                {props.actions ? (
                  <td className="kg-table-actions px-4 py-3">{props.actions(row)}</td>
                ) : null}
              </tr>
            )
          })}
        </tbody>
      </table>
    </div>
  )
}

type Props = {
  query?: string
  model?: string
  emptyMessage?: string
  deletable?: boolean
  /** Server-side pagination via useKQuery. */
  paginated?: boolean | KTablePaginationConfig
  /** Server-side KSearch via useKQuery. */
  searchable?: boolean | KTableSearchConfig
  onerror?: (message: string) => void
  className?: string
  /** @deprecated use className */
  class?: string
  refreshToken?: number
}

/**
 * Query-backed table. Fetches via useKQuery; owns KSearch / KPagination when enabled.
 */
export function KTable(props: Props) {
  const kq = useKQuery({
    query: props.query,
    model: props.model,
    paginated: props.paginated,
    searchable: props.searchable,
    refreshToken: props.refreshToken,
    onerror: props.onerror,
  })

  const deletable = props.deletable ?? true
  const className = props.className ?? props.class

  return (
    <KQueryShell query={kq} emptyMessage={props.emptyMessage} className={className}>
      <KTableView
        columns={kq.columns}
        rows={kq.items}
        groupBy={kq.groupByFields}
        fieldLabel={(field) =>
          kq.searchFields.find((f) => f.key === field)?.label ?? field
        }
        actions={
          deletable
            ? (row) => (
                <Button variant="ghost" size="sm" onClick={() => void kq.remove(String(row.id))}>
                  Delete
                </Button>
              )
            : undefined
        }
      />
    </KQueryShell>
  )
}
