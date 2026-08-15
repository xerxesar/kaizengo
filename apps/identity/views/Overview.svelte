<script lang="ts">
  import { onMount } from 'svelte'
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
  } from '@kaizengo/sdk-svelte/ui'
  import IdentityToolbar from '../lib/IdentityToolbar.svelte'
  import { identityState, initIdentity } from '../lib/state.svelte'

  const identity = identityState()
  let statsLoading = $state(false)

  function go(page: string) {
    navigateApp(menuPagePath('identity', page))
  }

  async function refreshCounts() {
    if (!identity.selectedOrg) return
    statsLoading = true
    try {
      const [users, units] = await Promise.all([
        listModelRecords('identity', 'user', ['id']),
        listModelRecords('identity', 'org_unit', ['id']),
      ])
      identity.onStats({ users: users.length, units: units.length })
    } catch {
      /* views show their own errors */
    } finally {
      statsLoading = false
    }
  }

  $effect(() => {
    if (identity.ready && identity.selectedOrg) {
      void refreshCounts()
    }
  })

  onMount(() => {
    void initIdentity()
  })
</script>

{#if identity.loading}
  <Spinner />
{:else}
  <IdentityToolbar />

  {#if identity.error}
    <Alert variant="danger" dismissible ondismiss={() => (identity.error = '')}>{identity.error}</Alert>
  {:else if !identity.selectedOrg}
    <Alert variant="warning">{t('identity.no_org')}</Alert>
  {:else}
    <div class="overview">
      <div class="stats">
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

      <div class="cards">
        <Card title={t('identity.overview.actions')}>
          <div class="actions">
            <Button onclick={() => go('users')}>{t('identity.overview.manage_users')}</Button>
            <Button variant="secondary" onclick={() => go('structure')}>{t('identity.overview.edit_structure')}</Button>
            <Button variant="ghost" onclick={() => go('settings')}>{t('identity.overview.org_settings')}</Button>
          </div>
        </Card>

        <Card title={t('identity.overview.about_title')}>
          <p class="about">{t('identity.overview.about')}</p>
        </Card>
      </div>
    </div>
  {/if}
{/if}

<KAppStatus />

<style>
  .overview {
    display: flex;
    flex-direction: column;
    gap: 1.25rem;
  }
  .stats {
    display: grid;
    grid-template-columns: repeat(auto-fit, minmax(14rem, 1fr));
    gap: 1rem;
  }
  .cards {
    display: grid;
    grid-template-columns: repeat(auto-fit, minmax(18rem, 1fr));
    gap: 1rem;
  }
  .actions {
    display: flex;
    flex-wrap: wrap;
    gap: 0.5rem;
  }
  .about {
    font-size: 0.875rem;
    color: var(--kg-text-secondary);
    line-height: 1.6;
  }
</style>
