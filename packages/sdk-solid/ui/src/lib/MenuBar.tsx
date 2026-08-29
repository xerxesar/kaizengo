import { Menu } from '@ark-ui/solid/menu'
import { For, Show, type JSX } from 'solid-js'
import { menubarLinkClass, menuContentClass, menuItemClass } from './ark/styles'
import { cn } from './cn'
import { menuItemHref, navigateApp } from './menu-route'
import type { MenuItem } from './types'

type Props = {
  app: string
  items?: MenuItem[]
  active?: string
  label?: string
}

type MenuNodesProps = {
  nodes: MenuItem[]
  depth?: number
  app: string
  active: string
}

function MenuNodes(props: MenuNodesProps): JSX.Element {
  const depth = () => props.depth ?? 0
  const nodes = () => props.nodes ?? []

  function isLeaf(item: MenuItem) {
    return !item.children?.length
  }

  function go(item: MenuItem) {
    navigateApp(menuItemHref(props.app, item))
  }

  return (
    <For each={nodes()}>
      {(node) => (
        <Show
          when={!isLeaf(node)}
          fallback={
            <Menu.Item
              value={node.id}
              class={cn(
                menuItemClass,
                props.active === node.id && 'font-semibold text-[var(--kg-primary)]',
              )}
              style={{ 'padding-left': `calc(1rem + ${depth()} * 1rem)` }}
              onSelect={() => go(node)}
            >
              {node.label}
            </Menu.Item>
          }
        >
          <div class="py-1">
            <div
              class="px-4 py-1 text-xs font-semibold uppercase tracking-wide text-[var(--kg-text-muted)]"
              style={{ 'padding-left': `calc(1rem + ${depth()} * 1rem)` }}
            >
              {node.label}
            </div>
            <MenuNodes nodes={node.children ?? []} depth={depth() + 1} app={props.app} active={props.active} />
          </div>
        </Show>
      )}
    </For>
  )
}

export function MenuBar(props: Props): JSX.Element {
  const items = () => props.items ?? []
  const active = () => props.active ?? ''

  function isLeaf(item: MenuItem) {
    return !item.children?.length
  }

  function hasActiveDescendant(item: MenuItem): boolean {
    if (item.id === active()) return true
    return (item.children ?? []).some(hasActiveDescendant)
  }

  function go(item: MenuItem, e?: MouseEvent) {
    e?.preventDefault()
    navigateApp(menuItemHref(props.app, item))
  }

  return (
    <nav
      class="kg-menubar w-full border-b border-[var(--kg-border)] bg-[var(--kg-surface,var(--kg-bg))]"
      aria-label={props.label ?? 'Menu'}
    >
      <div class="flex min-h-10 w-full items-stretch overflow-x-auto px-[var(--kg-page-padding-x)]">
        <For each={items()}>
          {(item) => (
            <Show
              when={isLeaf(item)}
              fallback={
                <Menu.Root positioning={{ placement: 'bottom-start' }}>
                  <Menu.Trigger
                    class={cn(
                      menubarLinkClass,
                      hasActiveDescendant(item) && 'border-[var(--kg-primary)] font-semibold text-[var(--kg-primary)]',
                    )}
                  >
                    <span>{item.label}</span>
                    <span class="text-xs opacity-85 -mt-0.5" aria-hidden="true">
                      ▾
                    </span>
                  </Menu.Trigger>
                  <Menu.Positioner>
                    <Menu.Content class={cn(menuContentClass, 'max-h-[min(24rem,70vh)]')}>
                      <MenuNodes nodes={item.children ?? []} app={props.app} active={active()} />
                    </Menu.Content>
                  </Menu.Positioner>
                </Menu.Root>
              }
            >
              <a
                class={cn(menubarLinkClass, active() === item.id && 'border-[var(--kg-primary)] font-semibold text-[var(--kg-primary)]')}
                href={menuItemHref(props.app, item)}
                aria-current={active() === item.id ? 'page' : undefined}
                onClick={(e) => go(item, e)}
              >
                {item.label}
              </a>
            </Show>
          )}
        </For>
      </div>
    </nav>
  )
}
