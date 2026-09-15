import { useMemo, useState, type ReactNode } from 'react'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import { Progress } from '@/components/ui/progress'
import type { ModelRecord } from '@/lib/model-client'
import type { Column } from '@/lib/types'
import { cn } from '@/lib/utils'
import { KKanbanBoard, type KKanbanColumn } from './KKanban'
import { KPagination } from './KPagination'
import { KQueryShell } from './KQueryShell'
import { KTableView } from './KTable'
import {
  DEFAULT_PAGE_SIZE,
  sliceItems,
  usePaginationParams,
  type PaginationParamsOptions,
} from './pagination'
import {
  enableConfig,
  useKQuery,
  type KQueryPaginationConfig,
  type KQuerySearchConfig,
} from './useKQuery'

export type KCollectionView = 'table' | 'kanban'

export type KCollectionGroupBy<T> = {
  id: string
  label: string
  valueOf: (item: T) => string
  labelOf?: (value: string) => string
  columns?: KKanbanColumn[]
}

export type KCollectionFilter<T> = {
  id: string
  label: string
  valueOf: (item: T) => string | boolean | Array<string | boolean>
  options?: Array<{ value: string; label: string }>
}

export type KCollectionPaginationConfig = KQueryPaginationConfig
export type KCollectionSearchConfig = KQuerySearchConfig

type SharedProps = {
  view?: KCollectionView
  defaultView?: KCollectionView
  onViewChange?: (view: KCollectionView) => void
  emptyMessage?: string
  className?: string
  /** @deprecated use className */
  class?: string
}

type QueryProps = SharedProps & {
  query?: string
  model?: string
  card: (item: ModelRecord) => ReactNode
  paginated?: boolean | KCollectionPaginationConfig
  searchable?: boolean | KCollectionSearchConfig
  /** Kanban column field (defaults to KSearch groupBy[0]). */
  groupByField?: string
  /** Override table columns from the list view. */
  columns?: Column<ModelRecord>[]
  keyOf?: (item: ModelRecord) => string
  actions?: (item: ModelRecord) => ReactNode
  deletable?: boolean
  onerror?: (message: string) => void
  refreshToken?: number
  items?: undefined
}

type ItemsProps<T extends Record<string, unknown>> = SharedProps & {
  items: T[]
  columns: Column<T>[]
  card: (item: T) => ReactNode
  searchIn?: Array<keyof T | ((item: T) => string)>
  searchPlaceholder?: string
  groupBy?: Array<KCollectionGroupBy<T>>
  defaultGroupBy?: string
  groupByValue?: string
  onGroupByChange?: (id: string) => void
  filters?: Array<KCollectionFilter<T>>
  keyOf?: (item: T) => string
  keyField?: string
  actions?: (item: T) => ReactNode
  /** Client-side pagination over filtered items (items mode only). */
  paginated?: boolean | KCollectionPaginationConfig
  loading?: boolean
  query?: undefined
  model?: undefined
}

export type KCollectionProps<T extends Record<string, unknown> = ModelRecord> =
  | QueryProps
  | ItemsProps<T>

type SelectOption = { value: string; label: string }

const ALL = '__all__'
const NONE = '__none__'

function asStrings(value: string | boolean | Array<string | boolean>): string[] {
  const list = Array.isArray(value) ? value : [value]
  return list.map((v) => String(v))
}

function isQueryMode(props: { query?: string; model?: string }): props is QueryProps {
  return Boolean(props.query?.trim() || props.model?.trim())
}

function ViewToggle(props: {
  view: KCollectionView
  onChange: (view: KCollectionView) => void
}) {
  return (
    <div
      className="inline-flex shrink-0 border border-[var(--kg-border-strong)] bg-[var(--kg-surface)]"
      role="group"
      aria-label="Collection view"
    >
      <Button
        type="button"
        size="sm"
        variant={props.view === 'table' ? 'secondary' : 'ghost'}
        aria-pressed={props.view === 'table'}
        onClick={() => props.onChange('table')}
      >
        Table
      </Button>
      <Button
        type="button"
        size="sm"
        variant={props.view === 'kanban' ? 'secondary' : 'ghost'}
        aria-pressed={props.view === 'kanban'}
        onClick={() => props.onChange('kanban')}
      >
        Kanban
      </Button>
    </div>
  )
}

