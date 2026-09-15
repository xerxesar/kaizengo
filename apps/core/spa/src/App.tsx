import { useCallback, useEffect, useState } from 'react'
import { BrowserRouter, Navigate, Route, Routes, useNavigate, useParams } from 'react-router'
import { ChevronDownIcon, Eye, EyeOff, LayoutGrid, LayoutGridIcon } from 'lucide-react'
import { motion } from 'motion/react'
import { Alert } from '@/components/ui/alert'
import { Button } from '@/components/ui/button'
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu'
import { Layout } from '@/components/shell/Layout'
import { LayoutMain } from '@/components/shell/LayoutMain'
import { LayoutMenu } from '@/components/shell/LayoutMenu'
import { ViewHost } from '@/components/shell/ViewHost'
import { AuthProvider } from '@/lib/auth-context'
import { fetchMe, logout, type AuthUser } from '@/lib/auth'
import {
  activateStoredDB,
  getStoredDB,
  getStoredDebug,
  syncPrefsFromURL,
  syncURLFromPrefs,
} from '@/lib/db-prefs'
import { setI18nLocale, syncDocumentLocale, t } from '@/lib'
import { KeymapProvider } from '@/lib/keymap'
import { getTheme, getThemeMode, themeIconHref, type ThemeId } from '@/lib/theme'
import type { NavEntry } from '@/lib/nav'
import { Login } from '@/pages/Login'

function DebugBar() {
  const [enabled, setEnabled] = useState(getStoredDebug())
  const [clientDB, setClientDB] = useState(getStoredDB())
  const [activeDB, setActiveDB] = useState('')
  const [logs, setLogs] = useState<string[]>([])
  const [showBar, setShowBar] = useState(false)

  function pushLog(msg: string) {
    const line = `[${new Date().toISOString().slice(11, 19)}] ${msg}`
    setLogs((prev) => [line, ...prev].slice(0, 80))
    console.log('[kaizengo:debug]', msg)
  }

  async function refreshActive() {
    try {
      const res = await fetch('/web/database/status', { credentials: 'include' })
      if (!res.ok) {
        pushLog(`status failed (${res.status})`)
        return
      }
      const data = await res.json()
      const active = String(data.active ?? '')
      setActiveDB(active)
      pushLog(`active=${active || '(none)'} client=${getStoredDB() || '(none)'}`)
    } catch (e) {
      pushLog(e instanceof Error ? e.message : String(e))
    }
  }

  function sync() {
    syncPrefsFromURL()
    syncURLFromPrefs()
    setEnabled(getStoredDebug())
    setClientDB(getStoredDB())
    if (getStoredDebug()) {
      pushLog(`prefs db=${getStoredDB() || '(none)'} debug=1`)
      void refreshActive()
    }
  }

  useEffect(() => {
    sync()
    const onPop = () => sync()
    window.addEventListener('popstate', onPop)
    window.addEventListener('kaizengo:db', onPop)
    return () => {
      window.removeEventListener('popstate', onPop)
      window.removeEventListener('kaizengo:db', onPop)
    }
  }, [])

  if (!enabled) return null

  return (
    <>
      {showBar ? (
        <div className="debug-bar" aria-live="polite">
          <div className="debug-tags">
            <span className="debug-tag active" title="Active platform database">
              active: {activeDB || '(none)'}
            </span>
            <span className="debug-tag client" title="Client selection (localStorage)">
              client: {clientDB || '(none)'}
            </span>
          </div>
          <div className="debug-log">
            {logs.map((line) => (
              <div key={line}>{line}</div>
            ))}
          </div>
        </div>
      ) : null}
      <Button
        variant="secondary"
        className="absolute bottom-0 right-0 z-[20000]"
        onClick={() => setShowBar(!showBar)}
      >
        Debug Bar {showBar ? <EyeOff size={16} /> : <Eye size={16} />}
      </Button>
    </>
  )
}

