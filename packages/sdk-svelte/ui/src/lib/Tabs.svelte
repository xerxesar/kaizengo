<script lang="ts">
  import type { TabItem } from './types'

  type Props = {
    tabs: TabItem[]
    active: string
    onchange?: (id: string) => void
  }

  let { tabs, active = $bindable(), onchange }: Props = $props()

  function select(id: string) {
    active = id
    onchange?.(id)
  }
</script>

<div class="kg-tabs-wrap">
  <div class="kg-tabs-list" role="tablist" aria-label="Views">
    {#each tabs as tab (tab.id)}
      <button
        type="button"
        role="tab"
        class="kg-tab"
        class:active={active === tab.id}
        aria-selected={active === tab.id}
        onclick={() => select(tab.id)}
      >
        {#if tab.icon}<span class="kg-tab-icon" aria-hidden="true">{tab.icon}</span>{/if}
        <span class="kg-tab-label">{tab.label}</span>
        {#if tab.badge !== undefined}
          <span class="kg-tab-badge">{tab.badge}</span>
        {/if}
      </button>
    {/each}
  </div>
</div>

<style>
  .kg-tabs-wrap {
    background: transparent;
  }

  .kg-tabs-list {
    display: flex;
    flex-wrap: wrap;
    gap: 0;
    width: 100%;
    border-bottom: 1px solid var(--kg-border);
  }

  .kg-tab {
    position: relative;
    display: inline-flex;
    align-items: center;
    gap: var(--kg-space-03);
    height: var(--kg-control-height);
    padding: 0 var(--kg-space-05);
    border: 0;
    background: transparent;
    color: var(--kg-text-secondary);
    font-family: var(--kg-font);
    font-size: 0.875rem;
    font-weight: 400;
    letter-spacing: 0.16px;
    cursor: pointer;
    white-space: nowrap;
    transition: background 70ms cubic-bezier(0.2, 0, 0.38, 0.9),
      color 70ms cubic-bezier(0.2, 0, 0.38, 0.9);
  }

  .kg-tab:hover:not(.active) {
    color: var(--kg-text);
    background: var(--kg-tab-hover);
  }

  .kg-tab.active {
    color: var(--kg-text);
    font-weight: 600;
  }

  .kg-tab.active::after {
    content: '';
    position: absolute;
    left: 0;
    right: 0;
    bottom: -1px;
    height: 2px;
    background: var(--kg-tab-indicator);
  }

  .kg-tab:focus {
    outline: none;
  }

  .kg-tab:focus-visible {
    outline: var(--kg-focus-ring);
    outline-offset: var(--kg-focus-offset);
  }

  .kg-tab-icon {
    font-size: 0.875rem;
    line-height: 1;
  }

  .kg-tab-badge {
    display: inline-flex;
    align-items: center;
    justify-content: center;
    min-width: 1.125rem;
    height: 1.125rem;
    padding: 0 var(--kg-space-02);
    background: var(--kg-surface-muted);
    color: var(--kg-text-secondary);
    font-size: 0.75rem;
    font-weight: 400;
    line-height: 1;
  }

  .kg-tab.active .kg-tab-badge {
    background: var(--kg-primary-subtle);
    color: var(--kg-primary);
  }
</style>
