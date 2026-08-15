<script lang="ts">
  import { setContext } from 'svelte'
  import { I18N_CTX, type I18nScope } from './i18n-context'
  import { bundledTranslator, t } from './i18n-runtime.svelte'
  import { inferAppName } from './layout-context'

  type Props = {
    /** App id; defaults to `/app/{name}` or host `data-kg-app`. */
    app?: string
    children: import('svelte').Snippet
  }

  let { app = '', children }: Props = $props()

  let root = $state<HTMLElement | null>(null)

  const appName = $derived(app || inferAppName(root))

  const scope = $derived.by(
    (): I18nScope => ({
      app: appName,
      translator: bundledTranslator(),
      t,
    }),
  )

  setContext(I18N_CTX, {
    get app() {
      return scope.app
    },
    get translator() {
      return scope.translator
    },
    get t() {
      return scope.t
    },
  })
</script>

<div bind:this={root} class="kg-i18n-provider">
  {@render children()}
</div>

<style>
  .kg-i18n-provider {
    display: flex;
    flex-direction: column;
    flex: 1;
    min-width: 0;
    min-height: 0;
  }
</style>
