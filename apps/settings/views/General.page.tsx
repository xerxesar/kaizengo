import { createSignal, onMount, Show } from 'solid-js'
import {
  Alert,
  Button,
  Card,
  FormActions,
  FormField,
  Input,
  KAppStatus,
  Select,
  Spinner,
  applyLocale,
} from '@kaizengo/sdk-solid/ui'
import { fetchSettings, updateSettings, type PlatformSettings } from '../lib/graphql'

export default function General() {
  const [settings, setSettings] = createSignal<PlatformSettings | null>(null)
  const [locale, setLocale] = createSignal('')
  const [defaultCalendar, setDefaultCalendar] = createSignal('')
  const [shellTitle, setShellTitle] = createSignal('')
  const [loading, setLoading] = createSignal(true)
  const [saving, setSaving] = createSignal(false)
  const [error, setError] = createSignal('')
  const [saved, setSaved] = createSignal(false)

  async function load() {
    setLoading(true)
    setError('')
    setSaved(false)
    try {
      const data = await fetchSettings()
      setSettings(data.settings)
      setLocale(data.settings.locale)
      setDefaultCalendar(data.settings.defaultCalendar)
      setShellTitle(data.settings.shellTitle)
      applyLocale(data.settings.locale, data.settings.dir)
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e))
    } finally {
      setLoading(false)
    }
  }

  async function submit(e: Event) {
    e.preventDefault()
    if (!settings()) return
    setSaving(true)
    setError('')
    setSaved(false)
    try {
      const data = await updateSettings({
        locale: locale(),
        defaultCalendar: defaultCalendar(),
        shellTitle: shellTitle(),
      })
      setSettings(data.updateSettings)
      applyLocale(data.updateSettings.locale, data.updateSettings.dir)
      setSaved(true)
      window.dispatchEvent(
        new CustomEvent('kaizengo:settings', {
          detail: {
            shellTitle: data.updateSettings.shellTitle,
            locale: data.updateSettings.locale,
            dir: data.updateSettings.dir,
          },
        }),
      )
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e))
    } finally {
      setSaving(false)
    }
  }

  onMount(() => {
    void load()
  })

  return (
    <>
      <Show when={error()}>
        <Alert variant="danger" dismissible onDismiss={() => setError('')}>
          {error()}
        </Alert>
      </Show>
      <Show when={saved() && settings()}>
        <Alert variant="success">{settings()!.labels.saved}</Alert>
      </Show>

      <Show when={!loading()} fallback={<Spinner />}>
        <Show when={settings()}>
          {(s) => (
            <Card>
              <form class="flex max-w-md flex-col gap-5" onSubmit={submit}>
                <FormField label={s().labels.locale}>
                  <Select
                    value={locale()}
                    options={s().locales.map((l) => ({
                      value: l.id,
                      label: `${l.name} (${l.dir})`,
                    }))}
                    onChange={setLocale}
                  />
                </FormField>

                <FormField label={s().labels.calendar}>
                  <Select
                    value={defaultCalendar()}
                    options={s().calendars.map((c) => ({ value: c.id, label: c.name }))}
                    onChange={setDefaultCalendar}
                  />
                </FormField>

                <FormField label={s().labels.shell}>
                  <Input value={shellTitle()} onChange={setShellTitle} />
                </FormField>

                <FormActions>
                  <Button type="submit" loading={saving()}>
                    {s().labels.save}
                  </Button>
                </FormActions>
              </form>
            </Card>
          )}
        </Show>
      </Show>

      <KAppStatus />
    </>
  )
}
