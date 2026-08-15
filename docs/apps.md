# Apps system

kaizengo is organized as a **platform kernel** plus **apps** (products):

- **Platform** (`internal/platform`, `internal/auth`) — shared features and extension points (time calendars, i18n, sessions, …). Drivers blank-import to register.
- **Apps** (`apps/*`) — loadable products that *use* the platform. Customize by forking an app, not by patching another app (see [platform.md](platform.md)).

## Concepts

| Piece | Role |
|-------|------|
| `internal/platform` | Kernel APIs + drivers |
| `internal/auth` | Session cookie / Bearer principal middleware |
| `apps/<name>` | A loadable app package (`app.yaml` + `module.go` + optional `views/`) |
| `internal/module` | Manifest, registry, `Host`, nav, GraphQL registry |
| `apps/apps.go` | Blank-imports bundled apps so `init()` runs |
| `core` | SPA shell + login gate + protected `/graphql` and `/api/apps` |
| `appman` | App manager — install / upgrade bundled apps, runs migrations |
| `identity` | Users, org structure, memberships |
| `auth` | Login, sessions, `/auth/*`, `me` query |
| `permissions` | RBAC service (no SPA); guards identity GraphQL |
| `settings` | Locale, default calendar, shell title |

Bundled apps include **appman**, **hellospec**, **inventory**, **settings**, **typesense**, and **audit**. See [auth.md](auth.md) for identity and permissions.
For the event-sourced module convention, see [sdk.md](sdk.md).

Every app has an **`app.yaml`** manifest (name, depends, nav, locales, models). Apps using **`packages/sdk-go/engine`** (`hellospec`, `status`, `identity`, `auth`) are spec-driven; others load the spec for manifest/nav and keep custom `Setup` code.

```text
apps/<name>/
  app.yaml       # required — declarative manifest
  module.go      # engine.New(...) or custom App
  hooks.go       # optional Go lifecycle hooks (engine apps)
  views/         # optional UI — one .svelte file per menu view
  lib/           # optional shared TS for views
```

## Lifecycle

1. Blank import → `init()` → `module.Register(&App{})`
2. `module.Load` resolves dependencies (`Depends`)
3. **Setup** (all apps, dep order): provide services, register nav, register GraphQL fields
4. **Mount** (all apps, dep order): HTTP routes, static assets

```go
type App interface {
	Manifest() module.Manifest
	Setup(host *module.Host) error
	Mount(host *module.Host) error
}
```

## Host bag

```go
host.Provide("my.service", svc)
host.Lookup("my.service")

host.RegisterNav(module.NavEntry{
	ID: "notes", TitleKey: "nav.notes", Route: "notes",
})
// TitleKey resolves via i18n when serving GET /api/apps (auth required).

host.GQL.RegisterQuery("…", &graphql.Field{…})
host.GQL.RegisterMutation("…", &graphql.Field{…})
```

## Shell injection

The core SPA (after login):

1. `GET /api/apps` → nav catalog (requires session; see [auth.md](auth.md))
2. On route change → resolve menu view → render matching `apps/<app>/views/<View>.svelte`

All views compile into the single central SPA at `apps/core/spa`. See [svelte.md](svelte.md).

## Creating an app without touching core

1. Add `apps/myapp` (or use `kaizengo new-app`)
2. Blank-import in `apps/apps.go`
3. Add views under `apps/myapp/views/` (one `.svelte` per menu view)
4. Do **not** edit `apps/core` for GraphQL or nav — register on `host` in your `Setup`
5. Locales load automatically from `apps/myapp/locale/*.po`; use `i18n.T` / `i18n.Error` from `kaizengo/packages/sdk-go/i18n`

For new modules, prefer SDK helpers:

```go
app.MustLoadLocales("myapp")
app.RegisterNavFromSpec(host, "myapp", spec)
```

## Enable / disable / install

Compiled-in apps appear in **App Manager**. Only **installed** apps are Setup/Mounted (nav, GraphQL, migrations).

Platform apps (`core`, `identity`, `auth`, `permissions`, `appman`, `settings`) set `autoInstall: true` and load on first boot.

```bash
# Dev override: load exactly this list (still compiled in)
KaizenGo_APPS=core,identity,auth,permissions,appman,inventory ./bin/server
```

Install and upgrade from the App Manager UI (or GraphQL `installApp` / `upgradeApp`). That applies `apps/<name>/migrations/*.sql` and stamps `kaizengo.installed_apps`.

Empty `KaizenGo_APPS` / `*` no longer means “every compiled app” — it means installed + auto-install.