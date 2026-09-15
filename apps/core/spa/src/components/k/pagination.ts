import { useCallback, useEffect, useMemo, useState } from 'react'

export const DEFAULT_PAGE_SIZE = 20
export const DEFAULT_PAGE_SIZE_OPTIONS = [10, 20, 50, 100]

export type PaginationParamsOptions = {
  defaultPageSize?: number
  pageSizeOptions?: number[]
  /** URL search param for page (default `page`) */
  pageParam?: string
  /** URL search param for page size (default `pageSize`) */
  pageSizeParam?: string
  /** Sync to URL (default true) */
  url?: boolean
}

export type PaginationState = {
  page: number
  pageSize: number
  pageSizeOptions: number[]
  setPage: (page: number) => void
  setPageSize: (pageSize: number) => void
  offset: number
}

function parsePositiveInt(raw: string | null, fallback: number): number {
  if (raw == null || raw.trim() === '') return fallback
  const n = Number.parseInt(raw, 10)
  return Number.isFinite(n) && n > 0 ? n : fallback
}

function readSearch(): URLSearchParams {
  return new URLSearchParams(window.location.search)
}

/** Write search params without wiping React Router's history.state. */
function writeSearch(next: URLSearchParams, replace = true) {
  const qs = next.toString()
  const url = window.location.pathname + (qs ? `?${qs}` : '') + window.location.hash
  const current = window.location.pathname + window.location.search + window.location.hash
  if (url === current) return
  if (replace) history.replaceState(history.state, '', url)
  else history.pushState(history.state, '', url)
  // Notify shell listeners without synthesizing popstate (avoids RR back-stack glitches).
  window.dispatchEvent(new CustomEvent('kaizengo:location'))
}

/**
 * Read/write `page` + `pageSize` from the real address bar.
 * Uses `window.location.search` (not only react-router) so values survive
 * the shell's hybrid history / replaceState usage.
 */
export function usePaginationParams(options?: PaginationParamsOptions): PaginationState {
  const defaultPageSize = options?.defaultPageSize ?? DEFAULT_PAGE_SIZE
  const pageParam = options?.pageParam ?? 'page'
  const pageSizeParam = options?.pageSizeParam ?? 'pageSize'
  const pageSizeOptions = options?.pageSizeOptions ?? DEFAULT_PAGE_SIZE_OPTIONS
  const useUrl = options?.url !== false

  const [locVersion, setLocVersion] = useState(0)
  const [localPage, setLocalPage] = useState(1)
  const [localSize, setLocalSize] = useState(defaultPageSize)

  useEffect(() => {
    if (!useUrl) return
    const sync = () => setLocVersion((v) => v + 1)
    window.addEventListener('popstate', sync)
    window.addEventListener('kaizengo:location', sync)
    return () => {
      window.removeEventListener('popstate', sync)
      window.removeEventListener('kaizengo:location', sync)
    }
  }, [useUrl])

  const search = useUrl ? readSearch() : null
  // locVersion forces a re-read after history updates.
  void locVersion

  const page = useUrl ? parsePositiveInt(search!.get(pageParam), 1) : localPage
  const pageSize = useUrl
    ? parsePositiveInt(search!.get(pageSizeParam), defaultPageSize)
    : localSize

  const setPage = useCallback(
    (next: number) => {
      const page = Math.max(1, Math.floor(next) || 1)
      if (!useUrl) {
        setLocalPage(page)
        return
      }
      const p = readSearch()
      if (page <= 1) p.delete(pageParam)
      else p.set(pageParam, String(page))
      writeSearch(p, true)
    },
    [pageParam, useUrl],
  )

  const setPageSize = useCallback(
    (next: number) => {
      const size = Math.max(1, Math.floor(next) || defaultPageSize)
      if (!useUrl) {
        setLocalSize(size)
        setLocalPage(1)
        return
      }
      const p = readSearch()
      if (size === defaultPageSize) p.delete(pageSizeParam)
      else p.set(pageSizeParam, String(size))
      // Changing page size resets to first page.
      p.delete(pageParam)
      writeSearch(p, true)
    },
    [defaultPageSize, pageParam, pageSizeParam, useUrl],
  )

  return useMemo(
    () => ({
      page,
      pageSize,
      pageSizeOptions,
      setPage,
      setPageSize,
      offset: (page - 1) * pageSize,
    }),
    [page, pageSize, pageSizeOptions, setPage, setPageSize],
  )
}

export function totalPages(total: number, pageSize: number): number {
  if (pageSize <= 0) return 1
  return Math.max(1, Math.ceil(total / pageSize))
}

export function sliceItems<T>(items: T[], page: number, pageSize: number): T[] {
  if (pageSize <= 0) return items
  const start = (Math.max(1, page) - 1) * pageSize
  return items.slice(start, start + pageSize)
}
