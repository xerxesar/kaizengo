<script lang="ts">
  import type { AlertVariant } from './types'

  type Props = {
    variant?: AlertVariant
    title?: string
    dismissible?: boolean
    ondismiss?: () => void
    children: import('svelte').Snippet
  }

  let {
    variant = 'info',
    title,
    dismissible = false,
    ondismiss,
    children,
  }: Props = $props()
</script>

<div class="kg-alert {variant}" role="alert">
  <div class="kg-alert-accent"></div>
  <div class="kg-alert-body">
    {#if title}<strong class="kg-alert-title">{title}</strong>{/if}
    <div class="kg-alert-content">{@render children()}</div>
  </div>
  {#if dismissible}
    <button type="button" class="kg-alert-dismiss" aria-label="Dismiss" onclick={ondismiss}>×</button>
  {/if}
</div>

<style>
  .kg-alert {
    position: relative;
    display: flex;
    align-items: flex-start;
    gap: var(--kg-space-05);
    padding: var(--kg-space-05) var(--kg-space-06);
    border-radius: var(--kg-radius);
    font-size: 0.875rem;
    letter-spacing: 0.16px;
    line-height: 1.42857;
    color: var(--kg-text);
  }

  .kg-alert-accent {
    position: absolute;
    inset-inline-start: 0;
    top: 0;
    bottom: 0;
    width: 3px;
  }

  .kg-alert.info {
    background: var(--kg-info-bg);
  }
  .kg-alert.info .kg-alert-accent {
    background: var(--kg-info-border);
  }

  .kg-alert.success {
    background: var(--kg-success-bg);
  }
  .kg-alert.success .kg-alert-accent {
    background: var(--kg-success-border);
  }

  .kg-alert.warning {
    background: var(--kg-warning-bg);
  }
  .kg-alert.warning .kg-alert-accent {
    background: var(--kg-warning-border);
  }

  .kg-alert.danger {
    background: var(--kg-danger-bg);
  }
  .kg-alert.danger .kg-alert-accent {
    background: var(--kg-danger-border);
  }

  .kg-alert-body {
    flex: 1;
    display: flex;
    flex-direction: column;
    gap: var(--kg-space-02);
    padding-inline-start: var(--kg-space-03);
  }

  .kg-alert-title {
    display: block;
    font-weight: 600;
  }

  .kg-alert-content {
    color: var(--kg-text);
  }

  .kg-alert-dismiss {
    border: 0;
    background: transparent;
    font-size: 1.25rem;
    line-height: 1;
    cursor: pointer;
    color: var(--kg-text-secondary);
    padding: 0;
    margin-inline-start: auto;
  }

  .kg-alert-dismiss:hover {
    color: var(--kg-text);
  }

  .kg-alert-dismiss:focus-visible {
    outline: var(--kg-focus-ring);
    outline-offset: var(--kg-focus-offset);
  }
</style>
