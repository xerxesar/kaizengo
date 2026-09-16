import type { ReactNode } from 'react'
import type { ModelRecord } from '@/lib/model-client'
import { cn } from '@/lib/utils'
import { nestedGroupsInColumn } from './group-by'
import { KQueryShell } from './KQueryShell'
import {
  useKQuery,
  type KQueryPaginationConfig,
  type KQuerySearchConfig,
} from './useKQuery'

export type KKanbanColumn = {
  id: string
  label: string
}

export type KKanbanPaginationConfig = KQueryPaginationConfig
export type KKanbanSearchConfig = KQuerySearchConfig

export type KKanbanBoardProps<T> = {
  items: T[]
  columns: KKanbanColumn[]
  columnOf: (item: T) => string
  /** Second-level group field — subdivides cards inside each column. */
  nestedGroupField?: string
  nestedGroupLabel?: (value: string) => string
  card: (item: T) => ReactNode
  keyOf?: (item: T) => string
  emptyMessage?: string
  className?: string
}

/** Presentational kanban board — no fetching / pagination. */
export function KKanbanBoard<T>(props: KKanbanBoardProps<T>) {
  function keyOf(item: T, index: number): string {
    if (props.keyOf) return props.keyOf(item)
    const rec = item as Record<string, unknown>
    if (rec.id != null) return String(rec.id)
    if (rec.name != null) return String(rec.name)
    return String(index)
  }

  function renderCards(items: T[]) {
    if (!items.length) {
      return (
        <p className="col-span-full px-1 py-6 text-center text-xs text-[var(--kg-text-muted)]">
          Empty
        </p>
      )
    }
    return items.map((item, i) => (
      <div
        key={keyOf(item, i)}
        className="h-full min-h-0 min-w-0 overflow-hidden border border-[var(--kg-border)] bg-[var(--kg-surface)] p-4 shadow-sm [&_:is(h1,h2,h3,h4,h5,h6)]:truncate [&_p]:line-clamp-2"
        data-kanban-card={keyOf(item, i)}
      >
        <div className="h-full min-h-0 overflow-hidden">{props.card(item)}</div>
      </div>
    ))
  }

  if (props.items.length === 0) {
    return (
      <div className="rounded border border-dashed border-[var(--kg-border)] bg-[var(--kg-surface)] px-6 py-10 text-center">
        <p className="text-base font-medium text-[var(--kg-text)]">
          {props.emptyMessage ?? 'No records found'}
        </p>
      </div>
    )
  }

  return (
    <div
      className={cn(
        'grid auto-cols-[minmax(16rem,1fr)] grid-flow-col gap-4 overflow-x-auto pb-2',
        props.className,
      )}
    >
      {props.columns.map((col) => {
        const columnItems = props.items.filter((item) => props.columnOf(item) === col.id)
        const nestedField = props.nestedGroupField?.trim()
        const subGroups = nestedField
          ? nestedGroupsInColumn(columnItems as Record<string, unknown>[], nestedField)
          : null

        return (
          <section
            key={col.id}
            className="flex min-h-[12rem] min-w-[16rem] flex-col border border-[var(--kg-border)] bg-[var(--kg-surface-muted,var(--kg-field-hover))]"
          >
            <header className="flex items-center justify-between gap-2 border-b border-[var(--kg-border)] px-4 py-3">
              <h3 className="text-sm font-semibold text-[var(--kg-text)]">{col.label}</h3>
              <span className="text-xs tabular-nums text-[var(--kg-text-muted)]">{columnItems.length}</span>
            </header>
            {subGroups ? (
              <div className="flex flex-1 flex-col gap-3 p-3">
                {subGroups.map((sub) => (
                  <div
                    key={sub.id}
                    className="rounded border border-[var(--kg-border)] bg-[var(--kg-surface)]/60 p-2"
                  >
                    <div className="mb-2 flex items-center justify-between gap-2 px-1">
                      <h4 className="text-xs font-semibold uppercase tracking-wide text-[var(--kg-text-muted)]">
                        {props.nestedGroupLabel?.(sub.label) ?? sub.label}
                      </h4>
                      <span className="text-xxs tabular-nums text-[var(--kg-text-muted)]">
                        {sub.items.length}
                      </span>
                    </div>
                    <div className="grid auto-rows-[9rem] grid-cols-[repeat(auto-fill,minmax(12rem,1fr))] content-start gap-3">
                      {renderCards(sub.items as T[])}
                    </div>
                  </div>
                ))}
              </div>
            ) : (
              <div className="grid flex-1 auto-rows-[9rem] grid-cols-[repeat(auto-fill,minmax(12rem,1fr))] content-start gap-3 p-3">
                {renderCards(columnItems)}
              </div>
            )}
          </section>
        )
      })}
    </div>
  )
}

