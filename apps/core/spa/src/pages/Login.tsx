import { useState } from 'react'
import { Alert } from '@/components/ui/alert'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Layout } from '@/components/shell/Layout'
import { LayoutMain } from '@/components/shell/LayoutMain'
import { t } from '@/lib/i18n-context'

type Props = {
  onlogin: () => void
}

export function Login({ onlogin }: Props) {
  const [email, setEmail] = useState('admin@kaizengo.local')
  const [password, setPassword] = useState('')
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState('')

  async function submit(e: React.FormEvent) {
    e.preventDefault()
    setLoading(true)
    setError('')
    try {
      const res = await fetch('/auth/login', {
        method: 'POST',
        credentials: 'include',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ email, password }),
      })
      if (!res.ok) {
        const text = await res.text()
        throw new Error(text || `Login failed (${res.status})`)
      }
      onlogin()
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err))
    } finally {
      setLoading(false)
    }
  }

  return (
    <div className="kg-page kg-app flex min-h-screen w-full flex-col bg-[var(--kg-bg)]">
      <Layout variant="centered" containerSize="sm" align="center">
        <LayoutMain>
          <div className="login-stack">
            <div className="login-brand">
              <div className="login-logo" aria-hidden="true">
                <img src="/static/icon-mono.png" alt="Kaizengo" />
              </div>
              <h1 className="login-title">{t('login.brand')}</h1>
              <p className="login-tagline">{t('login.tagline')}</p>
            </div>

            <form className="login-form" onSubmit={submit}>
              {error ? <Alert variant="danger">{error}</Alert> : null}

              <div className="flex min-w-[12rem] flex-1 flex-col gap-1">
                <Label>
                  {t('login.email')}
                  <span className="text-[var(--kg-danger)]"> *</span>
                </Label>
                <Input
                  type="email"
                  value={email}
                  onChange={(e) => setEmail(e.target.value)}
                  placeholder="admin@kaizengo.local"
                  required
                />
              </div>

              <div className="flex min-w-[12rem] flex-1 flex-col gap-1">
                <Label>
                  {t('login.password')}
                  <span className="text-[var(--kg-danger)]"> *</span>
                </Label>
                <Input
                  type="password"
                  value={password}
                  onChange={(e) => setPassword(e.target.value)}
                  placeholder={t('login.password_placeholder')}
                  required
                />
              </div>

              <Button type="submit" disabled={loading || !email || !password}>
                {loading ? '…' : t('login.submit')}
              </Button>

              <p className="login-hint">
                <a href="/web/database/manager">{t('login.manage_databases')}</a>
              </p>
            </form>
          </div>
        </LayoutMain>
      </Layout>
    </div>
  )
}
