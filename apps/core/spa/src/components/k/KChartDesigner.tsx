import { useEffect, useMemo, useState } from 'react'
import { Button } from '@/components/ui/button'
import { Checkbox } from '@/components/ui/checkbox'
import {
  Dialog,
  DialogContent,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import type { ModelRecord } from '@/lib/model-client'
import {
  DEFAULT_CHART_TYPES,
  type KChartConfig,
  type KChartMeasure,
  type KChartType,
} from './chart-config'
import { ChartCanvas, CHART_TYPE_LABELS, type ChartType } from './chart-canvas'
import { saveChartPreset } from './chart-presets'
import type { SearchField } from './search/types'

const TYPE_OPTIONS: KChartType[] = [
  'bar',
  'line',
  'area',
  'pie',
  'doughnut',
  'scatter',
  'radar',
]

const MEASURE_OPTIONS: KChartMeasure[] = ['count', 'sum', 'avg']

const NONE = '__none__'

export type KChartDesignerProps = {
  open: boolean
  onOpenChange: (open: boolean) => void
  rows: ModelRecord[] | Record<string, unknown>[]
  fields: SearchField[]
  /** When set, Save persists chart presets for this model. */
  model?: string
  seed?: Partial<KChartConfig>
  fieldLabel?: (field: string) => string
  onApply: (config: KChartConfig) => void
  onSaved?: (config: KChartConfig) => void
}

function FieldSelect(props: {
  id: string
  label: string
  value: string
  fields: SearchField[]
  allowNone?: boolean
  noneLabel?: string
  onChange: (value: string) => void
}) {
  const value = props.value || (props.allowNone ? NONE : '')
  return (
    <div className="space-y-1.5">
      <Label htmlFor={props.id}>{props.label}</Label>
      <Select
        value={value || undefined}
        onValueChange={(v) => props.onChange(v === NONE ? '' : v)}
      >
        <SelectTrigger id={props.id} aria-label={props.label}>
          <SelectValue placeholder={`Select ${props.label.toLowerCase()}…`} />
        </SelectTrigger>
        <SelectContent>
          {props.allowNone ? (
            <SelectItem value={NONE}>{props.noneLabel ?? 'None'}</SelectItem>
          ) : null}
          {props.fields.map((f) => (
            <SelectItem key={f.key} value={f.key}>
              {f.label}
            </SelectItem>
          ))}
        </SelectContent>
      </Select>
    </div>
  )
}

/**
 * Chart builder: edit a KChartConfig with live preview.
 * Apply uses it immediately; Save (when model is set) stores shared/default flags.
 */
export function KChartDesigner(props: KChartDesignerProps) {
  const [name, setName] = useState(props.seed?.label ?? '')
  const [type, setType] = useState<KChartType>(props.seed?.type ?? 'bar')
  const [xField, setXField] = useState(props.seed?.xField ?? '')
  const [yField, setYField] = useState(props.seed?.yField ?? '')
  const [seriesField, setSeriesField] = useState(props.seed?.seriesField ?? '')
  const [measure, setMeasure] = useState<KChartMeasure>(
    props.seed?.measure ?? (props.seed?.yField ? 'sum' : 'count'),
  )
  const [shared, setShared] = useState(props.seed?.shared ?? false)
  const [isDefault, setIsDefault] = useState(props.seed?.isDefault ?? false)
  const [saving, setSaving] = useState(false)
  const [error, setError] = useState('')

  useEffect(() => {
    if (!props.open) return
    setName(props.seed?.label ?? '')
    setType(props.seed?.type ?? 'bar')
    setXField(props.seed?.xField ?? props.fields[0]?.key ?? '')
    setYField(props.seed?.yField ?? '')
    setSeriesField(props.seed?.seriesField ?? '')
    setMeasure(props.seed?.measure ?? (props.seed?.yField ? 'sum' : 'count'))
    setShared(props.seed?.shared ?? false)
    setIsDefault(props.seed?.isDefault ?? false)
    setError('')
  }, [props.open, props.seed, props.fields])

  const draft: KChartConfig = useMemo(
    () => ({
      id: props.seed?.id,
      label: name.trim() || undefined,
      type,
      types: DEFAULT_CHART_TYPES,
      xField,
      yField: yField || undefined,
      seriesField: seriesField || undefined,
      measure,
      shared,
      isDefault,
    }),
    [props.seed?.id, name, type, xField, yField, seriesField, measure, shared, isDefault],
  )

  const canApply = Boolean(xField.trim())
  const canSave = Boolean(props.model?.trim() && name.trim() && canApply)

  function apply() {
    if (!canApply) return
    props.onApply({
      ...draft,
      label: name.trim() || draft.label || 'Custom chart',
    })
    props.onOpenChange(false)
  }

  async function save() {
    if (!canSave || !props.model?.trim()) return
    setSaving(true)
    setError('')
    try {
      const saved = await saveChartPreset({
        model: props.model,
        id: props.seed?.id,
        label: name.trim(),
        xField,
        yField: yField || undefined,
        seriesField: seriesField || undefined,
        measure,
        type,
        types: DEFAULT_CHART_TYPES,
        shared,
        isDefault,
      })
      const config: KChartConfig = {
        id: saved.id,
        label: saved.label,
        type: saved.type,
        types: saved.types,
        xField: saved.xField,
        yField: saved.yField,
        seriesField: saved.seriesField,
        measure: saved.measure,
        shared: saved.shared,
        isDefault: saved.isDefault,
      }
      props.onSaved?.(config)
      props.onApply(config)
      props.onOpenChange(false)
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e))
    } finally {
      setSaving(false)
    }
  }

  return (
    <Dialog open={props.open} onOpenChange={props.onOpenChange} className="max-w-4xl">
      <DialogContent className="gap-0 p-0">
        <DialogHeader>
          <DialogTitle>Chart builder</DialogTitle>
        </DialogHeader>

        <div className="grid gap-4 px-5 py-4 md:grid-cols-[minmax(14rem,18rem)_1fr]">
          <div className="flex flex-col gap-3">
            <div className="space-y-1.5">
              <Label htmlFor="chart-name">Name</Label>
              <Input
                id="chart-name"
                value={name}
                onChange={(e) => setName(e.target.value)}
                placeholder="My chart…"
              />
            </div>

            <div className="space-y-1.5">
              <Label htmlFor="chart-type">Type</Label>
              <Select value={type} onValueChange={(v) => setType(v as KChartType)}>
                <SelectTrigger id="chart-type" aria-label="Chart type">
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  {TYPE_OPTIONS.map((t) => (
                    <SelectItem key={t} value={t}>
                      {CHART_TYPE_LABELS[t as ChartType]}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </div>

            <FieldSelect
              id="chart-x"
              label="Category (X)"
              value={xField}
              fields={props.fields}
              onChange={setXField}
            />

            <FieldSelect
              id="chart-y"
              label="Value (Y)"
              value={yField}
              fields={props.fields}
              allowNone
              noneLabel="Count records"
              onChange={(v) => {
                setYField(v)
                if (!v) setMeasure('count')
                else if (measure === 'count') setMeasure('sum')
              }}
            />

            <div className="space-y-1.5">
              <Label htmlFor="chart-measure">Measure</Label>
              <Select
                value={measure}
                onValueChange={(v) => setMeasure(v as KChartMeasure)}
              >
                <SelectTrigger id="chart-measure" aria-label="Measure">
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  {MEASURE_OPTIONS.map((m) => (
                    <SelectItem key={m} value={m} disabled={m !== 'count' && !yField}>
                      {m}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </div>

            <FieldSelect
              id="chart-series"
              label="Series"
              value={seriesField}
              fields={props.fields}
              allowNone
              onChange={setSeriesField}
            />

            {props.model?.trim() ? (
              <div className="space-y-3 border-t border-[var(--kg-border)] pt-3">
                <label className="flex cursor-pointer items-start gap-2 text-sm">
                  <Checkbox checked={shared} onCheckedChange={(v) => setShared(v === true)} />
                  <span>
                    <span className="font-medium">Share with others</span>
                    <span className="mt-0.5 block text-xs text-[var(--kg-text-muted)]">
                      Visible to everyone in your organization
                    </span>
                  </span>
                </label>
                <label className="flex cursor-pointer items-start gap-2 text-sm">
                  <Checkbox checked={isDefault} onCheckedChange={(v) => setIsDefault(v === true)} />
                  <span>
                    <span className="font-medium">Set as default</span>
                    <span className="mt-0.5 block text-xs text-[var(--kg-text-muted)]">
                      Open this chart automatically for this list
                    </span>
                  </span>
                </label>
              </div>
            ) : null}

            {error ? (
              <p className="text-xs text-red-600 dark:text-red-400">{error}</p>
            ) : null}
          </div>

          <div className="min-h-[16rem] min-w-0">
            <p className="mb-2 text-xxs font-medium uppercase tracking-wide text-[var(--kg-text-muted)]">
              Preview
            </p>
            {xField ? (
              <ChartCanvas
                rows={props.rows as Record<string, unknown>[]}
                xField={xField}
                yField={yField || undefined}
                seriesField={seriesField || undefined}
                measure={measure}
                type={type}
                fieldLabel={props.fieldLabel}
                height={260}
                emptyMessage="No data for this configuration"
              />
            ) : (
              <div className="flex h-[260px] items-center justify-center border border-dashed border-[var(--kg-border)] text-sm text-[var(--kg-text-muted)]">
                Choose a category field to preview
              </div>
            )}
          </div>
        </div>

        <DialogFooter className="justify-between sm:justify-between">
          <Button type="button" variant="ghost" onClick={() => props.onOpenChange(false)}>
            Cancel
          </Button>
          <div className="flex gap-2">
            <Button type="button" variant="secondary" disabled={!canApply} onClick={apply}>
              Apply
            </Button>
            {props.model?.trim() ? (
              <Button type="button" disabled={!canSave || saving} onClick={() => void save()}>
                Save
              </Button>
            ) : null}
          </div>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}
