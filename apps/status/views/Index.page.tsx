import { t } from '@/lib'
import { KAppStatus } from '@/k'

export default function Index() {
  return (
    <>
      <p className="mb-5 text-[var(--kg-text-secondary)]">{t('status.lead')}</p>
      <KAppStatus />
    </>
  )
}
