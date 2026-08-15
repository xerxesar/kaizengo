<script lang="ts">
  type Props = {
    children: import('svelte').Snippet
    padding?: boolean
    boxed?: boolean
  }

  let { children, padding = true, boxed = false }: Props = $props()
</script>

<div class="kg-tab-panel" class:padded={padding} class:boxed={boxed}>
  {@render children()}
</div>

<style>
  .kg-tab-panel {
    width: 100%;
    min-width: 0;
    flex: 1;
    background: transparent;
  }

  .kg-tab-panel.padded {
    padding-top: 0;
  }

  .kg-tab-panel.boxed.padded {
    padding: var(--kg-space-06);
    background: var(--kg-surface);
    border: 1px solid var(--kg-border);
  }
</style>
