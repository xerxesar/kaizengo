import { createSignal, onMount, Show, type JSX } from 'solid-js'
import { Alert } from './Alert'
import { Card } from './Card'
import { Spinner } from './Spinner'
import { t } from './i18n-context'
import { inferAppName } from './layout-context'
import { fetchAppPing } from './model-client'

type Props = {
  app?: string
  onerror?: (message: string) => void
}

export function KAppStatus(props: Props): JSX.Element {
  let root: HTMLDivElement | undefined

  const [loading, setLoading] = createSignal(true)
  const [error, setError] = createSignal('')
  const [status, setStatus] = createSignal('')

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

  onMount(() => {
    void refresh(props.app || inferAppName(root ?? null))
  })

  const appName = () => props.app || inferAppName(root ?? null)

  return (
    <div ref={root} class="mt-6">
      <Show when={error()} fallback={
        <Show when={!loading()} fallback={<Spinner />}>
          <Card title={t(`${appName()}.ping`)}>
            <p>
              <code>{status()}</code>
            </p>
          </Card>
        </Show>
      }>
        <Alert variant="danger">{error()}</Alert>
      </Show>
    </div>
  )
}