function Shell() {
  const params = useParams()
  const navigate = useNavigate()
  const appRoute = params.app ?? ''

  const [apps, setApps] = useState<NavEntry[]>([])
  const [error, setError] = useState('')
  const [shellTitle, setShellTitle] = useState('KaizenGo')
  const [theme, setThemeState] = useState<ThemeId>(getTheme())
  const [user, setUser] = useState<AuthUser | null>(null)
  const [authLoading, setAuthLoading] = useState(true)
  const [appsMenuOpen, setAppsMenuOpen] = useState(false)
  const [dbReady, setDbReady] = useState(false)
  const [i18nReady, setI18nReady] = useState(false)

  const brandIcon = themeIconHref(getThemeMode(theme))
  const activeEntry = apps.find((a) => a.route === appRoute)

  const loadApps = useCallback(async () => {
    const res = await fetch('/api/apps', { credentials: 'include' })
    if (!res.ok) throw new Error(`Failed to load apps (${res.status})`)
    setApps(await res.json())
  }, [])

  const checkAuth = useCallback(async () => {
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
  }, [loadApps])

  async function handleLogout() {
    await logout()
    setUser(null)
    setApps([])
    navigate('/')
  }

  useEffect(() => {
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
    const onApps = () => void loadApps()
    const onToggleApps = () => setAppsMenuOpen(true)
    const onSignOut = () => void handleLogout()

    window.addEventListener('kaizengo:settings', onSettings)
    window.addEventListener('kaizengo:theme', onTheme)
    window.addEventListener('kaizengo:apps', onApps)
    window.addEventListener('kaizengo:shell.toggleApps', onToggleApps)
    window.addEventListener('kaizengo:shell.signOut', onSignOut)

    void (async () => {
      syncPrefsFromURL()
      syncURLFromPrefs()
      try {
        if (getStoredDB()) {
          await activateStoredDB()
          window.dispatchEvent(new Event('kaizengo:db'))
        }
      } catch (e) {
        setError(e instanceof Error ? e.message : String(e))
      } finally {
        setDbReady(true)
      }
      try {
        const { locale } = await syncDocumentLocale()
        setI18nLocale(locale)
      } finally {
        setI18nReady(true)
      }
      await checkAuth()
    })()

    return () => {
      window.removeEventListener('kaizengo:settings', onSettings)
      window.removeEventListener('kaizengo:theme', onTheme)
      window.removeEventListener('kaizengo:apps', onApps)
      window.removeEventListener('kaizengo:shell.toggleApps', onToggleApps)
      window.removeEventListener('kaizengo:shell.signOut', onSignOut)
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [checkAuth, loadApps])

  if (!i18nReady || !dbReady || authLoading) {
    return (
      <main className="auth-loading">
        <p>{t('shell.loading')}</p>
      </main>
    )
  }

  if (!user) {
    return <Login onlogin={() => void checkAuth()} />
  }

  return (
    <AuthProvider user={user}>
      <KeymapProvider>
        <div className="kg-page kg-app flex min-h-screen w-full flex-col bg-[var(--kg-bg)]">
          <header className="shell-bar">
            <a
              className="brand"
              href="/app/"
              data-keymap-id="go-home"
              onClick={(e) => {
                e.preventDefault()
                navigate('/')
              }}
            >
              <img src={brandIcon} alt="KaizenGo" className="brand-icon" />
              <span className="brand-text">{shellTitle}</span>
            </a>

            <DropdownMenu open={appsMenuOpen} onOpenChange={setAppsMenuOpen}>
              <DropdownMenuTrigger
                className="inline-flex h-12 justify-center items-center gap-2 border-0 bg-transparent px-4 text-sm text-[var(--kg-shell-text)] hover:bg-[var(--kg-shell-hover)]"
                data-keymap-id="toggle-apps"
              >
                <LayoutGridIcon className='h-6 w-6' />
                <span className='pt-0.5 flex flex-row justify-center items-center'>
                  {t('shell.apps')}
                </span>
              </DropdownMenuTrigger>
              <DropdownMenuContent>
                {apps.map((a) => (
                  <DropdownMenuItem key={a.route} onSelect={() => navigate(`/${a.route}`)}>
                    {a.title}
                  </DropdownMenuItem>
                ))}
                {!apps.length ? (
                  <div className="px-4 py-3 text-sm text-[var(--kg-text-muted)]">{t('shell.no_apps')}</div>
                ) : null}
              </DropdownMenuContent>
            </DropdownMenu>

            <div className="user-menu">
              <span className="user-name">{user.name}</span>
              {user.roles.includes('admin') ? (
                <span className="admin-badge">{t('shell.admin')}</span>
              ) : null}
              <Button
                type="button"
                className="logout-btn"
                data-keymap-id="sign-out"
                onClick={() => void handleLogout()}
              >
                {t('shell.sign_out')}
              </Button>
            </div>
          </header>

          <div className="shell-body">
            {!appRoute ? (
              <Layout title={t('shell.not_found')}>
                <LayoutMain>
                  <Alert variant="warning">{t('shell.app_not_found')}</Alert>
                </LayoutMain>
              </Layout>
            ) : !activeEntry ? (
              <Layout title={t('shell.apps')}>
                <LayoutMain>
                  <Alert variant="danger">{t('shell.no_apps')}</Alert>
                </LayoutMain>
              </Layout>
            ) : (
              <motion.div
                key={appRoute}
                initial={{ opacity: 0, y: 6 }}
                animate={{ opacity: 1, y: 0 }}
                transition={{ duration: 0.18 }}
                className="flex min-h-0 w-full flex-1 flex-col"
              >
                <Layout title={activeEntry.title}>
                  <LayoutMenu app={appRoute} />
                  <LayoutMain>
                    {error ? <Alert variant="danger">{error}</Alert> : null}
                    <ViewHost hostApp={appRoute} onerror={(message) => setError(message)} />
                  </LayoutMain>
                </Layout>
              </motion.div>
            )}
          </div>
        </div>
      </KeymapProvider>
    </AuthProvider>
  )
}

export default function App() {
  return (
    <BrowserRouter basename="/app">
      <Routes>
        <Route path="/" element={<Shell />} />
        <Route path="/:app/*" element={<Shell />} />
        <Route path="*" element={<Navigate to="/" replace />} />
      </Routes>
      <DebugBar />
    </BrowserRouter>
  )
}
