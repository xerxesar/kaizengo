<script lang="ts" generics="T extends Record<string, unknown>">
  import type { Column } from './types'
  import type { Snippet } from 'svelte'

  type Props = {
    columns: Column<T>[]
    rows: T[]
    keyField?: string
    loading?: boolean
    emptyMessage?: string
    onclick?: (row: T) => void
    actions?: Snippet<[T]>
  }

  let {
    columns,
    rows,
    keyField = 'id',
    loading = false,
    emptyMessage = 'No records found',
    onclick,
    actions,
  }: Props = $props()

  function cellValue(row: T, col: Column<T>): string {
    if (col.render) return col.render(row)
    const val = row[col.key]
    return val == null ? '' : String(val)
  }

  function rowKey(row: T, i: number): string {
    const k = row[keyField]
    return k == null ? String(i) : String(k)
  }
</script>

<div class="kg-table-wrap">
  <table class="kg-table">
    <thead>
      <tr>
        {#each columns as col (col.key)}
          <th style:width={col.width} style:text-align={col.align ?? 'left'}>{col.label}</th>
        {/each}
        {#if actions}<th class="kg-table-actions-col">Actions</th>{/if}
      </tr>
    </thead>
    <tbody>
      {#if loading}
        <tr><td colspan={columns.length + (actions ? 1 : 0)} class="kg-table-empty">Loading…</td></tr>
      {:else if rows.length === 0}
        <tr><td colspan={columns.length + (actions ? 1 : 0)} class="kg-table-empty">{emptyMessage}</td></tr>
      {:else}
        {#each rows as row, i (rowKey(row, i))}
          <tr class:clickable={!!onclick} onclick={() => onclick?.(row)}>
            {#each columns as col (col.key)}
              <td style:text-align={col.align ?? 'left'} class:mono={col.mono}>{cellValue(row, col)}</td>
            {/each}
            {#if actions}
              <td class="kg-table-actions" onclick={(e) => e.stopPropagation()}>
                {@render actions(row)}
              </td>
            {/if}
          </tr>
        {/each}
      {/if}
    </tbody>
  </table>
</div>

<style>
  .kg-table-wrap {
    overflow-x: auto;
    background: var(--kg-surface);
    border-block: 1px solid var(--kg-border);
  }

  .kg-table {
    width: 100%;
    border-collapse: collapse;
    font-size: 0.875rem;
    letter-spacing: 0.16px;
  }

  .kg-table th {
    padding: var(--kg-space-04) var(--kg-space-05);
    text-align: left;
    font-size: 0.875rem;
    font-weight: 600;
    color: var(--kg-text);
    background: var(--kg-surface-muted);
    border-bottom: 1px solid var(--kg-border);
    white-space: nowrap;
  }

  .kg-table td {
    padding: var(--kg-space-04) var(--kg-space-05);
    border-bottom: 1px solid var(--kg-border);
    color: var(--kg-text);
    vertical-align: middle;
    background: var(--kg-surface);
  }

  .kg-table tbody tr:last-child td {
    border-bottom: 0;
  }

  .kg-table tbody tr.clickable {
    cursor: pointer;
  }

  .kg-table tbody tr.clickable:hover td {
    background: var(--kg-surface-hover);
  }

  .kg-table td.mono {
    font-family: var(--kg-font-mono);
    font-size: 0.8125rem;
    color: var(--kg-text-secondary);
  }

  .kg-table-empty {
    text-align: center;
    color: var(--kg-text-muted);
    padding: var(--kg-space-09) var(--kg-space-05) !important;
  }

  .kg-table-actions-col {
    width: 1%;
  }

  .kg-table-actions {
    white-space: nowrap;
  }
</style>
