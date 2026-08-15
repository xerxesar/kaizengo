<script lang="ts">
  import type { Component } from 'svelte'
  import { resolveComponent } from './registry'

  export type ViewSlotSpec = {
    slot: string
    component: string
  }

  type Props = {
    slots: ViewSlotSpec[]
    slotName: string
  }

  let { slots, slotName }: Props = $props()

  const entries = $derived(
    slots
      .filter((s) => s.slot === slotName)
      .map((s) => ({ spec: s, component: resolveComponent(s.component) }))
      .filter((e): e is { spec: ViewSlotSpec; component: Component } => e.component != null),
  )
</script>

{#each entries as { component: SlotComponent } (SlotComponent)}
  <SlotComponent />
{/each}
