<script lang="ts">
  import type { MenuItem } from './types'
  import { menuItemHref, navigateApp } from './menu-route'

  type Props = {
    /** App id — used to build `/app/{app}/{page}` hrefs. */
    app: string
    /** Full menu tree from `{app}Menus` (folders become dropdowns). */
    items?: MenuItem[]
    /** Active leaf menu id. */
    active?: string
    label?: string
  }

  let {
    app,
    items = [],
    active = '',
    label = 'Menu',
  }: Props = $props()

  let openId = $state<string | null>(null)

  function isLeaf(item: MenuItem) {
    return !item.children?.length
  }

  function hasActiveDescendant(item: MenuItem): boolean {
    if (item.id === active) return true
    return (item.children ?? []).some(hasActiveDescendant)
  }

  function go(item: MenuItem, e?: MouseEvent) {
    if (!isLeaf(item)) {
      e?.preventDefault()
      openId = openId === item.id ? null : item.id
      return
    }
    e?.preventDefault()
    openId = null
    navigateApp(menuItemHref(app, item))
  }

  function onDocClick(e: MouseEvent) {
    const t = e.target as HTMLElement | null
    if (!t?.closest?.('.kg-menubar')) openId = null
  }
</script>

<svelte:window onclick={onDocClick} />

<nav class="kg-menubar" aria-label={label}>
  <div class="kg-menubar-inner">
    {#each items as item (item.id)}
      {#if isLeaf(item)}
        <a
          class="kg-menubar-link"
          class:active={active === item.id}
          href={menuItemHref(app, item)}
          aria-current={active === item.id ? 'page' : undefined}
          onclick={(e) => go(item, e)}
        >
          {item.label}
        </a>
      {:else}
        <div class="kg-menubar-group">
          <button
            type="button"
            class="kg-menubar-link kg-menubar-parent"
            class:active={hasActiveDescendant(item)}
            aria-expanded={openId === item.id}
            aria-haspopup="true"
            onclick={(e) => {
              e.stopPropagation()
              go(item, e)
            }}
          >
            <span>{item.label}</span>
            <span class="kg-menubar-caret" aria-hidden="true">▾</span>
          </button>
          {#if openId === item.id}
            <div class="kg-menubar-dropdown" role="menu">
              {@render menuNodes(item.children ?? [], 0)}
            </div>
          {/if}
        </div>
      {/if}
    {/each}
  </div>
</nav>

{#snippet menuNodes(nodes: MenuItem[], depth: number)}
  {#each nodes as node (node.id)}
    {#if isLeaf(node)}
      <a
        role="menuitem"
        class="kg-menubar-dropbtn"
        class:active={active === node.id}
        style:padding-left={`calc(var(--kg-space-04) + ${depth} * var(--kg-space-04))`}
        href={menuItemHref(app, node)}
        aria-current={active === node.id ? 'page' : undefined}
        onclick={(e) => go(node, e)}
      >
        {node.label}
      </a>
    {:else}
      <div class="kg-menubar-subgroup">
        <div
          class="kg-menubar-sublabel"
          style:padding-left={`calc(var(--kg-space-04) + ${depth} * var(--kg-space-04))`}
        >
          {node.label}
        </div>
        {@render menuNodes(node.children ?? [], depth + 1)}
      </div>
    {/if}
  {/each}
{/snippet}

<style>
  .kg-menubar {
    width: 100%;
    min-width: 0;
    background: var(--kg-surface, var(--kg-bg));
    border-bottom: 1px solid var(--kg-border);
  }

  .kg-menubar-inner {
    display: flex;
    flex-wrap: nowrap;
    align-items: stretch;
    gap: 0;
    width: 100%;
    min-height: 2.5rem;
    padding: 0 var(--kg-page-padding-x);
    overflow-x: auto;
    overflow: visible;
  }

  .kg-menubar-group {
    position: relative;
    display: flex;
    align-items: stretch;
  }

  .kg-menubar-link {
    display: inline-flex;
    align-items: center;
    justify-content: center;
    gap: var(--kg-space-02);
    height: 2.5rem;
    padding: 0 var(--kg-space-05);
    border: 0;
    border-bottom: 3px solid transparent;
    background: transparent;
    color: var(--kg-text);
    font-family: var(--kg-font);
    font-size: 0.875rem;
    font-weight: 400;
    letter-spacing: 0.16px;
    line-height: 1.28571;
    text-decoration: none;
    cursor: pointer;
    white-space: nowrap;
    box-sizing: border-box;
  }

  .kg-menubar-link:hover:not(.active) {
    background: var(--kg-surface-muted, var(--kg-field-hover));
    color: var(--kg-text);
  }

  .kg-menubar-link.active,
  .kg-menubar-parent.active {
    border-bottom-color: var(--kg-primary);
    color: var(--kg-primary);
    font-weight: 600;
  }

  .kg-menubar-link:focus {
    outline: none;
  }

  .kg-menubar-link:focus-visible {
    outline: var(--kg-focus-ring);
    outline-offset: var(--kg-focus-offset);
  }

  .kg-menubar-caret {
    font-size: 0.7em;
    opacity: 0.85;
  }

  .kg-menubar-dropdown {
    position: absolute;
    top: calc(100% + 1px);
    left: 0;
    z-index: 20;
    min-width: 12rem;
    max-height: min(24rem, 70vh);
    overflow: auto;
    padding: var(--kg-space-02);
    background: var(--kg-surface);
    border: 1px solid var(--kg-border-strong);
    box-shadow: 0 4px 12px color-mix(in srgb, var(--kg-text) 12%, transparent);
  }

  .kg-menubar-dropbtn {
    display: flex;
    width: 100%;
    align-items: center;
    height: var(--kg-control-height-sm);
    padding-right: var(--kg-space-04);
    border: 0;
    background: transparent;
    color: var(--kg-text);
    font-family: var(--kg-font);
    font-size: 0.875rem;
    text-align: left;
    text-decoration: none;
    cursor: pointer;
    box-sizing: border-box;
  }

  .kg-menubar-dropbtn:hover,
  .kg-menubar-dropbtn.active {
    background: var(--kg-primary-subtle, var(--kg-surface-muted));
    color: var(--kg-primary);
    font-weight: 600;
  }

  .kg-menubar-subgroup {
    padding-block: var(--kg-space-02);
  }

  .kg-menubar-sublabel {
    padding-top: var(--kg-space-02);
    padding-bottom: var(--kg-space-02);
    padding-right: var(--kg-space-04);
    color: var(--kg-text-muted);
    font-size: 0.75rem;
    font-weight: 600;
    letter-spacing: 0.02em;
    text-transform: uppercase;
  }
</style>
