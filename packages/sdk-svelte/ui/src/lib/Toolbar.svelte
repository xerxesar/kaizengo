<script lang="ts">
  type Props = {
    children?: import('svelte').Snippet
    start?: import('svelte').Snippet
    end?: import('svelte').Snippet
  }

  let { children, start, end }: Props = $props()
</script>

<div class="kg-toolbar">
  <div class="kg-toolbar-start">
    {#if start}
      {@render start()}
    {:else if children}
      {@render children()}
    {/if}
  </div>
  {#if end}
    <div class="kg-toolbar-end">
      {@render end()}
    </div>
  {/if}
</div>

<style>
  .kg-toolbar {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 0.75rem;
    flex-wrap: wrap;
    min-height: var(--kg-control-height);
    padding: 0 0 0.875rem;
    background: transparent;
    border-bottom: none;
  }

  .kg-toolbar-start,
  .kg-toolbar-end {
    display: flex;
    align-items: center;
    gap: 0.625rem;
    flex-wrap: wrap;
  }

  .kg-toolbar-start {
    flex: 1;
    min-width: 0;
  }

  .kg-toolbar-end {
    flex-shrink: 0;
  }
</style>
