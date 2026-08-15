<script lang="ts">
  import { onMount } from 'svelte'
  import {
    Alert,
    Card,
    Layout,
    LayoutMain,
    LayoutMenu,
    Page,
    getTheme,
    getThemeMode,
    setI18nLocale,
    syncDocumentLocale,
    t,
    themeIconHref,
  } from '@kaizengo/sdk-svelte/ui'
  import type { ThemeId } from '@kaizengo/sdk-svelte/ui'
  import type { NavEntry } from './lib/nav'
  import ViewHost from './lib/ViewHost.svelte'
  import { fetchMe, logout, type AuthUser } from './lib/auth'
  import Login from './views/Login.svelte'

  let i18nReady = $state(false)
  let apps = $state<NavEntry[]>([])
  let appRoute = $state(currentAppRoute())
  let error = $state('')
  let menuOpen = $state(false)
  let shellTitle = $state('KaizenGo')
  let theme = $state<ThemeId>(getTheme())
  let brandIcon = $derived(themeIconHref(getThemeMode(theme)))
  let user = $state<AuthUser | null>(null)
  let authLoading = $state(true)

  const activeEntry = $derived(apps.find((a) => a.route === appRoute))

  function currentAppRoute(): string {
    const path = window.location.pathname.replace(/\/+$/, '')
    const m = path.match(/^\/app\/([^/]+)/)
    return m?.[1] ?? ''
  }

  function syncRoute() {
    appRoute = currentAppRoute()
  }

  function navigate(to: string) {
    const url = to ? `/app/${to}` : '/app/'
    history.pushState({}, '', url)
    syncRoute()
    menuOpen = false
  }

  async function checkAuth() {
    authLoading = true
    try {
      user = await fetchMe()
      if (user) {
        const { locale } = await syncDocumentLocale()
        setI18nLocale(locale)
        await loadApps()
      }
    } catch (e) {
      user = null
      error = e instanceof Error ? e.message : String(e)
    } finally {
      authLoading = false
    }
  }

  async function loadApps() {
    const res = await fetch('/api/apps', { credentials: 'include' })
    if (!res.ok) throw new Error(`Failed to load apps (${res.status})`)
    apps = await res.json()
  }

  async function handleLogout() {
    await logout()
    user = null
    apps = []
    navigate('')
  }

  onMount(() => {
    const onPop = () => {
      syncRoute()
    }
    const onSettings = (e: Event) => {
      const detail = (e as CustomEvent<{ shellTitle?: string; locale?: string; dir?: 'ltr' | 'rtl' }>)
        .detail
      if (detail?.shellTitle) shellTitle = String(detail.shellTitle)
      if (detail?.locale) {
        setI18nLocale(detail.locale)
      }
      void loadApps()
    }
    const onTheme = (e: Event) => {
      const id = (e as CustomEvent<{ theme?: ThemeId }>).detail?.theme
      theme = id ?? getTheme()
    }
    const onApps = () => {
      void loadApps()
    }
    window.addEventListener('popstate', onPop)
    window.addEventListener('kaizengo:settings', onSettings)
    window.addEventListener('kaizengo:theme', onTheme)
    window.addEventListener('kaizengo:apps', onApps)
    void (async () => {
      try {
        const { locale } = await syncDocumentLocale()
        setI18nLocale(locale)
      } finally {
        i18nReady = true
      }
      await checkAuth()
    })()
    return () => {
      window.removeEventListener('popstate', onPop)
      window.removeEventListener('kaizengo:settings', onSettings)
      window.removeEventListener('kaizengo:theme', onTheme)
      window.removeEventListener('kaizengo:apps', onApps)
    }
  })
</script>

