<script lang="ts">
  type Props = {
    open?: boolean
    title: string
    size?: 'sm' | 'md' | 'lg'
    onclose?: () => void
    footer?: import('svelte').Snippet
    children: import('svelte').Snippet
  }

  let { open = $bindable(false), title, size = 'md', onclose, footer, children }: Props = $props()

  function close() {
    open = false
    onclose?.()
  }

  function onKeydown(e: KeyboardEvent) {
    if (e.key === 'Escape') close()
  }
</script>

<svelte:window onkeydown={onKeydown} />

{#if open}
  <div class="kg-modal-backdrop" onclick={close} role="presentation"></div>
  <div class="kg-modal {size}" role="dialog" aria-modal="true" aria-labelledby="kg-modal-title">
    <header class="kg-modal-header">
      <h2 id="kg-modal-title" class="kg-modal-title">{title}</h2>
      <button type="button" class="kg-modal-close" aria-label="Close" onclick={close}>×</button>
    </header>
    <div class="kg-modal-body">{@render children()}</div>
    {#if footer}
      <footer class="kg-modal-footer">{@render footer()}</footer>
    {/if}
  </div>
{/if}

<style>
  .kg-modal-backdrop {
    position: fixed;
    inset: 0;
    background: rgba(22, 22, 22, 0.5);
    z-index: 9000;
  }

  .kg-modal {
    position: fixed;
    top: 50%;
    left: 50%;
    transform: translate(-50%, -50%);
    z-index: 9001;
    background: var(--kg-surface);
    display: flex;
    flex-direction: column;
    max-height: 90vh;
    width: calc(100% - 2rem);
    box-shadow: var(--kg-shadow-lg);
  }

  .kg-modal.sm { max-width: 24rem; }
  .kg-modal.md { max-width: 36rem; }
  .kg-modal.lg { max-width: 48rem; }

  .kg-modal-header {
    display: flex;
    align-items: flex-start;
    justify-content: space-between;
    gap: var(--kg-space-05);
    padding: var(--kg-space-05) var(--kg-space-06) var(--kg-space-04);
    border-bottom: 1px solid var(--kg-border);
  }

  .kg-modal-title {
    font-size: 1.25rem;
    font-weight: 400;
    line-height: 1.4;
    letter-spacing: 0;
  }

  .kg-modal-close {
    border: 0;
    background: transparent;
    font-size: 1.5rem;
    line-height: 1;
    cursor: pointer;
    color: var(--kg-text-secondary);
    padding: 0;
  }

  .kg-modal-close:hover {
    color: var(--kg-text);
  }

  .kg-modal-body {
    padding: var(--kg-space-06);
    overflow-y: auto;
  }

  .kg-modal-footer {
    display: flex;
    justify-content: flex-end;
    gap: var(--kg-space-03);
    padding: var(--kg-space-05) var(--kg-space-06);
    border-top: 1px solid var(--kg-border);
    background: var(--kg-surface-muted);
  }
</style>
