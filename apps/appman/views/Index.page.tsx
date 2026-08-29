import { createMemo, createSignal, For, onMount, Show } from 'solid-js'
import {
  Alert,
  Badge,
  Button,
  Card,
  Input,
  KAppStatus,
  Spinner,
  t,
} from '@kaizengo/sdk-solid/ui'
import { fetchApps, installApp, upgradeApp, type App } from '../lib/graphql'

export default function Index() {
  const [loading, setLoading] = createSignal(true)
  const [error, setError] = createSignal('')
  const [apps, setApps] = createSignal<App[]>([])
  const [q, setQ] = createSignal('')
  const [busy, setBusy] = createSignal('')

  const filtered = createMemo(() =>
    apps().filter((a) => {
      const hay = `${a.name} ${a.title} ${a.summary}`.toLowerCase()
      return hay.includes(q().trim().toLowerCase())
    }),
  )

  async function refresh() {
    setLoading(true)
    setError('')
    try {
      setApps(await fetchApps())
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e))
    } finally {
      setLoading(false)
    }
  }

  function replace(updated: App) {
    setApps((prev) => prev.map((a) => (a.name === updated.name ? updated : a)))
  }

  async function install(name: string) {
    setBusy(name)
    setError('')
    try {
      replace(await installApp(name))
      window.dispatchEvent(new CustomEvent('kaizengo:apps'))
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
      replace(await upgradeApp(name))
      window.dispatchEvent(new CustomEvent('kaizengo:apps'))
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e))
    } finally {
      setBusy('')
    }
  }

  onMount(() => {
    void refresh()
  })

  return (
    <>
      <p class="mb-4 text-sm leading-relaxed text-[var(--kg-text-secondary)]">{t('appman.subtitle')}</p>

      <Show when={error()}>
        <Alert variant="danger" dismissible onDismiss={() => setError('')}>
          {error()}
        </Alert>
      </Show>

      <div class="mb-4 max-w-xs">
        <Input value={q()} onChange={setQ} placeholder={t('appman.search')} />
      </div>

      <Show when={!loading()} fallback={<Spinner />}>
        <Show
          when={filtered().length > 0}
          fallback={<p class="text-[var(--kg-text-muted)]">{t('appman.empty')}</p>}
        >
          <div class="grid grid-cols-[repeat(auto-fill,minmax(18rem,1fr))] gap-4">
            <For each={filtered()}>
              {(app) => (
                <Card
                  title={app.title}
                  actions={
                    <>
                      <Show
                        when={app.autoInstall}
                        fallback={
                          <Show
                            when={app.upgrade}
                            fallback={
                              <Show when={app.installed} fallback={<Badge>{t('appman.available')}</Badge>}>
                                <Badge variant="success">{t('appman.installed')}</Badge>
                              </Show>
                            }
                          >
                            <Badge variant="warning">{t('appman.upgrade')}</Badge>
                          </Show>
                        }
                      >
                        <Badge variant="muted">{t('appman.system')}</Badge>
                      </Show>
                    </>
                  }
                >
                  <p class="mb-2 text-sm leading-relaxed text-[var(--kg-text-secondary)]">{app.summary}</p>
                  <p class="m-0 text-xs text-[var(--kg-text-muted)]">
                    {t('appman.version', app.version)}
                    {app.installedVersion && app.installedVersion !== app.version
                      ? ` · ${t('appman.installed_version', app.installedVersion)}`
                      : ''}
                  </p>
                  <Show when={app.depends.length}>
                    <p class="m-0 text-xs text-[var(--kg-text-muted)]">
                      {t('appman.depends', app.depends.join(', '))}
                    </p>
                  </Show>
                  <Show when={!app.autoInstall}>
                    <div class="mt-3">
                      <Show
                        when={!app.installed}
                        fallback={
                          <Show when={app.upgrade}>
                            <Button
                              loading={busy() === app.name}
                              disabled={busy() !== '' && busy() !== app.name}
                              onClick={() => void upgrade(app.name)}
                            >
                              {t('appman.upgrade')}
                            </Button>
                          </Show>
                        }
                      >
                        <Button
                          loading={busy() === app.name}
                          disabled={busy() !== '' && busy() !== app.name}
                          onClick={() => void install(app.name)}
                        >
                          {t('appman.install')}
                        </Button>
                      </Show>
                    </div>
                  </Show>
                </Card>
              )}
            </For>
          </div>
        </Show>
      </Show>

      <KAppStatus />
    </>
  )
}
