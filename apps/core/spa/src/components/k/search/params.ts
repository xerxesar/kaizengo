import { useCallback, useEffect, useMemo, useState } from 'react'
import {
  decodeDomain,
  encodeDomain,
  EMPTY_SEARCH,
  type Domain,
  type SearchState,
} from './types'

export type KSearchParamsOptions = {
  qParam?: string
  searchInParam?: string
  domainParam?: string
  groupByParam?: string
  url?: boolean
  /** Called when filters change (e.g. reset page to 1). */
  onFilterChange?: () => void
}

function readSearch(): URLSearchParams {
  return new URLSearchParams(window.location.search)
}

function writeSearch(next: URLSearchParams) {
  const qs = next.toString()
  const url = window.location.pathname + (qs ? `?${qs}` : '') + window.location.hash
  const current = window.location.pathname + window.location.search + window.location.hash
  if (url === current) return
  history.replaceState(history.state, '', url)
  window.dispatchEvent(new CustomEvent('kaizengo:location'))
}

function parseCSV(raw: string | null): string[] {
  if (!raw?.trim()) return []
  return raw
    .split(',')
    .map((s) => s.trim())
    .filter(Boolean)
}

function readState(
  qParam: string,
  searchInParam: string,
  domainParam: string,
  groupByParam: string,
): SearchState {
  const p = readSearch()
  return {
    q: (p.get(qParam) ?? '').trim(),
    searchIn: parseCSV(p.get(searchInParam)),
    domain: decodeDomain(p.get(domainParam)),
    groupBy: parseCSV(p.get(groupByParam)),
    junction: 'and',
  }
}

function writeState(
  state: SearchState,
  qParam: string,
  searchInParam: string,
  domainParam: string,
  groupByParam: string,
) {
  const p = readSearch()
  if (state.q.trim()) p.set(qParam, state.q.trim())
  else p.delete(qParam)

  if (state.searchIn.length) p.set(searchInParam, state.searchIn.join(','))
  else p.delete(searchInParam)

  const dom = encodeDomain(state.domain)
  if (dom) p.set(domainParam, dom)
  else p.delete(domainParam)

  if (state.groupBy.length) p.set(groupByParam, state.groupBy.join(','))
  else p.delete(groupByParam)

  // Changing search resets pagination.
  p.delete('page')
  writeSearch(p)
}

/** URL-synced search/filter/groupBy state (mirrors usePaginationParams). */
export function useKSearchParams(options?: KSearchParamsOptions): SearchState & {
  setState: (next: Partial<SearchState> | ((prev: SearchState) => SearchState)) => void
  setQ: (q: string) => void
  setSearchIn: (fields: string[]) => void
  setDomain: (domain: Domain) => void
  setGroupBy: (fields: string[]) => void
  setJunction: (j: 'and' | 'or') => void
  clear: () => void
  toListOpts: () => {
    q?: string
    searchIn?: string[]
    domain?: string
    groupBy?: string[]
  }
} {
  const qParam = options?.qParam ?? 'q'
  const searchInParam = options?.searchInParam ?? 'searchIn'
  const domainParam = options?.domainParam ?? 'domain'
  const groupByParam = options?.groupByParam ?? 'groupBy'
  const useUrl = options?.url !== false

  const [locVersion, setLocVersion] = useState(0)
  const [local, setLocal] = useState<SearchState>(EMPTY_SEARCH)

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

  void locVersion
  const state = useUrl
    ? readState(qParam, searchInParam, domainParam, groupByParam)
    : local

  const commit = useCallback(
    (next: SearchState) => {
      if (useUrl) {
        writeState(next, qParam, searchInParam, domainParam, groupByParam)
        options?.onFilterChange?.()
      } else {
        setLocal(next)
        options?.onFilterChange?.()
      }
    },
    [useUrl, qParam, searchInParam, domainParam, groupByParam, options],
  )

  const setState = useCallback(
    (next: Partial<SearchState> | ((prev: SearchState) => SearchState)) => {
      const prev = useUrl
        ? readState(qParam, searchInParam, domainParam, groupByParam)
        : local
      const resolved = typeof next === 'function' ? next(prev) : { ...prev, ...next }
      commit(resolved)
    },
    [useUrl, qParam, searchInParam, domainParam, groupByParam, local, commit],
  )

  return useMemo(
    () => ({
      ...state,
      setState,
      setQ: (q: string) => setState({ q }),
      setSearchIn: (searchIn: string[]) => setState({ searchIn }),
      setDomain: (domain: Domain) => setState({ domain }),
      setGroupBy: (groupBy: string[]) => setState({ groupBy }),
      setJunction: (junction: 'and' | 'or') => setState({ junction }),
      clear: () => commit({ ...EMPTY_SEARCH }),
      toListOpts: () => {
        const opts: {
          q?: string
          searchIn?: string[]
          domain?: string
          groupBy?: string[]
        } = {}
        if (state.q.trim()) opts.q = state.q.trim()
        if (state.searchIn.length) opts.searchIn = state.searchIn
        const dom = encodeDomain(state.domain)
        if (dom) opts.domain = dom
        if (state.groupBy.length) opts.groupBy = state.groupBy
        return opts
      },
    }),
    [state, setState, commit],
  )
}
