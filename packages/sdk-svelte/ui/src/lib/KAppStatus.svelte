<script lang="ts">
  import Alert from './Alert.svelte'
  import Card from './Card.svelte'
  import Spinner from './Spinner.svelte'
  import { getI18n, t } from './i18n-context'
  import { inferAppName } from './layout-context'
  import { fetchAppPing } from './model-client'

  type Props = {
    /** App id; defaults to I18nProvider / route context. */
    app?: string
    onerror?: (message: string) => void
  }

  let { app = '', onerror }: Props = $props()

  let root = $state<HTMLElement | null>(null)
  let loading = $state(true)
  let error = $state('')
  let status = $state('')

  const i18n = getI18n()
  const appName = $derived(app || i18n.app || inferAppName(root))

  async function refresh(name: string) {
    error = ''
    if (!name) {
      error = 'app context not available'
      loading = false
      return
    }
    loading = true
    try {
      status = await fetchAppPing(name)
    } catch (e) {
      error = e instanceof Error ? e.message : String(e)
      onerror?.(error)
    } finally {
      loading = false
    }
  }

  $effect(() => {
    void refresh(appName)
  })
</script>

<div bind:this={root} class="kg-kapp-status">
  {#if error}
    <Alert variant="danger">{error}</Alert>
  {:else if loading}
    <Spinner />
  {:else}
    <Card title={t(`${appName}.ping`)}>
      <p><code>{status}</code></p>
    </Card>
  {/if}
</div>

<style>
  .kg-kapp-status {
    margin-top: var(--kg-space-06);
  }
</style>
