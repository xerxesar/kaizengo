<script lang="ts">
  import { getContext, onMount } from 'svelte'
  import MenuBar from './MenuBar.svelte'
  import {
    LAYOUT_CTX,
    inferAppName,
    type LayoutRegistry,
  } from './layout-context'
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
    /** Host app whose menus to load (required when used from core). */
    app?: string
    label?: string
    when?: boolean
  }

  let { app: appProp = '', label, when = true }: Props = $props()

  let appName = $state('')
  let items = $state<MenuItem[]>([])
  let active = $state('')
  let loading = $state(true)
  let loadError = $state('')
  let locationTick = $state(0)

  const selected = $derived.by(() => {
    locationTick
    if (!items.length) return null
    return resolveMenuSelection(items, currentMenuPage())
  })
  const ready = $derived(!loading && !loadError)
  const hasMenus = $derived(items.length > 0)
  const show = $derived(when && (loading || hasMenus || Boolean(loadError)))

  const registry = getContext<LayoutRegistry>(LAYOUT_CTX)

  function syncFromUrl() {
    locationTick++
    if (!items.length) {
      active = ''
      return
    }
    const leaf = resolveMenuSelection(items, currentMenuPage())
    active = leaf?.id ?? ''
    if (!appName || !leaf) return
    const page = currentMenuPage()
    const want = menuRouteOf(leaf)
    if (!page || page !== want) {
      navigateApp(menuPagePath(appName, want), true)
    }
  }

  $effect(() => {
    registry.setMenuState({
      app: appName,
      ready,
      error: loadError,
      active,
      selected,
      hasMenus,
    })
  })

  async function load(name: string) {
    if (!name) {
      loadError = 'Could not infer app name for menus'
      loading = false
      return
    }
    appName = name
    loading = true
    loadError = ''
    try {
      const menus = await fetchAppMenus(name)
      items = menus
      syncFromUrl()
    } catch (e) {
      loadError = e instanceof Error ? e.message : String(e)
      items = []
      active = ''
    } finally {
      loading = false
    }
  }

  onMount(() => {
    const onLoc = () => {
      if (!items.length) return
      syncFromUrl()
    }
    window.addEventListener('popstate', onLoc)
    window.addEventListener('kaizengo:location', onLoc)
    void load(appProp.trim() || inferAppName())
    return () => {
      window.removeEventListener('popstate', onLoc)
      window.removeEventListener('kaizengo:location', onLoc)
    }
  })

  $effect(() => {
    if (!show) {
      registry.setNav(null)
      return
    }
    registry.setNav(menuBar)
    return () => registry.setNav(null)
  })
</script>

{#snippet menuBar()}
  {#if loadError}
    <p class="kg-menubar-error">{loadError}</p>
  {:else if !loading && appName && hasMenus}
    <MenuBar app={appName} {items} {active} {label} />
  {/if}
{/snippet}

<style>
  :global(.kg-menubar-error) {
    margin: 0;
    padding: var(--kg-space-03) var(--kg-page-padding-x);
    color: var(--kg-danger, #da1e28);
    font-size: 0.875rem;
  }
</style>
