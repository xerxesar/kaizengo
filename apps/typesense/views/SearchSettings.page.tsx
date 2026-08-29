import { createMemo, createSignal, onMount, Show, type JSX } from 'solid-js'
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
} from '@kaizengo/sdk-solid/ui'
import {
  fetchSearchConfig,
  reindexSearchModel,
  updateSearchModelConfig,
  type SearchConfig,
  type SearchModelConfig,
} from '../lib/graphql'

type DraftEntry = { enabled: boolean; fields: Set<string> }

export default function SearchSettings(): JSX.Element {
  const [config, setConfig] = createSignal<SearchConfig | null>(null)
  const [loading, setLoading] = createSignal(true)
  const [saving, setSaving] = createSignal(false)
  const [reindexing, setReindexing] = createSignal<string | null>(null)
  const [error, setError] = createSignal('')
  const [saved, setSaved] = createSignal(false)
  const [reindexed, setReindexed] = createSignal('')
  const [draft, setDraft] = createSignal<Record<string, DraftEntry>>({})

  function draftKey(row: SearchModelConfig) {
    return `${row.app}.${row.model}`
  }

  function ensureDraft(row: SearchModelConfig) {
    const key = draftKey(row)
    if (key in draft()) return
    const fields = new Set(row.fields.filter((f) => f.selected).map((f) => f.name))
    setDraft((prev) => ({ ...prev, [key]: { enabled: row.enabled, fields } }))
  }

  function isFieldSelected(row: SearchModelConfig, fieldName: string) {
    const key = draftKey(row)
    const d = draft()[key]
    if (d) return d.fields.has(fieldName)
    const f = row.fields.find((x) => x.name === fieldName)
    return f?.selected ?? false
  }

  function isEnabled(row: SearchModelConfig) {
    const key = draftKey(row)
    const d = draft()[key]
    if (d) return d.enabled
    return row.enabled
  }

  function toggleEnabled(row: SearchModelConfig) {
    const key = draftKey(row)
    ensureDraft(row)
    setDraft((prev) => {
      const next = { ...prev[key] }
      next.enabled = !next.enabled
      return { ...prev, [key]: next }
    })
  }

  function toggleField(row: SearchModelConfig, fieldName: string) {
    const key = draftKey(row)
    ensureDraft(row)
    setDraft((prev) => {
      const next = { ...prev[key], fields: new Set(prev[key].fields) }
      if (next.fields.has(fieldName)) next.fields.delete(fieldName)
      else next.fields.add(fieldName)
      return { ...prev, [key]: next }
    })
  }

  async function load() {
    setLoading(true)
    setError('')
    setSaved(false)
    setReindexed('')
    try {
      const data = await fetchSearchConfig()
      setConfig(data.searchConfig)
      setDraft({})
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e))
      setConfig(null)
    } finally {
      setLoading(false)
    }
  }

  async function saveRow(row: SearchModelConfig) {
    const key = draftKey(row)
    ensureDraft(row)
    const d = draft()[key]
    setSaving(true)
    setError('')
    setSaved(false)
    setReindexed('')
    try {
      const data = await updateSearchModelConfig({
        app: row.app,
        model: row.model,
        enabled: d.enabled,
        fields: [...d.fields],
      })
      setConfig(data.updateSearchModelConfig)
      setDraft((prev) => {
        const next = { ...prev }
        delete next[key]
        return next
      })
      setSaved(true)
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e))
    } finally {
      setSaving(false)
    }
  }

  async function reindex(row: SearchModelConfig, field?: string) {
    const key = field ? `${draftKey(row)}.${field}` : draftKey(row)
    setReindexing(key)
    setError('')
    setSaved(false)
    setReindexed('')
    try {
      const data = await reindexSearchModel({
        app: row.app,
        model: row.model,
        field: field || undefined,
      })
      setConfig(data.reindexSearchModel.searchConfig)
      setReindexed(
        field
          ? t('typesense.search.reindexed_field', field, data.reindexSearchModel.indexed)
          : t('typesense.search.reindexed', data.reindexSearchModel.indexed),
      )
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e))
    } finally {
      setReindexing(null)
    }
  }

  function isReindexing(row: SearchModelConfig, field?: string) {
    const key = field ? `${draftKey(row)}.${field}` : draftKey(row)
    return reindexing() === key
  }

  const models = createMemo(() => config()?.models ?? [])

  const columns = createMemo<Column<SearchModelConfig>[]>(() => [
    { key: 'app', label: 'App', render: (r) => r.app },
    { key: 'model', label: 'Model', render: (r) => r.model },
    { key: 'collection', label: 'Collection', mono: true, render: (r) => r.collection },
    {
      key: 'status',
      label: 'Status',
      cell: (row) => (
        <Badge variant={isEnabled(row) ? 'success' : 'muted'}>
          {isEnabled(row) ? t('typesense.search.enabled') : t('typesense.search.disabled')}
        </Badge>
      ),
    },
    { key: 'documentCount', label: 'Indexed', render: (r) => String(r.documentCount) },
    { key: 'source', label: 'Source', render: (r) => r.source },
  ])

  onMount(() => {
    void load()
  })

  return (
    <>
      <Show when={error()}>
        <Alert variant="danger" dismissible onDismiss={() => setError('')}>
          {error()}
        </Alert>
      </Show>
      <Show when={!error() && saved()}>
        <Alert variant="success">{t('typesense.search.saved')}</Alert>
      </Show>
      <Show when={!error() && !saved() && reindexed()}>
        <Alert variant="success">{reindexed()}</Alert>
      </Show>

      <Show when={!loading()} fallback={<p class="text-[var(--kg-text-muted)]">{t('typesense.search.loading')}</p>}>
        <Show when={config()} fallback={<Alert variant="warning">{t('typesense.search.unavailable')}</Alert>}>
          {(cfg) => (
            <>
              <div class="mb-[var(--kg-space-06)] grid grid-cols-[repeat(auto-fit,minmax(12rem,1fr))] gap-[var(--kg-space-05)]">
                <StatCard
                  label={t('typesense.search.backend')}
                  value={cfg().backend}
                  hint={cfg().connected ? t('typesense.search.connected') : t('typesense.search.memory_hint')}
                />
                <StatCard
                  label={t('typesense.search.collections')}
                  value={models().filter((m) => m.enabled).length}
                  hint={t('typesense.search.collections_hint')}
                />
                <StatCard
                  label={t('typesense.search.documents')}
                  value={models().reduce((n, m) => n + m.documentCount, 0)}
                  hint={t('typesense.search.documents_hint')}
                />
              </div>

              <Card title={t('typesense.search.models_title')}>
                <Show
                  when={models().length > 0}
                  fallback={<p class="text-[var(--kg-text-muted)]">{t('typesense.search.no_models')}</p>}
                >
                  <Table
                    columns={columns()}
                    rows={models()}
                    actions={(row) => (
                      <div class="flex flex-row flex-wrap items-center gap-[var(--kg-space-03)]">
                        <Button size="sm" variant="ghost" onClick={() => toggleEnabled(row)}>
                          {isEnabled(row) ? t('typesense.search.disable') : t('typesense.search.enable')}
                        </Button>
                        <Button size="sm" loading={saving()} onClick={() => void saveRow(row)}>
                          {t('typesense.search.save')}
                        </Button>
                        <Button
                          size="sm"
                          variant="ghost"
                          loading={isReindexing(row)}
                          disabled={!isEnabled(row)}
                          onClick={() => void reindex(row)}
                        >
                          {t('typesense.search.reindex')}
                        </Button>
                      </div>
                    )}
                  />

                  <div class="mt-[var(--kg-space-06)] flex flex-col gap-[var(--kg-space-06)]">
                    {models().map((row) => (
                      <section>
                        <h3 class="mb-[var(--kg-space-03)] mt-0 text-base">
                          {row.app}.{row.model}
                        </h3>
                        <p class="text-[var(--kg-text-muted)]">{t('typesense.search.fields_hint')}</p>
                        <div class="my-[var(--kg-space-04)] grid grid-cols-[repeat(auto-fill,minmax(12rem,1fr))] gap-[var(--kg-space-03)]">
                          {row.fields.map((field) => (
                            <label class="flex flex-wrap items-center gap-[var(--kg-space-03)] text-sm">
                              <input
                                type="checkbox"
                                checked={isFieldSelected(row, field.name)}
                                onChange={() => toggleField(row, field.name)}
                              />
                              <span>{field.name}</span>
                              <Badge variant="muted">{field.type}</Badge>
                              <Button
                                size="sm"
                                variant="ghost"
                                loading={isReindexing(row, field.name)}
                                onClick={() => void reindex(row, field.name)}
                              >
                                {t('typesense.search.reindex_field')}
                              </Button>
                            </label>
                          ))}
                        </div>
                        <FormActions>
                          <Button size="sm" loading={saving()} onClick={() => void saveRow(row)}>
                            {t('typesense.search.save')}
                          </Button>
                        </FormActions>
                      </section>
                    ))}
                  </div>
                </Show>
              </Card>
            </>
          )}
        </Show>
      </Show>

      <KAppStatus />
    </>
  )
}
