import { useEffect, useMemo, useRef } from 'react'
import * as echarts from 'echarts/core'
import { BarChart, LineChart, PieChart, ScatterChart, RadarChart } from 'echarts/charts'
import {
  DatasetComponent,
  GridComponent,
  LegendComponent,
  RadarComponent,
  TooltipComponent,
} from 'echarts/components'
import { CanvasRenderer } from 'echarts/renderers'
import type { EChartsOption } from 'echarts'
import { cn } from '@/lib/utils'
import { aggregateChart, type ChartDatum, type MeasureKind } from './aggregate'

echarts.use([
  BarChart,
  LineChart,
  PieChart,
  ScatterChart,
  RadarChart,
  DatasetComponent,
  GridComponent,
  LegendComponent,
  RadarComponent,
  TooltipComponent,
  CanvasRenderer,
])

export type ChartType =
  | 'bar'
  | 'line'
  | 'area'
  | 'pie'
  | 'doughnut'
  | 'scatter'
  | 'radar'

export type ChartMeasure = MeasureKind

export const CHART_TYPE_LABELS: Record<ChartType, string> = {
  bar: 'Bar',
  line: 'Line',
  area: 'Area',
  pie: 'Pie',
  doughnut: 'Doughnut',
  scatter: 'Scatter',
  radar: 'Radar',
}

export function EmptyChart(props: { message: string }) {
  return (
    <div className="rounded border border-dashed border-[var(--kg-border)] bg-[var(--kg-surface)] px-6 py-10 text-center">
      <p className="text-base font-medium text-[var(--kg-text)]">{props.message}</p>
    </div>
  )
}

function readCssVar(name: string, fallback: string): string {
  if (typeof window === 'undefined') return fallback
  const v = getComputedStyle(document.documentElement).getPropertyValue(name).trim()
  return v || fallback
}

export function buildEchartsOption(input: {
  data: ChartDatum[]
  type: ChartType
  measure: ChartMeasure
  xLabel: string
  yLabel: string
}): EChartsOption {
  const textColor = readCssVar('--kg-text', '#0f172a')
  const muted = readCssVar('--kg-text-muted', '#64748b')
  const border = readCssVar('--kg-border', '#e2e8f0')
  const { data, type, measure, xLabel, yLabel } = input

  const categories: string[] = []
  const seriesNames: string[] = []
  const byCat = new Map<string, ChartDatum[]>()
  for (const d of data) {
    if (!byCat.has(d.label)) {
      byCat.set(d.label, [])
      categories.push(d.label)
    }
    byCat.get(d.label)!.push(d)
    const s = d.series ?? ''
    if (s && !seriesNames.includes(s)) seriesNames.push(s)
  }
  const multi = seriesNames.length > 1
  const names = multi ? seriesNames : ['']

  const tooltip: EChartsOption['tooltip'] = {
    trigger: type === 'pie' || type === 'doughnut' ? 'item' : 'axis',
  }
  const legend =
    multi || type === 'pie' || type === 'doughnut' || type === 'radar'
      ? { textStyle: { color: muted } }
      : undefined

  if (type === 'pie' || type === 'doughnut') {
    const pieData = categories.map((cat) => {
      const points = byCat.get(cat) ?? []
      const value = points.reduce((sum, p) => sum + p.value, 0)
      return { name: cat, value }
    })
    return {
      color: ['#0ea5e9', '#8b5cf6', '#f59e0b', '#10b981', '#f43f5e', '#06b6d4', '#84cc16'],
      tooltip,
      legend,
      series: [
        {
          type: 'pie',
          radius: type === 'doughnut' ? ['42%', '68%'] : '65%',
          data: pieData,
          label: { color: textColor },
        },
      ],
    }
  }

  if (type === 'radar') {
    const max = Math.max(...data.map((d) => d.value), 1)
    const indicators = categories.map((name) => ({ name, max: max * 1.15 || 1 }))
    const series = names.map((seriesName) => ({
      name: seriesName || yLabel,
      type: 'radar' as const,
      data: [
        {
          value: categories.map((cat) => {
            const points = byCat.get(cat) ?? []
            if (!seriesName) return points[0]?.value ?? 0
            return points.find((p) => p.series === seriesName)?.value ?? 0
          }),
          name: seriesName || yLabel,
        },
      ],
    }))
    return {
      color: ['#0ea5e9', '#8b5cf6', '#f59e0b', '#10b981'],
      tooltip,
      legend,
      radar: {
        indicator: indicators,
        axisName: { color: muted },
      },
      series,
    }
  }

  const series = names.map((seriesName) => {
    const values = categories.map((cat) => {
      const points = byCat.get(cat) ?? []
      if (!seriesName) return points[0]?.value ?? 0
      return points.find((p) => p.series === seriesName)?.value ?? 0
    })
    if (type === 'scatter') {
      return {
        name: seriesName || yLabel,
        type: 'scatter' as const,
        data: values.map((v, i) => [i, v]),
        symbolSize: 10,
      }
    }
    if (type === 'area') {
      return {
        name: seriesName || yLabel,
        type: 'line' as const,
        data: values,
        areaStyle: {},
        smooth: true,
      }
    }
    if (type === 'line') {
      return {
        name: seriesName || yLabel,
        type: 'line' as const,
        data: values,
        smooth: true,
      }
    }
    return {
      name: seriesName || yLabel,
      type: 'bar' as const,
      data: values,
      barMaxWidth: 48,
    }
  })

  return {
    color: ['#0ea5e9', '#8b5cf6', '#f59e0b', '#10b981', '#f43f5e', '#06b6d4'],
    tooltip,
    legend,
    grid: { left: 48, right: 16, top: legend ? 40 : 24, bottom: 40 },
    xAxis: {
      type: type === 'scatter' ? 'value' : 'category',
      data: type === 'scatter' ? undefined : categories,
      name: xLabel,
      nameTextStyle: { color: muted },
      axisLabel: { color: muted },
      axisLine: { lineStyle: { color: border } },
      splitLine: { lineStyle: { color: border, opacity: 0.4 } },
    },
    yAxis: {
      type: 'value',
      name: measure === 'count' ? 'Count' : yLabel,
      nameTextStyle: { color: muted },
      axisLabel: { color: muted },
      splitLine: { lineStyle: { color: border, opacity: 0.4 } },
    },
    series,
  }
}

