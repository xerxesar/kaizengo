import { useEffect, useState, type ReactNode } from 'react'
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu'
import { getLayoutRegistry, inferAppName } from '@/lib/layout-context'
import { fetchAppMenus } from '@/lib/menu-client'
import {
  currentMenuPage,
  menuItemHref,
  menuPagePath,
  menuRouteOf,
  navigateApp,
  resolveMenuSelection,
} from '@/lib/menu-route'
import { cn } from '@/lib/utils'
import { menubarLinkClass } from '@/lib/styles/classes'
import type { MenuItem } from '@/lib/types'

type Props = {
  app?: string
  label?: string
  when?: boolean
}

function MenuNodes({
  nodes,
  depth = 0,
  app,
  active,
}: {
  nodes: MenuItem[]
  depth?: number
  app: string
  active: string
}) {
  return (
    <>
      {nodes.map((node) => {
        const isLeaf = !node.children?.length
        if (isLeaf) {
          return (
            <DropdownMenuItem
              key={node.id}
              className={cn(active === node.id && 'font-semibold text-[var(--kg-primary)]')}
              style={{ paddingLeft: `calc(1rem + ${depth} * 1rem)` }}
              onSelect={() => navigateApp(menuItemHref(app, node))}
            >
              {node.label}
            </DropdownMenuItem>
          )
        }
        return (
          <div key={node.id} className="py-1">
            <DropdownMenuLabel style={{ paddingLeft: `calc(1rem + ${depth} * 1rem)` }}>
              {node.label}
            </DropdownMenuLabel>
            <MenuNodes nodes={node.children ?? []} depth={depth + 1} app={app} active={active} />
          </div>
        )
      })}
    </>
  )
}

function MenuBar({
  app,
  items = [],
  active = '',
  label,
}: {
  app: string
  items?: MenuItem[]
  active?: string
  label?: string
}) {
  function hasActiveDescendant(item: MenuItem): boolean {
    if (item.id === active) return true
    return (item.children ?? []).some(hasActiveDescendant)
  }

  return (
    <nav
      className="kg-menubar w-full border-b border-[var(--kg-border)] bg-[var(--kg-surface,var(--kg-bg))]"
      aria-label={label ?? 'Menu'}
    >
      <div className="flex min-h-10 w-full items-stretch overflow-x-auto px-[var(--kg-page-padding-x)]">
        {items.map((item) => {
          const isLeaf = !item.children?.length
          if (isLeaf) {
            return (
              <a
                key={item.id}
                className={cn(
                  menubarLinkClass,
                  active === item.id && 'border-[var(--kg-primary)] font-semibold text-[var(--kg-primary)]',
                )}
                href={menuItemHref(app, item)}
                aria-current={active === item.id ? 'page' : undefined}
                onClick={(e) => {
                  e.preventDefault()
                  navigateApp(menuItemHref(app, item))
                }}
              >
                {item.label}
              </a>
            )
          }
          return (
            <DropdownMenu key={item.id}>
              <DropdownMenuTrigger
                className={cn(
                  menubarLinkClass,
                  hasActiveDescendant(item) &&
                    'border-[var(--kg-primary)] font-semibold text-[var(--kg-primary)]',
                )}
              >
                <span>{item.label}</span>
                <span className="-mt-0.5 text-xs opacity-85" aria-hidden="true">
                  ▾
                </span>
              </DropdownMenuTrigger>
              <DropdownMenuContent className="max-h-[min(24rem,70vh)]">
                <MenuNodes nodes={item.children ?? []} app={app} active={active} />
              </DropdownMenuContent>
            </DropdownMenu>
          )
        })}
      </div>
    </nav>
  )
}

export function LayoutMenu(props: Props) {
  const registry = getLayoutRegistry()
  const when = props.when ?? true

  const [appName, setAppName] = useState('')
  const [items, setItems] = useState<MenuItem[]>([])
  const [active, setActive] = useState('')
  const [loading, setLoading] = useState(true)
  const [loadError, setLoadError] = useState('')
  const [, setUrlVersion] = useState(0)

  const selected =
    items.length > 0 ? resolveMenuSelection(items, currentMenuPage()) : null
  const ready = !loading && !loadError
  const hasMenus = items.length > 0
  const show = when && (loading || hasMenus || Boolean(loadError))

  function syncFromUrl(list: MenuItem[], name: string) {
    if (!list.length) {
      setActive('')
      return
    }
    const page = currentMenuPage()
    const leaf = resolveMenuSelection(list, page)
    setActive(leaf?.id ?? '')
    if (!name || !leaf) return
    const want = menuRouteOf(leaf)
    if (page !== want) navigateApp(menuPagePath(name, want), true, { keepSearch: true })
  }

  useEffect(() => {
    const name = props.app?.trim() || inferAppName()
    if (!name) {
      setLoadError('Could not infer app name for menus')
      setLoading(false)
      return
    }
    let cancelled = false
    ;(async () => {
      setAppName(name)
      setLoading(true)
      setLoadError('')
      try {
        const menus = await fetchAppMenus(name)
        if (cancelled) return
        setItems(menus)
        setUrlVersion((v) => v + 1)
        syncFromUrl(menus, name)
      } catch (e) {
        if (cancelled) return
        setLoadError(e instanceof Error ? e.message : String(e))
        setItems([])
        setActive('')
      } finally {
        if (!cancelled) setLoading(false)
      }
    })()
    return () => {
      cancelled = true
    }
  }, [props.app])

  useEffect(() => {
    const onLoc = () => {
      setUrlVersion((v) => v + 1)
      if (!items.length) return
      syncFromUrl(items, appName)
    }
    window.addEventListener('popstate', onLoc)
    window.addEventListener('kaizengo:location', onLoc)
    return () => {
      window.removeEventListener('popstate', onLoc)
      window.removeEventListener('kaizengo:location', onLoc)
      registry.setNav(null)
    }
  }, [items, appName, registry])

  useEffect(() => {
    registry.setMenuState({
      app: appName,
      ready,
      error: loadError,
      active,
      selected,
      hasMenus,
    })
  }, [appName, ready, loadError, active, selected?.id, hasMenus, registry])

  useEffect(() => {
    if (!show) {
      registry.setNav(null)
      return
    }
    const nav: ReactNode = loadError ? (
      <p className="kg-menubar-error px-6 py-2 text-sm text-red-600">{loadError}</p>
    ) : !loading && appName && hasMenus ? (
      <MenuBar app={appName} items={items} active={active} label={props.label} />
    ) : null
    registry.setNav(nav)
  }, [show, loadError, loading, appName, hasMenus, items, active, props.label, registry])

  return null
}
