import type { ReactNode } from 'react'
import { Button } from '@/components/ui/button'
import type { ModelRecord } from '@/lib/model-client'
import type { Column } from '@/lib/types'
import { cn } from '@/lib/utils'
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
  emptyMessage?: string
  actions?: (row: T) => ReactNode
  keyField?: string
  keyOf?: (row: T) => string
  className?: string
}

/** Presentational table — no fetching. */
export function KTableView<T extends Record<string, unknown>>(props: KTableViewProps<T>) {
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
          {props.rows.map((row, index) => {
            const key =
              props.keyOf?.(row) ??
              String(row[props.keyField ?? 'id'] ?? index)
            return (
              <tr key={key} className="border-b border-[var(--kg-border)]">
                {props.columns.map((col) => (
                  <td
                    key={col.key}
                    className={cn('px-4 py-3', col.mono && 'font-mono text-xs')}
                    style={{ textAlign: col.align ?? 'left' }}
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