function OptionSelect(props: {
  label: string
  value: string
  options: SelectOption[]
  onChange: (value: string) => void
  className?: string
}) {
  return (
    <div className={cn('flex min-w-[9rem] flex-col gap-1', props.className)}>
      <span className="text-xxs font-medium uppercase tracking-wide text-[var(--kg-text-muted)]">
        {props.label}
      </span>
      <Select value={props.value} onValueChange={props.onChange}>
        <SelectTrigger aria-label={props.label}>
          <SelectValue />
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

/**
 * Collection shell: table | kanban over one data source.
 * Prefer `query`/`model` (useKQuery + server search/pagination).
 * Pass `items` for non-CQRS lists (client filter / optional client page).
 */
export function KCollection<T extends Record<string, unknown> = ModelRecord>(
  props: KCollectionProps<T>,
) {
  if (isQueryMode(props)) {
    return <KCollectionQuery {...(props as QueryProps)} />
  }
  return <KCollectionItems {...(props as ItemsProps<T>)} />
}

function KCollectionQuery(props: QueryProps) {
  const [internalView, setInternalView] = useState<KCollectionView>(
    props.defaultView ?? 'kanban',
  )
  const view = props.view ?? internalView

  function setView(next: KCollectionView) {
    if (props.view == null) setInternalView(next)
    props.onViewChange?.(next)
  }

  const kq = useKQuery({
    query: props.query,
    model: props.model,
    paginated: props.paginated,
    searchable: props.searchable,
    groupByField: props.groupByField,
    refreshToken: props.refreshToken,
    onerror: props.onerror,
  })

  const columns = props.columns ?? kq.columns
  const columnField = kq.columnField
  const kanbanColumns = kq.groupColumns

  return (
    <KQueryShell
      query={kq}
      emptyMessage={props.emptyMessage}
      className={props.className ?? props.class}
      toolbar={
        <div className="flex justify-end">
          <ViewToggle view={view} onChange={setView} />
        </div>
      }
    >
      {view === 'kanban' ? (
        <KKanbanBoard
          items={kq.items}
          columns={kanbanColumns}
          columnOf={(item) =>
            columnField ? String(item[columnField] ?? '') : 'all'
          }
          card={props.card}
          keyOf={props.keyOf}
        />
      ) : (
        <KTableView
          columns={columns}
          rows={kq.items}
          keyOf={props.keyOf}
          actions={
            props.actions ??
            (props.deletable
              ? (row) => (
                  <Button
                    variant="ghost"
                    size="sm"
                    onClick={() => void kq.remove(String(row.id))}
                  >
                    Delete
                  </Button>
                )
              : undefined)
          }
        />
      )}
    </KQueryShell>
  )
}

/** Escape hatch for custom / non-CQRS item lists (e.g. appman). */
function KCollectionItems<T extends Record<string, unknown>>(props: ItemsProps<T>) {
  const paging = enableConfig<KCollectionPaginationConfig>(props.paginated)
  const pageOpts: PaginationParamsOptions = {
    defaultPageSize: paging?.pageSize ?? DEFAULT_PAGE_SIZE,
    pageSizeOptions: paging?.pageSizeOptions,
    pageParam: paging?.pageParam,
    pageSizeParam: paging?.pageSizeParam,
    url: paging ? paging.url !== false : false,
  }
  const pagination = usePaginationParams(pageOpts)

  const [internalView, setInternalView] = useState<KCollectionView>(
    props.defaultView ?? 'kanban',
  )
  const [query, setQuery] = useState('')
  const [internalGroupBy, setInternalGroupBy] = useState(
    props.defaultGroupBy ?? props.groupBy?.[0]?.id ?? '',
  )
  const [filterValues, setFilterValues] = useState<Record<string, string>>({})

  const view = props.view ?? internalView
  const groupById = props.groupByValue ?? internalGroupBy

  function setView(next: KCollectionView) {
    if (props.view == null) setInternalView(next)
    props.onViewChange?.(next)
  }

  function setGroupBy(id: string) {
    if (props.groupByValue == null) setInternalGroupBy(id)
    props.onGroupByChange?.(id)
  }

  function setFilter(id: string, value: string) {
    setFilterValues((prev) => ({ ...prev, [id]: value }))
  }

  function filterValue(id: string): string {
    return filterValues[id] ?? ALL
  }

  const activeGroup = useMemo(
    () => props.groupBy?.find((g) => g.id === groupById),
    [props.groupBy, groupById],
  )

  const searched = useMemo(() => {
    const q = query.trim().toLowerCase()
    if (!q) return props.items
    const keys = props.searchIn
    return props.items.filter((item) => {
      if (!keys?.length) {
        return Object.values(item).some((v) => String(v ?? '').toLowerCase().includes(q))
      }
      return keys.some((key) => {
        const text = typeof key === 'function' ? key(item) : String(item[key] ?? '')
        return text.toLowerCase().includes(q)
      })
    })
  }, [props.items, props.searchIn, query])

  const filtered = useMemo(() => {
    const filters = props.filters ?? []
    if (!filters.length) return searched
    return searched.filter((item) =>
      filters.every((filter) => {
        const selected = filterValue(filter.id)
        if (selected === ALL) return true
        return asStrings(filter.valueOf(item)).includes(selected)
      }),
    )
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [searched, props.filters, filterValues])

  const paged = useMemo(() => {
    if (!paging) return filtered
    return sliceItems(filtered, pagination.page, pagination.pageSize)
  }, [filtered, paging, pagination.page, pagination.pageSize])

  const groupOptions = useMemo<SelectOption[]>(
    () => [
      { value: NONE, label: 'None' },
      ...(props.groupBy ?? []).map((g) => ({ value: g.id, label: g.label })),
    ],
    [props.groupBy],
  )

  function filterOptions(filter: KCollectionFilter<T>): SelectOption[] {
    if (filter.options?.length) {
      return [{ value: ALL, label: 'All' }, ...filter.options]
    }
    const seen = new Map<string, string>()
    for (const item of props.items) {
      for (const raw of asStrings(filter.valueOf(item))) {
        if (!seen.has(raw)) seen.set(raw, raw)
      }
    }
    return [
      { value: ALL, label: 'All' },
      ...[...seen.entries()].map(([value, label]) => ({ value, label })),
    ]
  }

  const kanbanColumns = useMemo<KKanbanColumn[]>(() => {
    const group = activeGroup
    if (!group || groupById === NONE) return [{ id: 'all', label: 'All' }]
    if (group.columns?.length) return group.columns

    const order: string[] = []
    const seen = new Set<string>()
    for (const item of filtered) {
      const id = group.valueOf(item)
      if (!seen.has(id)) {
        seen.add(id)
        order.push(id)
      }
    }
    return order.map((id) => ({
      id,
      label: group.labelOf?.(id) ?? id,
    }))
  }, [activeGroup, filtered, groupById])

  const visible = paging ? paged : filtered

  const pagerProps = paging
    ? {
        total: filtered.length,
        page: pagination.page,
        pageSize: pagination.pageSize,
        onPageChange: pagination.setPage,
        onPageSizeChange: pagination.setPageSize,
        pageSizeOptions: pagination.pageSizeOptions,
        url: false as const,
      }
    : null

  return (
    <div className={cn('flex flex-col gap-4', props.className ?? props.class)}>
      <div className="flex flex-col gap-3 border border-[var(--kg-border)] bg-[var(--kg-surface)] p-3">
        <div className="flex min-h-10 flex-wrap items-end justify-between gap-3">
          <div className="flex min-w-0 flex-1 flex-wrap items-end gap-3">
            <div className="flex min-w-[14rem] max-w-sm flex-1 flex-col gap-1">
              <span className="text-xxs font-medium uppercase tracking-wide text-[var(--kg-text-muted)]">
                Search
              </span>
              <Input
                value={query}
                onChange={(e) => setQuery(e.target.value)}
                placeholder={props.searchPlaceholder ?? 'Search…'}
              />
            </div>

            {groupOptions.length > 1 ? (
              <OptionSelect
                label="Group by"
                value={groupById || NONE}
                options={groupOptions}
                onChange={setGroupBy}
              />
            ) : null}

            {(props.filters ?? []).map((filter) => (
              <OptionSelect
                key={filter.id}
                label={filter.label}
                value={filterValue(filter.id)}
                options={filterOptions(filter)}
                onChange={(value) => setFilter(filter.id, value)}
              />
            ))}
          </div>

          <ViewToggle view={view} onChange={setView} />
        </div>

        {query.trim() || Object.values(filterValues).some((v) => v && v !== ALL) ? (
          <div className="flex flex-wrap items-center gap-2 text-xs text-[var(--kg-text-muted)]">
            <span>
              Showing {filtered.length} of {props.items.length}
            </span>
            <Button
              type="button"
              size="sm"
              variant="ghost"
              onClick={() => {
                setQuery('')
                setFilterValues({})
              }}
            >
              Clear
            </Button>
          </div>
        ) : null}
      </div>

      {pagerProps ? <KPagination {...pagerProps} /> : null}

      {props.loading ? (
        <div className="flex items-center justify-center py-8" role="status" aria-label="Loading">
          <Progress indeterminate className="w-48 max-w-full" />
        </div>
      ) : view === 'kanban' ? (
        <KKanbanBoard
          items={visible}
          columns={kanbanColumns}
          columnOf={(item) =>
            activeGroup && groupById !== NONE ? activeGroup.valueOf(item) : 'all'
          }
          card={props.card}
          keyOf={props.keyOf}
          emptyMessage={props.emptyMessage}
        />
      ) : (
        <KTableView
          columns={props.columns}
          rows={visible}
          keyField={props.keyField ?? 'id'}
          keyOf={props.keyOf}
          emptyMessage={props.emptyMessage}
          actions={props.actions}
        />
      )}

      {pagerProps ? <KPagination {...pagerProps} /> : null}
    </div>
  )
}
