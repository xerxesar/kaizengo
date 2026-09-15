import { useCallback, useEffect, useMemo, useRef, useState } from 'react'
import {
  deleteModelRecord,
  fetchModelViews,
  fetchViewSlots,
  listModelGroups,
  listModelRecordsPage,
  listViewForModel,
  listViewForQuery,
  parseNamespace,
  resolveCqrsRef,
  type GroupBucket,
  type ModelField,
  type ModelRecord,
  type ModelView,
} from '@/lib/model-client'
import type { Column } from '@/lib/types'
import {
  DEFAULT_PAGE_SIZE,
  usePaginationParams,
  type PaginationParamsOptions,
} from './pagination'
import { useKSearchParams } from './search/params'
import type { SearchField } from './search/types'
import type { KPaginationProps } from './KPagination'
import type { KSearchProps } from './KSearch'

export type KQueryPaginationConfig = {
  pageSize?: number
  pageSizeOptions?: number[]
  pageParam?: string
  pageSizeParam?: string
  /** Sync to URL (default true) */
  url?: boolean
}

export type KQuerySearchConfig = {
  /** Sync to URL (default true) */
  url?: boolean
  searchable?: string[]
  filterable?: string[]
  groupable?: string[]
  placeholder?: string
}

export type UseKQueryOptions = {
  query?: string
  model?: string
  /**
   * Server-side pagination. Pass `true` for defaults, or a config object.
   * When enabled, page/pageSize are sent to listModelRecordsPage.
   */
  paginated?: boolean | KQueryPaginationConfig
  /**
   * Server-side KSearch (q / domain / groupBy).
   * Pass `true` for defaults, or a config object.
   */
  searchable?: boolean | KQuerySearchConfig
  /**
   * Also fetch group buckets (for kanban columns).
   * Defaults to search.groupBy[0] when searchable; override with an explicit field.
   */
  groupByField?: string
  /** Extra GraphQL selection keys beyond view columns. */
  fields?: string[]
  refreshToken?: number
  /** Skip fetching when false (default true). */
  enabled?: boolean
  onerror?: (message: string) => void
}

export type UseKQueryResult = {
  items: ModelRecord[]
  total: number
  columns: Column<ModelRecord>[]
  searchFields: SearchField[]
  loading: boolean
  error: string
  app: string
  model: string
  modelRef: string
  view: ModelView | null
  deleteCommand?: string
  groups: GroupBucket[]
  /** Kanban-ready columns derived from groups / groupByField. */
  groupColumns: Array<{ id: string; label: string }>
  columnField: string
  refresh: () => Promise<void>
  remove: (id: string) => Promise<void>
  paginated: boolean
  searchable: boolean
  page: number
  pageSize: number
  setPage: (page: number) => void
  setPageSize: (pageSize: number) => void
  /** Ready for `<KPagination {...pagerProps} />` (null when not paginated). */
  pagerProps: Omit<KPaginationProps, 'className' | 'class'> | null
  /** Ready for `<KSearch {...searchProps} />` (null until model resolved / not searchable). */
  searchProps: Omit<KSearchProps, 'className'> | null
}

export function enableConfig<T extends object>(
  value: boolean | T | undefined,
): T | null {
  if (!value) return null
  if (value === true) return {} as T
  return value
}

function toSearchFields(
  fields: ModelField[] | undefined,
  columns: { key: string; label: string }[],
): SearchField[] {
  if (fields?.length) {
    return fields.map((f) => ({
      key: f.key,
      label: f.label,
      type: f.type,
      values: f.values,
      relation: f.relation,
    }))
  }
  return columns.map((c) => ({ key: c.key, label: c.label, type: 'string' }))
}

function formatCell(key: string, value: unknown): string {
  if (value == null) return ''
  if (key === 'updatedAt' || key === 'createdAt') {
    try {
      return new Date(String(value)).toLocaleString()
    } catch {
      return String(value)
    }
  }
  return String(value)
}

function buildColumns(
  keys: { key: string; label: string; width?: string; align?: string }[],
): Column<ModelRecord>[] {
  return keys.map((col) => ({
    key: col.key,
    label: col.label,
    width: col.width,
    align: (col.align as 'left' | 'center' | 'right' | undefined) ?? 'left',
    render: (row: ModelRecord) => formatCell(col.key, row[col.key]),
  }))
}

