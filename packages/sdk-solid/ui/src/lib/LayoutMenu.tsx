import { createEffect, createSignal, on, onCleanup, onMount, Show, type JSX } from 'solid-js'
import { MenuBar } from './MenuBar'
import { getLayoutRegistry, inferAppName } from './layout-context'
import type { LayoutSlot } from './layout-context'
import { fetchAppMenus } from './menu-client'
import {
  currentMenuPage,
  menuPagePath,
  menuRouteOf,
  navigateApp,
  resolveMenuSelection,
} from './menu-route'
import type { MenuItem } from './types'

type Props = {
  app?: string
  label?: string
  when?: boolean
}

export function LayoutMenu(props: Props): JSX.Element {
  const registry = getLayoutRegistry()
  const when = () => props.when ?? true

  const [appName, setAppName] = createSignal('')
  const [items, setItems] = createSignal<MenuItem[]>([])
  const [active, setActive] = createSignal('')
  const [loading, setLoading] = createSignal(true)
  const [loadError, setLoadError] = createSignal('')
  const [urlVersion, setUrlVersion] = createSignal(0)

  const selected = () => {
    urlVersion()
    const list = items()
    if (!list.length) return null
    return resolveMenuSelection(list, currentMenuPage())
  }

  const ready = () => !loading() && !loadError()
  const hasMenus = () => items().length > 0
  const show = () => when() && (loading() || hasMenus() || Boolean(loadError()))

  const navSlot: LayoutSlot = () => (
    <Show
      when={loadError()}
      fallback={
        <Show when={!loading() && appName() && hasMenus()}>
          <MenuBar app={appName()} items={items()} active={active()} label={props.label} />
        </Show>
      }
    >
      <p class="kg-menubar-error px-6 py-2 text-sm text-red-600">{loadError()}</p>
    </Show>
  )

  function syncFromUrl(list: MenuItem[], name: string) {
    if (!list.length) {
      if (active()) setActive('')
      return
    }

    const page = currentMenuPage()
    const leaf = resolveMenuSelection(list, page)
    const nextActive = leaf?.id ?? ''
    if (nextActive !== active()) setActive(nextActive)

    if (!name || !leaf) return

    const want = menuRouteOf(leaf)
    if (page !== want) {
      navigateApp(menuPagePath(name, want), true)
    }
  }

  async function load(name: string) {
    if (!name) {
      setLoadError('Could not infer app name for menus')
      setLoading(false)
      return
    }
    setAppName(name)
    setLoading(true)
    setLoadError('')
    try {
      const menus = await fetchAppMenus(name)
      setItems(menus)
      setUrlVersion((v) => v + 1)
      syncFromUrl(menus, name)
    } catch (e) {
      setLoadError(e instanceof Error ? e.message : String(e))
      setItems([])
      setActive('')
    } finally {
      setLoading(false)
    }
  }

  onMount(() => {
    const onLoc = () => {
      setUrlVersion((v) => v + 1)
      const list = items()
      if (!list.length) return
      syncFromUrl(list, appName())
    }
    window.addEventListener('popstate', onLoc)
    window.addEventListener('kaizengo:location', onLoc)
    onCleanup(() => {
      window.removeEventListener('popstate', onLoc)
      window.removeEventListener('kaizengo:location', onLoc)
      registry.setNav(null)
    })
  })

  createEffect(() => {
    const name = props.app?.trim() || inferAppName()
    if (name) void load(name)
  })

  createEffect(
    on(
      () =>
        [
          appName(),
          ready(),
          loadError(),
          active(),
          selected()?.id ?? '',
          hasMenus(),
        ] as const,
      ([app, isReady, err, act, selectedId, menus]) => {
        const sel = selected()
        registry.setMenuState({
          app,
          ready: isReady,
          error: err,
          active: act,
          selected: sel,
          hasMenus: menus,
        })
      },
    ),
  )

  createEffect(() => {
    registry.setNav(show() ? navSlot : null)
  })

  return null
}