export function EChartsHost(props: {
  option: EChartsOption
  height: number | string
  className?: string
}) {
  const elRef = useRef<HTMLDivElement>(null)
  const chartRef = useRef<echarts.EChartsType | null>(null)

  useEffect(() => {
    const el = elRef.current
    if (!el) return
    const chart = echarts.init(el, undefined, { renderer: 'canvas' })
    chartRef.current = chart
    const onResize = () => chart.resize()
    window.addEventListener('resize', onResize)
    const ro = typeof ResizeObserver !== 'undefined' ? new ResizeObserver(onResize) : null
    ro?.observe(el)
    return () => {
      window.removeEventListener('resize', onResize)
      ro?.disconnect()
      chart.dispose()
      chartRef.current = null
    }
  }, [])

  useEffect(() => {
    chartRef.current?.setOption(props.option, true)
  }, [props.option])

  return (
    <div
      ref={elRef}
      className={cn('w-full', props.className)}
      style={{ height: props.height }}
      role="img"
      aria-label="Chart"
    />
  )
}

/** Canvas-only chart preview (no toolbar). Used by KChartView and KChartDesigner. */
export function ChartCanvas(props: {
  rows: Record<string, unknown>[]
  xField: string
  yField?: string
  seriesField?: string
  measure?: ChartMeasure
  type?: ChartType
  fieldLabel?: (field: string) => string
  emptyMessage?: string
  height?: number | string
  className?: string
}) {
  const measure = props.measure ?? (props.yField ? 'sum' : 'count')
  const chartType = props.type ?? 'bar'
  const labelOf = props.fieldLabel ?? ((field: string) => field)

  const data = useMemo(
    () =>
      props.xField
        ? aggregateChart(props.rows, props.xField, {
            yField: props.yField,
            measure,
            seriesField: props.seriesField,
          })
        : [],
    [props.rows, props.xField, props.yField, measure, props.seriesField],
  )

  const option = useMemo(
    () =>
      buildEchartsOption({
        data,
        type: chartType,
        measure,
        xLabel: labelOf(props.xField),
        yLabel: props.yField ? labelOf(props.yField) : 'Count',
      }),
    [data, chartType, measure, props.xField, props.yField, labelOf],
  )

  if (!props.rows.length) {
    return <EmptyChart message={props.emptyMessage ?? 'No records found'} />
  }
  if (!props.xField.trim()) {
    return <EmptyChart message="Chart requires an xField" />
  }
  if (!data.length) {
    return <EmptyChart message={props.emptyMessage ?? 'No chart data'} />
  }

  return (
    <div className={cn('border border-[var(--kg-border)] bg-[var(--kg-surface)] p-4', props.className)}>
      <EChartsHost option={option} height={props.height ?? 320} />
    </div>
  )
}
