import { createEffect, createMemo, createSignal, on, onMount, Show } from 'solid-js'
import {
  Alert,
  Button,
  Card,
  KAppStatus,
  Spinner,
  StatCard,
  listModelRecords,
  menuPagePath,
  navigateApp,
  t,
} from '@kaizengo/sdk-solid/ui'
import { IdentityToolbar } from '../lib/IdentityToolbar'
import { identityState, initIdentity } from '../lib/state'

export default function Overview() {
  const identity = identityState()
  const [statsLoading, setStatsLoading] = createSignal(false)

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
      identity.onStats({ users: users.length, units: units.length })
    } catch {
      /* views show their own errors */
    } finally {
      setStatsLoading(false)
    }
  }

  createEffect(
    on(
      () => (identity.ready ? identity.selectedOrg?.id : undefined),
      (orgId) => {
        if (orgId) void refreshCounts()
      },
    ),
  )

  onMount(() => {
    void initIdentity()
  })

  return (
    <Show when={!identity.loading} fallback={<Spinner />}>
      <IdentityToolbar />

      <Show
        when={!identity.error}
        fallback={
          <Alert variant="danger" dismissible onDismiss={() => (identity.error = '')}>
            {identity.error}
          </Alert>
        }
      >
        <Show when={identity.selectedOrg} fallback={<Alert variant="warning">{t('identity.no_org')}</Alert>}>
          <div class="flex flex-col gap-5">
            <div class="grid grid-cols-[repeat(auto-fit,minmax(14rem,1fr))] gap-4">
              <StatCard
                label={t('identity.stat.org')}
                value={identity.selectedOrg!.name}
                hint={identity.selectedOrg!.slug}
                icon="🏢"
              />
              <StatCard
                label={t('identity.stat.users')}
                value={statsLoading() ? '…' : identity.userCount}
                hint={t('identity.stat.users_hint')}
                icon="👥"
              />
              <StatCard
                label={t('identity.stat.units')}
                value={statsLoading() ? '…' : identity.unitCount}
                hint={t('identity.stat.units_hint')}
                icon="⎔"
              />
            </div>

            <div class="grid grid-cols-[repeat(auto-fit,minmax(18rem,1fr))] gap-4">
              <Card title={t('identity.overview.actions')}>
                <div class="flex flex-wrap gap-2">
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
                <p class="text-sm leading-relaxed text-[var(--kg-text-secondary)]">
                  {t('identity.overview.about')}
                </p>
              </Card>
            </div>
          </div>
        </Show>
      </Show>

      <KAppStatus />
    </Show>
  )
}