async function resolveListView(
  query?: string,
  model?: string,
): Promise<{ app: string; view: ModelView }> {
  const queryRef = query?.trim()
  if (queryRef) {
    const { app, field } = resolveCqrsRef(queryRef)
    const views = await fetchModelViews(app)
    const view = listViewForQuery(views, field)
    if (!view?.columns?.length) {
      throw new Error(`no list view bound to query ${queryRef} (${field})`)
    }
    return { app, view }
  }

  const modelRef = model?.trim()
  if (modelRef) {
    const { app, name } = parseNamespace(modelRef)
    const views = await fetchModelViews(app)
    const view = listViewForModel(views, name)
    if (!view?.columns?.length) {
      throw new Error(`no list view found for model ${modelRef}`)
    }
    return { app, view }
  }

  throw new Error('useKQuery requires query (preferred) or model')
}

/**
 * Shared CQRS list query: resolves the list view, owns pagination + search
 * params, and always fetches via listModelRecordsPage (server-side).
 */
export function useKQuery(options: UseKQueryOptions): UseKQueryResult {
  const paging = useMemo(
    () => enableConfig<KQueryPaginationConfig>(options.paginated),
    [options.paginated],
  )
  const searching = useMemo(
    () => enableConfig<KQuerySearchConfig>(options.searchable),
    [options.searchable],
  )
  const enabled = options.enabled !== false

  const pageOpts: PaginationParamsOptions = {
    defaultPageSize: paging?.pageSize ?? DEFAULT_PAGE_SIZE,
    pageSizeOptions: paging?.pageSizeOptions,
    pageParam: paging?.pageParam,
    pageSizeParam: paging?.pageSizeParam,
    url: paging ? paging.url !== false : false,
  }
  const pagination = usePaginationParams(pageOpts)
  const resetPage = useCallback(() => pagination.setPage(1), [pagination.setPage])

  const search = useKSearchParams({
    url: searching ? searching.url !== false : false,
    onFilterChange: resetPage,
  })

  const [loading, setLoading] = useState(true)
  const [error, setError] = useState('')
  const [items, setItems] = useState<ModelRecord[]>([])
  const [total, setTotal] = useState(0)
  const [columns, setColumns] = useState<Column<ModelRecord>[]>([])
  const [searchFields, setSearchFields] = useState<SearchField[]>([])
  const [app, setApp] = useState('')
  const [model, setModel] = useState('')
  const [modelRef, setModelRef] = useState('')
  const [view, setView] = useState<ModelView | null>(null)
  const [deleteCommand, setDeleteCommand] = useState<string | undefined>()
  const [groups, setGroups] = useState<GroupBucket[]>([])

  const columnField =
    (searching ? search.groupBy[0]?.trim() : '') ||
    options.groupByField?.trim() ||
    ''

  const searchOptsKey = searching
    ? `${search.q}|${search.searchIn.join(',')}|${JSON.stringify(search.domain)}|${search.groupBy.join(',')}`
    : ''

  const listFilterOpts = searching ? search.toListOpts() : {}
  const searchGroupBy = searching ? search.groupBy : []
  const listFilterOptsRef = useRef(listFilterOpts)
  const searchGroupByRef = useRef(searchGroupBy)
  listFilterOptsRef.current = listFilterOpts
  searchGroupByRef.current = searchGroupBy

  const reportError = useCallback(
    (message: string) => {
      setError(message)
      options.onerror?.(message)
    },
    // eslint-disable-next-line react-hooks/exhaustive-deps
    [options.onerror],
  )

  const refresh = useCallback(async () => {
    if (!enabled) {
      setLoading(false)
      return
    }
    setError('')
    setLoading(true)
    try {
      const resolved = await resolveListView(options.query, options.model)
      const { app: nextApp, view: nextView } = resolved
      setApp(nextApp)
      setModel(nextView.model)
      setModelRef(`${nextApp}.${nextView.model}`)
      setView(nextView)
      setDeleteCommand(nextView.deleteCommand?.trim() || undefined)
      setColumns(buildColumns(nextView.columns ?? []))
      setSearchFields(toSearchFields(nextView.fields, nextView.columns ?? []))

      const fieldKeys = [
        ...(nextView.columns ?? []).map((c) => c.key),
        ...(options.fields ?? []),
        ...(columnField ? [columnField] : []),
      ]
      const listQuery = nextView.listQuery?.trim() || undefined
      const filters = listFilterOptsRef.current
      const listOpts =
        paging || searching
          ? {
              page: paging ? pagination.page : 1,
              pageSize: paging ? pagination.pageSize : 100_000,
              ...filters,
            }
          : undefined

      const page = await listModelRecordsPage(
        nextApp,
        nextView.model,
        fieldKeys,
        listOpts,
        listQuery,
      )
      setItems(page.items)
      setTotal(page.total)

      const activeGroupBy = searchGroupByRef.current
      const groupKeys =
        activeGroupBy.length > 0 ? activeGroupBy : columnField ? [columnField] : []
      if (groupKeys.length) {
        const buckets = await listModelGroups(
          nextApp,
          nextView.model,
          {
            groupBy: groupKeys,
            ...filters,
          },
          listQuery,
        )
        setGroups(buckets)
      } else {
        setGroups([])
      }

      await fetchViewSlots(nextApp, nextView.name)
    } catch (e) {
      reportError(e instanceof Error ? e.message : String(e))
    } finally {
      setLoading(false)
    }
  }, [
    enabled,
    options.query,
    options.model,
    options.fields,
    columnField,
    Boolean(paging),
    Boolean(searching),
    pagination.page,
    pagination.pageSize,
    searchOptsKey,
    reportError,
  ])

  const remove = useCallback(
    async (id: string) => {
      setError('')
      try {
        await deleteModelRecord(app, model, id, deleteCommand)
        if (paging || searching) await refresh()
        else {
          setItems((prev) => prev.filter((row) => String(row.id) !== id))
          setTotal((n) => Math.max(0, n - 1))
        }
      } catch (e) {
        reportError(e instanceof Error ? e.message : String(e))
      }
    },
    [app, model, deleteCommand, paging, searching, refresh, reportError],
  )

  useEffect(() => {
    void refresh()
  }, [refresh])

  useEffect(() => {
    if (options.refreshToken != null && options.refreshToken > 0) void refresh()
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [options.refreshToken])

  const groupColumns = useMemo(() => {
    if (!columnField) return [{ id: 'all', label: 'All' }]
    if (groups.length) {
      return groups.map((g) => {
        const id = g.values[0] ?? ''
        return { id, label: id || '(empty)' }
      })
    }
    const seen = new Set<string>()
    const order: string[] = []
    for (const item of items) {
      const id = String(item[columnField] ?? '')
      if (!seen.has(id)) {
        seen.add(id)
        order.push(id)
      }
    }
    return order.map((id) => ({ id, label: id || '(empty)' }))
  }, [columnField, groups, items])

  const pagerProps = paging
    ? {
        total,
        page: pagination.page,
        pageSize: pagination.pageSize,
        onPageChange: pagination.setPage,
        onPageSizeChange: pagination.setPageSize,
        pageSizeOptions: pagination.pageSizeOptions,
        url: false as const,
      }
    : null

  const searchProps =
    searching && modelRef
      ? {
          model: modelRef,
          fields: searchFields,
          searchable: searching.searchable,
          filterable: searching.filterable,
          groupable: searching.groupable,
          placeholder: searching.placeholder,
          q: search.q,
          searchIn: search.searchIn,
          domain: search.domain,
          groupBy: search.groupBy,
          junction: search.junction,
          onChange: search.setState,
          url: false as const,
        }
      : null

  return {
    items,
    total,
    columns,
    searchFields,
    loading,
    error,
    app,
    model,
    modelRef,
    view,
    deleteCommand,
    groups,
    groupColumns,
    columnField,
    refresh,
    remove,
    paginated: Boolean(paging),
    searchable: Boolean(searching),
    page: pagination.page,
    pageSize: pagination.pageSize,
    setPage: pagination.setPage,
    setPageSize: pagination.setPageSize,
    pagerProps,
    searchProps,
  }
}
