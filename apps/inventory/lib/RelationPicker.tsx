import { createSignal, onMount, type JSX } from 'solid-js'
import {
  Select,
  fetchModelViews,
  listModelRecords,
  listViewForModel,
  parseNamespace,
  type ModelRecord,
} from '@kaizengo/sdk-solid/ui'

type Props = {
  relation: string
  fromApp: string
  value?: string
  placeholder?: string
  onChange?: (id: string) => void
}

export function RelationPicker(props: Props): JSX.Element {
  const [options, setOptions] = createSignal<{ value: string; label: string }[]>([])
  const [loading, setLoading] = createSignal(true)

  function resolve(): { app: string; model: string } {
    const rel = props.relation.trim()
    if (rel.includes('.')) {
      const ns = parseNamespace(rel)
      return { app: ns.app, model: ns.name }
    }
    return { app: props.fromApp, model: rel }
  }

  function labelOf(row: ModelRecord, keys: string[]): string {
    for (const key of ['name', 'sku', 'code', 'serial', 'symbol', 'identifier']) {
      if (keys.includes(key) && row[key]) return String(row[key])
    }
    return String(row.id)
  }

  onMount(async () => {
    try {
      const { app, model } = resolve()
      const views = await fetchModelViews(app)
      const list = listViewForModel(views, model)
      const keys = (list?.columns ?? []).map((c) => c.key)
      const prefer = ['name', 'sku', 'code', 'serial', 'symbol', 'identifier'].filter((k) => keys.includes(k))
      const rows = await listModelRecords(app, model, prefer.length ? prefer : ['id'])
      setOptions(rows.map((row) => ({ value: String(row.id), label: labelOf(row, prefer) })))
    } finally {
      setLoading(false)
    }
  })

  return (
    <Select
      value={props.value ?? ''}
      options={options()}
      placeholder={loading() ? '…' : props.placeholder}
      disabled={loading()}
      onChange={(value) => props.onChange?.(value)}
    />
  )
}
