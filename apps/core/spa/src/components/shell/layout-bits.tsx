import type { ReactNode } from 'react'
import { Progress } from '@/components/ui/progress'
import type { Column } from '@/lib/types'
import { cn } from '@/lib/utils'

export function Card({
  title,
  actions,
  className,
  children,
}: {
  title?: string
  actions?: ReactNode
  className?: string
  children?: ReactNode
}) {
  return (
    <section className={cn('border border-[var(--kg-border)] bg-[var(--kg-surface)]', className)}>
      {(title || actions) && (
        <header className="flex items-center justify-between gap-3 border-b border-[var(--kg-border)] px-5 py-4">
          {title ? <h3 className="text-sm font-semibold text-[var(--kg-text)]">{title}</h3> : <span />}
          {actions}
        </header>
      )}
      <div className="p-5">{children}</div>
    </section>
  )
}

export function Toolbar({
  start,
  end,
  children,
}: {
  start?: ReactNode
  end?: ReactNode
  children?: ReactNode
}) {
  return (
    <div className="flex flex-wrap items-center justify-between gap-3">
      <div className="flex flex-wrap items-center gap-2">{start ?? children}</div>
      {end ? <div className="flex flex-wrap items-center gap-2">{end}</div> : null}
    </div>
  )
}

export function StatCard({
  label,
  value,
  hint,
  icon,
}: {
  label: string
  value: ReactNode
  hint?: string
  icon?: ReactNode
}) {
  return (
    <div className="border border-[var(--kg-border)] bg-[var(--kg-surface)] px-5 py-4">
      <div className="flex items-start justify-between gap-3">
        <div>
          <p className="text-xs uppercase tracking-wide text-[var(--kg-text-muted)]">{label}</p>
          <p className="mt-1 text-2xl font-semibold text-[var(--kg-text)]">{value}</p>
          {hint ? <p className="mt-1 text-xs text-[var(--kg-text-secondary)]">{hint}</p> : null}
        </div>
        {icon}
      </div>
    </div>
  )
}

export function FormActions({ children }: { children?: ReactNode }) {
  return <div className="flex flex-wrap items-center gap-3 pt-2">{children}</div>
}

export function FormSection({
  title,
  description,
  children,
}: {
  title: string
  description?: string
  children?: ReactNode
}) {
  return (
    <section className="flex flex-col gap-3">
      <div>
        <h3 className="text-sm font-semibold text-[var(--kg-text)]">{title}</h3>
        {description ? <p className="text-sm text-[var(--kg-text-muted)]">{description}</p> : null}
      </div>
      {children}
    </section>
  )
}

export function DataTable<T extends Record<string, unknown>>({
  columns,
  rows,
  keyField = 'id',
  keyOf,
  loading,
  emptyMessage,
  onRowClick,
  actions,
  className,
}: {
  columns: Column<T>[]
  rows: T[]
  keyField?: string
  keyOf?: (row: T) => string
  loading?: boolean
  emptyMessage?: string
  onRowClick?: (row: T) => void
  actions?: (row: T) => ReactNode
  className?: string
}) {
  if (loading) {
    return (
      <div className="flex items-center justify-center py-8" role="status" aria-label="Loading">
        <Progress indeterminate className="w-48 max-w-full" />
      </div>
    )
  }

  if (!rows.length) {
    return (
      <div className="rounded border border-dashed border-[var(--kg-border)] bg-[var(--kg-surface)] px-6 py-10 text-center">
        <p className="text-base font-medium text-[var(--kg-text)]">{emptyMessage ?? 'No records found'}</p>
      </div>
    )
  }

  return (
    <div className={cn('overflow-x-auto border-y border-[var(--kg-border)] bg-[var(--kg-surface)]', className)}>
      <table className="kg-table w-full border-collapse text-sm">
        <thead>
          <tr className="border-b border-[var(--kg-border)] bg-[var(--kg-surface-muted,var(--kg-field-hover))]">
            {columns.map((col) => (
              <th
                key={col.key}
                className="px-4 py-3 text-left font-semibold text-[var(--kg-text)]"
                style={{ width: col.width, textAlign: col.align ?? 'left' }}
              >
                {col.label}
              </th>
            ))}
            {actions ? <th className="px-4 py-3 text-left">Actions</th> : null}
          </tr>
        </thead>
        <tbody>
          {rows.map((row, index) => {
            const key = keyOf?.(row) ?? String(row[keyField] ?? index)
            return (
              <tr
                key={key}
                className={cn('border-b border-[var(--kg-border)]', onRowClick && 'cursor-pointer hover:bg-[var(--kg-surface-hover)]')}
                onClick={onRowClick ? () => onRowClick(row) : undefined}
              >
                {columns.map((col) => (
                  <td
                    key={col.key}
                    className={cn('px-4 py-3', col.mono && 'font-mono text-xs')}
                    style={{ textAlign: col.align ?? 'left' }}
                  >
                    {col.cell
                      ? col.cell(row)
                      : col.render
                        ? col.render(row)
                        : row[col.key] == null
                          ? ''
                          : String(row[col.key])}
                  </td>
                ))}
                {actions ? <td className="px-4 py-3">{actions(row)}</td> : null}
              </tr>
            )
          })}
        </tbody>
      </table>
    </div>
  )
}
