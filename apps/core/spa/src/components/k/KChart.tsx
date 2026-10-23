import { useEffect, useMemo, useRef, useState } from 'react'
import { Pencil } from 'lucide-react'
import { Button } from '@/components/ui/button'
import type { ModelRecord } from '@/lib/model-client'
import { t } from '@/lib'
import type { Column } from '@/lib/types'
import { cn } from '@/lib/utils'
import {
  ChartCanvas,
  CHART_TYPE_LABELS,
} from './chart-canvas'
import {
  DEFAULT_CHART_TYPES,
  type KChartConfig,
  type KChartMeasure,
  type KChartType,
} from './chart-config'
import {
  fetchDefaultChartPreset,
  listChartPresets,
  listChartPresetsSync,
  savedToChartConfig,
  type SavedChartPreset,
} from './chart-presets'
import { KChartDesigner } from './KChartDesigner'
import { KQueryShell } from './KQueryShell'
import type { SearchField } from './search/types'
import {
  useKQuery,
  type KQueryPaginationConfig,
  type KQuerySearchConfig,
} from './useKQuery'
import { useChartUrlState } from './url-state'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'

export type { KChartConfig, KChartMeasure, KChartType } from './chart-config'
export { DEFAULT_CHART_TYPES } from './chart-config'

export type KChartPaginationConfig = KQueryPaginationConfig
export type KChartSearchConfig = KQuerySearchConfig

export type KChartViewProps<T extends Record<string, unknown> = ModelRecord> = {
  rows: T[]
  xField?: string
  yField?: string
  measure?: KChartMeasure
  seriesField?: string
  type?: KChartType
  types?: KChartType[]
  presets?: KChartConfig[]
  chartId?: string
  onChartIdChange?: (id: string) => void
  onTypeChange?: (type: KChartType) => void
  /** Model fields for the chart builder. */
  fields?: SearchField[]
  /** Model ref (`app.model`) — enables saving user chart presets. */
  model?: string
  /** Show preset/type toolbar (default true). */
  toolbar?: boolean
  /** Sync chart preset/type/draft fields to URL (default true). */
  url?: boolean
  fieldLabel?: (field: string) => string
  emptyMessage?: string
  className?: string
  height?: number | string
}

function chartPresetLabel(p: KChartConfig & { shared?: boolean; isDefault?: boolean }): string {
  const base = p.labelKey ? t(p.labelKey) : p.label || p.id || 'Chart'
  const tags: string[] = []
  if (p.isDefault) tags.push('default')
  if (p.shared) tags.push('shared')
  return tags.length ? `${base} (${tags.join(', ')})` : base
}

function normalizeChartType(raw?: string | null): KChartType | undefined {
  const value = raw?.trim() as KChartType | undefined
  if (!value) return undefined
  if (value in CHART_TYPE_LABELS) return value
  return undefined
}

export function chartConfigFromPreset(preset: {
  id?: string
  label?: string | null
  labelKey?: string | null
  type?: string | null
  types?: string[] | null
  xField?: string | null
  yField?: string | null
  seriesField?: string | null
  measure?: string | null
}): KChartConfig | null {
  const xField = preset.xField?.trim()
  if (!xField) return null
  const types = (preset.types ?? [])
    .map((t) => normalizeChartType(t))
    .filter((t): t is KChartType => Boolean(t))
  return {
    id: preset.id,
    label: preset.label ?? undefined,
    labelKey: preset.labelKey ?? undefined,
    type: normalizeChartType(preset.type) ?? 'bar',
    types: types.length ? types : DEFAULT_CHART_TYPES,
    xField,
    yField: preset.yField?.trim() || undefined,
    seriesField: preset.seriesField?.trim() || undefined,
    measure: (preset.measure?.trim() as KChartMeasure | undefined) || undefined,
  }
}

