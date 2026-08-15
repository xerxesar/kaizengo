<script lang="ts">
  import { onMount } from 'svelte'
  import {
    Alert,
    Badge,
    Button,
    Card,
    FormActions,
    KAppStatus,
    StatCard,
    Table,
    t,
    type Column,
  } from '@kaizengo/sdk-svelte/ui'
  import {
    fetchSearchConfig,
    reindexSearchModel,
    updateSearchModelConfig,
    type SearchConfig,
    type SearchModelConfig,
  } from '../lib/graphql'

  let config = $state<SearchConfig | null>(null)
  let loading = $state(true)
  let saving = $state(false)
  let reindexing = $state<string | null>(null)
  let error = $state('')
  let saved = $state(false)
  let reindexed = $state('')
  let draft = $state<Record<string, { enabled: boolean; fields: Set<string> }>>({})

  const columns: Column<SearchModelConfig>[] = [
    { key: 'app', label: 'App', render: (r) => r.app },
    { key: 'model', label: 'Model', render: (r) => r.model },
    { key: 'collection', label: 'Collection', mono: true, render: (r) => r.collection },
    {
      key: 'enabled',
      label: 'Search',
      render: (r) => (draftKey(r) in draft ? (draft[draftKey(r)].enabled ? 'on' : 'off') : r.enabled ? 'on' : 'off'),
    },
    {
      key: 'documentCount',
      label: 'Indexed',
      render: (r) => String(r.documentCount),
    },
    { key: 'source', label: 'Source', render: (r) => r.source },
  ]

  function draftKey(row: SearchModelConfig) {
    return `${row.app}.${row.model}`
  }

  function isFieldSelected(row: SearchModelConfig, fieldName: string) {
    const key = draftKey(row)
    if (key in draft) {
      return draft[key].fields.has(fieldName)
    }
    const f = row.fields.find((x) => x.name === fieldName)
    return f?.selected ?? false
  }

  function isEnabled(row: SearchModelConfig) {
    const key = draftKey(row)
    if (key in draft) return draft[key].enabled
    return row.enabled
  }

  function toggleEnabled(row: SearchModelConfig) {
    const key = draftKey(row)
    ensureDraft(row)
    draft[key].enabled = !draft[key].enabled
    draft = { ...draft }
  }

  function toggleField(row: SearchModelConfig, fieldName: string) {
    const key = draftKey(row)
    ensureDraft(row)
    if (draft[key].fields.has(fieldName)) {
      draft[key].fields.delete(fieldName)
    } else {
      draft[key].fields.add(fieldName)
    }
    draft = { ...draft }
  }

  function ensureDraft(row: SearchModelConfig) {
    const key = draftKey(row)
    if (key in draft) return
    const fields = new Set(row.fields.filter((f) => f.selected).map((f) => f.name))
    draft[key] = { enabled: row.enabled, fields }
  }

  export async function load() {
    loading = true
    error = ''
    saved = false
    reindexed = ''
    try {
      const data = await fetchSearchConfig()
      config = data.searchConfig
      draft = {}
    } catch (e) {
      error = e instanceof Error ? e.message : String(e)
      config = null
    } finally {
      loading = false
    }
  }

  async function saveRow(row: SearchModelConfig) {
    const key = draftKey(row)
    ensureDraft(row)
    const d = draft[key]
    saving = true
    error = ''
    saved = false
    reindexed = ''
    try {
      const data = await updateSearchModelConfig({
        app: row.app,
        model: row.model,
        enabled: d.enabled,
        fields: [...d.fields],
      })
      config = data.updateSearchModelConfig
      delete draft[key]
      draft = { ...draft }
      saved = true
    } catch (e) {
      error = e instanceof Error ? e.message : String(e)
    } finally {
      saving = false
    }
  }

  async function reindex(row: SearchModelConfig, field?: string) {
    const key = field ? `${draftKey(row)}.${field}` : draftKey(row)
    reindexing = key
    error = ''
    saved = false
    reindexed = ''
    try {
      const data = await reindexSearchModel({
        app: row.app,
        model: row.model,
        field: field || undefined,
      })
      config = data.reindexSearchModel.searchConfig
      reindexed = field
        ? t('typesense.search.reindexed_field', field, data.reindexSearchModel.indexed)
        : t('typesense.search.reindexed', data.reindexSearchModel.indexed)
    } catch (e) {
      error = e instanceof Error ? e.message : String(e)
    } finally {
      reindexing = null
    }
  }

  function isReindexing(row: SearchModelConfig, field?: string) {
    const key = field ? `${draftKey(row)}.${field}` : draftKey(row)
    return reindexing === key
  }

  const models = $derived(config?.models ?? [])

  onMount(async () => {
    await load()
  })
