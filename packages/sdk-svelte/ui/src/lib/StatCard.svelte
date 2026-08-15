<script lang="ts">
  type Props = {
    label: string
    value: string | number
    hint?: string
    icon?: string
  }

  let { label, value, hint, icon }: Props = $props()
</script>

<div class="kg-stat">
  {#if icon}<span class="kg-stat-icon" aria-hidden="true">{icon}</span>{/if}
  <div class="kg-stat-content">
    <span class="kg-stat-label">{label}</span>
    <span class="kg-stat-value">{value}</span>
    {#if hint}<span class="kg-stat-hint">{hint}</span>{/if}
  </div>
</div>

<style>
  .kg-stat {
    display: flex;
    align-items: flex-start;
    gap: var(--kg-space-05);
    padding: var(--kg-space-05);
    background: var(--kg-surface);
    border: 1px solid var(--kg-border);
  }

  .kg-stat-icon {
    font-size: 1.25rem;
    line-height: 1;
    color: var(--kg-text-secondary);
  }

  .kg-stat-content {
    display: flex;
    flex-direction: column;
    gap: var(--kg-space-02);
    min-width: 0;
  }

  .kg-stat-label {
    font-size: 0.75rem;
    font-weight: 400;
    letter-spacing: 0.32px;
    color: var(--kg-text-secondary);
  }

  .kg-stat-value {
    font-size: 1.75rem;
    font-weight: 300;
    color: var(--kg-text);
    line-height: 1.28571;
    letter-spacing: 0;
  }

  .kg-stat-hint {
    font-size: 0.875rem;
    color: var(--kg-text-muted);
    letter-spacing: 0.16px;
  }
</style>
