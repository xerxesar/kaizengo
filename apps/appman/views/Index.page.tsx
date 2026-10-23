import { useState } from 'react'
import { Alert } from '@/components/ui/alert'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { t, type Column, badgeVariants } from '@/lib'
import type { ModelRecord } from '@/lib/model-client'
import { KAppStatus, KCollection } from '@/k'
import { installApp, upgradeApp } from '../lib/graphql'

type AppLane = 'available' | 'installed' | 'upgrade' | 'system'

function laneOf(row: ModelRecord): AppLane {
  const status = String(row.status ?? '')
  if (status === 'system' || status === 'upgrade' || status === 'installed' || status === 'available') {
    return status
  }
  if (row.autoInstall) return 'system'
  if (row.upgrade) return 'upgrade'
  if (row.installed) return 'installed'
  return 'available'
}

function laneLabel(lane: string): string {
  switch (lane) {
    case 'system':
      return t('appman.system')
    case 'upgrade':
      return t('appman.upgrade')
    case 'installed':
      return t('appman.installed')
    case 'available':
      return t('appman.available')
    default:
      return lane
  }
}

function laneBadgeVariant(lane: string): keyof typeof badgeVariants {
  switch (lane) {
    case 'system':
      return 'muted'
    case 'upgrade':
      return 'warning'
    case 'installed':
      return 'success'
    default:
      return 'default'
  }
}

const tableColumns: Column<ModelRecord>[] = [
  {
    key: 'title',
    label: 'Title',
    cell: (app) => (
      <div className="flex min-w-0 flex-col gap-0.5">
        <span className="font-medium text-[var(--kg-text)]">{String(app.title ?? '')}</span>
        <span className="font-mono text-xs text-[var(--kg-text-muted)]">{String(app.name ?? '')}</span>
      </div>
    ),
  },
  {
    key: 'summary',
    label: 'Summary',
    cell: (app) => (
      <span className="text-[var(--kg-text-secondary)]">{String(app.summary ?? '')}</span>
    ),
  },
  {
    key: 'version',
    label: 'Version',
    mono: true,
    cell: (app) => {
      const version = String(app.version ?? '')
      const installed = app.installedVersion != null ? String(app.installedVersion) : ''
      return (
        <span>
          {version}
          {installed && installed !== version ? ` · ${installed}` : ''}
        </span>
      )
    },
  },
  {
    key: 'status',
    label: 'Status',
    cell: (app) => {
      const lane = laneOf(app)
      return <Badge variant={laneBadgeVariant(lane)}>{laneLabel(lane)}</Badge>
    },
  },
]

export default function Index() {
  const [error, setError] = useState('')
  const [busy, setBusy] = useState('')
  const [refreshToken, setRefreshToken] = useState(0)

  async function install(name: string) {
    setBusy(name)
    setError('')
    try {
      await installApp(name)
      window.dispatchEvent(new CustomEvent('kaizengo:apps'))
      setRefreshToken((n) => n + 1)
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e))
    } finally {
      setBusy('')
    }
  }

  async function upgrade(name: string) {
    setBusy(name)
    setError('')
    try {
      await upgradeApp(name)
      window.dispatchEvent(new CustomEvent('kaizengo:apps'))
      setRefreshToken((n) => n + 1)
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e))
    } finally {
      setBusy('')
    }
  }

  function appActions(app: ModelRecord) {
    const name = String(app.name ?? app.id ?? '')
    if (app.autoInstall) return null
    if (!app.installed) {
      return (
        <Button
          size="sm"
          disabled={busy !== '' && busy !== name}
          onClick={() => void install(name)}
        >
          {busy === name ? '…' : t('appman.install')}
        </Button>
      )
    }
    if (app.upgrade) {
      return (
        <Button
          size="sm"
          disabled={busy !== '' && busy !== name}
          onClick={() => void upgrade(name)}
        >
          {busy === name ? '…' : t('appman.upgrade')}
        </Button>
      )
    }
    return null
  }

  return (
    <>
      <p className="mb-4 text-sm leading-relaxed text-[var(--kg-text-secondary)]">
        {t('appman.subtitle')}
      </p>

      {error ? (
        <Alert variant="danger" className="mb-4">
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

      <KCollection
        query="appman.apps"
        paginated
        searchable
        groupByField="status"
        views={['table', 'kanban', 'chart', 'pivot']}
        defaultView="kanban"
        columns={tableColumns}
        keyOf={(app) => String(app.name ?? app.id)}
        actions={(app) => appActions(app)}
        emptyMessage={t('appman.empty')}
        refreshToken={refreshToken}
        onerror={setError}
        card={(app) => {
          const lane = laneOf(app)
          const depends = String(app.depends ?? '')
          const installedVersion =
            app.installedVersion != null ? String(app.installedVersion) : ''
          return (
            <div className="flex min-h-0 flex-col gap-2 overflow-hidden">
              <div className="flex items-start justify-between gap-2">
                <div className="min-w-0">
                  <h4 className="truncate text-sm font-semibold text-[var(--kg-text)]">
                    {String(app.title ?? '')}
                  </h4>
                  <p className="truncate font-mono text-xs text-[var(--kg-text-muted)]">
                    {String(app.name ?? '')}
                  </p>
                </div>
                <Badge variant={laneBadgeVariant(lane)} className="shrink-0">
                  {laneLabel(lane)}
                </Badge>
              </div>
              <p
                className="line-clamp-2 text-sm leading-relaxed text-[var(--kg-text-secondary)]"
                title={String(app.summary ?? '')}
              >
                {String(app.summary ?? '')}
              </p>
              <p className="m-0 truncate text-xs text-[var(--kg-text-muted)]">
                {t('appman.version', String(app.version ?? ''))}
                {installedVersion && installedVersion !== String(app.version ?? '')
                  ? ` · ${t('appman.installed_version', installedVersion)}`
                  : ''}
              </p>
              {depends ? (
                <p className="m-0 truncate text-xs text-[var(--kg-text-muted)]">
                  {t('appman.depends', depends)}
                </p>
              ) : null}
              {appActions(app) ? <div className="mt-1 shrink-0">{appActions(app)}</div> : null}
            </div>
          )
        }}
      />

      <KAppStatus />
    </>
  )
}
