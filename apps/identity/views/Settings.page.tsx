import { useEffect, useMemo } from 'react'
import { Alert } from '@/components/ui/alert'
import { Progress } from '@/components/ui/progress'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import { Label } from '@/components/ui/label'
import { formatDateTime, getTheme, setTheme, t, THEMES, type ThemeId, Card, FormSection } from '@/lib'
import { KAppStatus, KForm, KFormField } from '@/k'
import { IdentityToolbar } from '../lib/IdentityToolbar'
import { initIdentity, updateSelectedOrg, useIdentityState } from '../lib/state'

export default function Settings() {
  const identity = useIdentityState()
  const org = identity.selectedOrg
  const theme = getTheme()
  const themeOptions = useMemo(
    () => THEMES.map((th) => ({ value: th.id, label: th.label })),
    [],
  )

  function onThemeChange(value: string) {
    setTheme(value as ThemeId)
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

  useEffect(() => {
    void initIdentity()
  }, [])

  if (identity.loading) {
    return (
      <div className="flex items-center justify-center py-8" role="status" aria-label="Loading">
        <Progress indeterminate className="w-48 max-w-full" />
      </div>
    )
  }

  return (
    <>
      <IdentityToolbar />

      {identity.error ? (
        <Alert variant="danger">
          <div className="min-w-0 flex-1">{identity.error}</div>
        </Alert>
      ) : !org ? (
        <Alert variant="warning">
          <div className="min-w-0 flex-1">{t('identity.no_org')}</div>
        </Alert>
      ) : (
        <div className="flex w-full flex-col gap-5">
          <Card title={t('identity.settings.profile')}>
            {identity.isAdmin ? (
              <FormSection
                title={t('identity.settings.general')}
                description={t('identity.settings.general_desc')}
              >
                <KForm
                  command="identity.reviseOrganization"
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
            ) : (
              <FormSection title={t('identity.settings.general')}>
                <dl className="settings-grid">
                  <div>
                    <dt>{t('identity.settings.name')}</dt>
                    <dd>{org.name}</dd>
                  </div>
                  <div>
                    <dt>{t('identity.settings.slug')}</dt>
                    <dd>
                      <code>{org.slug}</code>
                    </dd>
                  </div>
                </dl>
              </FormSection>
            )}

            <dl className="settings-grid readonly">
              <div>
                <dt>{t('identity.settings.id')}</dt>
                <dd className="mono">{org.id}</dd>
              </div>
              <div>
                <dt>{t('identity.settings.created')}</dt>
                <dd>{formatDateTime(org.createdAt)}</dd>
              </div>
            </dl>
          </Card>

          <Card title={t('identity.settings.appearance')}>
            <FormSection
              title={t('identity.settings.theme')}
              description={t('identity.settings.theme_desc')}
            >
              <div className="flex min-w-[12rem] flex-1 flex-col gap-1">
                <Label>{t('identity.settings.theme_label')}</Label>
                <Select value={theme} onValueChange={onThemeChange}>
                  <SelectTrigger aria-label={t('identity.settings.theme_label')}>
                    <SelectValue />
                  </SelectTrigger>
                  <SelectContent>
                    {themeOptions.map((opt) => (
                      <SelectItem key={opt.value} value={opt.value}>
                        {opt.label}
                      </SelectItem>
                    ))}
                  </SelectContent>
                </Select>
              </div>
            </FormSection>
          </Card>

          <Card title={t('identity.settings.security')}>
            <p className="info">{t('identity.settings.security_body')}</p>
            <dl className="settings-grid compact">
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
    </>
  )
}
