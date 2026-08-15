<script lang="ts">
  type Props = {
    title: string
    subtitle?: string
    actions?: import('svelte').Snippet
    children?: import('svelte').Snippet
  }

  let { title, subtitle, actions, children }: Props = $props()
</script>

<header class="kg-page-header">
  <div class="kg-page-header-text">
    <h1 class="kg-page-title">{title}</h1>
    {#if subtitle}<p class="kg-page-subtitle">{subtitle}</p>{/if}
  </div>
  {#if actions}
    <div class="kg-page-actions">{@render actions()}</div>
  {/if}
</header>
{#if children}
  <div class="kg-page-extra">{@render children()}</div>
{/if}

<style>
  .kg-page-header {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: var(--kg-space-06);
    padding-block: var(--kg-space-06);
    background: transparent;
  }

  .kg-page-title {
    font-family: var(--kg-font-display);
    font-size: 2rem;
    font-weight: 400;
    line-height: 1.25;
    color: var(--kg-text);
    letter-spacing: 0;
  }

  .kg-page-subtitle {
    margin-top: var(--kg-space-03);
    font-size: 0.875rem;
    font-weight: 400;
    color: var(--kg-text-secondary);
    max-width: 40rem;
    line-height: 1.42857;
    letter-spacing: 0.16px;
  }

  .kg-page-actions {
    display: flex;
    align-items: center;
    gap: var(--kg-space-03);
    flex-shrink: 0;
  }

  .kg-page-extra {
    padding-block: 0 0 var(--kg-space-05);
  }
</style>
