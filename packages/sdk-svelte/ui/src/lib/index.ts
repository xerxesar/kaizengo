export { default as Alert } from './Alert.svelte'
export { default as Badge } from './Badge.svelte'
export { default as Button } from './Button.svelte'
export { default as Card } from './Card.svelte'
export { default as Container } from './Container.svelte'
export { default as EmptyState } from './EmptyState.svelte'
export { default as FormActions } from './FormActions.svelte'
export { default as FormField } from './FormField.svelte'
export { default as FormSection } from './FormSection.svelte'
export { default as Input } from './Input.svelte'
export { default as I18nProvider } from './I18nProvider.svelte'
export { default as KAppStatus } from './KAppStatus.svelte'
export { default as KForm } from './KForm.svelte'
export { default as KFormField } from './KFormField.svelte'
export { default as KInput } from './KInput.svelte'
export { default as KTable } from './KTable.svelte'
export { default as KViewSlots } from './KViewSlots.svelte'
export { default as Layout } from './Layout.svelte'
export { default as LayoutActions } from './LayoutActions.svelte'
export { default as LayoutAlerts } from './LayoutAlerts.svelte'
export { default as LayoutMain } from './LayoutMain.svelte'
export { default as LayoutMenu } from './LayoutMenu.svelte'
export { default as LayoutTabs } from './LayoutTabs.svelte'
export { default as MenuBar } from './MenuBar.svelte'
export { default as MenuOutlet } from './MenuOutlet.svelte'
export { default as Modal } from './Modal.svelte'
export { default as Page } from './Page.svelte'
export { default as PageHeader } from './PageHeader.svelte'
export { default as SearchInput } from './SearchInput.svelte'
export { default as Select } from './Select.svelte'
export { default as Spinner } from './Spinner.svelte'
export { default as StatCard } from './StatCard.svelte'
export { default as TabPanel } from './TabPanel.svelte'
export { default as Table } from './Table.svelte'
export { default as Tabs } from './Tabs.svelte'
export { default as Toolbar } from './Toolbar.svelte'
export { default as TreeView } from './TreeView.svelte'

export type * from './types'
export type { KFormActionsContext, KFormContext, KFormFieldContext } from './kform-types'
export { getKFormContext } from './kform-context'
export { registerViewComponent, resolveViewComponent } from './view-components'
export { firstMenuLeaf, flattenMenuItems } from './menu'
export { fetchAppMenus } from './menu-client'
export {
  createModelRecord,
  deleteModelRecord,
  fetchAppPing,
  fetchModelViews,
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
export { getMenuContext, inferAppName, type MenuContext } from './layout-context'
export {
  appModuleUrl,
  contentAppForMenu,
  type SpaMountContext,
} from './spa-mount'
export { fetchI18n, applyLocale, syncDocumentLocale, type I18nBundle, type I18nEntry, type TextDirection, type Translator } from './i18n'
export { getI18n, t, type I18nScope } from './i18n-context'
export { setI18nLocale, getI18nLocale } from './i18n-runtime.svelte'
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
