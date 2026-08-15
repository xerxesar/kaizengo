<script lang="ts">
  import { setContext } from 'svelte'
  import Container from './Container.svelte'
  import PageHeader from './PageHeader.svelte'
  import Spinner from './Spinner.svelte'
  import {
    LAYOUT_CTX,
    MENU_CTX,
    type LayoutRegistry,
    type MenuContext,
    type MenuState,
  } from './layout-context'
  import type { ContainerAlign, ContainerSize } from './types'

  type Props = {
    title?: string
    subtitle?: string
    loading?: boolean
    variant?: 'app' | 'centered'
    containerSize?: ContainerSize
    align?: ContainerAlign
    fullWidth?: boolean
    children: import('svelte').Snippet
  }

  let {
    title,
    subtitle,
    loading = false,
    variant = 'app',
    containerSize = 'fluid',
    align = 'start',
    fullWidth = false,
    children,
  }: Props = $props()

  let slotActions = $state<import('svelte').Snippet | null>(null)
  let slotAlerts = $state<import('svelte').Snippet | null>(null)
  let slotNav = $state<import('svelte').Snippet | null>(null)
  let slotMain = $state<import('svelte').Snippet | null>(null)
  let menu = $state<MenuState>({
    app: '',
    ready: false,
    error: '',
    active: '',
    selected: null,
    hasMenus: false,
  })

  setContext<MenuContext>(MENU_CTX, {
    app: () => menu.app,
    ready: () => menu.ready,
    error: () => menu.error,
    active: () => menu.active,
    selected: () => menu.selected,
    hasMenus: () => menu.hasMenus,
  })

  function setNav(value: import('svelte').Snippet | null) {
    slotNav = value
  }

  setContext<LayoutRegistry>(LAYOUT_CTX, {
    setActions(value) {
      slotActions = value
    },
    setAlerts(value) {
      slotAlerts = value
    },
    setNav,
    setTabs: setNav,
    setMain(value) {
      slotMain = value
    },
    setMenuState(state) {
      menu = state
    },
  })

  const pageAlign = $derived(variant === 'centered' ? 'center' : align)
  const hasHeader = $derived(Boolean(title))
</script>

<div class="kg-layout" class:centered={variant === 'centered'}>
  {#if slotNav}
    <div class="kg-layout-menubar">
      {@render slotNav()}
    </div>
  {/if}

  {#if hasHeader}
    <div class="kg-layout-header">
      <Container size={containerSize} align={pageAlign}>
        <PageHeader title={title!} {subtitle} actions={slotActions} />
      </Container>
    </div>
  {/if}

  {#if fullWidth}
    <main class="kg-layout-main">
      {#if loading}
        <Spinner />
      {:else if slotMain}
        {@render slotMain()}
      {/if}
    </main>
  {:else}
    <Container size={containerSize} align={pageAlign}>
      <div class="kg-layout-content">
        {#if slotAlerts}
          <div class="kg-layout-alerts">
            {@render slotAlerts()}
          </div>
        {/if}

        <main class="kg-layout-main">
          {#if loading}
            <Spinner />
          {:else if slotMain}
            <div class="kg-layout-body">
              {@render slotMain()}
            </div>
          {/if}
        </main>
      </div>
    </Container>
  {/if}

  <div class="kg-layout-slots" hidden aria-hidden="true">
    {@render children()}
  </div>
</div>

<style>
  .kg-layout {
    display: flex;
    flex-direction: column;
    flex: 1;
    min-height: 100%;
    width: 100%;
    background: var(--kg-bg);
  }

  .kg-layout.centered {
    justify-content: center;
    min-height: calc(100vh - 3rem);
  }

  .kg-layout-menubar {
    width: 100%;
    flex-shrink: 0;
  }

  .kg-layout-menubar:not(:has(:global(.kg-menubar))):not(:has(:global(.kg-tabs-list))):not(
      :has(:global(.kg-menubar-error))
    ) {
    display: none;
  }

  .kg-layout-header {
    width: 100%;
    background: var(--kg-header-bg);
    border-bottom: 1px solid var(--kg-border);
  }

  .kg-layout-content {
    display: flex;
    flex-direction: column;
    gap: var(--kg-space-05);
    width: 100%;
    min-width: 0;
    padding-block: var(--kg-page-padding-y);
  }

  .kg-layout-alerts:not(:has(:global(.kg-alert))) {
    display: none;
  }

  .kg-layout-alerts :global(.kg-alert) {
    margin-bottom: 0;
  }

  .kg-layout-alerts,
  .kg-layout-main {
    width: 100%;
    min-width: 0;
  }

  .kg-layout-main {
    flex: 1;
    padding-block: 0;
  }

  .kg-layout.centered .kg-layout-content {
    padding-block: var(--kg-page-padding-y);
  }

  .kg-layout-body {
    display: flex;
    flex-direction: column;
    gap: var(--kg-layout-gap);
    width: 100%;
    min-width: 0;
  }

  .kg-layout-slots {
    display: none;
  }
</style>
