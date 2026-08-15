<script lang="ts">
  import { getContext } from 'svelte'
  import Tabs from './Tabs.svelte'
  import { LAYOUT_CTX, type LayoutRegistry } from './layout-context'
  import type { TabItem } from './types'

  type Props = {
    tabs: TabItem[]
    active: string
    onchange?: (id: string) => void
    when?: boolean
  }

  let { tabs, active = $bindable(), onchange, when = true }: Props = $props()
  const registry = getContext<LayoutRegistry>(LAYOUT_CTX)

  $effect(() => {
    if (!when) {
      registry.setNav(null)
      return
    }
    registry.setNav(tabBar)
    return () => registry.setNav(null)
  })
</script>

{#snippet tabBar()}
  <Tabs {tabs} bind:active {onchange} />
{/snippet}
