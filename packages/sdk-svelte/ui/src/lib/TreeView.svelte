<script lang="ts">
  import type { TreeNode } from './types'

  type Props<T> = {
    nodes: TreeNode<T>[]
    selectedId?: string | null
    onselect?: (node: TreeNode<T>) => void
  }

  let { nodes, selectedId = null, onselect }: Props<unknown> = $props()
</script>

{#snippet renderNodes(items: TreeNode<unknown>[], depth: number)}
  <ul class="kg-tree" class:nested={depth > 0}>
    {#each items as node (node.id)}
      <li>
        <button
          type="button"
          class="kg-tree-node"
          class:selected={selectedId === node.id}
          style:padding-left="calc(var(--kg-space-05) + {depth} * var(--kg-space-05))"
          onclick={() => onselect?.(node)}
        >
          {#if node.children?.length}
            <span class="kg-tree-toggle" aria-hidden="true">▾</span>
          {:else}
            <span class="kg-tree-toggle spacer" aria-hidden="true"></span>
          {/if}
          <span class="kg-tree-label">{node.label}</span>
          {#if node.meta}<span class="kg-tree-meta">{node.meta}</span>{/if}
        </button>
        {#if node.children?.length}
          {@render renderNodes(node.children, depth + 1)}
        {/if}
      </li>
    {/each}
  </ul>
{/snippet}

<div class="kg-tree-wrap">
  {#if nodes.length === 0}
    <p class="kg-tree-empty">No items</p>
  {:else}
    {@render renderNodes(nodes, 0)}
  {/if}
</div>

<style>
  .kg-tree-wrap {
    border: 1px solid var(--kg-border);
    background: var(--kg-surface);
    overflow: hidden;
  }

  .kg-tree {
    list-style: none;
    margin: 0;
    padding: 0;
  }

  .kg-tree.nested {
    border-left: 1px solid var(--kg-border);
    margin-left: var(--kg-space-05);
  }

  .kg-tree-node {
    display: flex;
    align-items: center;
    gap: var(--kg-space-03);
    width: 100%;
    padding: var(--kg-space-04) var(--kg-space-05);
    border: 0;
    border-bottom: 1px solid var(--kg-border);
    background: transparent;
    text-align: left;
    cursor: pointer;
    font-size: 0.875rem;
    letter-spacing: 0.16px;
    color: var(--kg-text);
    transition: background 70ms cubic-bezier(0.2, 0, 0.38, 0.9);
  }

  .kg-tree-node:hover {
    background: var(--kg-surface-hover);
  }

  .kg-tree-node.selected {
    background: var(--kg-primary-subtle);
    color: var(--kg-primary);
    border-left: 3px solid var(--kg-primary);
  }

  .kg-tree-toggle {
    font-size: 0.65rem;
    color: var(--kg-text-muted);
    width: 0.75rem;
    flex-shrink: 0;
  }

  .kg-tree-toggle.spacer {
    visibility: hidden;
  }

  .kg-tree-label {
    font-weight: 400;
    flex: 1;
  }

  .kg-tree-meta {
    font-size: 0.75rem;
    color: var(--kg-text-muted);
    background: var(--kg-surface-muted);
    padding: var(--kg-space-01) var(--kg-space-03);
  }

  .kg-tree-node.selected .kg-tree-meta {
    background: rgba(15, 98, 254, 0.12);
    color: var(--kg-primary);
  }

  .kg-tree-empty {
    padding: var(--kg-space-09);
    text-align: center;
    color: var(--kg-text-muted);
    font-size: 0.875rem;
  }
</style>
