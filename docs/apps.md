# Apps system

kaizengo is organized as a **platform kernel** plus **apps** (products):

- **Platform** (`internal/platform`) — shared features and extension points (time calendars, i18n, …). Drivers blank-import to register.
- **Apps** (`apps/*`) — loadable products that *use* the platform. Customize by forking an app, not by patching another app (see [platform.md](platform.md)).

## Concepts

| Piece | Role |
|-------|------|
| `internal/platform` | Kernel APIs + drivers |
| `apps/<name>` | A loadable app package |
| `internal/module` | Manifest, registry, `Host`, nav, GraphQL registry |
| `apps/apps.go` | Blank-imports bundled apps so `init()` runs |
| `core` | SPA shell + shared host mount (`/app`, `/graphql`, …) |
| `settings` | Configure locale, default calendar, shell title |

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
	ID: "notes", Title: "Notes", Route: "notes",
	ModuleURL: "/app-assets/notes/spa.js",
})

host.GQL.RegisterQuery("…", &graphql.Field{…})
host.GQL.RegisterMutation("…", &graphql.Field{…})
```

## Shell injection

The core SPA:

1. `GET /api/apps` → nav catalog
2. On route change → `import(moduleUrl)`
3. Calls `default.mount(el)` / `unmount()`

**Contract** (framework-agnostic):

```js
export default {
  async mount(el) { /* render into el */ },
  unmount(el) { /* cleanup */ },
}
```

Works with vanilla JS, Svelte, React, Vue, or static HTML (`el.innerHTML = …`).

## Creating an app without touching core

1. Add `apps/myapp` (or use `kaizengo new-app`)
2. Blank-import in `apps/apps.go`
3. Do **not** edit `apps/core` for GraphQL or nav — register on `host` in your `Setup`

## Enable / disable

```bash
KaizenGo_APPS=core,myapp ./bin/server
```
