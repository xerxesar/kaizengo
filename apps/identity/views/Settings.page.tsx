import { createMemo, createSignal, onMount, Show } from 'solid-js'
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
} from '@kaizengo/sdk-solid/ui'
import { IdentityToolbar } from '../lib/IdentityToolbar'
import { identityState, initIdentity, updateSelectedOrg } from '../lib/state'

export default function Settings() {
  const identity = identityState()
  const [theme, setThemeState] = createSignal<ThemeId>(getTheme())
  const org = createMemo(() => identity.selectedOrg)

  function onThemeChange(value: string) {
    const id = value as ThemeId
    setThemeState(id)
    setTheme(id)
  }

  function onOrgSaved(record: Record<string, unknown>) {
    const current = org()
    if (!current) return
    updateSelectedOrg({
      id: String(record.id ?? current.id),
      name: String(record.name ?? current.name),
      slug: String(record.slug ?? current.slug),
      createdAt: String(record.createdAt ?? current.createdAt),
    })
  }

  onMount(() => {
    void initIdentity()
  })

  return (
    <Show when={!identity.loading} fallback={<Spinner />}>
      <IdentityToolbar />

      <Show when={identity.error} fallback={
        <Show when={org()} fallback={<Alert variant="warning">{t('identity.no_org')}</Alert>}>
          {(currentOrg) => (
            <div class="flex w-full flex-col gap-5">
              <Card title={t('identity.settings.profile')}>
                <Show
                  when={identity.isAdmin}
                  fallback={
                    <FormSection title={t('identity.settings.general')}>
                      <dl class="settings-grid">
                        <div>
                          <dt>{t('identity.settings.name')}</dt>
                          <dd>{currentOrg().name}</dd>
                        </div>
                        <div>
                          <dt>{t('identity.settings.slug')}</dt>
                          <dd>
                            <code>{currentOrg().slug}</code>
                          </dd>
                        </div>
                      </dl>
                    </FormSection>
                  }
                >
                  <FormSection
                    title={t('identity.settings.general')}
                    description={t('identity.settings.general_desc')}
                  >
                    <KForm
                      model="identity.organization"
                      id={currentOrg().id}
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
                </Show>

                <dl class="settings-grid readonly">
                  <div>
                    <dt>{t('identity.settings.id')}</dt>
                    <dd class="mono">{currentOrg().id}</dd>
                  </div>
                  <div>
                    <dt>{t('identity.settings.created')}</dt>
                    <dd>{formatDateTime(currentOrg().createdAt)}</dd>
                  </div>
                </dl>
              </Card>

              <Card title={t('identity.settings.appearance')}>
                <FormSection title={t('identity.settings.theme')} description={t('identity.settings.theme_desc')}>
                  <FormField label={t('identity.settings.theme_label')}>
                    <Select
                      value={theme()}
                      options={THEMES.map((th) => ({ value: th.id, label: th.label }))}
                      onChange={onThemeChange}
                    />
                  </FormField>
                </FormSection>
              </Card>

              <Card title={t('identity.settings.security')}>
                <p class="info">{t('identity.settings.security_body')}</p>
                <dl class="settings-grid compact">
                  <div>
                    <dt>{t('identity.settings.admin_email')}</dt>
                    <dd>
                      <code>KaizenGo_ADMIN_EMAIL</code>
                    </dd>
                  </div>
                  <div>
                    <dt>{t('identity.settings.admin_password')}</dt>
                    <dd>
                      <code>KaizenGo_ADMIN_PASSWORD</code>
                    </dd>
                  </div>
                </dl>
              </Card>
            </div>
          )}
        </Show>
      }>
        <Alert variant="danger">{identity.error}</Alert>
      </Show>

      <KAppStatus />

      <style>{`
        .settings-grid { display: grid; gap: 1rem; margin: 0; }
        .settings-grid.readonly { margin-top: 1.25rem; padding-top: 1.25rem; border-top: 1px solid var(--kg-border); }
        .settings-grid.compact { margin-top: 1rem; }
        .settings-grid div { display: flex; flex-direction: column; gap: 0.25rem; }
        .settings-grid dt { font-size: 0.75rem; font-weight: 600; text-transform: uppercase; letter-spacing: 0.04em; color: var(--kg-text-muted); }
        .settings-grid dd { margin: 0; font-size: 0.9375rem; }
        .settings-grid code, .mono { font-family: var(--kg-font-mono); font-size: 0.8125rem; background: var(--kg-surface-muted); padding: 0.15rem 0.4rem; border-radius: var(--kg-radius-sm); }
        .mono { background: none; padding: 0; word-break: break-all; }
        .info { font-size: 0.875rem; color: var(--kg-text-secondary); line-height: 1.6; }
      `}</style>
    </Show>
  )
}
