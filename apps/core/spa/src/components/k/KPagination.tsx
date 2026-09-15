import { useEffect, useId, useMemo, useRef, useState, type FormEvent } from 'react'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import { cn } from '@/lib/utils'
import { KEYMAP_ID_ATTR } from '@/lib/keymap/types'
import {
  DEFAULT_PAGE_SIZE,
  DEFAULT_PAGE_SIZE_OPTIONS,
  totalPages,
  usePaginationParams,
} from './pagination'

export type KPaginationProps = {
  /** Total number of records (unpaged). */
  total: number
  /** Controlled page (1-based). Pair with onPageChange. */
  page?: number
  /** Controlled page size. Pair with onPageSizeChange. */
  pageSize?: number
  onPageChange?: (page: number) => void
  onPageSizeChange?: (pageSize: number) => void
  pageSizeOptions?: number[]
  /**
   * Sync page/pageSize to URL search params.
   * Default true when page/pageSize are not controlled.
   */
  url?: boolean
  pageParam?: string
  pageSizeParam?: string
  defaultPageSize?: number
  className?: string
  /** @deprecated use className */
  class?: string
}

/**
 * Composable pagination: prev/next, manual page entry, page size.
 * Defaults to `?page=` / `?pageSize=` via react-router when uncontrolled.
 */
