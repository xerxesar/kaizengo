import { createSignal, Show, type JSX, type ParentProps } from 'solid-js'
import { Spinner } from './Spinner'
import {
  LayoutContext,
  MenuContext,
  type LayoutRegistry,
  type LayoutSlot,
  type MenuContext as MenuContextValue,
  type MenuState,
} from './layout-context'
import type { ContainerAlign, ContainerSize } from './types'

type Props = ParentProps & {
  title?: string
  subtitle?: string
  loading?: boolean
  variant?: 'app' | 'centered'
  containerSize?: ContainerSize
  align?: ContainerAlign
  fullWidth?: boolean
}

function Container(props: ParentProps & { size?: ContainerSize; align?: ContainerAlign }) {
  const sizeClass =
    props.size === 'sm'
      ? 'max-w-[var(--kg-container-sm)]'
      : props.size === 'md'
        ? 'max-w-[var(--kg-container-md)]'
        : props.size === 'lg'
          ? 'max-w-[var(--kg-container-lg)]'
          : props.size === 'xl'
            ? 'max-w-[var(--kg-container-xl)]'
            : 'max-w-none'

  const alignClass =
    props.align === 'center' ? 'mx-auto' : props.align === 'end' ? 'ms-auto' : 'me-auto'

  return (
    <div class={`w-full px-[var(--kg-page-padding-x)] ${sizeClass} ${alignClass}`}>
      {props.children}
    </div>
  )
}

function setLayoutSlot(
  setter: (value: LayoutSlot | null) => void,
  slot: LayoutSlot | null,
): void {
  if (slot === null) {
    setter(null)
    return
  }
  setter((prev) => (prev === slot ? prev : slot))
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

function renderSlot(slot: LayoutSlot | null | undefined): JSX.Element {
  if (!slot) return undefined as unknown as JSX.Element
  return slot()
}

function PageHeader(props: { title: string; subtitle?: string; actions?: LayoutSlot | null }) {
  return (
    <div class="flex flex-wrap items-start justify-between gap-4 py-4">
      <div>
        <h1 class="text-2xl font-normal text-[var(--kg-text)]">{props.title}</h1>
        <Show when={props.subtitle}>
          <p class="mt-1 text-sm text-[var(--kg-text-secondary)]">{props.subtitle}</p>
        </Show>
      </div>
      <Show when={props.actions}>{renderSlot(props.actions ?? null)}</Show>
    </div>
  )
}

function MainBody(props: ParentProps & { loading?: boolean }) {
  return (
    <Show
      when={!props.loading}
      fallback={
        <div class="kg-layout-body flex w-full min-w-0 flex-col gap-5">
          <Spinner />
        </div>
      }
    >
      <div class="kg-layout-body flex w-full min-w-0 flex-col gap-5">{props.children}</div>
    </Show>
  )
}

export function Layout(props: Props): JSX.Element {
  const [slotActions, setSlotActions] = createSignal<LayoutSlot | null>(null)
  const [slotAlerts, setSlotAlerts] = createSignal<LayoutSlot | null>(null)
  const [slotNav, setSlotNav] = createSignal<LayoutSlot | null>(null)
  const [menu, setMenu] = createSignal<MenuState>({
    app: '',
    ready: false,
    error: '',
    active: '',
    selected: null,
    hasMenus: false,
  })

  const menuCtx: MenuContextValue = {
    app: () => menu().app,
    ready: () => menu().ready,
    error: () => menu().error,
    active: () => menu().active,
    selected: () => menu().selected,
    hasMenus: () => menu().hasMenus,
  }

  const registry: LayoutRegistry = {
    setActions: (slot) => setLayoutSlot(setSlotActions, slot),
    setAlerts: (slot) => setLayoutSlot(setSlotAlerts, slot),
    setNav: (slot) => setLayoutSlot(setSlotNav, slot),
    setTabs: (slot) => setLayoutSlot(setSlotNav, slot),
    setMain: () => {},
    setMenuState: (state) => setMenu((prev) => (menuStateEqual(prev, state) ? prev : state)),
  }

  const pageAlign = () => (props.variant === 'centered' ? 'center' : (props.align ?? 'start'))
  const hasHeader = () => Boolean(props.title)

  return (
    <LayoutContext.Provider value={registry}>
      <MenuContext.Provider value={menuCtx}>
        <div
          class={`kg-layout flex min-h-full w-full flex-1 flex-col bg-[var(--kg-bg)] ${props.variant === 'centered' ? 'min-h-[calc(100vh-3rem)] justify-center' : ''}`}
        >
          <Show when={slotNav()}>
            <div class="kg-layout-menubar w-full shrink-0">{renderSlot(slotNav())}</div>
          </Show>

          <Show when={hasHeader()}>
            <div class="w-full border-b border-[var(--kg-border)] bg-[var(--kg-header-bg)]">
              <Container size={props.containerSize ?? 'fluid'} align={pageAlign()}>
                <PageHeader title={props.title!} subtitle={props.subtitle} actions={slotActions()} />
              </Container>
            </div>
          </Show>

          <Show
            when={props.fullWidth}
            fallback={
              <Container size={props.containerSize ?? 'fluid'} align={pageAlign()}>
                <div class="flex w-full min-w-0 flex-col gap-5 py-[var(--kg-page-padding-y)]">
                  <Show when={slotAlerts()}>
                    <div class="kg-layout-alerts w-full">{renderSlot(slotAlerts())}</div>
                  </Show>
                  <main class="kg-layout-main w-full min-w-0 flex-1">
                    <MainBody loading={props.loading}>{props.children}</MainBody>
                  </main>
                </div>
              </Container>
            }
          >
            <main class="kg-layout-main w-full min-w-0 flex-1">
              <MainBody loading={props.loading}>{props.children}</MainBody>
            </main>
          </Show>
        </div>
      </MenuContext.Provider>
    </LayoutContext.Provider>
  )
}