type QueryProps = {
  query?: string
  model?: string
  card: (item: ModelRecord) => ReactNode
  /** Field used to place cards into columns (also used when KSearch has no groupBy). */
  groupByField?: string
  /** Explicit columns; defaults to useKQuery groupColumns. */
  columns?: KKanbanColumn[]
  keyOf?: (item: ModelRecord) => string
  paginated?: boolean | KKanbanPaginationConfig
  searchable?: boolean | KKanbanSearchConfig
  emptyMessage?: string
  onerror?: (message: string) => void
  className?: string
  /** @deprecated use className */
  class?: string
  refreshToken?: number
}

type ItemsProps<T> = {
  items: T[]
  columns: KKanbanColumn[]
  columnOf: (item: T) => string
  card: (item: T) => ReactNode
  keyOf?: (item: T) => string
  emptyMessage?: string
  className?: string
  /** @deprecated use className */
  class?: string
  // Disallow query props on items mode
  query?: undefined
  model?: undefined
}

export type KKanbanProps<T = ModelRecord> = QueryProps | ItemsProps<T>

function isQueryMode(props: { query?: string; model?: string }): props is QueryProps {
  return Boolean(props.query?.trim() || props.model?.trim())
}

/**
 * Kanban board. Prefer `query`/`model` (fetches via useKQuery).
 * Pass `items` + `columnOf` for a presentational escape hatch.
 */
export function KKanban<T = ModelRecord>(props: KKanbanProps<T>) {
  if (isQueryMode(props)) {
    return <KKanbanQuery {...(props as QueryProps)} />
  }
  const itemsProps = props as ItemsProps<T>
  return (
    <KKanbanBoard
      items={itemsProps.items}
      columns={itemsProps.columns}
      columnOf={itemsProps.columnOf}
      card={itemsProps.card}
      keyOf={itemsProps.keyOf}
      emptyMessage={itemsProps.emptyMessage}
      className={itemsProps.className ?? itemsProps.class}
    />
  )
}

function KKanbanQuery(props: QueryProps) {
  const kq = useKQuery({
    query: props.query,
    model: props.model,
    paginated: props.paginated,
    searchable: props.searchable,
    groupByField: props.groupByField,
    refreshToken: props.refreshToken,
    onerror: props.onerror,
  })

  const columns = props.columns ?? kq.groupColumns
  const columnField = kq.columnField

  return (
    <KQueryShell
      query={kq}
      emptyMessage={props.emptyMessage}
      className={props.className ?? props.class}
    >
      <KKanbanBoard
        items={kq.items}
        columns={columns}
        columnOf={(item) =>
          columnField ? String(item[columnField] ?? '') : 'all'
        }
        nestedGroupField={kq.nestedGroupField || undefined}
        nestedGroupLabel={(value) => {
          const field = kq.searchFields.find((f) => f.key === kq.nestedGroupField)
          return field ? `${field.label}: ${value}` : value
        }}
        card={props.card}
        keyOf={props.keyOf}
      />
    </KQueryShell>
  )
}
