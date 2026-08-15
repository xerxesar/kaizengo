<script lang="ts">
  import { onMount } from 'svelte'
  import SearchBar from '../../../search/SearchBar.svelte'
  import type { SearchHit } from '../../../search/client'
  import Alert from './Alert.svelte'
  import Button from './Button.svelte'
  import EmptyState from './EmptyState.svelte'
  import Spinner from './Spinner.svelte'
  import Table from './Table.svelte'
  import Toolbar from './Toolbar.svelte'
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

  type Props = {
    /** Namespaced model ref (`hellospec.greeting`, `notes.note`, …). */
    model: string
    emptyMessage?: string
    deletable?: boolean
    onerror?: (message: string) => void
    class?: string
  }

  let {
    model,
    emptyMessage = 'No records found',
    deletable = true,
    onerror,
    class: className = '',
  }: Props = $props()

  let loading = $state(true)
  let error = $state('')
  let rows = $state<ModelRecord[]>([])
  let columns = $state<Column<ModelRecord>[]>([])
  let searchEnabled = $state(false)
  let searchHits = $state<SearchHit[] | null>(null)

  const ns = $derived(parseNamespace(model))
  const searchCollection = $derived([model])
  const displayedRows = $derived(
    searchHits === null
      ? rows
      : rows.filter((row) => searchHits!.some((hit) => hit.id === String(row.id))),
  )

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
    error = message
    onerror?.(message)
  }

  function onSearchResults(hits: SearchHit[] | null) {
    searchHits = hits
  }

  async function refresh() {
    error = ''
    searchHits = null
    searchEnabled = false
    try {
      const { app, name } = ns
      const views = await fetchModelViews(app)
      const view = listViewForModel(views, name)
      if (!view?.columns?.length) {
        throw new Error(`no list view found for model ${model}`)
      }

      columns = buildColumns(view.columns)
      const fieldKeys = view.columns.map((c) => c.key)
      rows = await listModelRecords(app, name, fieldKeys)

      const slots = await fetchViewSlots(app, view.name)
      searchEnabled = slots.some(
        (slot) => slot.component === 'platform.SearchBar' && (slot.slot === 'toolbar' || slot.slot === ''),
      )
    } catch (e) {
      reportError(e instanceof Error ? e.message : String(e))
    } finally {
      loading = false
    }
  }

  async function remove(id: string) {
    error = ''
    try {
      const { app, name } = ns
      await deleteModelRecord(app, name, id)
      rows = rows.filter((row) => String(row.id) !== id)
    } catch (e) {
      reportError(e instanceof Error ? e.message : String(e))
    }
  }

  onMount(() => {
    void refresh()
  })

  export { refresh }
</script>

<div class="kg-ktable {className}">
  {#if error}
    <Alert variant="danger">{error}</Alert>
  {/if}

  {#if loading}
    <Spinner />
  {:else}
    {#if searchEnabled}
      <Toolbar>
        {#snippet start()}
          <SearchBar
            collections={searchCollection}
            placeholder={t(`${ns.app}.${ns.name}.search`)}
            onResults={onSearchResults}
          />
        {/snippet}
      </Toolbar>
    {/if}

    {#if displayedRows.length === 0}
      <EmptyState title={emptyMessage} />
    {:else}
      <Table {columns} rows={displayedRows}>
        {#if deletable}
          {#snippet actions(row)}
            <Button variant="ghost" size="sm" onclick={() => void remove(String(row.id))}>
              Delete
            </Button>
          {/snippet}
        {/if}
      </Table>
    {/if}
  {/if}
</div>

<style>
  .kg-ktable {
    display: flex;
    flex-direction: column;
    gap: var(--kg-space-05);
  }
</style>
