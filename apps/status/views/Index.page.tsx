import { KAppStatus, t } from '@kaizengo/sdk-solid/ui'

export default function Index() {
  return (
    <>
      <p class="mb-5 text-[var(--kg-text-secondary)]">{t('status.lead')}</p>
      <KAppStatus />
    </>
  )
}
