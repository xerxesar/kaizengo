import { createEffect, createMemo, createSignal, on, Show, type Component } from 'solid-js'
import { Dynamic } from 'solid-js/web'
import { I18nProvider, Spinner, contentAppForMenu, getMenuContext } from '@kaizengo/sdk-solid/ui'
import type { MenuItem } from '@kaizengo/sdk-solid/ui'
import { resolveView } from './views/registry'

type Props = {
  hostApp: string
  onerror?: (message: string) => void
}

export function ViewHost(props: Props) {
  const menu = getMenuContext()
  const ready = () => menu.ready()
  const error = () => menu.error()
  const selected = () => menu.selected()
  const hasMenus = () => menu.hasMenus()

  const [reportedError, setReportedError] = createSignal('')

  const contentApp = createMemo(() => {
    if (hasMenus() && selected()) return contentAppForMenu(selected()!, props.hostApp)
    return props.hostApp
  })

  const ViewComponent = createMemo<Component | null>(() => {
    const app = contentApp()
    if (!app) return null
    const item = hasMenus() ? selected() : null
    return resolveView({
      app,
      view: item?.view || undefined,
      component: item?.component || undefined,
    })
  })

  const viewMountKey = createMemo(() => {
    const View = ViewComponent()
    if (!View) return ''
    const item = hasMenus() ? selected() : null
    return `${contentApp()}::${item?.view ?? ''}::${item?.component ?? ''}`
  })

  function missingViewMessage(app: string, item: MenuItem | null): string {
    if (item?.component) return `No component registered for ${item.component}`
    if (item?.view) return `No view ${app}.${item.view}`
    return `No default view for app ${app}`
  }

  createEffect(
    on(
      () => {
        const err = error()
        if (err) return err
        if (!ready()) return null
        if (hasMenus() && !selected()) return null
        const view = ViewComponent()
        return view ? '' : missingViewMessage(contentApp(), selected())
      },
      (message) => {
        if (message == null) return
        if (message === reportedError()) return
        setReportedError(message)
        props.onerror?.(message)
      },
    ),
  )

  return (
    <div class="kg-view-host flex min-h-0 w-full min-w-0 flex-1 flex-col">
      <Show when={ready() || error()} fallback={<Spinner />}>
        <Show when={viewMountKey()} keyed>
          {(_key) => (
            <I18nProvider app={contentApp()}>
              <Dynamic component={ViewComponent()!} />
            </I18nProvider>
          )}
        </Show>
      </Show>
    </div>
  )
}
