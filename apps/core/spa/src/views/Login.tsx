import { createSignal, Show } from 'solid-js'
import {
  Alert,
  Button,
  FormField,
  Input,
  Layout,
  LayoutMain,
  Page,
  t,
} from '@kaizengo/sdk-solid/ui'

type Props = {
  onlogin: () => void
}

export function Login(props: Props) {
  const [email, setEmail] = createSignal('admin@kaizengo.local')
  const [password, setPassword] = createSignal('')
  const [loading, setLoading] = createSignal(false)
  const [error, setError] = createSignal('')

  async function submit(e: Event) {
    e.preventDefault()
    setLoading(true)
    setError('')
    try {
      const res = await fetch('/auth/login', {
        method: 'POST',
        credentials: 'include',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ email: email(), password: password() }),
      })
      if (!res.ok) {
        const text = await res.text()
        throw new Error(text || `Login failed (${res.status})`)
      }
      props.onlogin()
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err))
    } finally {
      setLoading(false)
    }
  }

  return (
    <Page>
      <Layout variant="centered" containerSize="sm" align="center">
        <LayoutMain>
          <div class="login-stack">
            <div class="login-brand">
              <div class="login-logo" aria-hidden="true">
                <img src="/static/icon-mono.png" alt="Kaizengo" />
              </div>
              <h1 class="login-title">{t('login.brand')}</h1>
              <p class="login-tagline">{t('login.tagline')}</p>
            </div>

            <form class="login-form" onSubmit={submit}>
              <Show when={error()}>
                <Alert variant="danger">{error()}</Alert>
              </Show>

              <FormField label={t('login.email')} required>
                <Input type="email" value={email()} onChange={setEmail} placeholder="admin@kaizengo.local" />
              </FormField>

              <FormField label={t('login.password')} required>
                <Input
                  type="password"
                  value={password()}
                  onChange={setPassword}
                  placeholder={t('login.password_placeholder')}
                />
              </FormField>

              <Button type="submit" loading={loading()} disabled={!email() || !password()}>
                {t('login.submit')}
              </Button>

              <p class="login-hint">{t('login.hint', 'admin@kaizengo.local', 'changeme')}</p>
            </form>
          </div>
        </LayoutMain>
      </Layout>
    </Page>
  )
}
