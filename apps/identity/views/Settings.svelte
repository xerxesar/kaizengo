<script lang="ts">
  import { onMount } from 'svelte'
  import {
    Alert,
    Card,
    FormField,
    FormSection,
    KAppStatus,
    KForm,
    KFormField,
    Select,
    Spinner,
    formatDateTime,
    getTheme,
    setTheme,
    t,
    THEMES,
    type ThemeId,
  } from '@kaizengo/sdk-svelte/ui'
  import IdentityToolbar from '../lib/IdentityToolbar.svelte'
  import { identityState, initIdentity, updateSelectedOrg } from '../lib/state.svelte'

  const identity = identityState()

  let theme = $state<ThemeId>(getTheme())
  const org = $derived(identity.selectedOrg)

  function onThemeChange(e: Event) {
    const id = (e.currentTarget as HTMLSelectElement).value as ThemeId
    theme = id
    setTheme(id)
  }

  function onOrgSaved(record: Record<string, unknown>) {
    if (!org) return
    updateSelectedOrg({
      id: String(record.id ?? org.id),
      name: String(record.name ?? org.name),
      slug: String(record.slug ?? org.slug),
      createdAt: String(record.createdAt ?? org.createdAt),
    })
  }

  onMount(() => {
    void initIdentity()
  })
</script>

{#if identity.loading}
  <Spinner />
{:else}
  <IdentityToolbar />

  {#if identity.error}
    <Alert variant="danger">{identity.error}</Alert>
  {:else if !org}
    <Alert variant="warning">{t('identity.no_org')}</Alert>
  {:else}
    <div class="settings">
      <Card title={t('identity.settings.profile')}>
        {#if identity.isAdmin}
          <FormSection title={t('identity.settings.general')} description={t('identity.settings.general_desc')}>
            <KForm
              model="identity.organization"
              id={org.id}
              submitLabel={t('identity.settings.save')}
              successMessage={t('identity.settings.saved')}
              onsuccess={onOrgSaved}
            >
              <KFormField field="name" label={t('identity.settings.name')} />
              <KFormField
                field="slug"
                label={t('identity.settings.slug')}
                hint={t('identity.settings.slug_hint')}
              />
            </KForm>
          </FormSection>
        {:else}
          <FormSection title={t('identity.settings.general')}>
            <dl class="settings-grid">
              <div><dt>{t('identity.settings.name')}</dt><dd>{org.name}</dd></div>
              <div><dt>{t('identity.settings.slug')}</dt><dd><code>{org.slug}</code></dd></div>
            </dl>
          </FormSection>
        {/if}

        <dl class="settings-grid readonly">
          <div><dt>{t('identity.settings.id')}</dt><dd class="mono">{org.id}</dd></div>
          <div><dt>{t('identity.settings.created')}</dt><dd>{formatDateTime(org.createdAt)}</dd></div>
        </dl>
      </Card>

      <Card title={t('identity.settings.appearance')}>
        <FormSection title={t('identity.settings.theme')} description={t('identity.settings.theme_desc')}>
          <FormField label={t('identity.settings.theme_label')}>
            {#snippet children()}
              <Select
                value={theme}
                options={THEMES.map((th) => ({ value: th.id, label: th.label }))}
                onchange={onThemeChange}
              />
            {/snippet}
          </FormField>
        </FormSection>
      </Card>

      <Card title={t('identity.settings.security')}>
        <p class="info">{t('identity.settings.security_body')}</p>
        <dl class="settings-grid compact">
          <div><dt>{t('identity.settings.admin_email')}</dt><dd><code>KaizenGo_ADMIN_EMAIL</code></dd></div>
          <div><dt>{t('identity.settings.admin_password')}</dt><dd><code>KaizenGo_ADMIN_PASSWORD</code></dd></div>
        </dl>
      </Card>
    </div>
  {/if}
{/if}

<KAppStatus />

<style>
  .settings {
    display: flex;
    flex-direction: column;
    gap: var(--kg-space-05);
    width: 100%;
  }
  .settings-grid {
    display: grid;
    gap: 1rem;
    margin: 0;
  }
  .settings-grid.readonly {
    margin-top: 1.25rem;
    padding-top: 1.25rem;
    border-top: 1px solid var(--kg-border);
  }
  .settings-grid.compact {
    margin-top: 1rem;
  }
  .settings-grid div {
    display: flex;
    flex-direction: column;
    gap: 0.25rem;
  }
  .settings-grid dt {
    font-size: 0.75rem;
    font-weight: 600;
    text-transform: uppercase;
    letter-spacing: 0.04em;
    color: var(--kg-text-muted);
  }
  .settings-grid dd {
    margin: 0;
    font-size: 0.9375rem;
  }
  .settings-grid code,
  .mono {
    font-family: var(--kg-font-mono);
    font-size: 0.8125rem;
    background: var(--kg-surface-muted);
    padding: 0.15rem 0.4rem;
    border-radius: var(--kg-radius-sm);
  }
  .mono {
    background: none;
    padding: 0;
    word-break: break-all;
  }
  .info {
    font-size: 0.875rem;
    color: var(--kg-text-secondary);
    line-height: 1.6;
  }
</style>
