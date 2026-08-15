<script lang="ts">
  import type { BadgeVariant } from './types'

  type Props = {
    variant?: BadgeVariant
    children: import('svelte').Snippet
  }

  let { variant = 'default', children }: Props = $props()
</script>

<span class="kg-badge {variant}">{@render children()}</span>

<style>
  .kg-badge {
    display: inline-flex;
    align-items: center;
    min-height: 1.5rem;
    padding: 0 var(--kg-space-04);
    border-radius: 1rem;
    font-size: 0.75rem;
    font-weight: 400;
    letter-spacing: 0.16px;
    white-space: nowrap;
  }

  .kg-badge.default {
    background: var(--kg-surface-muted);
    color: var(--kg-text);
  }

  .kg-badge.success {
    background: var(--kg-success-bg);
    color: var(--kg-success);
  }

  .kg-badge.warning {
    background: var(--kg-warning-bg);
    color: var(--kg-warning);
  }

  .kg-badge.danger {
    background: var(--kg-danger-bg);
    color: var(--kg-danger);
  }

  .kg-badge.info {
    background: var(--kg-info-bg);
    color: var(--kg-info);
  }

  .kg-badge.muted {
    background: var(--kg-surface-muted);
    color: var(--kg-text-muted);
  }
</style>
