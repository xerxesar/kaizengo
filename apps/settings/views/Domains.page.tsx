import { useEffect, useMemo, useState } from 'react'
import { Alert } from '@/components/ui/alert'
import { Button } from '@/components/ui/button'
import { Label } from '@/components/ui/label'
import { Progress } from '@/components/ui/progress'
import { Textarea } from '@/components/ui/textarea'
import { KAppStatus } from '@/k'
import { Card, cn, t } from '@/lib'
import { DOMAINS_PENDING_EVENT, type DomainsPending } from '@/lib/platform-shell'
import {
  applyDomain,
  clearDomainsPending,
  domainDiff,
  fetchConflicts,
  fetchDomains,
  fetchMigratePending,
  readDomainsPending,
  rebaseDomain,
  sqlForGuided,
  suggestGuided,
  type DomainConflict,
  type DomainRow,
  type GuidedAction,
  type MigrateHint,
} from '../lib/domains'

type Tab = 'overview' | 'conflicts' | 'migrate'

export default function Domains() {
  const [rows, setRows] = useState<DomainRow[]>([])
  const [app, setApp] = useState('')
  const [tab, setTab] = useState<Tab>('overview')
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState('')
  const [message, setMessage] = useState('')

  const [conflicts, setConflicts] = useState<DomainConflict[]>([])
  const [resolutions, setResolutions] = useState<Record<string, 'ours' | 'theirs'>>({})

  const [proposedSpec, setProposedSpec] = useState('')
  const [hints, setHints] = useState<MigrateHint[]>([])
  const [guided, setGuided] = useState<Record<string, GuidedAction>>({})
  const [customSQL, setCustomSQL] = useState<Record<string, string>>({})
  const [advancedOpen, setAdvancedOpen] = useState(false)
  const [migratePre, setMigratePre] = useState('')
  const [migratePost, setMigratePost] = useState('')
  const [busy, setBusy] = useState(false)

  const selected = useMemo(() => rows.find((r) => r.app === app) ?? null, [rows, app])

  function applyHints(list: MigrateHint[]) {
    setHints(list)
    setGuided((prev) => {
      const next = { ...prev }
      for (const h of list) {
        if (!next[h.path]) next[h.path] = suggestGuided(h)
      }
      return next
    })
  }

  function applyMigrateDraft(spec: string, migrateHints?: MigrateHint[]) {
    const trimmed = spec.trim()
    if (!trimmed) return
    setTab('migrate')
    setProposedSpec(trimmed)
    if (migrateHints?.length) applyHints(migrateHints)
    void domainDiff(trimmed)
      .then((data) => {
        applyHints(data.platformDomainDiff.migrateHints ?? migrateHints ?? [])
      })
      .catch(() => {
        /* keep pending hints if analyze fails */
      })
  }

  function applyPending(pending: DomainsPending | null, list: DomainRow[]) {
    if (!pending) return
    if (pending.app) setApp(pending.app)
    else {
      const urlApp = new URLSearchParams(window.location.search).get('app')?.trim() || ''
      if (urlApp) setApp(urlApp)
    }
    if (pending.tab === 'migrate' || pending.proposedSpec || pending.migrateHints?.length) {
      if (pending.proposedSpec?.trim()) {
        applyMigrateDraft(pending.proposedSpec, pending.migrateHints)
      } else {
        setTab('migrate')
        if (pending.migrateHints?.length) applyHints(pending.migrateHints)
      }
    } else if (pending.tab === 'conflicts' || list.some((r) => r.app === (pending.app || app) && r.rebaseStatus === 'conflict')) {
      setTab('conflicts')
    }
    clearDomainsPending()
  }

  async function loadServerMigratePending(forApp: string) {
    if (!forApp) return
    try {
      const data = await fetchMigratePending(forApp)
      const pending = data.platformDomainMigratePending
      if (!pending?.proposedSpec?.trim()) return
      applyMigrateDraft(pending.proposedSpec, pending.migrateHints)
    } catch {
      /* ignore — empty migrate tab stays empty */
    }
  }

  async function loadList(preferApp?: string) {
    setLoading(true)
    setError('')
    try {
      const data = await fetchDomains()
      const list = data.platformDomains ?? []
      setRows(list)
      const pending = readDomainsPending()
      const urlApp = new URLSearchParams(window.location.search).get('app')?.trim() || ''
      const nextApp =
        preferApp ||
        pending?.app ||
        urlApp ||
        list.find((r) => r.migratePending)?.app ||
        list.find((r) => r.rebaseStatus === 'conflict')?.app ||
        list[0]?.app ||
        ''
      setApp(nextApp)
      applyPending(pending, list)
      const row = list.find((r) => r.app === nextApp)
      if (
        !pending?.proposedSpec?.trim() &&
        (pending?.tab === 'migrate' || row?.migratePending) &&
        nextApp
      ) {
        setTab('migrate')
        await loadServerMigratePending(nextApp)
      }
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e))
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => {
    void loadList()
    const onPending = (e: Event) => {
      const pending = (e as CustomEvent<DomainsPending>).detail ?? readDomainsPending()
      applyPending(pending, rows)
      if (pending?.app) setApp(pending.app)
    }
    const onLoc = () => {
      const pending = readDomainsPending()
      if (pending) applyPending(pending, rows)
    }
    window.addEventListener(DOMAINS_PENDING_EVENT, onPending)
    window.addEventListener('kaizengo:location', onLoc)
    return () => {
      window.removeEventListener(DOMAINS_PENDING_EVENT, onPending)
      window.removeEventListener('kaizengo:location', onLoc)
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [])

  useEffect(() => {
    if (!app || tab !== 'conflicts') return
    let cancelled = false
    ;(async () => {
      try {
        const data = await fetchConflicts(app)
        if (cancelled) return
        const list = data.platformDomainConflicts ?? []
        setConflicts(list)
        setResolutions((prev) => {
          const next = { ...prev }
          for (const c of list) {
            if (!next[c.path]) next[c.path] = 'ours'
          }
          return next
        })
      } catch (e) {
        if (!cancelled) setError(e instanceof Error ? e.message : String(e))
      }
    })()
    return () => {
      cancelled = true
    }
  }, [app, tab])

  useEffect(() => {
    if (!app || tab !== 'migrate') return
    if (proposedSpec.trim()) return
    let cancelled = false
    ;(async () => {
      try {
        const data = await fetchMigratePending(app)
        if (cancelled) return
        const pending = data.platformDomainMigratePending
        if (!pending?.proposedSpec?.trim()) return
        applyMigrateDraft(pending.proposedSpec, pending.migrateHints)
      } catch {
        /* ignore */
      }
    })()
    return () => {
      cancelled = true
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [app, tab])

  async function refreshHints() {
    if (!proposedSpec.trim()) {
      setHints([])
      return
    }
    setBusy(true)
    setError('')
    try {
      const data = await domainDiff(proposedSpec)
      const list = data.platformDomainDiff.migrateHints ?? []
      setHints(list)
      setGuided((prev) => {
        const next = { ...prev }
        for (const h of list) {
          if (!next[h.path]) next[h.path] = suggestGuided(h)
        }
        return next
      })
      if (!app && data.platformDomainDiff.app) setApp(data.platformDomainDiff.app)
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e))
    } finally {
      setBusy(false)
    }
  }

  function rebuildMigrateSQL(nextGuided = guided, nextCustom = customSQL, nextHints = hints) {
    const pre: string[] = []
    const post: string[] = []
    for (const h of nextHints) {
      const action = nextGuided[h.path] ?? suggestGuided(h)
      const { phase, sql } = sqlForGuided(h, action, nextCustom[h.path] ?? '')
      if (!sql) continue
      if (phase === 'pre') pre.push(sql)
      else if (phase === 'post') post.push(sql)
    }
    setMigratePre(pre.join('\n'))
    setMigratePost(post.join('\n'))
  }

  useEffect(() => {
    rebuildMigrateSQL()
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [hints, guided, customSQL])

  async function onRebase(force = false) {
    if (!app) return
    setBusy(true)
    setError('')
    setMessage('')
    try {
      const resList = Object.entries(resolutions).map(([path, side]) => ({ path, side }))
      const data = await rebaseDomain(app, { force, resolutions: force ? [] : resList })
      const res = data.platformRebaseDomain
      if (res.domain.rebaseStatus === 'conflict' && (res.domain.conflictCount ?? 0) > 0) {
        setMessage(t('settings.domains.still_conflicts'))
        await loadList(app)
        setTab('conflicts')
      } else {
        setMessage(t('settings.domains.rebased'))
        await loadList(app)
        setTab('overview')
      }
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e))
    } finally {
      setBusy(false)
    }
  }

  async function onApplyMigrate(noMigrate = false) {
    if (!proposedSpec.trim()) {
      setError(t('settings.domains.need_spec'))
      return
    }
    setBusy(true)
    setError('')
    setMessage('')
    try {
      const data = await applyDomain(proposedSpec, {
        migratePre: noMigrate ? undefined : migratePre || undefined,
        migratePost: noMigrate ? undefined : migratePost || undefined,
        noMigrate,
      })
      const res = data.platformApplyDomain
      if (res.migrateRequired) {
        setHints(res.migrateHints ?? [])
        setError(t('settings.domains.still_migrate'))
      } else {
        setMessage(t('settings.domains.applied'))
        setProposedSpec('')
        setHints([])
        await loadList(res.domain.app)
        setTab('overview')
      }
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e))
    } finally {
      setBusy(false)
    }
  }

  if (loading) {
    return (
      <div className="flex items-center justify-center py-12" role="status">
        <Progress indeterminate className="w-48" />
      </div>
    )
  }

  return (
    <div className="mx-auto flex w-full max-w-5xl flex-col gap-6 p-4 md:p-6">
      <header>
        <h1 className="text-2xl font-semibold text-[var(--kg-text)]">{t('settings.domains.title')}</h1>
        <p className="mt-1 text-sm text-[var(--kg-text-muted)]">{t('settings.domains.subtitle')}</p>
      </header>

      {error ? <Alert variant="danger">{error}</Alert> : null}
      {message ? <Alert>{message}</Alert> : null}

      <Card className="flex flex-col gap-3 p-4">
        <Label htmlFor="domain-app">{t('settings.domains.app')}</Label>
        <select
          id="domain-app"
          className="h-10 rounded border border-[var(--kg-border-strong)] bg-[var(--kg-surface)] px-3 text-sm"
          value={app}
          onChange={(e) => setApp(e.target.value)}
        >
          {rows.map((r) => (
            <option key={r.app} value={r.app}>
              {r.title || r.app}
              {r.rebaseStatus === 'conflict' ? ` (${r.conflictCount ?? 0} conflicts)` : ''}
              {r.migratePending ? ' (migrate pending)' : ''}
            </option>
          ))}
        </select>
        {selected?.error ? (
          <p className="text-sm text-[var(--kg-danger)]">{selected.error}</p>
        ) : null}

        <div className="flex flex-wrap gap-2">
          {(['overview', 'conflicts', 'migrate'] as Tab[]).map((id) => (
            <Button
              key={id}
              type="button"
              size="sm"
              variant={tab === id ? 'primary' : 'secondary'}
              onClick={() => setTab(id)}
            >
              {t(`settings.domains.tab_${id}`)}
              {id === 'conflicts' && selected?.rebaseStatus === 'conflict'
                ? ` (${selected.conflictCount ?? 0})`
                : ''}
              {id === 'migrate' && selected?.migratePending ? ' •' : ''}
            </Button>
          ))}
        </div>
      </Card>

      {tab === 'overview' && selected ? (
        <Card className="space-y-2 p-4 text-sm">
          <p>
            <span className="text-[var(--kg-text-muted)]">{t('settings.domains.kind')}: </span>
            {selected.kind}
          </p>
          <p>
            <span className="text-[var(--kg-text-muted)]">{t('settings.domains.revision')}: </span>
            {selected.revision} ({selected.source})
          </p>
          <p>
            <span className="text-[var(--kg-text-muted)]">{t('settings.domains.rebase')}: </span>
            {selected.rebaseStatus || '—'}
          </p>
          {selected.rebaseStatus === 'conflict' ? (
            <Button type="button" size="sm" onClick={() => setTab('conflicts')}>
              {t('settings.domains.resolve_conflicts')}
            </Button>
          ) : null}
          {selected.migratePending ? (
            <Button type="button" size="sm" onClick={() => setTab('migrate')}>
              {t('settings.domains.resolve_migrate')}
            </Button>
          ) : null}
        </Card>
      ) : null}

      {tab === 'conflicts' ? (
        <div className="flex flex-col gap-4">
          {!conflicts.length ? (
            <Alert>{t('settings.domains.no_conflicts')}</Alert>
          ) : (
            conflicts.map((c) => (
              <Card key={c.path} className="flex flex-col gap-3 p-4">
                <div className="font-mono text-sm font-medium">{c.path}</div>
                <div className="grid gap-3 md:grid-cols-3">
                  <Side label={t('settings.domains.ancestor')} value={c.ancestor ?? '—'} />
                  <Side
                    label={t('settings.domains.ours')}
                    value={c.ours}
                    active={resolutions[c.path] === 'ours'}
                    onPick={() => setResolutions((r) => ({ ...r, [c.path]: 'ours' }))}
                  />
                  <Side
                    label={t('settings.domains.theirs')}
                    value={c.theirs}
                    active={resolutions[c.path] === 'theirs'}
                    onPick={() => setResolutions((r) => ({ ...r, [c.path]: 'theirs' }))}
                  />
                </div>
              </Card>
            ))
          )}
          <div className="flex flex-wrap gap-2">
            <Button type="button" disabled={busy || !conflicts.length} onClick={() => void onRebase(false)}>
              {t('settings.domains.apply_resolutions')}
            </Button>
            <Button type="button" variant="secondary" disabled={busy} onClick={() => void onRebase(true)}>
              {t('settings.domains.take_all_code')}
            </Button>
          </div>
        </div>
      ) : null}

      {tab === 'migrate' ? (
        <div className="flex flex-col gap-4">
          <Card className="flex flex-col gap-2 p-4">
            <Label htmlFor="proposed-spec">{t('settings.domains.proposed_spec')}</Label>
            <Textarea
              id="proposed-spec"
              rows={12}
              className="font-mono text-xs"
              value={proposedSpec}
              onChange={(e) => setProposedSpec(e.target.value)}
              placeholder="name: myapp…"
            />
            <div className="flex flex-wrap gap-2">
              <Button type="button" size="sm" variant="secondary" disabled={busy} onClick={() => void refreshHints()}>
                {t('settings.domains.analyze')}
              </Button>
            </div>
          </Card>

          {hints.map((h) => {
            const action = guided[h.path] ?? suggestGuided(h)
            return (
              <Card key={h.path} className="flex flex-col gap-3 p-4">
                <div className="font-mono text-sm font-medium">{h.path}</div>
                <p className="text-sm text-[var(--kg-text-muted)]">{h.hint}</p>
                <div className="flex flex-wrap gap-2">
                  {(['rename', 'backfill', 'skip', 'custom'] as GuidedAction[]).map((a) => (
                    <Button
                      key={a}
                      type="button"
                      size="sm"
                      variant={action === a ? 'primary' : 'secondary'}
                      onClick={() => setGuided((g) => ({ ...g, [h.path]: a }))}
                    >
                      {t(`settings.domains.action_${a}`)}
                    </Button>
                  ))}
                </div>
                {(action === 'custom' || action === 'rename' || action === 'backfill') && (
                  <Textarea
                    rows={3}
                    className="font-mono text-xs"
                    value={customSQL[h.path] ?? ''}
                    placeholder={h.stub}
                    onChange={(e) => setCustomSQL((c) => ({ ...c, [h.path]: e.target.value }))}
                  />
                )}
              </Card>
            )
          })}

          <button
            type="button"
            className="text-left text-sm text-[var(--kg-primary)] underline"
            onClick={() => setAdvancedOpen((v) => !v)}
          >
            {advancedOpen ? t('settings.domains.hide_advanced') : t('settings.domains.show_advanced')}
          </button>
          {advancedOpen ? (
            <Card className="grid gap-3 p-4 md:grid-cols-2">
              <div>
                <Label>{t('settings.domains.migrate_pre')}</Label>
                <Textarea
                  rows={6}
                  className="mt-1 font-mono text-xs"
                  value={migratePre}
                  onChange={(e) => setMigratePre(e.target.value)}
                />
              </div>
              <div>
                <Label>{t('settings.domains.migrate_post')}</Label>
                <Textarea
                  rows={6}
                  className="mt-1 font-mono text-xs"
                  value={migratePost}
                  onChange={(e) => setMigratePost(e.target.value)}
                />
              </div>
            </Card>
          ) : null}

          <div className="flex flex-wrap gap-2">
            <Button type="button" disabled={busy || !proposedSpec.trim()} onClick={() => void onApplyMigrate(false)}>
              {t('settings.domains.apply_with_migrate')}
            </Button>
            <Button
              type="button"
              variant="secondary"
              disabled={busy || !proposedSpec.trim()}
              onClick={() => void onApplyMigrate(true)}
            >
              {t('settings.domains.apply_no_migrate')}
            </Button>
          </div>
        </div>
      ) : null}

      <KAppStatus />
    </div>
  )
}

function Side(props: { label: string; value: string; active?: boolean; onPick?: () => void }) {
  return (
    <button
      type="button"
      disabled={!props.onPick}
      onClick={props.onPick}
      className={cn(
        'rounded border p-2 text-left',
        props.active ? 'border-[var(--kg-primary)] bg-[var(--kg-surface-hover)]' : 'border-[var(--kg-border)]',
        props.onPick ? 'cursor-pointer' : 'cursor-default opacity-80',
      )}
    >
      <div className="mb-1 text-xs font-medium text-[var(--kg-text-muted)]">{props.label}</div>
      <pre className="max-h-40 overflow-auto whitespace-pre-wrap break-words font-mono text-[11px]">{props.value}</pre>
    </button>
  )
}
