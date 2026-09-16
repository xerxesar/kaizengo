import { useCallback, useEffect, useMemo, useState } from 'react'

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

/** Subscribe to URL changes (popstate + kaizengo:location). */
export function useLocationVersion(enabled = true): number {
  const [locVersion, setLocVersion] = useState(0)
  useEffect(() => {
    if (!enabled) return
    const sync = () => setLocVersion((v) => v + 1)
    window.addEventListener('popstate', sync)
    window.addEventListener('kaizengo:location', sync)
    return () => {
      window.removeEventListener('popstate', sync)
      window.removeEventListener('kaizengo:location', sync)
    }
  }, [enabled])
  return locVersion
}

export function getUrlParam(name: string): string | null {
  return readSearch().get(name)
}

export function setUrlParams(
  updates: Record<string, string | null | undefined>,
): void {
  const p = readSearch()
  let changed = false
  for (const [key, value] of Object.entries(updates)) {
    if (value === undefined) continue
    const next = value == null || !value.trim() ? null : value.trim()
    const prev = p.get(key)
    if (next == null) {
      if (prev != null) {
        p.delete(key)
        changed = true
      }
    } else if (prev !== next) {
      p.set(key, next)
      changed = true
    }
  }
  if (changed) writeSearch(p)
}

export function parseCsvParam(raw: string | null): string[] {
  if (!raw?.trim()) return []
  return raw
    .split(',')
    .map((s) => s.trim())
    .filter(Boolean)
}

export function toCsvParam(values: string[] | undefined | null): string | null {
  const list = (values ?? []).map((s) => s.trim()).filter(Boolean)
  return list.length ? list.join(',') : null
}

/**
 * Single string URL param with local fallback when url=false.
 * Empty string clears the param.
 */
export function useUrlStringParam(
  name: string,
  options?: { url?: boolean; defaultValue?: string },
): [string, (next: string) => void] {
  const useUrl = options?.url !== false
  const fallback = options?.defaultValue ?? ''
  const locVersion = useLocationVersion(useUrl)
  const [local, setLocal] = useState(fallback)
  void locVersion

  const value = useUrl ? (getUrlParam(name) ?? fallback) : local

  const setValue = useCallback(
    (next: string) => {
      if (useUrl) setUrlParams({ [name]: next.trim() || null })
      else setLocal(next)
    },
    [name, useUrl],
  )

  return [value, setValue]
}

/** CSV list URL param. */
export function useUrlCsvParam(
  name: string,
  options?: { url?: boolean; defaultValue?: string[] },
): [string[], (next: string[]) => void] {
  const useUrl = options?.url !== false
  const fallback = options?.defaultValue ?? []
  const locVersion = useLocationVersion(useUrl)
  const [local, setLocal] = useState(fallback)
  void locVersion

  const value = useUrl ? parseCsvParam(getUrlParam(name)) : local
  // When URL empty and we have a default, surface default without writing yet
  const effective =
    useUrl && value.length === 0 && fallback.length > 0 ? fallback : value

  const setValue = useCallback(
    (next: string[]) => {
      if (useUrl) setUrlParams({ [name]: toCsvParam(next) })
      else setLocal(next)
    },
    [name, useUrl],
  )

  return [effective, setValue]
}

export type CollectionViewParamOptions = {
  url?: boolean
  allowed: string[]
  defaultView: string
  param?: string
}

/** Collection presentation mode (`?view=table|kanban|chart|pivot`). */
export function useCollectionViewParam(
  options: CollectionViewParamOptions,
): [string, (next: string) => void] {
  const param = options.param ?? 'view'
  const useUrl = options.url !== false
  const locVersion = useLocationVersion(useUrl)
  const [local, setLocal] = useState(options.defaultView)
  void locVersion

  const raw = useUrl ? getUrlParam(param) : local
  const view =
    raw && options.allowed.includes(raw) ? raw : options.defaultView

  const setView = useCallback(
    (next: string) => {
      if (!options.allowed.includes(next)) return
      if (useUrl) {
        setUrlParams({ [param]: next === options.defaultView ? null : next })
      } else {
        setLocal(next)
      }
    },
    [options.allowed, options.defaultView, param, useUrl],
  )

  return [view, setView]
}

export type ChartUrlState = {
  chartId: string
  chartType: string
  xField: string
  yField: string
  seriesField: string
  measure: string
}

export type ChartUrlOptions = {
  url?: boolean
  /** Prefix for param names (default none → chart, chartType, …). */
  prefix?: string
}

function chartParamNames(prefix = '') {
  const p = prefix
  return {
    id: `${p}chart`,
    type: `${p}chartType`,
    x: `${p}chartX`,
    y: `${p}chartY`,
    series: `${p}chartSeries`,
    measure: `${p}chartMeasure`,
  }
}