/** Presentational chart — Apache ECharts, no fetching. */
export function KChartView<T extends Record<string, unknown>>(props: KChartViewProps<T>) {
  const labelOf = props.fieldLabel ?? ((field: string) => field)
  const showToolbar = props.toolbar !== false
  const designable = Boolean(props.fields?.length)
  const syncUrl = props.url !== false
  const chartUrl = useChartUrlState({ url: syncUrl })

  const [userPresets, setUserPresets] = useState<SavedChartPreset[]>(() =>
    props.model ? listChartPresetsSync(props.model) : [],
  )
  const [designerOpen, setDesignerOpen] = useState(false)
  /** Ephemeral override from Apply (not yet a named preset) — mirrored to URL when syncUrl. */
  const [draftOverride, setDraftOverride] = useState<KChartConfig | null>(null)
  const [internalPresetId, setInternalPresetId] = useState<string | undefined>(props.chartId)
  const defaultAppliedRef = useRef(false)

  useEffect(() => {
    if (!props.model) return
    let cancelled = false
    void (async () => {
      const items = await listChartPresets(props.model!)
      if (cancelled) return
      setUserPresets(items)
      if (
        !defaultAppliedRef.current &&
        props.chartId == null &&
        !chartUrl.hasUrlChartId &&
        !chartUrl.hasUrlDraft
      ) {
        defaultAppliedRef.current = true
        const def =
          items.find((p) => p.isDefault) ?? (await fetchDefaultChartPreset(props.model!))
        if (def?.id && !cancelled) {
          setInternalPresetId(def.id)
          chartUrl.setChartId(def.id)
          props.onChartIdChange?.(def.id)
        }
      }
    })()
    return () => {
      cancelled = true
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [props.model])

  const specPresets = props.presets ?? []
  const mergedPresets = useMemo(() => {
    const byId = new Map<string, KChartConfig>()
    for (const p of specPresets) if (p.id) byId.set(p.id, p)
    for (const p of userPresets.map(savedToChartConfig)) {
      if (!p.id) continue
      byId.set(p.id, { ...byId.get(p.id), ...p })
    }
    return [...byId.values()]
  }, [specPresets, userPresets])

  const urlDraft: KChartConfig | null =
    chartUrl.hasUrlDraft
      ? {
          xField: chartUrl.state.xField,
          yField: chartUrl.state.yField || undefined,
          seriesField: chartUrl.state.seriesField || undefined,
          measure: (chartUrl.state.measure as KChartMeasure | undefined) || undefined,
          type: normalizeChartType(chartUrl.state.chartType) ?? 'bar',
        }
      : null

  const presetId =
    (chartUrl.hasUrlChartId ? chartUrl.state.chartId : undefined) ??
    props.chartId ??
    internalPresetId ??
    mergedPresets[0]?.id

  const activePreset =
    draftOverride ??
    urlDraft ??
    mergedPresets.find((p) => p.id === presetId) ??
    mergedPresets[0] ??
    null

  const xField = activePreset?.xField?.trim() || props.xField?.trim() || ''
  const yField = activePreset?.yField?.trim() || props.yField?.trim()
  const seriesField = activePreset?.seriesField?.trim() || props.seriesField?.trim()
  const measure =
    activePreset?.measure ?? props.measure ?? (yField ? 'sum' : 'count')
  const allowedTypes =
    props.types?.length
      ? props.types
      : activePreset?.types?.length
        ? activePreset.types
        : DEFAULT_CHART_TYPES

  const urlType = normalizeChartType(chartUrl.state.chartType)
  const defaultType =
    urlType ?? props.type ?? activePreset?.type ?? allowedTypes[0] ?? 'bar'

  const [chartType, setChartType] = useState<KChartType>(defaultType)

  useEffect(() => {
    setChartType(defaultType)
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [presetId, props.type, activePreset?.type, draftOverride?.type, urlType])

  function setPreset(id: string) {
    setDraftOverride(null)
    if (props.chartId == null) setInternalPresetId(id)
    chartUrl.setChartId(id)
    props.onChartIdChange?.(id)
  }

  function setType(next: KChartType) {
    setChartType(next)
    chartUrl.setChartType(next)
    props.onTypeChange?.(next)
  }

  function refreshUserPresets() {
    if (!props.model) return
    void listChartPresets(props.model).then(setUserPresets)
  }

  function applyDesignerConfig(config: KChartConfig) {
    if (config.id) {
      setDraftOverride(null)
      if (props.chartId == null) setInternalPresetId(config.id)
      chartUrl.setChartId(config.id)
      props.onChartIdChange?.(config.id)
    } else {
      setDraftOverride(config)
      chartUrl.setDraftFields({
        xField: config.xField,
        yField: config.yField,
        seriesField: config.seriesField,
        measure: config.measure,
      })
    }
    if (config.type) {
      setChartType(config.type)
      chartUrl.setChartType(config.type)
    }
  }

  const activeUserMeta = userPresets.find((p) => p.id === activePreset?.id)
  const designerSeed: Partial<KChartConfig> = {
    id: activePreset?.id,
    label: activePreset?.label || (activePreset?.labelKey ? t(activePreset.labelKey) : ''),
    type: chartType,
    types: allowedTypes,
    xField,
    yField,
    seriesField,
    measure,
    shared: activeUserMeta?.shared ?? activePreset?.shared,
    isDefault: activeUserMeta?.isDefault ?? activePreset?.isDefault,
  }

  const isCustomDraft = Boolean(
    (draftOverride && !draftOverride.id) || (urlDraft && !chartUrl.hasUrlChartId),
  )

  if (!props.rows.length) {
    return (
      <div className={cn('flex flex-col gap-3', props.className)}>
        {showToolbar && designable ? (
          <div className="flex justify-end">
            <Button
              type="button"
              size="sm"
              variant="ghost"
              aria-label="Chart builder"
              onClick={() => setDesignerOpen(true)}
            >
              <Pencil className="size-4" aria-hidden />
              <span className="ml-1">Build</span>
            </Button>
          </div>
        ) : null}
        <div className="rounded border border-dashed border-[var(--kg-border)] bg-[var(--kg-surface)] px-6 py-10 text-center">
          <p className="text-base font-medium text-[var(--kg-text)]">
            {props.emptyMessage ?? 'No records found'}
          </p>
        </div>
        {designable ? (
          <KChartDesigner
            open={designerOpen}
            onOpenChange={setDesignerOpen}
            rows={[]}
            fields={props.fields!}
            model={props.model}
            seed={designerSeed}
            fieldLabel={labelOf}
            onApply={applyDesignerConfig}
            onSaved={() => refreshUserPresets()}
          />
        ) : null}
      </div>
    )
  }

  const showPresetSwitcher = mergedPresets.length > 1
  const showTypeSwitcher = allowedTypes.length > 1

  return (
    <div className={cn('flex flex-col gap-3', props.className)}>
      {showToolbar ? (
        <div className="flex flex-wrap items-end justify-between gap-3">
          <div className="min-w-0 text-xs text-[var(--kg-text-muted)]">
            <span className="font-medium text-[var(--kg-text)]">{labelOf(xField || '—')}</span>
            {xField ? (
              <>
                {' · '}
                {measure}
                {yField ? ` of ${labelOf(yField)}` : ''}
              </>
            ) : null}
            {isCustomDraft ? (
              <span className="ml-2 text-[var(--kg-text-muted)]">(custom)</span>
            ) : null}
          </div>
          <div className="flex flex-wrap items-center gap-2">
            {showPresetSwitcher ? (
              <Select
                value={
                  isCustomDraft
                    ? undefined
                    : (activePreset?.id ?? mergedPresets[0]?.id)
                }
                onValueChange={setPreset}
              >
                <SelectTrigger className="h-8 w-[12rem]" aria-label="Chart preset">
                  <SelectValue placeholder="Preset" />
                </SelectTrigger>
                <SelectContent>
                  {mergedPresets.map((p) => (
                    <SelectItem key={p.id} value={p.id!}>
                      {chartPresetLabel(p)}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
            ) : null}
            {showTypeSwitcher ? (
              <Select value={chartType} onValueChange={(v) => setType(v as KChartType)}>
                <SelectTrigger className="h-8 w-[9rem]" aria-label="Chart type">
                  <SelectValue placeholder="Type" />
                </SelectTrigger>
                <SelectContent>
                  {allowedTypes.map((t) => (
                    <SelectItem key={t} value={t}>
                      {CHART_TYPE_LABELS[t]}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
            ) : null}
            {designable ? (
              <Button
                type="button"
                size="sm"
                variant="ghost"
                aria-label="Chart builder"
                onClick={() => setDesignerOpen(true)}
              >
                <Pencil className="size-4" aria-hidden />
                <span className="ml-1">Build</span>
              </Button>
            ) : null}
          </div>
        </div>
      ) : null}

      <ChartCanvas
        rows={props.rows as Record<string, unknown>[]}
        xField={xField}
        yField={yField}
        seriesField={seriesField}
        measure={measure}
        type={chartType}
        fieldLabel={labelOf}
        emptyMessage={props.emptyMessage}
        height={props.height}
      />

      {designable ? (
        <KChartDesigner
          open={designerOpen}
          onOpenChange={setDesignerOpen}
          rows={props.rows as ModelRecord[]}
          fields={props.fields!}
          model={props.model}
          seed={designerSeed}
          fieldLabel={labelOf}
          onApply={applyDesignerConfig}
          onSaved={() => {
            refreshUserPresets()
          }}
        />
      ) : null}
    </div>
  )
}

type QueryProps = {
  query?: string
  model?: string
  chartId?: string
  xField?: string
  yField?: string
  measure?: KChartMeasure
  seriesField?: string
  type?: KChartType
  types?: KChartType[]
  paginated?: boolean | KChartPaginationConfig
  searchable?: boolean | KChartSearchConfig
  /** Sync chart settings to URL (default true). */
  url?: boolean
  emptyMessage?: string
  onerror?: (message: string) => void
  className?: string
  /** @deprecated use className */
  class?: string
  refreshToken?: number
}

type ItemsProps<T extends Record<string, unknown>> = {
  rows: T[]
  xField?: string
  yField?: string
  measure?: KChartMeasure
  seriesField?: string
  type?: KChartType
  types?: KChartType[]
  presets?: KChartConfig[]
  chartId?: string
  fields?: SearchField[]
  model?: string
  fieldLabel?: (field: string) => string
  /** Sync chart settings to URL (default true). */
  url?: boolean
  emptyMessage?: string
  className?: string
  /** @deprecated use className */
  class?: string
  query?: undefined
}

export type KChartProps<T extends Record<string, unknown> = ModelRecord> =
  | QueryProps
  | ItemsProps<T>

function isQueryMode(props: { query?: string; model?: string }): props is QueryProps {
  return Boolean(props.query?.trim() || props.model?.trim())
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

/**
 * Chart view (Apache ECharts). Prefer `query`/`model` (fetches via useKQuery).
 * Pass `rows` + `xField` for a presentational escape hatch.
 */
export function KChart<T extends Record<string, unknown> = ModelRecord>(
  props: KChartProps<T>,
) {
  if (isQueryMode(props)) {
    return <KChartQuery {...(props as QueryProps)} />
  }
  const itemsProps = props as ItemsProps<T>
  return (
    <KChartView
      rows={itemsProps.rows}
      xField={itemsProps.xField}
      yField={itemsProps.yField}
      measure={itemsProps.measure}
      seriesField={itemsProps.seriesField}
      type={itemsProps.type}
      types={itemsProps.types}
      presets={itemsProps.presets}
      chartId={itemsProps.chartId}
      fields={itemsProps.fields}
      model={itemsProps.model}
      fieldLabel={itemsProps.fieldLabel}
      emptyMessage={itemsProps.emptyMessage}
      className={itemsProps.className ?? itemsProps.class}
      url={itemsProps.url}
    />
  )
}

function KChartQuery(props: QueryProps) {
  const kq = useKQuery({
    query: props.query,
    model: props.model,
    paginated: props.paginated,
    searchable: props.searchable,
    refreshToken: props.refreshToken,
    onerror: props.onerror,
  })

  const presets = useMemo(
    () =>
      (kq.view?.chartPresets ?? [])
        .map((p) => chartConfigFromPreset(p))
        .filter((p): p is KChartConfig => Boolean(p)),
    [kq.view?.chartPresets],
  )

  const fallbackX = defaultCategoryField(kq.columns, kq.groupByFields, props.xField)

  return (
    <KQueryShell
      query={kq}
      emptyMessage={props.emptyMessage}
      className={props.className ?? props.class}
    >
      <KChartView
        rows={kq.items}
        presets={presets}
        chartId={props.chartId}
        xField={props.xField?.trim() || (presets.length ? undefined : fallbackX)}
        yField={props.yField}
        measure={props.measure}
        seriesField={props.seriesField?.trim() || kq.nestedGroupField || undefined}
        type={props.type}
        types={props.types}
        fields={kq.searchFields}
        model={kq.modelRef || undefined}
        fieldLabel={(field) =>
          kq.searchFields.find((f) => f.key === field)?.label ?? field
        }
        url={props.url}
      />
    </KQueryShell>
  )
}
