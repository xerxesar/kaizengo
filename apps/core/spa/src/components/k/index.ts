/** Spec-driven CQRS UI — prefer importing from `@/k`. */

export { KAppStatus } from './KAppStatus'
export { KForm } from './KForm'
export { KFormField } from './KFormField'
export { KTable, KTableView } from './KTable'
export type { KTablePaginationConfig, KTableSearchConfig, KTableViewProps } from './KTable'
export { KChart, KChartView } from './KChart'
export { KChartDesigner } from './KChartDesigner'
export type {
  KChartConfig,
  KChartMeasure,
  KChartPaginationConfig,
  KChartProps,
  KChartSearchConfig,
  KChartType,
  KChartViewProps,
} from './KChart'
export { DEFAULT_CHART_TYPES, chartConfigFromPreset } from './KChart'
export {
  listChartPresets,
  listChartPresetsSync,
  saveChartPreset,
  deleteChartPreset,
  fetchDefaultChartPreset,
} from './chart-presets'
export { KPivot, KPivotView } from './KPivot'
export type {
  KPivotFieldOption,
  KPivotMeasure,
  KPivotPaginationConfig,
  KPivotProps,
  KPivotSearchConfig,
  KPivotViewProps,
} from './KPivot'
export { KKanban, KKanbanBoard } from './KKanban'
export type {
  KKanbanColumn,
  KKanbanPaginationConfig,
  KKanbanProps,
  KKanbanSearchConfig,
  KKanbanBoardProps,
} from './KKanban'
export { KCollection } from './KCollection'
export type {
  KCollectionChartConfig,
  KCollectionFilter,
  KCollectionGroupBy,
  KCollectionPaginationConfig,
  KCollectionPivotConfig,
  KCollectionProps,
  KCollectionSearchConfig,
  KCollectionView,
} from './KCollection'
export { KPagination } from './KPagination'
export type { KPaginationProps } from './KPagination'
export { KSearch } from './KSearch'
export type { KSearchProps } from './KSearch'
export { KQueryShell } from './KQueryShell'
export type { KQueryShellProps } from './KQueryShell'
export {
  useKQuery,
  enableConfig,
  type KQueryPaginationConfig,
  type KQuerySearchConfig,
  type UseKQueryOptions,
  type UseKQueryResult,
} from './useKQuery'
export {
  DEFAULT_PAGE_SIZE,
  DEFAULT_PAGE_SIZE_OPTIONS,
  sliceItems,
  totalPages,
  usePaginationParams,
} from './pagination'
export type { PaginationParamsOptions, PaginationState } from './pagination'
export { useKSearchParams } from './search/params'
export type { KSearchParamsOptions } from './search/params'
export {
  useCollectionViewParam,
  useChartUrlState,
  usePivotUrlState,
  useLocationVersion,
  getUrlParam,
  setUrlParams,
} from './url-state'
export type {
  ChartUrlState,
  CollectionViewParamOptions,
  PivotUrlState,
} from './url-state'
export {
  decodeDomain,
  encodeDomain,
  extractLeaves,
  opsForFieldType,
  serializeLeaves,
  EMPTY_SEARCH,
} from './search/types'
export type {
  Domain,
  DomainLeaf,
  DomainNode,
  DomainOp,
  SearchField,
  SearchState,
  SearchTemplate,
} from './search/types'
export {
  deleteSearchTemplate,
  listSearchTemplates,
  saveSearchTemplate,
} from './search/templates'
export { getKFormContext, useKForm } from './kform-context'
export type { KFormActionsContext, KFormContext, KFormFieldContext } from './kform-types'
