import { useMemo, useState, type ReactNode } from 'react'
import { BarChart3, Columns3, Grid3x3, Table2 } from 'lucide-react'
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
import { KChartView, chartConfigFromPreset, DEFAULT_CHART_TYPES, type KChartMeasure, type KChartType } from './KChart'
import { KKanbanBoard, type KKanbanColumn } from './KKanban'
import { KPagination } from './KPagination'
import { KPivotView, type KPivotMeasure } from './KPivot'
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
import { useCollectionViewParam } from './url-state'

export type KCollectionView = 'table' | 'kanban' | 'chart' | 'pivot'

export type KCollectionChartConfig = {
  /** Prefer a named appspec chart preset when available. */
  chartId?: string
  xField?: string
  yField?: string
  measure?: KChartMeasure
  seriesField?: string
  type?: KChartType
  types?: KChartType[]
}

export type KCollectionPivotConfig = {
  rowFields?: string[]
  colFields?: string[]
  measureField?: string
  measure?: KPivotMeasure
}

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

const DEFAULT_VIEWS: KCollectionView[] = ['table', 'kanban']

type SharedProps = {
  view?: KCollectionView
  defaultView?: KCollectionView
  onViewChange?: (view: KCollectionView) => void
  /** Which views appear in the toggle (default: table + kanban). */
  views?: KCollectionView[]
  /** Sync selected view to `?view=` (default true when uncontrolled). */
  url?: boolean
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
  chart?: KCollectionChartConfig
  pivot?: KCollectionPivotConfig
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
  chart?: KCollectionChartConfig
  pivot?: KCollectionPivotConfig
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

const VIEW_META: Record<
  KCollectionView,
  { label: string; icon: typeof Table2 }
> = {
  table: { label: 'Table', icon: Table2 },
  kanban: { label: 'Kanban', icon: Columns3 },
  chart: { label: 'Chart', icon: BarChart3 },
  pivot: { label: 'Pivot', icon: Grid3x3 },
}

function normalizeViews(views?: KCollectionView[]): KCollectionView[] {
  const list = views?.length ? views : DEFAULT_VIEWS
  return [...new Set(list)]
}

function initialView(
  views: KCollectionView[],
  preferred?: KCollectionView,
): KCollectionView {
  if (preferred && views.includes(preferred)) return preferred
  return views[0] ?? 'table'
}

function ViewToggle(props: {
  view: KCollectionView
  views: KCollectionView[]
  onChange: (view: KCollectionView) => void
}) {
  if (props.views.length <= 1) return null

  return (
    <div
      className="inline-flex shrink-0 border border-[var(--kg-border-strong)] bg-[var(--kg-surface)]"
      role="group"
      aria-label="Collection view"
    >
      {props.views.map((id) => {
        const meta = VIEW_META[id]
        const Icon = meta.icon
        return (
          <Button
            key={id}
            type="button"
            size="sm"
            variant={props.view === id ? 'secondary' : 'ghost'}
            aria-pressed={props.view === id}
            aria-label={meta.label}
            onClick={() => props.onChange(id)}
          >
            <Icon className="size-4" aria-hidden />
            <span className="sr-only">{meta.label}</span>
          </Button>
        )
      })}
    </div>
  )
}

function fieldLabelOf(
  fields: Array<{ key: string; label: string }>,
  field: string,
): string {
  return fields.find((f) => f.key === field)?.label ?? field
}

function defaultCategoryField(
  columns: Column<ModelRecord>[],
  groupBy: string[],
  explicit?: string,
): string {
  if (explicit?.trim()) return explicit.trim()
  if (groupBy[0]?.trim()) return groupBy[0].trim()
  const skip = new Set(['id', 'updatedAt', 'createdAt'])
  const col = columns.find((c) => !skip.has(c.key))
  return col?.key ?? columns[0]?.key ?? ''
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
  const availableViews = useMemo(
    () => normalizeViews(props.views),
    [props.views],
  )
  const defaultView = initialView(availableViews, props.defaultView ?? 'kanban')
  const useUrl = props.view == null && props.url !== false
  const [urlView, setUrlView] = useCollectionViewParam({
    url: useUrl,
    allowed: availableViews,
    defaultView,
  })
  const view = (props.view ?? urlView) as KCollectionView

  function setView(next: KCollectionView) {
    if (!availableViews.includes(next)) return
    if (props.view == null) setUrlView(next)
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
  const pivotFields = defaultPivotFields(
    columns,
    kq.groupByFields,
    props.pivot?.rowFields,
    props.pivot?.colFields,
  )
  const chartPresets = useMemo(
    () =>
      (kq.view?.chartPresets ?? [])
        .map((p) => chartConfigFromPreset(p))
        .filter((p): p is NonNullable<typeof p> => Boolean(p)),
    [kq.view?.chartPresets],
  )

  return (
    <KQueryShell
      query={kq}
      emptyMessage={props.emptyMessage}
      className={props.className ?? props.class}
      toolbar={
        <div className="flex justify-end">
          <ViewToggle view={view} views={availableViews} onChange={setView} />
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
          nestedGroupField={kq.nestedGroupField || undefined}
          nestedGroupLabel={(value) => {
            const field = kq.searchFields.find((f) => f.key === kq.nestedGroupField)
            return field ? `${field.label}: ${value}` : value
          }}
          card={props.card}
          keyOf={props.keyOf}
        />
      ) : view === 'chart' ? (
        <KChartView
          rows={kq.items}
          presets={chartPresets}
          chartId={props.chart?.chartId}
          xField={
            props.chart?.xField?.trim() ||
            (chartPresets.length
              ? undefined
              : defaultCategoryField(columns, kq.groupByFields))
          }
          yField={props.chart?.yField}
          measure={props.chart?.measure}
          seriesField={
            props.chart?.seriesField ?? (kq.nestedGroupField || undefined)
          }
          type={props.chart?.type}
          types={props.chart?.types}
          fields={kq.searchFields}
          model={kq.modelRef || undefined}
          fieldLabel={(field) => fieldLabelOf(kq.searchFields, field)}
          url={props.url !== false}
        />
      ) : view === 'pivot' ? (
        <KPivotView
          rows={kq.items}
          rowFields={pivotFields.rowFields}
          colFields={pivotFields.colFields}
          measureField={props.pivot?.measureField}
          measure={props.pivot?.measure}
          fields={kq.searchFields}
          fieldLabel={(field) => fieldLabelOf(kq.searchFields, field)}
          url={props.url !== false}
        />
      ) : (
        <KTableView
          columns={columns}
          rows={kq.items}
          groupBy={kq.groupByFields}
          fieldLabel={(field) => fieldLabelOf(kq.searchFields, field)}
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

  const availableViews = useMemo(
    () => normalizeViews(props.views),
    [props.views],
  )
  const defaultView = initialView(availableViews, props.defaultView ?? 'kanban')
  const useUrl = props.view == null && props.url !== false
  const [urlView, setUrlView] = useCollectionViewParam({
    url: useUrl,
    allowed: availableViews,
    defaultView,
  })
  const [query, setQuery] = useState('')
  const [internalGroupBy, setInternalGroupBy] = useState(
    props.defaultGroupBy ?? props.groupBy?.[0]?.id ?? '',
  )
  const [filterValues, setFilterValues] = useState<Record<string, string>>({})

  const view = (props.view ?? urlView) as KCollectionView
  const groupById = props.groupByValue ?? internalGroupBy

  function setView(next: KCollectionView) {
    if (!availableViews.includes(next)) return
    if (props.view == null) setUrlView(next)
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

          <ViewToggle view={view} views={availableViews} onChange={setView} />
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
      ) : view === 'chart' ? (
        <KChartView
          rows={visible}
          xField={
            props.chart?.xField ??
            (activeGroup && groupById !== NONE ? activeGroup.id : props.columns[0]?.key ?? '')
          }
          yField={props.chart?.yField}
          measure={props.chart?.measure}
          seriesField={props.chart?.seriesField}
          type={props.chart?.type}
          types={props.chart?.types ?? DEFAULT_CHART_TYPES}
          emptyMessage={props.emptyMessage}
          url={props.url !== false}
        />
      ) : view === 'pivot' ? (
        <KPivotView
          rows={visible}
          rowFields={
            props.pivot?.rowFields ??
            (activeGroup && groupById !== NONE ? [activeGroup.id] : [props.columns[0]?.key ?? ''])
          }
          colFields={props.pivot?.colFields}
          measureField={props.pivot?.measureField}
          measure={props.pivot?.measure}
          fields={props.columns.map((c) => ({ key: c.key, label: c.label ?? c.key }))}
          emptyMessage={props.emptyMessage}
          url={props.url !== false}
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
