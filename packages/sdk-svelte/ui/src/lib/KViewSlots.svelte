<script lang="ts">
  import type { Component } from 'svelte'
  import type { ViewSlot } from './model-client'
  import { resolveViewComponent } from './view-components'

  type Props = {
    slots: ViewSlot[]
    slotName: string
    resolve?: (componentId: string) => Component | null
  }

  let { slots, slotName, resolve = resolveViewComponent }: Props = $props()

  const entries = $derived(
    slots
      .filter((slot) => slot.slot === slotName)
      .map((slot) => ({ slot, component: resolve(slot.component) }))
      .filter((entry): entry is { slot: ViewSlot; component: Component } => entry.component != null),
  )
</script>

{#each entries as { component: SlotComponent } (SlotComponent)}
  <div class="kg-view-slot" data-slot={slotName}>
    <SlotComponent />
  </div>
{/each}

<style>
  .kg-view-slot {
    display: contents;
  }
</style>
