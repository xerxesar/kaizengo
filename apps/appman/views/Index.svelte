<script lang="ts">
  import { onMount } from 'svelte'
  import {
    Alert,
    Badge,
    Button,
    Card,
    Input,
    KAppStatus,
    Spinner,
    t,
  } from '@kaizengo/sdk-svelte/ui'
  import {
    fetchApps,
    installApp,
    upgradeApp,
    type App,
  } from '../lib/graphql'

  let loading = $state(true)
  let error = $state('')
  let apps = $state<App[]>([])
  let q = $state('')
  let busy = $state('')

  const filtered = $derived(
    apps.filter((a) => {
      const hay = `${a.name} ${a.title} ${a.summary}`.toLowerCase()
      return hay.includes(q.trim().toLowerCase())
    }),
  )

  async function refresh() {
    loading = true
    error = ''
    try {
      apps = await fetchApps()
    } catch (e) {
      error = e instanceof Error ? e.message : String(e)
    } finally {
      loading = false
    }
  }

  function replace(updated: App) {
    apps = apps.map((a) => (a.name === updated.name ? updated : a))
  }

  async function install(name: string) {
    busy = name
    error = ''
    try {
      replace(await installApp(name))
      window.dispatchEvent(new CustomEvent('kaizengo:apps'))
    } catch (e) {
      error = e instanceof Error ? e.message : String(e)
    } finally {
      busy = ''
    }
  }

  async function upgrade(name: string) {
    busy = name
    error = ''
    try {
      replace(await upgradeApp(name))
      window.dispatchEvent(new CustomEvent('kaizengo:apps'))
    } catch (e) {
      error = e instanceof Error ? e.message : String(e)
    } finally {
      busy = ''
    }
  }

  onMount(() => {
    void refresh()
  })
</script>

<p class="lead">{t('appman.subtitle')}</p>

{#if error}
  <Alert variant="danger" dismissible ondismiss={() => (error = '')}>{error}</Alert>
{/if}

<div class="toolbar">
  <Input bind:value={q} placeholder={t('appman.search')} />
</div>

{#if loading}
  <Spinner />
{:else if !filtered.length}
  <p class="empty">{t('appman.empty')}</p>
{:else}
  <div class="grid">
    {#each filtered as app (app.name)}
      <Card title={app.title}>
        {#snippet actions()}
          {#if app.autoInstall}
            <Badge variant="muted">{t('appman.system')}</Badge>
          {:else if app.upgrade}
            <Badge variant="warning">{t('appman.upgrade')}</Badge>
          {:else if app.installed}
            <Badge variant="success">{t('appman.installed')}</Badge>
          {:else}
            <Badge>{t('appman.available')}</Badge>
          {/if}
        {/snippet}

        <p class="summary">{app.summary}</p>
        <p class="meta">
          {t('appman.version', app.version)}
          {#if app.installedVersion && app.installedVersion !== app.version}
            · {t('appman.installed_version', app.installedVersion)}
          {/if}
        </p>
        {#if app.depends.length}
          <p class="meta">{t('appman.depends', app.depends.join(', '))}</p>
        {/if}

        {#if !app.autoInstall}
          <div class="row">
            {#if !app.installed}
              <Button
                loading={busy === app.name}
                disabled={busy !== '' && busy !== app.name}
                onclick={() => void install(app.name)}
              >
                {t('appman.install')}
              </Button>
            {:else if app.upgrade}
              <Button
                loading={busy === app.name}
                disabled={busy !== '' && busy !== app.name}
                onclick={() => void upgrade(app.name)}
              >
                {t('appman.upgrade')}
              </Button>
            {/if}
          </div>
        {/if}
      </Card>
    {/each}
  </div>
{/if}

<KAppStatus />

<style>
  .lead {
    margin: 0 0 1rem;
    color: var(--kg-text-secondary);
    font-size: 0.875rem;
    line-height: 1.5;
  }
  .toolbar {
    max-width: 20rem;
    margin-bottom: 1rem;
  }
  .grid {
    display: grid;
    grid-template-columns: repeat(auto-fill, minmax(18rem, 1fr));
    gap: 1rem;
  }
  .summary {
    margin: 0 0 0.5rem;
    font-size: 0.875rem;
    color: var(--kg-text-secondary);
    line-height: 1.5;
  }
  .meta {
    margin: 0;
    font-size: 0.75rem;
    color: var(--kg-text-muted);
  }
  .row {
    margin-top: 0.75rem;
  }
  .empty {
    color: var(--kg-text-muted);
  }
</style>
