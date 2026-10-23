import { getCurrentUser, t } from '@/lib'
import { KAppStatus } from '@/k'

export default function Home() {
  const currentUser = getCurrentUser()

  return (
    <>
      <section className="border border-[var(--kg-border)] bg-[var(--kg-surface)]">
        <header className="flex items-center justify-between border-b border-[var(--kg-border)] px-5 py-4">
          <h3 className="text-sm font-semibold text-[var(--kg-text)]">
            {t('shell.welcome', currentUser.name)}
          </h3>
        </header>
        <div className="p-5">{t('shell.signed_in', currentUser.email)}</div>
      </section>
      <div className="h-4 w-full" />
      <section className="border border-[var(--kg-border)] bg-[var(--kg-surface)]">
        <header className="flex items-center justify-between border-b border-[var(--kg-border)] px-5 py-4">
          <h3 className="text-sm font-semibold text-[var(--kg-text)]">{t('shell.get_started')}</h3>
        </header>
        <div className="p-5">
          <p className="hint">{t('shell.get_started_hint')}</p>
        </div>
      </section>
      <KAppStatus />
    </>
  )
}
