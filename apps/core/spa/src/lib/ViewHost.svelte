<script lang="ts">
  import { I18nProvider, Spinner, contentAppForMenu, getMenuContext, type MenuItem } from '@kaizengo/sdk-svelte/ui'
  import { resolveView } from './views/registry'

  type Props = {
    /** URL/host app (owns the menu tree). */
    hostApp: string
    onerror?: (message: string) => void
  }

  let { hostApp, onerror }: Props = $props()

  const menu = getMenuContext()
  const ready = $derived(menu.ready())
  const error = $derived(menu.error())
  const selected = $derived(menu.selected())
  const hasMenus = $derived(menu.hasMenus())

  const contentApp = $derived.by(() => {
    if (hasMenus && selected) return contentAppForMenu(selected, hostApp)
    return hostApp
  })

  const ViewComponent = $derived.by(() => {
    if (!contentApp) return null
    const item = hasMenus ? selected : null
    return resolveView({
      app: contentApp,
      view: item?.view || undefined,
      component: item?.component || undefined,
    })
  })

  $effect(() => {
    if (error) {
      onerror?.(error)
      return
    }
    if (!ready) return
    if (hasMenus && !selected) return
    if (!ViewComponent) {
      onerror?.(missingViewMessage(contentApp, selected))
      return
    }
    onerror?.('')
  })

  function missingViewMessage(app: string, item: MenuItem | null): string {
    if (item?.component) return `No component registered for ${item.component}`
    if (item?.view) return `No view ${app}.${item.view}`
    return `No default view for app ${app}`
  }
</script>

{#if !ready && !error}
  <Spinner />
{:else if ViewComponent}
  {#key `${contentApp}::${selected?.view ?? ''}::${selected?.component ?? ''}`}
    <I18nProvider app={contentApp}>
      {#snippet children()}
        <ViewComponent />
      {/snippet}
    </I18nProvider>
  {/key}
{/if}

<style>
  :global(.kg-view-host) {
    display: flex;
    flex-direction: column;
    flex: 1;
    min-width: 0;
    min-height: 0;
    width: 100%;
  }
</style>
