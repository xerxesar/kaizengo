<script lang="ts">
  type Props = {
    title?: string
    children: import('svelte').Snippet
    actions?: import('svelte').Snippet
  }

  let { title, children, actions }: Props = $props()
</script>

<section class="kg-card">
  {#if title || actions}
    <header class="kg-card-header">
      {#if title}<h3 class="kg-card-title">{title}</h3>{/if}
      {#if actions}<div class="kg-card-actions">{@render actions()}</div>{/if}
    </header>
  {/if}
  <div class="kg-card-body">{@render children()}</div>
</section>

<style>
  .kg-card {
    background: var(--kg-surface);
    border: 1px solid var(--kg-border);
    border-radius: var(--kg-radius);
  }

  .kg-card-header {
    display: flex;
    align-items: center;
    justify-content: space-between;
    padding: var(--kg-space-05) var(--kg-space-05);
    border-bottom: 1px solid var(--kg-border);
  }

  .kg-card-title {
    font-size: 0.875rem;
    font-weight: 600;
    letter-spacing: 0.16px;
  }

  .kg-card-actions {
    display: flex;
    gap: var(--kg-space-03);
  }

  .kg-card-body {
    padding: var(--kg-space-05);
  }
</style>
