# Internals & SDKs

How KaizenGo boots, how apps plug in, and which SDK packages you call from Go and Svelte.

For shipping an app step-by-step, use the [tutorial](../tutorial/index.md). For exhaustive field/hook catalogs, see [SDK architecture](../sdk.md).

## Layer map

```text
cmd/server          → process entry, Host, Load, HTTP listen
internal/module     → App interface, registry, Host bag, GraphQL registry
internal/platform   → i18n, time, config, postgres, search
internal/auth       → session middleware / principal
packages/sdk-go     → engine, appspec, events, extension, gql guards
packages/sdk-svelte → UI kit, model client, identity/search clients
apps/<name>         → app.yaml + module.go + views + migrations
```

| Layer | Owns | Must not own |
|-------|------|--------------|
| `internal/` | Kernel, lifecycle, middleware | Per-app business rules |
| `packages/sdk-go` | Spec → CRUD/GQL/events | App-specific schemas |
| `packages/sdk-svelte` | Shared UI + GraphQL clients | Product screens |
| `apps/` | Spec, hooks, views, migrations | Platform kernel forks |

## Boot sequence

`cmd/server/main.go` wires the host, then loads apps:

1. Blank-import `kaizengo/apps` so every bundled app’s `init()` runs `module.Register`
2. Create `module.Host` (router + service bag + GraphQL registry)
3. Connect Postgres and attach it to the host
4. Install session middleware (looks up `auth` service at request time)
5. Resolve **wanted** apps (`KaizenGo_APPS` or installed + `autoInstall`)
6. `module.Load` — capability check → Setup (dep order) → Mount (dep order)

```go
host := module.NewHost(r, slog.Default())
db, err := postgres.Connect(context.Background(), config.PostgresDSN())
postgres.Attach(host, db)

mgr := engine.NewManager(host, module.Default, store)
wanted, err := mgr.Wanted(context.Background(), module.ParseAppList(os.Getenv("KaizenGo_APPS")))
if err := module.Load(host, module.Default, wanted); err != nil {
	log.Fatal(err)
}
```

Apps register themselves via blank imports in `apps/apps.go`:

```go
import (
	_ "kaizengo/apps/core"
	_ "kaizengo/apps/hellospec"
	// …
)
```

### `module.App` contract

Every loadable package implements:

```go
type App interface {
	Manifest() Manifest
	Setup(host *Host) error
	Mount(host *Host) error
}
```

`Load` resolves dependency order, validates `provides` / `uses`, applies cross-app `exports` / `extends`, then runs Setup for all apps before any Mount:

```go
for _, a := range apps {
	if err := a.Setup(host); err != nil { /* … */ }
}
for _, a := range apps {
	if err := a.Mount(host); err != nil { /* … */ }
}
```

### Host bag

Apps share services through `Provide` / `Lookup` — they do not import each other’s packages for runtime wiring:

```go
host.Provide("my.service", svc)
raw, ok := host.Lookup("my.service")

host.GQL.RegisterQuery("myField", &graphql.Field{ /* … */ })
host.RegisterNav(module.NavEntry{ID: "demo", Route: "demo"})
```

## Spec → runtime (engine)

Most apps are one line of Go plus YAML. `engine.New` loads `apps/<name>/app.yaml` and wires locales, nav, event-sourced models, and GraphQL:

```go
// apps/hellospec/module.go
func init() {
	app := engine.New(engine.Options{
		AppName: "hellospec",
		Version: "0.3.0",
	})
	module.Register(app.Hooks("greeting", greetingHooks()))
}
```

During `Setup`, the engine:

1. Loads the appspec
2. Loads `.po` catalogs and registers shell nav
3. Opens the shared Postgres pool and event store (`SetupEvents`)
4. Registers CRUD GraphQL + `{app}Views` / `{app}Menus`
5. Provides the model registry on the host (`ModelsKey(appName)`)

```go
func (a *App) Setup(host *module.Host) error {
	spec, err := a.loadSpec()
	// …
	events, err := SetupEvents(host, a.opts.AppName, spec, a.opts.Hooks)
	host.Provide(ModelsKey(a.opts.AppName), events.Models)
	host.Provide(a.opts.AppName, a)
	return nil
}
```

Write path for a model mutation:

```text
normalize → app Before* hooks → extension Before*
→ validate → append event → project read model
→ app After* hooks → extension After*
```

## Frontend path

There is **one** shell SPA (`apps/core/spa`). App pages live under `apps/<name>/views/*.page.svelte` and are discovered at build time:

```ts
// apps/core/spa/src/lib/views/registry.ts
const appViewModules = import.meta.glob<ViewModule>(
  '../../../../../*/views/**/*.svelte',
  { eager: true },
)
// maps "hellospec.GreetingList" → component
```

Menus from GraphQL (`{app}Menus`) pick a leaf `view`; the shell resolves `{app}.{view}` and mounts it. Shared UI and GraphQL helpers come from `@kaizengo/sdk-svelte/*`.

## Package index

| Package | Import | Docs |
|---------|--------|------|
| Engine | `kaizengo/packages/sdk-go/engine` | [Go SDK](go-sdk.md) |
| Appspec | `kaizengo/packages/sdk-go/appspec` | [Go SDK](go-sdk.md) |
| Events / projection | `…/events`, `…/projection` | [Go SDK](go-sdk.md) |
| Extension points | `…/extension` | [Go SDK](go-sdk.md), [extension platform](../extension-platform.md) |
| GQL guards | `…/gql` | [Go SDK](go-sdk.md), [GraphQL](../graphql.md) |
| UI + model client | `@kaizengo/sdk-svelte/ui` | [Svelte SDK](svelte-sdk.md) |
| Identity client | `@kaizengo/sdk-svelte/identity` | [Svelte SDK](svelte-sdk.md) |
| Search client | `@kaizengo/sdk-svelte/search` | [Svelte SDK](svelte-sdk.md) |

## Worked example: hellospec

| File | Role |
|------|------|
| `apps/hellospec/app.yaml` | Model `greeting`, menus, search fields |
| `apps/hellospec/module.go` | `engine.New` + hooks registration |
| `apps/hellospec/hooks.go` | Trim, prefix, protect deletes |
| `apps/hellospec/views/GreetingList.page.svelte` | `<KTable model="hellospec.greeting" />` |
| `apps/hellospec/views/GreetingForm.page.svelte` | `<KForm>` + `<KFormField>` |
| `apps/hellospec/migrations/` | Event store + `greetings_read` |

List page (entire UI for the menu leaf):

```svelte
<script lang="ts">
  import { KTable, KAppStatus, t } from '@kaizengo/sdk-svelte/ui'
</script>

<KTable model="hellospec.greeting" emptyMessage={t('hellospec.empty')} />
<KAppStatus />
```

`KTable` / `KForm` call the engine GraphQL naming convention (`hellospecGreetings`, `createHellospecGreeting`, …) via `model-client.ts` — you rarely hand-write those queries.

## Next

- [Go SDK](go-sdk.md) — packages, hooks, events, extensions
- [Svelte SDK](svelte-sdk.md) — UI, clients, Vite aliases
- [Apps system](../apps.md) — install / load / Host details
- [Platform](../platform.md) — i18n, calendars, config
