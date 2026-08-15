<script lang="ts">
  import { onMount } from 'svelte'
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
  } from '@kaizengo/sdk-svelte/ui'
  import { fetchSettings, updateSettings, type PlatformSettings } from '../lib/graphql'
  let settings = $state<PlatformSettings | null>(null)
  let locale = $state('')
  let defaultCalendar = $state('')
  let shellTitle = $state('')
  let loading = $state(true)
  let saving = $state(false)
  let error = $state('')
  let saved = $state(false)

  export async function load() {
    loading = true
    error = ''
    saved = false
    try {
      const data = await fetchSettings()
      settings = data.settings
      locale = data.settings.locale
      defaultCalendar = data.settings.defaultCalendar
      shellTitle = data.settings.shellTitle
      applyLocale(data.settings.locale, data.settings.dir)
    } catch (e) {
      error = e instanceof Error ? e.message : String(e)
    } finally {
      loading = false
    }
  }

  async function submit(e: Event) {
    e.preventDefault()
    if (!settings) return
    saving = true
    error = ''
    saved = false
    try {
      const data = await updateSettings({ locale, defaultCalendar, shellTitle })
      settings = data.updateSettings
      applyLocale(data.updateSettings.locale, data.updateSettings.dir)
      saved = true
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
      error = e instanceof Error ? e.message : String(e)
    } finally {
      saving = false
    }
  }

  onMount(() => {
    void load()
  })
</script>

{#if error}
  <Alert variant="danger" dismissible ondismiss={() => (error = '')}>{error}</Alert>
{:else if saved && settings}
  <Alert variant="success">{settings.labels.saved}</Alert>
{/if}

{#if loading}
  <Spinner />
{:else if settings}
  <Card>
    <form class="settings-form" onsubmit={submit}>
      <FormField label={settings.labels.locale}>
        {#snippet children()}
          <Select
            bind:value={locale}
            options={settings.locales.map((l) => ({
              value: l.id,
              label: `${l.name} (${l.dir})`,
            }))}
          />
        {/snippet}
      </FormField>

      <FormField label={settings.labels.calendar}>
        {#snippet children()}
          <Select
            bind:value={defaultCalendar}
            options={settings.calendars.map((c) => ({ value: c.id, label: c.name }))}
          />
        {/snippet}
      </FormField>

      <FormField label={settings.labels.shell}>
        {#snippet children()}
          <Input bind:value={shellTitle} />
        {/snippet}
      </FormField>

      <FormActions>
        <Button type="submit" loading={saving}>{settings.labels.save}</Button>
      </FormActions>
    </form>
  </Card>
{/if}

<KAppStatus />

<style>
  .settings-form {
    display: flex;
    flex-direction: column;
    gap: var(--kg-space-05);
    max-width: 28rem;
  }
</style>
