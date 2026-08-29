import { createEffect, createSignal, onMount, Show, type JSX } from 'solid-js'
import { Alert } from './Alert'
import { Button } from './Button'
import { EmptyState } from './EmptyState'
import { Spinner } from './Spinner'
import { Table } from './Table'
import { t } from './i18n-context'
import {
  deleteModelRecord,
  fetchModelViews,
  fetchViewSlots,
  listModelRecords,
  listViewForModel,
  parseNamespace,
  type ModelRecord,
} from './model-client'
import type { Column } from './types'
import { cn } from './cn'

type Props = {
  model: string
  emptyMessage?: string
  deletable?: boolean
  onerror?: (message: string) => void
  class?: string
  refreshToken?: number
}

export function KTable(props: Props): JSX.Element {
  const [loading, setLoading] = createSignal(true)
  const [error, setError] = createSignal('')
  const [rows, setRows] = createSignal<ModelRecord[]>([])
  const [columns, setColumns] = createSignal<Column<ModelRecord>[]>([])

  const ns = () => parseNamespace(props.model)

  function formatCell(key: string, value: unknown): string {
    if (value == null) return ''
    if (key === 'updatedAt' || key === 'createdAt') {
      try {
        return new Date(String(value)).toLocaleString()
      } catch {
        return String(value)
      }
    }
    return String(value)
  }

  function buildColumns(keys: { key: string; label: string; width?: string; align?: string }[]) {
    return keys.map((col) => ({
      key: col.key,
      label: col.label,
      width: col.width,
      align: (col.align as 'left' | 'center' | 'right' | undefined) ?? 'left',
      render: (row: ModelRecord) => formatCell(col.key, row[col.key]),
    }))
  }

  function reportError(message: string) {
    setError(message)
    props.onerror?.(message)
  }

  async function refresh() {
    setError('')
    try {
      const { app, name } = ns()
      const views = await fetchModelViews(app)
      const view = listViewForModel(views, name)
      if (!view?.columns?.length) {
        throw new Error(`no list view found for model ${props.model}`)
      }

      setColumns(buildColumns(view.columns))
      const fieldKeys = view.columns.map((c) => c.key)
      setRows(await listModelRecords(app, name, fieldKeys))
      await fetchViewSlots(app, view.name)
    } catch (e) {
      reportError(e instanceof Error ? e.message : String(e))
    } finally {
      setLoading(false)
    }
  }

  async function remove(id: string) {
    setError('')
    try {
      const { app, name } = ns()
      await deleteModelRecord(app, name, id)
      setRows((prev) => prev.filter((row) => String(row.id) !== id))
    } catch (e) {
      reportError(e instanceof Error ? e.message : String(e))
    }
  }

  onMount(() => {
    void refresh()
  })

  createEffect(() => {
    if (props.refreshToken != null && props.refreshToken > 0) {
      void refresh()
    }
  })

  const deletable = () => props.deletable ?? true

  return (
    <div class={cn('flex flex-col gap-5', props.class)}>
      <Show when={error()}>
        <Alert variant="danger">{error()}</Alert>
      </Show>

      <Show when={!loading()} fallback={<Spinner />}>
        <Show when={rows().length > 0} fallback={<EmptyState title={props.emptyMessage ?? 'No records found'} />}>
          <Table
            columns={columns()}
            rows={rows()}
            actions={
              deletable()
                ? (row) => (
                    <Button variant="ghost" size="sm" onClick={() => void remove(String(row.id))}>
                      Delete
                    </Button>
                  )
                : undefined
            }
          />
        </Show>
      </Show>
    </div>
  )
}
