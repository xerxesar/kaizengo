export { Alert } from './src/lib/Alert'
export { Badge } from './src/lib/Badge'
export { Button } from './src/lib/Button'
export { Card } from './src/lib/Card'
export { Checkbox, CheckboxInput } from './src/lib/Checkbox'
export { Combobox } from './src/lib/Combobox'
export { SearchableCombobox } from './src/lib/SearchableCombobox'
export { EmptyState } from './src/lib/EmptyState'
export { FormActions } from './src/lib/FormActions'
export { FormField } from './src/lib/FormField'
export { FormSection } from './src/lib/FormSection'
export { I18nProvider } from './src/lib/I18nProvider'
export { Input } from './src/lib/Input'
export { KAppStatus } from './src/lib/KAppStatus'
export { KForm } from './src/lib/KForm'
export { KFormField } from './src/lib/KFormField'
export { KTable } from './src/lib/KTable'
export { Layout } from './src/lib/Layout'
export { LayoutActions } from './src/lib/LayoutActions'
export { LayoutMain } from './src/lib/LayoutMain'
export { LayoutMenu } from './src/lib/LayoutMenu'
export { Modal } from './src/lib/Modal'
export { Page } from './src/lib/Page'
export { PageHeader } from './src/lib/PageHeader'
export { Select } from './src/lib/Select'
export { Spinner } from './src/lib/Spinner'
export { StatCard } from './src/lib/StatCard'
export { Table } from './src/lib/Table'
export { Toolbar } from './src/lib/Toolbar'
export { TreeView } from './src/lib/TreeView'

export { menuContentClass, menuItemClass, selectTriggerClass } from './src/lib/ark/styles'

export type * from './src/lib/types'
export type { KFormActionsContext, KFormContext, KFormFieldContext } from './src/lib/kform-types'
export { getKFormContext } from './src/lib/kform-context'
export { registerViewComponent, resolveViewComponent } from './src/lib/view-components'
export { firstMenuLeaf, flattenMenuItems } from './src/lib/menu'
export { fetchAppMenus } from './src/lib/menu-client'
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
  listViewForModel,
  parseNamespace,
  updateModelRecord,
  type ModelColumn,
  type ModelField,
  type ModelRecord,
  type ModelView,
  type Namespace,
  type SecuredResource,
  type ViewSlot,
} from './src/lib/model-client'
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
} from './src/lib/menu-route'
export { getMenuContext, inferAppName, type MenuContext } from './src/lib/layout-context'
export {
  appModuleUrl,
  contentAppForMenu,
  type SpaMountContext,
} from './src/lib/spa-mount'
export { fetchI18n, applyLocale, syncDocumentLocale, type I18nBundle, type I18nEntry, type TextDirection, type Translator } from './src/lib/i18n'
export { getI18n, t, type I18nScope } from './src/lib/i18n-context'
export { setI18nLocale, getI18nLocale } from './src/lib/i18n-runtime'
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
} from './src/lib/theme'
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
  type KeymapBinding,
  type KeymapCatalog,
  type KeymapContextValue,
  type KeymapOverrides,
  type KeymapScope,
} from './src/lib/keymap'
export { useHotkeyRecorder, useFormatHotkey } from '@ark-ui/solid'

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

export function statusBadgeVariant(status: string): import('./src/lib/types').BadgeVariant {
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
