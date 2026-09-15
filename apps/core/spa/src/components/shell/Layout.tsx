import { useMemo, useState, type ReactNode } from 'react'
import { Progress } from '@/components/ui/progress'
import {
  LayoutContext,
  MenuContext,
  type LayoutRegistry,
  type LayoutSlot,
  type MenuState,
} from '@/lib/layout-context'
import type { ContainerAlign, ContainerSize } from '@/lib/types'

type Props = {
  title?: string
  subtitle?: string
  loading?: boolean
  variant?: 'app' | 'centered'
  containerSize?: ContainerSize
  align?: ContainerAlign
  fullWidth?: boolean
  children?: ReactNode
}

function Container({
  children,
  size = 'fluid',
  align = 'start',
}: {
  children?: ReactNode
  size?: ContainerSize
  align?: ContainerAlign
}) {
  const sizeClass =
    size === 'sm'
      ? 'max-w-[var(--kg-container-sm)]'
      : size === 'md'
        ? 'max-w-[var(--kg-container-md)]'
        : size === 'lg'
          ? 'max-w-[var(--kg-container-lg)]'
          : size === 'xl'
            ? 'max-w-[var(--kg-container-xl)]'
            : 'max-w-none'
  const alignClass = align === 'center' ? 'mx-auto' : 'me-auto'
  return <div className={`w-full px-[var(--kg-page-padding-x)] ${sizeClass} ${alignClass}`}>{children}</div>
}

function menuStateEqual(a: MenuState, b: MenuState): boolean {
  return (
    a.app === b.app &&
    a.ready === b.ready &&
    a.error === b.error &&
    a.active === b.active &&
    a.selected?.id === b.selected?.id &&
    a.hasMenus === b.hasMenus
  )
}

export function Layout(props: Props) {
  const [slotActions, setSlotActions] = useState<LayoutSlot | null>(null)
  const [slotAlerts, setSlotAlerts] = useState<LayoutSlot | null>(null)
  const [slotNav, setSlotNav] = useState<LayoutSlot | null>(null)
  const [menu, setMenu] = useState<MenuState>({
    app: '',
    ready: false,
    error: '',
    active: '',
    selected: null,
    hasMenus: false,
  })

  const registry = useMemo<LayoutRegistry>(
    () => ({
      setActions: setSlotActions,
      setAlerts: setSlotAlerts,
      setNav: setSlotNav,
      setTabs: setSlotNav,
      setMain: () => {},
      setMenuState: (state) => setMenu((prev) => (menuStateEqual(prev, state) ? prev : state)),
    }),
    [],
  )

  const pageAlign = props.variant === 'centered' ? 'center' : (props.align ?? 'start')

  return (
    <LayoutContext.Provider value={registry}>
      <MenuContext.Provider value={menu}>
        <div
          className={`kg-layout flex min-h-full w-full flex-1 flex-col bg-[var(--kg-bg)] ${
            props.variant === 'centered' ? 'min-h-[calc(100vh-3rem)] justify-center' : ''
          }`}
        >
          {slotNav ? <div className="kg-layout-menubar w-full shrink-0">{slotNav}</div> : null}

          {props.title ? (
            <div className="w-full border-b border-[var(--kg-border)] bg-[var(--kg-header-bg)]">
              <Container size={props.containerSize ?? 'fluid'} align={pageAlign}>
                <div className="flex flex-wrap items-start justify-between gap-4 py-4">
                  <div>
                    <h1 className="text-2xl font-normal text-[var(--kg-text)]">{props.title}</h1>
                    {props.subtitle ? (
                      <p className="mt-1 text-sm text-[var(--kg-text-secondary)]">{props.subtitle}</p>
                    ) : null}
                  </div>
                  {slotActions}
                </div>
              </Container>
            </div>
          ) : null}

          {props.fullWidth ? (
            <main className="kg-layout-main w-full min-w-0 flex-1">
              <MainBody loading={props.loading}>{props.children}</MainBody>
            </main>
          ) : (
            <Container size={props.containerSize ?? 'fluid'} align={pageAlign}>
              <div className="flex w-full min-w-0 flex-col gap-5 py-[var(--kg-page-padding-y)]">
                {slotAlerts ? <div className="kg-layout-alerts w-full">{slotAlerts}</div> : null}
                <main className="kg-layout-main w-full min-w-0 flex-1">
                  <MainBody loading={props.loading}>{props.children}</MainBody>
                </main>
              </div>
            </Container>
          )}
        </div>
      </MenuContext.Provider>
    </LayoutContext.Provider>
  )
}

function MainBody({ children, loading }: { children?: ReactNode; loading?: boolean }) {
  if (loading) {
    return (
      <div className="kg-layout-body flex w-full min-w-0 flex-col gap-5">
        <div className="flex items-center justify-center py-8" role="status" aria-label="Loading">
          <Progress indeterminate className="w-48 max-w-full" />
        </div>
      </div>
    )
  }
  return <div className="kg-layout-body flex w-full min-w-0 flex-col gap-5">{children}</div>
}
