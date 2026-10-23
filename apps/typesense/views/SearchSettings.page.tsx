import { useEffect, useMemo, useState } from 'react'
import { Alert } from '@/components/ui/alert'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { t, type Column, Card, StatCard, FormActions, DataTable } from '@/lib'
import { KAppStatus } from '@/k'
import {
  fetchSearchConfig,
  reindexSearchModel,
  updateSearchModelConfig,
  type SearchConfig,
  type SearchModelConfig,
} from '../lib/graphql'

type DraftEntry = { enabled: boolean; fields: Set<string> }

export default function SearchSettings() {
  const [config, setConfig] = useState<SearchConfig | null>(null)
  const [loading, setLoading] = useState(true)
  const [saving, setSaving] = useState(false)
  const [reindexing, setReindexing] = useState<string | null>(null)
  const [error, setError] = useState('')
  const [saved, setSaved] = useState(false)
  const [reindexed, setReindexed] = useState('')
  const [draft, setDraft] = useState<Record<string, DraftEntry>>({})

  function draftKey(row: SearchModelConfig) {
    return `${row.app}.${row.model}`
  }

  function ensureDraft(row: SearchModelConfig) {
    const key = draftKey(row)
    if (key in draft) return
    const fields = new Set(row.fields.filter((f) => f.selected).map((f) => f.name))
    setDraft((prev) => ({ ...prev, [key]: { enabled: row.enabled, fields } }))
  }

  function isFieldSelected(row: SearchModelConfig, fieldName: string) {
    const key = draftKey(row)
    const d = draft[key]
    if (d) return d.fields.has(fieldName)
    const f = row.fields.find((x) => x.name === fieldName)
    return f?.selected ?? false
  }

  function isEnabled(row: SearchModelConfig) {
    const key = draftKey(row)
    const d = draft[key]
    if (d) return d.enabled
    return row.enabled
  }

  function toggleEnabled(row: SearchModelConfig) {
    const key = draftKey(row)
    ensureDraft(row)
    setDraft((prev) => {
      const cur = prev[key] ?? {
        enabled: row.enabled,
        fields: new Set(row.fields.filter((f) => f.selected).map((f) => f.name)),
      }
      return { ...prev, [key]: { ...cur, enabled: !cur.enabled } }
    })
  }

  function toggleField(row: SearchModelConfig, fieldName: string) {
    const key = draftKey(row)
    ensureDraft(row)
    setDraft((prev) => {
      const cur = prev[key]!
      const fields = new Set(cur.fields)
      if (fields.has(fieldName)) fields.delete(fieldName)
      else fields.add(fieldName)
      return { ...prev, [key]: { ...cur, fields } }
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
    const d = draft[key] ?? {
      enabled: row.enabled,
      fields: new Set(row.fields.filter((f) => f.selected).map((f) => f.name)),
    }
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
    return reindexing === key
  }

  const models = config?.models ?? []

  const columns = useMemo<Column<SearchModelConfig>[]>(
    () => [
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
    ],
    // eslint-disable-next-line react-hooks/exhaustive-deps
    [draft],
  )

  useEffect(() => {
    void load()
  }, [])

  return (
    <>
      {error ? (
        <Alert variant="danger">
          <div className="min-w-0 flex-1">{error}</div>
          <button
            type="button"
            className="shrink-0 text-current opacity-70 hover:opacity-100"
            aria-label="Dismiss"
            onClick={() => setError('')}
          >
            ×
          </button>
        </Alert>
      ) : null}
      {!error && saved ? <Alert variant="success">{t('typesense.search.saved')}</Alert> : null}
      {!error && !saved && reindexed ? <Alert variant="success">{reindexed}</Alert> : null}

      {loading ? (
        <p className="text-[var(--kg-text-muted)]">{t('typesense.search.loading')}</p>
      ) : !config ? (
        <Alert variant="warning">{t('typesense.search.unavailable')}</Alert>
      ) : (
        <>
          <div className="mb-[var(--kg-space-06)] grid grid-cols-[repeat(auto-fit,minmax(12rem,1fr))] gap-[var(--kg-space-05)]">
            <StatCard
              label={t('typesense.search.backend')}
              value={config.backend}
              hint={
                config.connected
                  ? t('typesense.search.connected')
                  : t('typesense.search.memory_hint')
              }
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
            {models.length === 0 ? (
              <p className="text-[var(--kg-text-muted)]">{t('typesense.search.no_models')}</p>
            ) : (
              <>
                <DataTable
                  columns={columns}
                  rows={models}
                  actions={(row) => (
                    <div className="flex flex-row flex-wrap items-center gap-[var(--kg-space-03)]">
                      <Button size="sm" variant="ghost" onClick={() => toggleEnabled(row)}>
                        {isEnabled(row)
                          ? t('typesense.search.disable')
                          : t('typesense.search.enable')}
                      </Button>
                      <Button size="sm" disabled={saving} onClick={() => void saveRow(row)}>
                        {saving ? '…' : t('typesense.search.save')}
                      </Button>
                      <Button
                        size="sm"
                        variant="ghost"
                        disabled={!isEnabled(row) || isReindexing(row)}
                        onClick={() => void reindex(row)}
                      >
                        {isReindexing(row) ? '…' : t('typesense.search.reindex')}
                      </Button>
                    </div>
                  )}
                />

                <div className="mt-[var(--kg-space-06)] flex flex-col gap-[var(--kg-space-06)]">
                  {models.map((row) => (
                    <section key={draftKey(row)}>
                      <h3 className="mb-[var(--kg-space-03)] mt-0 text-base">
                        {row.app}.{row.model}
                      </h3>
                      <p className="text-[var(--kg-text-muted)]">{t('typesense.search.fields_hint')}</p>
                      <div className="my-[var(--kg-space-04)] grid grid-cols-[repeat(auto-fill,minmax(12rem,1fr))] gap-[var(--kg-space-03)]">
                        {row.fields.map((field) => (
                          <label
                            key={field.name}
                            className="flex flex-wrap items-center gap-[var(--kg-space-03)] text-sm"
                          >
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
                              disabled={isReindexing(row, field.name)}
                              onClick={() => void reindex(row, field.name)}
                            >
                              {isReindexing(row, field.name)
                                ? '…'
                                : t('typesense.search.reindex_field')}
                            </Button>
                          </label>
                        ))}
                      </div>
                      <FormActions>
                        <Button size="sm" disabled={saving} onClick={() => void saveRow(row)}>
                          {saving ? '…' : t('typesense.search.save')}
                        </Button>
                      </FormActions>
                    </section>
                  ))}
                </div>
              </>
            )}
          </Card>
        </>
      )}

      <KAppStatus />
    </>
  )
}
