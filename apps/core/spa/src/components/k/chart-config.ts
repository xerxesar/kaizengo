import type { ChartMeasure, ChartType } from './chart-canvas'

export type KChartMeasure = ChartMeasure
export type KChartType = ChartType

export const DEFAULT_CHART_TYPES: KChartType[] = [
  'bar',
  'line',
  'area',
  'pie',
  'doughnut',
]

/** Spec / runtime chart config (mirrors appspec ChartPresetSpec). */
export type KChartConfig = {
  id?: string
  label?: string
  labelKey?: string
  type?: KChartType
  /** Allowed types for the in-toolbar switcher. */
  types?: KChartType[]
  xField: string
  yField?: string
  seriesField?: string
  measure?: KChartMeasure
  shared?: boolean
  isDefault?: boolean
}
