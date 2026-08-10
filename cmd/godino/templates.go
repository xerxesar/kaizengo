package main

const moduleGoTmpl = `package {{.Pkg}}

import (
	"net/http"

	"kaizengo/internal/module"
{{- if .WithGraphQL}}
	"github.com/graphql-go/graphql"
{{- end}}
)

func init() {
	module.Register(&App{})
}

// App is the {{.Name}} application.
type App struct{}

func (a *App) Manifest() module.Manifest {
	return module.Manifest{
		Name:        "{{.Name}}",
		Version:     "0.1.0",
		Summary:     "{{.Summary}}",
		Depends:     []string{"core"},
		Installable: true,
	}
}

func (a *App) Setup(host *module.Host) error {
{{- if .WithGraphQL}}
	host.GQL.RegisterQuery("{{.Pkg}}Ping", &graphql.Field{
		Type: graphql.NewNonNull(graphql.String),
		Resolve: func(graphql.ResolveParams) (any, error) {
			return "{{.Name}} ok", nil
		},
	})

{{- end}}
	host.RegisterNav(module.NavEntry{
		ID:        "{{.Name}}",
		Title:     "{{.Title}}",
		Route:     "{{.Route}}",
		ModuleURL: "{{.ModuleURL}}",
	})
	return nil
}

func (a *App) Mount(host *module.Host) error {
	host.Router.Handle(
		"/app-assets/{{.Name}}/*",
		http.StripPrefix("/app-assets/{{.Name}}/", http.FileServer(http.Dir("{{.AssetsDir}}"))),
	)
	return nil
}
`

const vanillaSpaJSTmpl = "/**\n" +
	" * {{.Title}} — vanilla ESM module for the core SPA shell.\n" +
	" * Contract: export default { mount(el), unmount?(el) }\n" +
	" */\n" +
	"const plugin = {\n" +
	"  async mount(el) {\n" +
	"    el.innerHTML = `\n" +
	"      <h1>{{.Title}}</h1>\n" +
	"      <p>Bootstrapped vanilla app <code>{{.Name}}</code>.</p>\n" +
	"{{- if .WithGraphQL}}\n" +
	"      <p id=\"ping\">Loading GraphQL…</p>\n" +
	"{{- end}}\n" +
	"    `\n" +
	"{{- if .WithGraphQL}}\n" +
	"    try {\n" +
	"      const res = await fetch('/graphql', {\n" +
	"        method: 'POST',\n" +
	"        headers: { 'Content-Type': 'application/json' },\n" +
	"        body: JSON.stringify({ query: '{ {{.Pkg}}Ping }' }),\n" +
	"      })\n" +
	"      const body = await res.json()\n" +
	"      const ping = el.querySelector('#ping')\n" +
	"      if (body.errors?.length) {\n" +
	"        ping.textContent = body.errors.map((e) => e.message).join(', ')\n" +
	"      } else {\n" +
	"        ping.textContent = 'GraphQL: ' + body.data.{{.Pkg}}Ping\n" +
	"      }\n" +
	"    } catch (e) {\n" +
	"      el.querySelector('#ping').textContent = String(e)\n" +
	"    }\n" +
	"{{- end}}\n" +
	"  },\n" +
	"  unmount(el) {\n" +
	"    el.innerHTML = ''\n" +
	"  },\n" +
	"}\n" +
	"\n" +
	"export default plugin\n"

const svelteMainTmpl = `import { mount, unmount, type Component } from 'svelte'
import App from './App.svelte'

type Mounted = Record<string, unknown>

let instance: Mounted | null = null

function ensureCSS() {
  if (document.querySelector('link[data-app-css="{{.Name}}"]')) return
  const cssLink = document.createElement('link')
  cssLink.rel = 'stylesheet'
  cssLink.href = '/app-assets/{{.Name}}/spa.css'
  cssLink.dataset.appCss = '{{.Name}}'
  document.head.appendChild(cssLink)
}

export default {
  async mount(el: HTMLElement) {
    ensureCSS()
    instance = mount(App as Component, { target: el }) as Mounted
  },
  unmount() {
    if (instance) {
      unmount(instance)
      instance = null
    }
  },
}
`

const svelteAppTmpl = `<script lang="ts">
{{- if .WithGraphQL}}
  import { onMount } from 'svelte'
  import { ping } from './graphql'

  let message = $state('…')
  let error = $state('')

  onMount(async () => {
    try {
      message = await ping()
    } catch (e) {
      error = e instanceof Error ? e.message : String(e)
    }
  })
{{- end}}
</script>

<div class="wrap">
  <h1>{{.Title}}</h1>
  <p>Bootstrapped Svelte app <code>{{.Name}}</code>.</p>
{{- if .WithGraphQL}}
  {#if error}
    <p class="err">{error}</p>
  {:else}
    <p>GraphQL: {message}</p>
  {/if}
{{- end}}
</div>

<style>
  .wrap {
    text-align: center;
  }
  .err {
    color: #b00020;
  }
</style>
`

const svelteGraphQLTSTmpl = `async function gql<T>(query: string, variables?: Record<string, unknown>): Promise<T> {
  const res = await fetch('/graphql', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ query, variables }),
  })
  if (!res.ok) throw new Error(` + "`GraphQL HTTP ${res.status}`" + `)
  const body = await res.json()
  if (body.errors?.length) {
    throw new Error(body.errors.map((e: { message: string }) => e.message).join(', '))
  }
  return body.data as T
}

export async function ping(): Promise<string> {
  const data = await gql<{ {{.Pkg}}Ping: string }>('query { {{.Pkg}}Ping }')
  return data.{{.Pkg}}Ping
}
`

const svelteViteConfig = `import { defineConfig } from 'vite'
import { svelte } from '@sveltejs/vite-plugin-svelte'
import { dirname, resolve } from 'node:path'
import { fileURLToPath } from 'node:url'

const dir = dirname(fileURLToPath(import.meta.url))

export default defineConfig({
  root: dir,
  plugins: [svelte()],
  build: {
    lib: {
      entry: resolve(dir, 'main.ts'),
      formats: ['es'],
      fileName: () => 'spa.js',
      cssFileName: 'spa',
    },
    outDir: resolve(dir, 'dist'),
    emptyOutDir: true,
    cssCodeSplit: false,
  },
})
`

const svelteConfigJS = `import { vitePreprocess } from '@sveltejs/vite-plugin-svelte'

/** @type {import('@sveltejs/vite-plugin-svelte').SvelteConfig} */
export default {
  preprocess: vitePreprocess(),
}
`

const svelteTSConfig = `{
  "compilerOptions": {
    "target": "ES2022",
    "module": "ESNext",
    "moduleResolution": "bundler",
    "verbatimModuleSyntax": true,
    "strict": true,
    "skipLibCheck": true
  },
  "include": ["./**/*.ts", "./**/*.svelte"]
}
`

const sveltePackageJSONTmpl = `{
  "name": "kaizengo-{{.Name}}-spa",
  "private": true,
  "type": "module",
  "scripts": {
    "build": "vite build"
  },
  "devDependencies": {
    "@sveltejs/vite-plugin-svelte": "file:../../core/spa/node_modules/@sveltejs/vite-plugin-svelte",
    "svelte": "file:../../core/spa/node_modules/svelte",
    "vite": "file:../../core/spa/node_modules/vite",
    "typescript": "file:../../core/spa/node_modules/typescript"
  }
}
`
