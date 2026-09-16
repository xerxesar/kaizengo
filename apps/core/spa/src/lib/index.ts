/** Platform barrel for app views. */

export { cn } from './cn'
export {
  alertVariants,
  badgeVariants,
  buttonSizes,
  buttonVariants,
  checkboxControlClass,
  comboboxControlClass,
  comboboxInputClass,
  comboboxTriggerClass,
  dialogBackdropClass,
  dialogContentClass,
  dialogPositionerClass,
  fieldErrorClass,
  fieldHintClass,
  fieldLabelClass,
  inputClass,
  kindTagClass,
  menuContentClass,
  menuItemClass,
  menubarLinkClass,
  progressRangeClass,
  progressTrackClass,
  selectContentClass,
  selectItemClass,
  selectTriggerClass,
  textareaClass,
  treeItemClass,
} from './styles/classes'

export type * from './types'
export { registerViewComponent, resolveViewComponent } from './view-components'
export { firstMenuLeaf, flattenMenuItems } from './menu'
export { fetchAppMenus } from './menu-client'
export {
  createModelRecord,
  deleteModelRecord,
  fetchAppPing,
  fetchACLActions,
  fetchModelViews,
  fetchResources,
  fetchViewSlots,
  formViewForModel,
  formatNamespace,
  getModelRecord,
  isNamespaced,
  listModelRecords,
  listModelRecordsPage,
  listModelGroups,
  listViewForModel,
  listViewForQuery,
  formViewForCommand,
  parseNamespace,
  resolveCqrsRef,
  updateModelRecord,
  type CqrsRef,
  type GroupBucket,
  type ModelColumn,
  type ModelField,
  type ModelListPage,
  type ModelListPageOpts,
  type ModelRecord,
  type ModelView,
  type Namespace,
  type SecuredResource,
  type ViewSlot,
} from './model-client'
export {
  appPath,
  currentAppRoute,
  currentMenuPage,
  findMenuByRoute,
  menuItemHref,
  menuPagePath,
  menuRouteOf,
  navigateApp,
  resolveMenuSelection,
} from './menu-route'
export {
  getMenuContext,
  getLayoutRegistry,
  inferAppName,
  useMenuContext,
  useLayoutRegistry,
  type MenuState,
  type LayoutRegistry,
} from './layout-context'
export { appModuleUrl, contentAppForMenu, type SpaMountContext } from './spa-mount'
export {
  fetchI18n,
  applyLocale,
  syncDocumentLocale,
  type I18nBundle,
  type I18nEntry,
  type TextDirection,
  type Translator,
} from './i18n'
export { getI18n, t, useI18n, type I18nScope } from './i18n-context'
export { I18nProvider } from './I18nProvider'
export { AuthProvider, getCurrentUser, useAuth, type AuthContextValue, type AuthUser } from './auth-context'
export { setI18nLocale, getI18nLocale } from './i18n-runtime'
export {
  getTheme,
  getThemeDef,
  getThemeMode,
  initTheme,
  setTheme,
  themeIconHref,
  THEMES,
  type ThemeDef,
  type ThemeId,
  type ThemeMode,
} from './theme'
export {
  KeymapProvider,
  KeymapHints,
  useKeymap,
  useKeymapAction,
  useSimpleHotkeyRecorder,
  fetchKeymap,
  runKeymapAction,
  registerKeymapAction,
  getKeymapOverrides,
  setKeymapOverride,
  clearKeymapOverride,
  clearAllKeymapOverrides,
  setKeymapRecording,
  KEYMAP_ID_ATTR,
  keymapLabelFor,
  useHotkeys,
  useFormatHotkey,
  type KeymapBinding,
  type KeymapCatalog,
  type KeymapContextValue,
  type KeymapOverrides,
  type KeymapScope,
} from './keymap'

export {
  KAppStatus,
  KForm,
  KFormField,
  KTable,
  KTableView,
  KChart,
  KChartView,
  KPivot,
  KPivotView,
  KKanban,
  KKanbanBoard,
  KCollection,
  KPagination,
  KSearch,
  KQueryShell,
  getKFormContext,
  useKQuery,
  usePaginationParams,
  useKSearchParams,
  DEFAULT_PAGE_SIZE,
  DEFAULT_PAGE_SIZE_OPTIONS,
  sliceItems,
  totalPages,
  type KKanbanColumn,
  type KCollectionChartConfig,
  type KCollectionFilter,
  type KCollectionGroupBy,
  type KCollectionPivotConfig,
  type KCollectionView,
  type KChartProps,
  type KChartViewProps,
  type KPivotProps,
  type KPivotViewProps,
  type KFormActionsContext,
  type KFormContext,
  type KFormFieldContext,
  type KPaginationProps,
  type KSearchProps,
  type UseKQueryResult,
} from '../k'

export { Layout } from '@/components/shell/Layout'
export { LayoutMain } from '@/components/shell/LayoutMain'
export { LayoutMenu } from '@/components/shell/LayoutMenu'
export { SearchableCombobox } from '@/components/shell/SearchableCombobox'
export { SearchableMultiSelect } from '@/components/shell/SearchableMultiSelect'
export { TreeView } from '@/components/shell/TreeView'
export { Card, Toolbar, StatCard, FormActions, FormSection, DataTable } from '@/components/shell/layout-bits'
export { SearchBar } from '@/components/shell/SearchBar'

export function formatDate(iso: string): string {
  if (!iso) return '—'
  try {
    return new Intl.DateTimeFormat(undefined, {
      year: 'numeric',
      month: 'short',
      day: 'numeric',
    }).format(new Date(iso))
  } catch {
    return iso
  }
}

export function formatDateTime(iso: string): string {
  if (!iso) return '—'
  try {
    return new Intl.DateTimeFormat(undefined, {
      year: 'numeric',
      month: 'short',
      day: 'numeric',
      hour: '2-digit',
      minute: '2-digit',
    }).format(new Date(iso))
  } catch {
    return iso
  }
}

export function statusBadgeVariant(status: string): import('./types').BadgeVariant {
  switch (status.toLowerCase()) {
    case 'active':
      return 'success'
    case 'suspended':
      return 'danger'
    case 'invited':
      return 'warning'
    default:
      return 'muted'
  }
}
