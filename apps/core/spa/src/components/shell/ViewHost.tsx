import { useEffect, useMemo, useState } from 'react'
import { Progress } from '@/components/ui/progress'
import { I18nProvider } from '@/lib/I18nProvider'
import { contentAppForMenu } from '@/lib/spa-mount'
import { getMenuContext } from '@/lib/layout-context'
import type { MenuItem } from '@/lib/types'
import { resolveView } from '@/lib/views/registry'

type Props = {
  hostApp: string
  onerror?: (message: string) => void
}

export function ViewHost({ hostApp, onerror }: Props) {
  const menu = getMenuContext()
  const [reportedError, setReportedError] = useState('')

  const contentApp = useMemo(() => {
    if (menu.hasMenus && menu.selected) return contentAppForMenu(menu.selected, hostApp)
    return hostApp
  }, [menu.hasMenus, menu.selected, hostApp])

  const ViewComponent = useMemo(() => {
    if (!contentApp) return null
    const item = menu.hasMenus ? menu.selected : null
    return resolveView({
      app: contentApp,
      view: item?.view || undefined,
      component: item?.component || undefined,
    })
  }, [contentApp, menu.hasMenus, menu.selected])

  const viewMountKey = useMemo(() => {
    if (!ViewComponent) return ''
    const item = menu.hasMenus ? menu.selected : null
    return `${contentApp}::${item?.view ?? ''}::${item?.component ?? ''}`
  }, [ViewComponent, contentApp, menu.hasMenus, menu.selected])

  useEffect(() => {
    let message: string | null = null
    if (menu.error) message = menu.error
    else if (!menu.ready) message = null
    else if (menu.hasMenus && !menu.selected) message = null
    else if (!ViewComponent) message = missingViewMessage(contentApp, menu.selected)
    else message = ''

    if (message == null) return
    if (message === reportedError) return
    setReportedError(message)
    onerror?.(message)
  }, [menu.error, menu.ready, menu.hasMenus, menu.selected, ViewComponent, contentApp, reportedError, onerror])

  if (!menu.ready && !menu.error) {
    return (
      <div className="kg-view-host flex min-h-0 w-full min-w-0 flex-1 flex-col">
        <div className="flex items-center justify-center py-8" role="status" aria-label="Loading">
          <Progress indeterminate className="w-48 max-w-full" />
        </div>
      </div>
    )
  }

  return (
    <div className="kg-view-host flex min-h-0 w-full min-w-0 flex-1 flex-col">
      {viewMountKey && ViewComponent ? (
        <I18nProvider app={contentApp} key={viewMountKey}>
          <ViewComponent />
        </I18nProvider>
      ) : null}
    </div>
  )
}

function missingViewMessage(app: string, item: MenuItem | null): string {
  if (item?.component) return `No component registered for ${item.component}`
  if (item?.view) return `No view ${app}.${item.view}`
  return `No default view for app ${app}`
}