</script>

{#if error}
  <Alert variant="danger" dismissible ondismiss={() => (error = '')}>{error}</Alert>
{:else if saved}
  <Alert variant="success">{t('typesense.search.saved')}</Alert>
{:else if reindexed}
  <Alert variant="success">{reindexed}</Alert>
{/if}

{#if loading}
  <p class="muted">{t('typesense.search.loading')}</p>
{:else if !config}
  <Alert variant="warning">{t('typesense.search.unavailable')}</Alert>
{:else}
  <div class="stats">
    <StatCard
      label={t('typesense.search.backend')}
      value={config.backend}
      hint={config.connected ? t('typesense.search.connected') : t('typesense.search.memory_hint')}
    />
    <StatCard
      label={t('typesense.search.collections')}
      value={models.filter((m) => m.enabled).length}
      hint={t('typesense.search.collections_hint')}
    />
    <StatCard
      label={t('typesense.search.documents')}
      value={models.reduce((n, m) => n + m.documentCount, 0)}
      hint={t('typesense.search.documents_hint')}
    />
  </div>

  <Card title={t('typesense.search.models_title')}>
    {#if models.length === 0}
      <p class="muted">{t('typesense.search.no_models')}</p>
    {:else}
      <Table {columns} rows={models}>
        {#snippet actions(row)}
          <div class="row-actions">
            <Badge variant={isEnabled(row) ? 'success' : 'muted'}>
              {isEnabled(row) ? t('typesense.search.enabled') : t('typesense.search.disabled')}
            </Badge>
            <Button size="sm" variant="ghost" onclick={() => toggleEnabled(row)}>
              {isEnabled(row) ? t('typesense.search.disable') : t('typesense.search.enable')}
            </Button>
            <Button size="sm" loading={saving} onclick={() => void saveRow(row)}>
              {t('typesense.search.save')}
            </Button>
            <Button
              size="sm"
              variant="ghost"
              loading={isReindexing(row)}
              disabled={!isEnabled(row)}
              onclick={() => void reindex(row)}
            >
              {t('typesense.search.reindex')}
            </Button>
          </div>
        {/snippet}
      </Table>

      <div class="field-panels">
        {#each models as row (draftKey(row))}
          <section class="field-panel">
            <h3>{row.app}.{row.model}</h3>
            <p class="muted">{t('typesense.search.fields_hint')}</p>
            <div class="field-grid">
              {#each row.fields as field (field.name)}
                <label class="field-check">
                  <input
                    type="checkbox"
                    checked={isFieldSelected(row, field.name)}
                    onchange={() => toggleField(row, field.name)}
                  />
                  <span>{field.name}</span>
                  <Badge variant="muted">{field.type}</Badge>
                  <Button
                    size="sm"
                    variant="ghost"
                    loading={isReindexing(row, field.name)}
                    onclick={() => void reindex(row, field.name)}
                  >
                    {t('typesense.search.reindex_field')}
                  </Button>
                </label>
              {/each}
            </div>
            <FormActions>
              <Button size="sm" loading={saving} onclick={() => void saveRow(row)}>
                {t('typesense.search.save')}
              </Button>
            </FormActions>
          </section>
        {/each}
      </div>
    {/if}
  </Card>
{/if}

<KAppStatus />

<style>
  .stats {
    display: grid;
    grid-template-columns: repeat(auto-fit, minmax(12rem, 1fr));
    gap: var(--kg-space-05);
    margin-bottom: var(--kg-space-06);
  }
  .row-actions {
    display: flex;
    flex-wrap: wrap;
    gap: var(--kg-space-03);
    align-items: center;
  }
  .field-panels {
    display: flex;
    flex-direction: column;
    gap: var(--kg-space-06);
    margin-top: var(--kg-space-06);
  }
  .field-panel h3 {
    margin: 0 0 var(--kg-space-03);
    font-size: 1rem;
  }
  .field-grid {
    display: grid;
    grid-template-columns: repeat(auto-fill, minmax(12rem, 1fr));
    gap: var(--kg-space-03);
    margin: var(--kg-space-04) 0;
  }
  .field-check {
    display: flex;
    flex-wrap: wrap;
    align-items: center;
    gap: var(--kg-space-03);
    font-size: 0.875rem;
  }
  .muted {
    color: var(--kg-text-muted);
  }
</style>
