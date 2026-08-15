<script lang="ts">
  import type { ContainerAlign, ContainerSize } from './types'

  type Props = {
    size?: ContainerSize
    align?: ContainerAlign
    padding?: boolean
    center?: boolean
    children: import('svelte').Snippet
  }

  let {
    size = 'fluid',
    align = 'start',
    padding = true,
    center = false,
    children,
  }: Props = $props()

  const resolvedAlign = $derived(center ? 'center' : align)
</script>

<div
  class="kg-container"
  data-size={size}
  data-align={resolvedAlign}
  class:no-padding={!padding}
>
  {@render children()}
</div>

<style>
  .kg-container {
    width: 100%;
    margin-inline: 0;
    padding-inline: var(--kg-page-padding-x);
  }

  .kg-container.no-padding {
    padding-inline: 0;
  }

  .kg-container[data-align='center'] {
    margin-inline: auto;
  }

  .kg-container[data-size='sm'] {
    max-width: var(--kg-container-sm);
  }

  .kg-container[data-size='md'] {
    max-width: var(--kg-container-md);
  }

  .kg-container[data-size='lg'] {
    max-width: var(--kg-container-lg);
  }

  .kg-container[data-size='xl'] {
    max-width: var(--kg-container-xl);
  }

  .kg-container[data-size='fluid'] {
    max-width: none;
  }
</style>
