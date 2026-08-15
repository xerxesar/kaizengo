<script lang="ts">
  import { onMount } from 'svelte'
  import {
    Select,
    fetchModelViews,
    listModelRecords,
    listViewForModel,
    parseNamespace,
    type ModelRecord,
  } from '@kaizengo/sdk-svelte/ui'

  type Props = {
    relation: string
    fromApp: string
    value?: string
    placeholder?: string
    onchange?: (id: string) => void
  }

  let { relation, fromApp, value = '', placeholder = '', onchange }: Props = $props()

  let options = $state<{ value: string; label: string }[]>([])
  let loading = $state(true)

  function resolve(): { app: string; model: string } {
    const rel = relation.trim()
    if (rel.includes('.')) {
      const ns = parseNamespace(rel)
      return { app: ns.app, model: ns.name }
    }
    return { app: fromApp, model: rel }
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
      options = rows.map((row) => ({ value: String(row.id), label: labelOf(row, prefer) }))
    } finally {
      loading = false
    }
  })
</script>

<Select
  {value}
  {options}
  placeholder={loading ? '…' : placeholder}
  disabled={loading}
  onchange={(e) => onchange?.((e.currentTarget as HTMLSelectElement).value)}
/>
