import { useEffect, useRef, useState } from 'react'
import { Alert } from '@/components/ui/alert'
import { Progress } from '@/components/ui/progress'
import { t } from '@/lib/i18n-context'
import { inferAppName } from '@/lib/layout-context'
import { fetchAppPing } from '@/lib/model-client'
import { getStoredDebug } from '@/lib/db-prefs'

type Props = {
  app?: string
  onerror?: (message: string) => void
}

export function KAppStatus(props: Props) {
  const rootRef = useRef<HTMLDivElement>(null)

  const [loading, setLoading] = useState(true)
  const [error, setError] = useState('')
  const [status, setStatus] = useState('')
  const [isDebugMode, setIsDebugMode] = useState(getStoredDebug())

  async function refresh(name: string) {
    setError('')
    if (!name) {
      setError('app context not available')
      setLoading(false)
      return
    }
    setLoading(true)
    try {
      setStatus(await fetchAppPing(name))
    } catch (e) {
      const message = e instanceof Error ? e.message : String(e)
      setError(message)
      props.onerror?.(message)
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => {
    setIsDebugMode(getStoredDebug())
    void refresh(props.app || inferAppName(rootRef.current))
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [props.app])

  const appName = props.app || inferAppName(rootRef.current)

  if (!isDebugMode) return null

  return (
    <div ref={rootRef} className="mt-6">
      {error ? (
        <Alert variant="danger">{error}</Alert>
      ) : loading ? (
        <div className="flex items-center justify-center py-8" role="status" aria-label="Loading">
          <Progress indeterminate className="w-48 max-w-full" />
        </div>
      ) : (
        <section className="border border-[var(--kg-border)] bg-[var(--kg-surface)]">
          <header className="flex items-center justify-between border-b border-[var(--kg-border)] px-5 py-4">
            <h3 className="text-sm font-semibold text-[var(--kg-text)]">{t(`${appName}.ping`)}</h3>
          </header>
          <div className="p-5">
            <p>
              <code>{status}</code>
            </p>
          </div>
        </section>
      )}
    </div>
  )
}
