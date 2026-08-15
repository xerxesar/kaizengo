<script lang="ts">
  import type { Snippet } from 'svelte'
  import { getMenuContext } from './layout-context'
  import type { MenuItem } from './types'
  import Spinner from './Spinner.svelte'

  type Props = {
    /** Render the active leaf menu. Receives `null` when there is no selection. */
    children: Snippet<[MenuItem | null]>
    onselect?: (item: MenuItem | null) => void
  }

  let { children, onselect }: Props = $props()
  const menu = getMenuContext()
  const ready = $derived(menu.ready())
  const error = $derived(menu.error())
  const selected = $derived(menu.selected())
  const hasMenus = $derived(menu.hasMenus())

  $effect(() => {
    onselect?.(selected)
  })
</script>

{#if !ready && !error}
  <Spinner />
{:else if hasMenus}
  {@render children(selected)}
{:else}
  {@render children(null)}
{/if}
