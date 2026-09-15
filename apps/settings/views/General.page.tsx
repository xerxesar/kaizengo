import { useEffect, useState } from 'react'
import { Alert } from '@/components/ui/alert'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Progress } from '@/components/ui/progress'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import { applyLocale, Card, FormActions } from '@/lib'
import { KAppStatus } from '@/k'
import { fetchSettings, updateSettings, type PlatformSettings } from '../lib/graphql'

function OptionSelect(props: {
  value: string
  options: { value: string; label: string }[]
  onChange: (value: string) => void
  'aria-label'?: string
}) {
  return (
    <Select value={props.value || undefined} onValueChange={props.onChange}>
      <SelectTrigger aria-label={props['aria-label']}>
        <SelectValue />
      </SelectTrigger>
      <SelectContent>
        {props.options.map((opt) => (
          <SelectItem key={opt.value} value={opt.value}>
            {opt.label}
          </SelectItem>
        ))}
      </SelectContent>
    </Select>
  )
}

export default function General() {
  const [settings, setSettings] = useState<PlatformSettings | null>(null)
  const [locale, setLocale] = useState('')
  const [defaultCalendar, setDefaultCalendar] = useState('')
  const [shellTitle, setShellTitle] = useState('')
  const [loading, setLoading] = useState(true)
  const [saving, setSaving] = useState(false)
  const [error, setError] = useState('')
  const [saved, setSaved] = useState(false)

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

  async function submit(e: React.FormEvent) {
    e.preventDefault()
    if (!settings) return
    setSaving(true)
    setError('')
    setSaved(false)
    try {
      const data = await updateSettings({
        locale,
        defaultCalendar,
        shellTitle,
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

  useEffect(() => {
    void load()
  }, [])

  return (
    <>
      {error ? (
        <Alert variant="danger">
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
      {saved && settings ? (
        <Alert variant="success">
          <div className="min-w-0 flex-1">{settings.labels.saved}</div>
        </Alert>
      ) : null}

      {loading ? (
        <div className="flex items-center justify-center py-8" role="status" aria-label="Loading">
          <Progress indeterminate className="w-48 max-w-full" />
        </div>
      ) : settings ? (
        <Card>
          <form className="flex max-w-md flex-col gap-5" onSubmit={submit}>
            <div className="flex min-w-[12rem] flex-1 flex-col gap-1">
              <Label>{settings.labels.locale}</Label>
              <OptionSelect
                value={locale}
                options={settings.locales.map((l) => ({
                  value: l.id,
                  label: `${l.name} (${l.dir})`,
                }))}
                onChange={setLocale}
                aria-label={settings.labels.locale}
              />
            </div>

            <div className="flex min-w-[12rem] flex-1 flex-col gap-1">
              <Label>{settings.labels.calendar}</Label>
              <OptionSelect
                value={defaultCalendar}
                options={settings.calendars.map((c) => ({ value: c.id, label: c.name }))}
                onChange={setDefaultCalendar}
                aria-label={settings.labels.calendar}
              />
            </div>

            <div className="flex min-w-[12rem] flex-1 flex-col gap-1">
              <Label>{settings.labels.shell}</Label>
              <Input value={shellTitle} onChange={(e) => setShellTitle(e.target.value)} />
            </div>

            <FormActions>
              <Button type="submit" disabled={saving}>
                {saving ? '…' : settings.labels.save}
              </Button>
            </FormActions>
          </form>
        </Card>
      ) : null}

      <KAppStatus />
    </>
  )
}
