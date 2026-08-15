package main

const moduleGoTmpl = `package {{.Pkg}}

import (
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
		TitleKey:  "nav.{{.Name}}",
		Title:     "{{.Title}}", // fallback until nav.{{.Name}} is added to the i18n catalog
		Route:     "{{.Route}}",
	})
	return nil
}

func (a *App) Mount(host *module.Host) error {
	_ = host
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

const svelteViewTmpl = `<script lang="ts">
  import { KAppStatus, t } from '@kaizengo/sdk-svelte/ui'
</script>

<p class="lead">{t('{{.Name}}.subtitle')}</p>

<KAppStatus />

<style>
  .lead {
    margin: 0 0 var(--kg-space-05);
    color: var(--kg-text-secondary);
  }
</style>
`

const svelteListViewTmpl = `<script lang="ts">
  import { KTable, KAppStatus, t } from '@kaizengo/sdk-svelte/ui'
</script>

<KTable model="{{.Name}}.item" emptyMessage={t('{{.Name}}.empty')} />

<KAppStatus />
`

const svelteFormViewTmpl = `<script lang="ts">
  import { KForm, KAppStatus } from '@kaizengo/sdk-svelte/ui'
</script>

<KForm model="{{.Name}}.item" />

<KAppStatus />
`

const svelteGraphQLTSTmpl = `async function gql<T>(query: string, variables?: Record<string, unknown>): Promise<T> {
  const res = await fetch('/graphql', {
    method: 'POST',
    credentials: 'include',
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

const svelteViteConfig = `// Per-app Vite builds are no longer used — views compile into apps/core/spa.
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
    "build": "vite build",
    "dev": "vite build --watch --mode development"
  },
  "dependencies": {
    "@kaizengo/sdk-svelte": "file:../../../packages/sdk-svelte"
  },
  "devDependencies": {
    "@sveltejs/vite-plugin-svelte": "file:../../core/spa/node_modules/@sveltejs/vite-plugin-svelte",
    "svelte": "file:../../core/spa/node_modules/svelte",
    "vite": "file:../../core/spa/node_modules/vite",
    "typescript": "file:../../core/spa/node_modules/typescript"
  }
}
`

const eventsMigrationSQLTmpl = `-- Event store tables (per-app schema)

CREATE TABLE IF NOT EXISTS streams (
    stream_id   TEXT PRIMARY KEY,
    stream_type TEXT NOT NULL,
    version     BIGINT NOT NULL DEFAULT 0,
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS events (
    id          BIGSERIAL PRIMARY KEY,
    event_id    TEXT NOT NULL UNIQUE,
    stream_id   TEXT NOT NULL,
    stream_type TEXT NOT NULL,
    version     BIGINT NOT NULL,
    event_type  TEXT NOT NULL,
    payload     JSONB NOT NULL,
    metadata    JSONB,
    occurred_at TIMESTAMPTZ NOT NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE(stream_id, version)
);

CREATE INDEX IF NOT EXISTS idx_events_stream_id ON events(stream_id, version);
CREATE INDEX IF NOT EXISTS idx_events_id ON events(id);
`

const readModelMigrationSQLTmpl = `-- Read model for {{.Name}}.item

CREATE TABLE IF NOT EXISTS items_read (
    id         TEXT PRIMARY KEY,
    org_id     TEXT NOT NULL,
    author_id  TEXT NOT NULL,
    deleted    BOOLEAN NOT NULL DEFAULT false,
    created_at TIMESTAMPTZ NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL,
    title      TEXT NOT NULL DEFAULT ''
);

CREATE INDEX IF NOT EXISTS idx_items_read_org ON items_read(org_id, updated_at DESC);
`

const appSpecYAMLTmpl = `name: {{.Name}}
title: {{.Title}}
summary: {{.Summary}}
depends:
  - core
  - identity
  - auth
  - permissions
nav:
  labelKey: nav.{{.Name}}
  route: {{.Route}}
models:
  - name: item
    fields:
      - name: title
        type: string
        required: true
menus:
  - id: items
    labelKey: {{.Name}}.menu.items
    children:
      - id: item_list
        labelKey: {{.Name}}.menu.list
        view: Items
      - id: item_form
        labelKey: {{.Name}}.menu.new
        view: NewItem
locales:
  - id: en
    name: English
    dir: ltr
  - id: fa
    name: Persian
    dir: rtl
`

const eventSourcedModuleGoTmpl = `package {{.Pkg}}

//go:generate go run ../../cmd/godino gen-types {{.Name}}

import (
	"kaizengo/internal/module"
	"kaizengo/packages/sdk-go/engine"
)

func init() {
	app := engine.New(engine.Options{
		AppName: "{{.Name}}",
		Version: "0.1.0",
	})
	module.Register(app.Hooks("item", itemHooks()))
}
`

const eventSourcedHooksGoTmpl = `package {{.Pkg}}

import (
	"strings"

	"kaizengo/packages/sdk-go/engine"
)

func itemHooks() engine.Hooks {
	return engine.Hooks{
		BeforeCreate: trimItemTitle,
		BeforeUpdate: trimItemTitle,
	}
}

func trimItemTitle(hc engine.HookContext) error {
	if title, ok := hc.Fields["title"].(string); ok {
		hc.Fields["title"] = strings.TrimSpace(title)
	}
	return nil
}
`

const localeEnTmpl = `msgid ""
msgstr ""
"Language: en\n"
"Content-Type: text/plain; charset=UTF-8\n"

msgid "{{.Name}}.title"
msgstr "{{.Title}}"

msgid "{{.Name}}.subtitle"
msgstr "{{.Summary}}"

msgid "{{.Name}}.ping"
msgstr "API status"

msgid "{{.Name}}.empty"
msgstr "No records yet."

msgid "{{.Name}}.new_placeholder"
msgstr "Enter a title…"

msgid "{{.Name}}.create"
msgstr "Add item"

msgid "{{.Name}}.menu.items"
msgstr "Items"

msgid "{{.Name}}.menu.list"
msgstr "All items"

msgid "{{.Name}}.menu.new"
msgstr "New item"
`

const localeFaTmpl = `msgid ""
msgstr ""
"Language: fa\n"
"Content-Type: text/plain; charset=UTF-8\n"

msgid "{{.Name}}.title"
msgstr "{{.Title}}"

msgid "{{.Name}}.subtitle"
msgstr "{{.Summary}}"

msgid "{{.Name}}.ping"
msgstr "وضعیت API"

msgid "{{.Name}}.empty"
msgstr "هنوز رکوردی نیست."

msgid "{{.Name}}.new_placeholder"
msgstr "عنوان را وارد کنید…"

msgid "{{.Name}}.create"
msgstr "افزودن مورد"

msgid "{{.Name}}.menu.items"
msgstr "موارد"

msgid "{{.Name}}.menu.list"
msgstr "همه موارد"

msgid "{{.Name}}.menu.new"
msgstr "مورد جدید"
`

const addonAppYamlTmpl = `name: {{.Name}}
title: {{.Title}}
summary: {{.Summary}}
depends:
  - core
spa: false
i18n: false
events:
  enabled: false
extends:
  - point: model.*.*.afterCreate
    handler: onAfterCreate
    priority: 100
`

const addonModuleGoTmpl = `package {{.Pkg}}

import (
	"log/slog"

	"kaizengo/internal/module"
	"kaizengo/packages/sdk-go/app"
	"kaizengo/packages/sdk-go/extension"
)

const appName = "{{.Name}}"
const appVersion = "0.1.0"

func init() {
	extension.RegisterNamed("onAfterCreate", onAfterCreate)
	module.Register(&App{})
}

type App struct{}

func (a *App) Manifest() module.Manifest {
	return app.ManifestFromSpec(app.MustAppSpec(appName), appVersion)
}

func (a *App) Setup(host *module.Host) error {
	_ = host
	spec := app.MustAppSpec(appName)
	return extension.SetupAddon(spec)
}

func (a *App) Mount(host *module.Host) error {
	_ = host
	return nil
}

func onAfterCreate(ctx extension.Context) error {
	slog.Info("{{.Name}} extension",
		"point", ctx.Point,
		"app", ctx.App.Name,
		"model", ctx.Model.Name,
		"recordId", ctx.RecordID,
	)
	return nil
}
`

