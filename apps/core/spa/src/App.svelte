<script lang="ts">
  import { onMount } from 'svelte'
  import type { NavEntry, SpaAppModule } from './lib/nav'

  let apps = $state<NavEntry[]>([])
  let route = $state(currentRoute())
  let hostEl = $state<HTMLElement | null>(null)
  let active: SpaAppModule | null = null
  let error = $state('')
  let menuOpen = $state(false)
  let shellTitle = $state('KaizenGo')

  function currentRoute(): string {
    const base = '/app'
    let path = window.location.pathname
    if (path.startsWith(base)) path = path.slice(base.length)
    return path.replace(/^\/+|\/+$/g, '')
  }

  function navigate(to: string) {
    const url = to ? `/app/${to}` : '/app/'
    history.pushState({}, '', url)
    route = currentRoute()
    menuOpen = false
  }

  function activeEntry(): NavEntry | undefined {
    return apps.find((a) => a.route === route)
  }

  async function mountApp(entry: NavEntry) {
    error = ''
    if (active?.unmount && hostEl) active.unmount(hostEl)
    active = null
    if (hostEl) hostEl.innerHTML = ''
    if (!hostEl) return

    try {
      const mod = (await import(/* @vite-ignore */ entry.moduleUrl)) as {
        default: SpaAppModule
      }
      active = mod.default
      await active.mount(hostEl)
    } catch (e) {
      error = e instanceof Error ? e.message : String(e)
    }
  }

  function clearApp() {
    if (active?.unmount && hostEl) active.unmount(hostEl)
    active = null
    if (hostEl) hostEl.innerHTML = ''
  }

  $effect(() => {
    const entry = activeEntry()
    const host = hostEl
    if (!entry || !host) {
      if (!entry) clearApp()
      return
    }
    void mountApp(entry)
  })

  onMount(() => {
    const onPop = () => {
      route = currentRoute()
    }
    const onSettings = (ev: Event) => {
      const detail = (ev as CustomEvent<{ shellTitle?: string }>).detail
      if (detail?.shellTitle) shellTitle = String(detail.shellTitle)
    }
    window.addEventListener('popstate', onPop)
    window.addEventListener('kaizengo:settings', onSettings)

    void fetch('/api/apps')
      .then((r) => r.json())
      .then((list: NavEntry[]) => {
        apps = Array.isArray(list) ? list : []
      })
      .catch((e) => {
        error = e instanceof Error ? e.message : String(e)
      })

    void fetch('/graphql', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ query: '{ settings { shellTitle } }' }),
    })
      .then((r) => r.json())
      .then((body) => {
        if (body.data?.settings?.shellTitle) {
          shellTitle = body.data.settings.shellTitle
        }
      })
      .catch(() => {
        /* settings app optional */
      })

    return () => {
      window.removeEventListener('popstate', onPop)
      window.removeEventListener('kaizengo:settings', onSettings)
    }
  })
</script>

<main>
  <header class="shell-bar">
    <a
      class="brand"
      href="/app/"
      onclick={(e) => {
        e.preventDefault()
        navigate('')
      }}>{shellTitle}</a
    >

    <div class="menu">
      <button
        type="button"
        class="menu-btn"
        aria-haspopup="listbox"
        aria-expanded={menuOpen}
        onclick={() => (menuOpen = !menuOpen)}
      >
        Apps
        <span class="caret">▾</span>
      </button>
      {#if menuOpen}
        <ul class="menu-list" role="listbox">
          <li>
            <button type="button" class:active={route === ''} onclick={() => navigate('')}>
              Home
            </button>
          </li>
          {#each apps as a}
            <li>
              <button
                type="button"
                class:active={a.route === route}
                onclick={() => navigate(a.route)}
              >
                {a.title}
              </button>
            </li>
          {/each}
          {#if !apps.length}
            <li class="empty">No apps registered</li>
          {/if}
        </ul>
      {/if}
    </div>
  </header>

  {#if route === ''}
    <section class="panel">
      <h1>Core</h1>
      <p>SPA shell. Apps load via <code>import()</code> + <code>mount(el)</code>.</p>
      <p class="hint">Open the <strong>Apps</strong> menu to launch one.</p>
    </section>
  {:else}
    <section class="panel">
      {#if error}
        <div class="error">{error}</div>
      {/if}
      <div bind:this={hostEl} class="host"></div>
    </section>
  {/if}
</main>

{#if menuOpen}
  <button
    type="button"
    class="backdrop"
    aria-label="Close menu"
    onclick={() => (menuOpen = false)}
  ></button>
{/if}