export function useChartUrlState(options?: ChartUrlOptions): {
  state: ChartUrlState
  setChartId: (id: string) => void
  setChartType: (type: string) => void
  setDraftFields: (fields: {
    xField?: string
    yField?: string
    seriesField?: string
    measure?: string
  }) => void
  clearDraftFields: () => void
  hasUrlChartId: boolean
  hasUrlDraft: boolean
} {
  const useUrl = options?.url !== false
  const prefix = options?.prefix ?? ''
  const locVersion = useLocationVersion(useUrl)
  const [local, setLocal] = useState<ChartUrlState>({
    chartId: '',
    chartType: '',
    xField: '',
    yField: '',
    seriesField: '',
    measure: '',
  })
  void locVersion

  const names = useMemo(() => chartParamNames(prefix), [prefix])

  const state: ChartUrlState = useUrl
    ? {
        chartId: getUrlParam(names.id) ?? '',
        chartType: getUrlParam(names.type) ?? '',
        xField: getUrlParam(names.x) ?? '',
        yField: getUrlParam(names.y) ?? '',
        seriesField: getUrlParam(names.series) ?? '',
        measure: getUrlParam(names.measure) ?? '',
      }
    : local

  const patch = useCallback(
    (updates: Partial<ChartUrlState>) => {
      if (useUrl) {
        setUrlParams({
          [names.id]: updates.chartId !== undefined ? updates.chartId || null : undefined,
          [names.type]:
            updates.chartType !== undefined ? updates.chartType || null : undefined,
          [names.x]: updates.xField !== undefined ? updates.xField || null : undefined,
          [names.y]: updates.yField !== undefined ? updates.yField || null : undefined,
          [names.series]:
            updates.seriesField !== undefined ? updates.seriesField || null : undefined,
          [names.measure]:
            updates.measure !== undefined ? updates.measure || null : undefined,
        })
      } else {
        setLocal((prev) => ({ ...prev, ...updates }))
      }
    },
    [names, useUrl],
  )

  return useMemo(
    () => ({
      state,
      setChartId: (chartId: string) =>
        patch({
          chartId,
          // Selecting a preset clears draft field overrides in the URL.
          xField: '',
          yField: '',
          seriesField: '',
          measure: '',
        }),
      setChartType: (chartType: string) => patch({ chartType }),
      setDraftFields: (fields) =>
        patch({
          chartId: '',
          xField: fields.xField ?? '',
          yField: fields.yField ?? '',
          seriesField: fields.seriesField ?? '',
          measure: fields.measure ?? '',
        }),
      clearDraftFields: () =>
        patch({ xField: '', yField: '', seriesField: '', measure: '' }),
      hasUrlChartId: Boolean(state.chartId.trim()),
      hasUrlDraft: Boolean(state.xField.trim()),
    }),
    [state, patch],
  )
}

export type PivotUrlState = {
  rowFields: string[]
  colFields: string[]
  measureField: string
  measure: string
}

export type PivotUrlOptions = {
  url?: boolean
  prefix?: string
}

function pivotParamNames(prefix = '') {
  const p = prefix
  return {
    rows: `${p}pivotRows`,
    cols: `${p}pivotCols`,
    y: `${p}pivotY`,
    measure: `${p}pivotMeasure`,
  }
}

export function usePivotUrlState(options?: PivotUrlOptions): {
  state: PivotUrlState
  setRowFields: (fields: string[]) => void
  setColFields: (fields: string[]) => void
  setMeasureField: (field: string) => void
  setMeasure: (measure: string) => void
  setAll: (next: PivotUrlState) => void
  hasUrlRows: boolean
} {
  const useUrl = options?.url !== false
  const prefix = options?.prefix ?? ''
  const locVersion = useLocationVersion(useUrl)
  const [local, setLocal] = useState<PivotUrlState>({
    rowFields: [],
    colFields: [],
    measureField: '',
    measure: '',
  })
  void locVersion

  const names = useMemo(() => pivotParamNames(prefix), [prefix])

  const state: PivotUrlState = useUrl
    ? {
        rowFields: parseCsvParam(getUrlParam(names.rows)),
        colFields: parseCsvParam(getUrlParam(names.cols)),
        measureField: getUrlParam(names.y) ?? '',
        measure: getUrlParam(names.measure) ?? '',
      }
    : local

  const patch = useCallback(
    (updates: Partial<PivotUrlState>) => {
      if (useUrl) {
        setUrlParams({
          [names.rows]:
            updates.rowFields !== undefined ? toCsvParam(updates.rowFields) : undefined,
          [names.cols]:
            updates.colFields !== undefined ? toCsvParam(updates.colFields) : undefined,
          [names.y]:
            updates.measureField !== undefined
              ? updates.measureField || null
              : undefined,
          [names.measure]:
            updates.measure !== undefined ? updates.measure || null : undefined,
        })
      } else {
        setLocal((prev) => ({ ...prev, ...updates }))
      }
    },
    [names, useUrl],
  )

  return useMemo(
    () => ({
      state,
      setRowFields: (rowFields: string[]) => patch({ rowFields }),
      setColFields: (colFields: string[]) => patch({ colFields }),
      setMeasureField: (measureField: string) => patch({ measureField }),
      setMeasure: (measure: string) => patch({ measure }),
      setAll: (next: PivotUrlState) => patch(next),
      hasUrlRows: state.rowFields.length > 0,
    }),
    [state, patch],
  )
}