export function KPagination(props: KPaginationProps) {
  const pageInputId = useId()
  const sizeInputId = useId()
  const sizeInputRef = useRef<HTMLInputElement>(null)
  const defaultPageSize = props.defaultPageSize ?? DEFAULT_PAGE_SIZE
  const pageSizeOptions = props.pageSizeOptions ?? DEFAULT_PAGE_SIZE_OPTIONS
  const pageControlled = props.page != null
  const sizeControlled = props.pageSize != null
  const useUrl = props.url ?? !(pageControlled || sizeControlled)

  const fromUrl = usePaginationParams({
    defaultPageSize,
    pageSizeOptions,
    pageParam: props.pageParam,
    pageSizeParam: props.pageSizeParam,
    url: useUrl,
  })

  const [localPage, setLocalPage] = useState(1)
  const [localSize, setLocalSize] = useState(defaultPageSize)
  const [sizeEditing, setSizeEditing] = useState(false)

  const page = pageControlled ? props.page! : useUrl ? fromUrl.page : localPage
  const pageSize = sizeControlled ? props.pageSize! : useUrl ? fromUrl.pageSize : localSize
  const pages = totalPages(props.total, pageSize)

  const sizeSelectOptions = useMemo(() => {
    const set = new Set(pageSizeOptions)
    if (pageSize > 0) set.add(pageSize)
    return [...set].sort((a, b) => a - b)
  }, [pageSizeOptions, pageSize])

  const [draft, setDraft] = useState(String(page))
  const [sizeDraft, setSizeDraft] = useState(String(pageSize))
  useEffect(() => {
    setDraft(String(page))
  }, [page])
  useEffect(() => {
    setSizeDraft(String(pageSize))
  }, [pageSize])
  useEffect(() => {
    if (sizeEditing) sizeInputRef.current?.focus()
  }, [sizeEditing])

  function applyPage(next: number) {
    const clamped = Math.min(pages, Math.max(1, next))
    if (props.onPageChange) props.onPageChange(clamped)
    else if (useUrl) fromUrl.setPage(clamped)
    else setLocalPage(clamped)
  }

  // Keep page in range when total/pageSize shrink (only once we know the total).
  useEffect(() => {
    if (props.total <= 0) return
    if (page > pages) applyPage(pages)
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [pages, page, props.total])

  function changeSize(next: number) {
    const size = Math.max(1, Math.floor(next) || 1)
    if (props.onPageSizeChange) props.onPageSizeChange(size)
    else if (useUrl) fromUrl.setPageSize(size)
    else {
      setLocalSize(size)
      setLocalPage(1)
    }
  }

  function commitSizeEdit() {
    const n = Number.parseInt(sizeDraft, 10)
    if (Number.isFinite(n) && n > 0) changeSize(n)
    else setSizeDraft(String(pageSize))
    setSizeEditing(false)
  }

  function startSizeEdit() {
    setSizeDraft(String(pageSize))
    setSizeEditing(true)
  }

  function submitPage(e: FormEvent) {
    e.preventDefault()
    const n = Number.parseInt(draft, 10)
    if (Number.isFinite(n)) applyPage(n)
    else setDraft(String(page))
  }

  function submitSize(e: FormEvent) {
    e.preventDefault()
    commitSizeEdit()
  }

  if (props.total <= 0) return null

  const from = Math.min(props.total, (page - 1) * pageSize + 1)
  const to = Math.min(props.total, page * pageSize)
  const className = props.className ?? props.class

  return (
    <div
      className={cn(
        'flex flex-wrap items-center justify-between gap-3 border border-[var(--kg-border)] bg-[var(--kg-surface)] px-3 py-2',
        className,
      )}
      role="navigation"
      aria-label="Pagination"
    >
      <p className="text-xs tabular-nums text-[var(--kg-text-muted)]">
        {from}–{to} of {props.total}
      </p>

      <div className="flex flex-wrap items-center gap-2">
        <Button
          type="button"
          size="sm"
          variant="secondary"
          disabled={page <= 1}
          onClick={() => applyPage(page - 1)}
          aria-label="Previous page"
          {...{ [KEYMAP_ID_ATTR]: 'k-page-prev' }}
        >
          Prev
        </Button>

        <form className="flex items-center gap-1.5" onSubmit={submitPage}>
          <label className="sr-only" htmlFor={pageInputId}>
            Page
          </label>
          <Input
            id={pageInputId}
            type="number"
            min={1}
            max={pages}
            value={draft}
            onChange={(e) => setDraft(e.target.value)}
            onBlur={() => {
              const n = Number.parseInt(draft, 10)
              if (Number.isFinite(n)) applyPage(n)
              else setDraft(String(page))
            }}
            className="h-8 w-14 px-2 text-center tabular-nums"
            aria-label="Page number"
          />
          <span className="text-xs text-[var(--kg-text-muted)]">/ {pages}</span>
        </form>

        <Button
          type="button"
          size="sm"
          variant="secondary"
          disabled={page >= pages}
          onClick={() => applyPage(page + 1)}
          aria-label="Next page"
          {...{ [KEYMAP_ID_ATTR]: 'k-page-next' }}
        >
          Next
        </Button>

        <div className="flex items-center gap-1.5">
          <span className="text-xxs font-medium uppercase tracking-wide text-[var(--kg-text-muted)]">
            Size
          </span>
          {sizeEditing ? (
            <form onSubmit={submitSize}>
              <label className="sr-only" htmlFor={sizeInputId}>
                Page size
              </label>
              <Input
                ref={sizeInputRef}
                id={sizeInputId}
                type="number"
                min={1}
                value={sizeDraft}
                onChange={(e) => setSizeDraft(e.target.value)}
                onBlur={commitSizeEdit}
                onKeyDown={(e) => {
                  if (e.key === 'Escape') {
                    setSizeDraft(String(pageSize))
                    setSizeEditing(false)
                  }
                }}
                className="h-8 w-[4.5rem] px-2 text-center tabular-nums"
                aria-label="Page size"
              />
            </form>
          ) : (
            <Select value={String(pageSize)} onValueChange={(v) => changeSize(Number(v))}>
              <SelectTrigger className="h-8 w-[4.5rem] gap-1 px-2" aria-label="Page size">
                <span
                  role="button"
                  tabIndex={0}
                  className="min-w-0 flex-1 truncate text-left tabular-nums outline-none"
                  title="Click to type a custom size"
                  onPointerDown={(e) => {
                    e.preventDefault()
                    e.stopPropagation()
                    startSizeEdit()
                  }}
                  onKeyDown={(e) => {
                    if (e.key === 'Enter' || e.key === ' ') {
                      e.preventDefault()
                      e.stopPropagation()
                      startSizeEdit()
                    }
                  }}
                >
                  <SelectValue />
                </span>
              </SelectTrigger>
              <SelectContent>
                {sizeSelectOptions.map((n) => (
                  <SelectItem key={n} value={String(n)}>
                    {n}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
          )}
        </div>
      </div>
    </div>
  )
}
