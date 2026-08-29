import { For, Show, type JSX } from 'solid-js'
import { cn } from './cn'
import type { Column } from './types'

type Props<T extends Record<string, unknown>> = {
  columns: Column<T>[]
  rows: T[]
  keyField?: string
  loading?: boolean
  emptyMessage?: string
  onRowClick?: (row: T) => void
  actions?: (row: T) => JSX.Element
  class?: string
}

export function Table<T extends Record<string, unknown>>(props: Props<T>): JSX.Element {
  const keyField = () => props.keyField ?? 'id'

  function cellText(row: T, col: Column<T>): string {
    if (col.render) return col.render(row)
    const val = row[col.key]
    return val == null ? '' : String(val)
  }

  function rowKey(row: T, i: number): string {
    const k = row[keyField()]
    return k == null ? String(i) : String(k)
  }

  return (
    <div class={cn('overflow-x-auto border-y border-[var(--kg-border)] bg-[var(--kg-surface)]', props.class)}>
      <table class="kg-table w-full border-collapse text-sm">
        <thead>
          <tr class="border-b border-[var(--kg-border)] bg-[var(--kg-surface-muted,var(--kg-field-hover))]">
            <For each={props.columns}>
              {(col) => (
                <th
                  class="px-4 py-3 text-left font-semibold text-[var(--kg-text)]"
                  style={{ width: col.width, 'text-align': col.align ?? 'left' }}
                >
                  {col.label}
                </th>
              )}
            </For>
            <Show when={props.actions}>
              <th class="kg-table-actions-col px-4 py-3 text-left">Actions</th>
            </Show>
          </tr>
        </thead>
        <tbody>
          <Show
            when={!props.loading && props.rows.length > 0}
            fallback={
              <tr>
                <td
                  class="px-4 py-6 text-center text-[var(--kg-text-muted)]"
                  colspan={props.columns.length + (props.actions ? 1 : 0)}
                >
                  {props.loading ? 'Loading…' : props.emptyMessage ?? 'No records found'}
                </td>
              </tr>
            }
          >
            <For each={props.rows}>
              {(row, i) => (
                <tr
                  class={cn(
                    'border-b border-[var(--kg-border)]',
                    props.onRowClick && 'cursor-pointer hover:bg-[var(--kg-surface-hover)]',
                  )}
                  onClick={() => props.onRowClick?.(row)}
                >
                  <For each={props.columns}>
                    {(col) => (
                      <td
                        class={cn('px-4 py-3', col.mono && 'font-mono text-xs')}
                        style={{ 'text-align': col.align ?? 'left' }}
                      >
                        <Show when={col.cell} fallback={cellText(row, col)}>
                          {col.cell!(row)}
                        </Show>
                      </td>
                    )}
                  </For>
                  <Show when={props.actions}>
                    <td
                      class="kg-table-actions px-4 py-3"
                      onClick={(e) => e.stopPropagation()}
                    >
                      {props.actions!(row)}
                    </td>
                  </Show>
                </tr>
              )}
            </For>
          </Show>
        </tbody>
      </table>
    </div>
  )
}
