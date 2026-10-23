import { useEffect, useState } from 'react'
import { Alert } from '@/components/ui/alert'
import { Button } from '@/components/ui/button'
import { Progress } from '@/components/ui/progress'
import { listModelRecords, menuPagePath, navigateApp, t, Card, StatCard } from '@/lib'
import { KAppStatus } from '@/k'
import { IdentityToolbar } from '../lib/IdentityToolbar'
import { identityState, initIdentity, useIdentityState } from '../lib/state'

export default function Overview() {
  const identity = useIdentityState()
  const [statsLoading, setStatsLoading] = useState(false)

  function go(page: string) {
    navigateApp(menuPagePath('identity', page))
  }

  async function refreshCounts() {
    if (!identity.selectedOrg) return
    setStatsLoading(true)
    try {
      const [users, units] = await Promise.all([
        listModelRecords('identity', 'user', ['id']),
        listModelRecords('identity', 'org_unit', ['id']),
      ])
      identityState().onStats({ users: users.length, units: units.length })
    } catch {
      /* views show their own errors */
    } finally {
      setStatsLoading(false)
    }
  }

  useEffect(() => {
    void initIdentity()
  }, [])

  useEffect(() => {
    if (identity.ready && identity.selectedOrg?.id) {
      void refreshCounts()
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [identity.ready, identity.selectedOrg?.id])

  if (identity.loading) {
    return (
      <div className="flex items-center justify-center py-8" role="status" aria-label="Loading">
        <Progress indeterminate className="w-48 max-w-full" />
      </div>
    )
  }

  return (
    <>
      <IdentityToolbar />

      {identity.error ? (
        <Alert variant="danger">
          <div className="min-w-0 flex-1">{identity.error}</div>
          <button
            type="button"
            className="shrink-0 text-current opacity-70 hover:opacity-100"
            aria-label="Dismiss"
            onClick={() => {
              identityState().error = ''
            }}
          >
            ×
          </button>
        </Alert>
      ) : !identity.selectedOrg ? (
        <Alert variant="warning">
          <div className="min-w-0 flex-1">{t('identity.no_org')}</div>
        </Alert>
      ) : (
        <div className="flex flex-col gap-5">
          <div className="grid grid-cols-[repeat(auto-fit,minmax(14rem,1fr))] gap-4">
            <StatCard
              label={t('identity.stat.org')}
              value={identity.selectedOrg.name}
              hint={identity.selectedOrg.slug}
              icon="🏢"
            />
            <StatCard
              label={t('identity.stat.users')}
              value={statsLoading ? '…' : identity.userCount}
              hint={t('identity.stat.users_hint')}
              icon="👥"
            />
            <StatCard
              label={t('identity.stat.units')}
              value={statsLoading ? '…' : identity.unitCount}
              hint={t('identity.stat.units_hint')}
              icon="⎔"
            />
          </div>

          <div className="grid grid-cols-[repeat(auto-fit,minmax(18rem,1fr))] gap-4">
            <Card title={t('identity.overview.actions')}>
              <div className="flex flex-wrap gap-2">
                <Button onClick={() => go('users')}>{t('identity.overview.manage_users')}</Button>
                <Button variant="secondary" onClick={() => go('structure')}>
                  {t('identity.overview.edit_structure')}
                </Button>
                <Button variant="ghost" onClick={() => go('settings')}>
                  {t('identity.overview.org_settings')}
                </Button>
              </div>
            </Card>

            <Card title={t('identity.overview.about_title')}>
              <p className="text-sm leading-relaxed text-[var(--kg-text-secondary)]">
                {t('identity.overview.about')}
              </p>
            </Card>
          </div>
        </div>
      )}

      <KAppStatus />
    </>
  )
}
