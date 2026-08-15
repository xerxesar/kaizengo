<script lang="ts">
  import { Button, FormField, Input, Layout, LayoutMain, Page, Alert } from '@kaizengo/sdk-svelte/ui'

  type Props = {
    t: (key: string, ...args: Array<string | number>) => string
    onlogin: () => void
  }

  let { t, onlogin }: Props = $props()

  let email = $state('admin@kaizengo.local')
  let password = $state('')
  let loading = $state(false)
  let error = $state('')

  async function submit(e: Event) {
    e.preventDefault()
    loading = true
    error = ''
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
      error = err instanceof Error ? err.message : String(err)
    } finally {
      loading = false
    }
  }
</script>

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

      <form class="login-form" onsubmit={submit}>
        {#if error}
          <Alert variant="danger">{error}</Alert>
        {/if}

        <FormField label={t('login.email')} required>
          {#snippet children()}
            <Input type="email" bind:value={email} placeholder="admin@kaizengo.local" />
          {/snippet}
        </FormField>

        <FormField label={t('login.password')} required>
          {#snippet children()}
            <Input type="password" bind:value={password} placeholder={t('login.password_placeholder')} />
          {/snippet}
        </FormField>

        <Button type="submit" loading={loading} disabled={!email || !password}>{t('login.submit')}</Button>

        <p class="hint">{t('login.hint', 'admin@kaizengo.local', 'changeme')}</p>
      </form>
    </div>
    </LayoutMain>
  </Layout>
</Page>

<style>
  .login-stack {
    display: flex;
    flex-direction: column;
    align-items: stretch;
    gap: var(--kg-space-07);
    width: 100%;
  }
  .login-brand {
    text-align: center;
  }
  .login-logo {
    width: 6rem;
    height: 6rem;
    margin: 0 auto 0;
  }
  .login-logo img {
    width: 100%;
    height: 100%;
    object-fit: contain;
  }
  .login-title {
    font-size: 2rem;
    font-weight: 400;
    line-height: 1.25;
    color: var(--kg-text);
    margin: 0;
  }
  .login-tagline {
    margin: var(--kg-space-03) 0 0;
    font-size: 0.875rem;
    color: var(--kg-text-secondary);
  }
  .login-form {
    width: 100%;
    display: flex;
    flex-direction: column;
    gap: var(--kg-space-05);
    background: var(--kg-surface);
    border: 1px solid var(--kg-border);
    padding: var(--kg-space-06);
  }
  .hint {
    font-size: 0.75rem;
    color: var(--kg-text-muted);
    margin: 0;
  }
</style>
