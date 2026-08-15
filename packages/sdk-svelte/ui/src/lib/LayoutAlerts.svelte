<script lang="ts">
  import { getContext } from 'svelte'
  import { LAYOUT_CTX, type LayoutRegistry } from './layout-context'

  type Props = {
    children: import('svelte').Snippet
  }

  let { children }: Props = $props()
  const registry = getContext<LayoutRegistry>(LAYOUT_CTX)

  $effect(() => {
    registry.setAlerts(children)
    return () => registry.setAlerts(null)
  })
</script>
