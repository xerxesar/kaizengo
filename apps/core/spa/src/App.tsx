import { createSignal, onCleanup, onMount, Show } from 'solid-js'
import { Menu } from '@ark-ui/solid/menu'
import {
  Alert,
  Button,
  Card,
  FormField,
  Input,
  KeymapProvider,
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
  navigateApp,
  menuContentClass,
  menuItemClass,
  type ThemeId,
} from '@kaizengo/sdk-solid/ui'
import type { NavEntry } from './lib/nav'
import { ViewHost } from './lib/ViewHost'
import { fetchMe, logout, type AuthUser } from './lib/auth'
import { Login } from './views/Login'

function currentAppRoute(): string {
  const path = window.location.pathname.replace(/\/+$/, '')
  const m = path.match(/^\/app\/([^/]+)/)
  return m?.[1] ?? ''
}

export default function App() {
  const [i18nReady, setI18nReady] = createSignal(false)
  const [apps, setApps] = createSignal<NavEntry[]>([])
  const [appRoute, setAppRoute] = createSignal(currentAppRoute())
  const [error, setError] = createSignal('')
  const [shellTitle, setShellTitle] = createSignal('KaizenGo')
  const [theme, setThemeState] = createSignal<ThemeId>(getTheme())
  const [user, setUser] = createSignal<AuthUser | null>(null)
  const [authLoading, setAuthLoading] = createSignal(true)
  const [appsMenuOpen, setAppsMenuOpen] = createSignal(false)

  const brandIcon = () => themeIconHref(getThemeMode(theme()))
  const activeEntry = () => apps().find((a) => a.route === appRoute())

  function syncRoute() {
    setAppRoute(currentAppRoute())
  }

  function navigate(to: string) {
    navigateApp(to ? `/app/${to}` : '/app/')
  }

  async function loadApps() {
    const res = await fetch('/api/apps', { credentials: 'include' })
    if (!res.ok) throw new Error(`Failed to load apps (${res.status})`)
    setApps(await res.json())
  }

  async function checkAuth() {
    setAuthLoading(true)
    try {
      const me = await fetchMe()
      setUser(me)
      if (me) {
        const { locale } = await syncDocumentLocale()
        setI18nLocale(locale)
        await loadApps()
      }
    } catch (e) {
      setUser(null)
      setError(e instanceof Error ? e.message : String(e))
    } finally {
      setAuthLoading(false)
    }
  }

  async function handleLogout() {
    await logout()
    setUser(null)
    setApps([])
    navigate('')
  }

  onMount(() => {
    const onPop = () => syncRoute()
    const onSettings = (e: Event) => {
      const detail = (e as CustomEvent<{ shellTitle?: string; locale?: string }>).detail
      if (detail?.shellTitle) setShellTitle(String(detail.shellTitle))
      if (detail?.locale) setI18nLocale(detail.locale)
      void loadApps()
    }
    const onTheme = (e: Event) => {
      const id = (e as CustomEvent<{ theme?: ThemeId }>).detail?.theme
      setThemeState(id ?? getTheme())
    }
    const onApps = () => {
      void loadApps()
    }
    const onToggleApps = () => setAppsMenuOpen(true)
    const onSignOut = () => {
      void handleLogout()
    }

    window.addEventListener('popstate', onPop)
    window.addEventListener('kaizengo:settings', onSettings)
    window.addEventListener('kaizengo:theme', onTheme)
    window.addEventListener('kaizengo:apps', onApps)
    window.addEventListener('kaizengo:shell.toggleApps', onToggleApps)
    window.addEventListener('kaizengo:shell.signOut', onSignOut)

    void (async () => {
      try {
        const { locale } = await syncDocumentLocale()
        setI18nLocale(locale)
      } finally {
        setI18nReady(true)
      }
      await checkAuth()
    })()

    onCleanup(() => {
      window.removeEventListener('popstate', onPop)
      window.removeEventListener('kaizengo:settings', onSettings)
      window.removeEventListener('kaizengo:theme', onTheme)
      window.removeEventListener('kaizengo:apps', onApps)
      window.removeEventListener('kaizengo:shell.toggleApps', onToggleApps)
      window.removeEventListener('kaizengo:shell.signOut', onSignOut)
    })
  })

  return (
    <Show
      when={i18nReady() && !authLoading()}
      fallback={
        <main class="auth-loading">
          <p>{t('shell.loading')}</p>
        </main>
      }
    >
      <Show when={user()} fallback={<Login onlogin={() => void checkAuth()} />}>
        {(currentUser) => (
          <KeymapProvider>
          <Page>
            <header class="shell-bar">
              <a
                class="brand"
                href="/app/"
                data-keymap-id="go-home"
                onClick={(e) => {
                  e.preventDefault()
                  navigate('')
                }}
              >
                <img src={brandIcon()} alt="KaizenGo" class="brand-icon" />
                <span class="brand-text">{shellTitle()}</span>
              </a>

              <Menu.Root
                open={appsMenuOpen()}
                onOpenChange={(e) => setAppsMenuOpen(e.open)}
                positioning={{ placement: 'bottom-start' }}
              >
                <Menu.Trigger
                  class="inline-flex h-8 items-center gap-2 border-0 bg-transparent px-4 text-sm text-[var(--kg-shell-text)] hover:bg-[var(--kg-shell-hover)]"
                  data-keymap-id="toggle-apps"
                >
                  {t('shell.apps')}
                  <span class="text-xs opacity-85">▾</span>
                </Menu.Trigger>
                <Menu.Positioner>
                  <Menu.Content class={menuContentClass}>
                    <Menu.Item
                      value="home"
                      class={menuItemClass}
                      onSelect={() => navigate('')}
                    >
                      {t('shell.home')}
                    </Menu.Item>
                    {apps().map((a) => (
                      <Menu.Item
                        value={a.route}
                        class={menuItemClass}
                        onSelect={() => navigate(a.route)}
                      >
                        {a.title}
                      </Menu.Item>
                    ))}
                    <Show when={!apps().length}>
                      <div class="px-4 py-3 text-sm text-[var(--kg-text-muted)]">{t('shell.no_apps')}</div>
                    </Show>
                  </Menu.Content>
                </Menu.Positioner>
              </Menu.Root>

              <div class="user-menu">
                <span class="user-name">{currentUser().name}</span>
                <Show when={currentUser().roles.includes('admin')}>
                  <span class="admin-badge">{t('shell.admin')}</span>
                </Show>
                <button type="button" class="logout-btn" data-keymap-id="sign-out" onClick={() => void handleLogout()}>
                  {t('shell.sign_out')}
                </button>
              </div>
            </header>

            <div class="shell-body">
              <Show
                when={appRoute()}
                fallback={
                  <Layout title={t('shell.welcome', currentUser().name)} subtitle={t('shell.signed_in', currentUser().email)}>
                    <LayoutMain>
                      <Card title={t('shell.get_started')}>
                        <p class="hint">{t('shell.get_started_hint')}</p>
                      </Card>
                    </LayoutMain>
                  </Layout>
                }
              >
                <Show
                  when={activeEntry()}
                  fallback={
                    <Layout title={t('shell.apps')}>
                      <LayoutMain>
                        <Alert variant="danger">{t('shell.no_apps')}</Alert>
                      </LayoutMain>
                    </Layout>
                  }
                >
                  {(entry) => (
                    <Show when={appRoute()} keyed>
                      {(route) => (
                        <Layout title={entry().title}>
                          <LayoutMenu app={route} />
                          <LayoutMain>
                            <Show when={error()}>
                              <Alert variant="danger">{error()}</Alert>
                            </Show>
                            <ViewHost hostApp={route} onerror={(message) => setError(message)} />
                          </LayoutMain>
                        </Layout>
                      )}
                    </Show>
                  )}
                </Show>
              </Show>
            </div>
          </Page>
          </KeymapProvider>
        )}
      </Show>
    </Show>
  )
}