{#if !i18nReady || authLoading}
  <main class="auth-loading"><p>{t('shell.loading')}</p></main>
{:else if !user}
  <Login {t} onlogin={() => void checkAuth()} />
{:else}
  <Page>
    <header class="shell-bar">
      <a
        class="brand"
        href="/app/"
        onclick={(e) => {
          e.preventDefault()
          navigate('')
        }}>
        <img src={brandIcon} alt="KaizenGo" class="brand-icon" />
        <span class="brand-text">{shellTitle}</span>
        </a
      >

      <div class="menu">
        <button
          type="button"
          class="menu-btn"
          aria-haspopup="listbox"
          aria-expanded={menuOpen}
          onclick={() => (menuOpen = !menuOpen)}
        >
          {t('shell.apps')}
          <span class="caret">▾</span>
        </button>
        {#if menuOpen}
          <ul class="menu-list" role="listbox">
            <li>
              <button type="button" class:active={appRoute === ''} onclick={() => navigate('')}>
                {t('shell.home')}
              </button>
            </li>
            {#each apps as a}
              <li>
                <button
                  type="button"
                  class:active={a.route === appRoute}
                  onclick={() => navigate(a.route)}
                >
                  {a.title}
                </button>
              </li>
            {/each}
            {#if !apps.length}
              <li class="empty">{t('shell.no_apps')}</li>
            {/if}
          </ul>
        {/if}
      </div>

      <div class="user-menu">
        <span class="user-name">{user.name}</span>
        {#if user.roles.includes('admin')}
          <span class="admin-badge">{t('shell.admin')}</span>
        {/if}
        <button type="button" class="logout-btn" onclick={() => void handleLogout()}>{t('shell.sign_out')}</button>
      </div>
    </header>

    <div class="shell-body">
      {#if appRoute === ''}
        <Layout title={t('shell.welcome', user.name)} subtitle={t('shell.signed_in', user.email)}>
          <LayoutMain>
            <Card title={t('shell.get_started')}>
              <p class="hint">{t('shell.get_started_hint')}</p>
            </Card>
          </LayoutMain>
        </Layout>
      {:else if activeEntry}
        {#key appRoute}
          <Layout title={activeEntry.title}>
            <LayoutMenu app={appRoute} />
            <LayoutMain>
              {#if error}
                <Alert variant="danger">{error}</Alert>
              {/if}
              <ViewHost
                hostApp={appRoute}
                onerror={(message) => (error = message)}
              />
            </LayoutMain>
          </Layout>
        {/key}
      {:else}
        <Layout title={t('shell.apps')}>
          <LayoutMain>
            <Alert variant="danger">{t('shell.no_apps')}</Alert>
          </LayoutMain>
        </Layout>
      {/if}
    </div>
  </Page>

  {#if menuOpen}
    <button
      type="button"
      class="backdrop"
      aria-label={t('shell.close_menu')}
      onclick={() => (menuOpen = false)}
    ></button>
  {/if}
{/if}

<style>
  .auth-loading {
    display: flex;
    justify-content: center;
    padding: var(--kg-space-09);
    color: var(--kg-text-secondary);
  }
  .hint {
    color: var(--kg-text-secondary);
    margin: 0;
  }
  .user-menu {
    margin-inline-start: auto;
    display: flex;
    align-items: center;
    gap: var(--kg-space-04);
    font-size: 0.875rem;
  }
  .user-name {
    color: var(--kg-shell-text);
    font-weight: 400;
  }
  .admin-badge {
    background: var(--kg-shell-hover);
    color: var(--kg-shell-text-muted);
    font-size: 0.6875rem;
    font-weight: 600;
    text-transform: uppercase;
    letter-spacing: 0.32px;
    padding: var(--kg-space-01) var(--kg-space-03);
  }
  .logout-btn {
    height: var(--kg-control-height-sm);
    padding: 0 var(--kg-space-05);
    border: 1px solid var(--kg-shell-border);
    background: transparent;
    color: var(--kg-shell-text);
    cursor: pointer;
    font: inherit;
    font-size: 0.875rem;
    transition: background 70ms cubic-bezier(0.2, 0, 0.38, 0.9);
  }
  .logout-btn:hover {
    background: var(--kg-shell-hover);
  }
  .logout-btn:focus-visible {
    outline: 2px solid var(--kg-primary);
    outline-offset: -2px;
  }
</style>
