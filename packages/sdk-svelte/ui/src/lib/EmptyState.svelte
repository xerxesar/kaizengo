<script lang="ts">
  type Props = {
    title?: string
    message?: string
    action?: import('svelte').Snippet
  }

  let { title = 'No data', message = '', action }: Props = $props()
</script>

<div class="kg-empty">
  <div class="kg-empty-icon">📋</div>
  <h3 class="kg-empty-title">{title}</h3>
  {#if message}<p class="kg-empty-msg">{message}</p>{/if}
  {#if action}<div class="kg-empty-action">{@render action()}</div>{/if}
</div>

<style>
  .kg-empty {
    text-align: center;
    padding: 3rem 1.5rem;
    color: var(--kg-text-muted);
  }
  .kg-empty-icon {
    font-size: 2rem;
    margin-bottom: 0.75rem;
    opacity: 0.5;
  }
  .kg-empty-title {
    font-size: 1rem;
    font-weight: 600;
    color: var(--kg-text-secondary);
    margin-bottom: 0.35rem;
  }
  .kg-empty-msg {
    font-size: 0.875rem;
    max-width: 24rem;
    margin: 0 auto;
  }
  .kg-empty-action {
    margin-top: 1rem;
  }
</style>
