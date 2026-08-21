# Go SDK (`packages/sdk-go`)

Public Go libraries for spec-driven apps. Prefer these over reaching into `internal/` from app code (except `internal/module` for `Register` / `Host`).

## Package map

| Path | Purpose |
|------|---------|
| `engine` | `New` / `Options` — load `app.yaml`, wire CRUD, GQL, hooks |
| `appspec` | Load and validate YAML; capability checks |
| `app` | Locales, nav helpers, installed-apps store, Postgres helpers |
| `events` | Event / store interfaces |
| `events/pgstore` | Postgres event store |
| `projection` | Projection runner + read-model sinks |
| `extension` | Global lifecycle handlers + yaml `extends` / `exports` |
| `gql` | Principal + RBAC helpers for resolvers |
| `i18n` | `T` / `Error` over loaded `.po` catalogs |
| `views` | Menu / view helpers used by the engine |
| `codegen` | Type and `.pot` generation (`godino`) |

Import path prefix: `kaizengo/packages/sdk-go/…`.

## Engine: one-liner app

```go
package hellospec

import (
	"kaizengo/internal/module"
	"kaizengo/packages/sdk-go/engine"
)

func init() {
	app := engine.New(engine.Options{
		AppName: "hellospec",
		Version: "0.3.0",
	})
	module.Register(app.Hooks("greeting", greetingHooks()))
}
```

`Options` highlights:

| Field | Role |
|-------|------|
| `AppName` | Directory under `apps/` and default route |
| `Version` | Manifest version |
| `Hooks` | Per-model lifecycle (usually via `.Hooks(...)`) |
| `Setup` | Extra Go after engine wiring (seed, custom services) |
| `Mount` | HTTP routes after all apps finished Setup |

Hybrid example pattern (auth): `engine.New` plus `Setup` / `Mount` for `/auth/*`.

## Lifecycle hooks

Hooks sit on the model name from `app.yaml`. `Before*` can mutate `Fields` or abort; `After*` see the projected `Record`.

```go
type HookContext struct {
	Context  context.Context
	App      appspec.AppSpec
	Model    appspec.ModelSpec
	OrgID    string
	UserID   string
	RecordID string
	Fields   map[string]any
	Record   Record
}

type Hooks struct {
	BeforeCreate, AfterCreate func(HookContext) error
	BeforeUpdate, AfterUpdate func(HookContext) error
	BeforeDelete, AfterDelete func(HookContext) error
}
```

From `apps/hellospec/hooks.go`:

```go
func beforeCreateGreeting(hc engine.HookContext) error {
	for _, fn := range []func(engine.HookContext) error{
		trimGreetingMessage,
		ensureGreetingPrefix,
		rejectProfanity,
	} {
		if err := fn(hc); err != nil {
			return err
		}
	}
	return nil
}

func greetingHooks() engine.Hooks {
	return engine.Hooks{
		BeforeCreate: beforeCreateGreeting,
		AfterCreate:  logGreetingCreated,
		BeforeUpdate: beforeUpdateGreeting,
		BeforeDelete: protectPinnedGreetings,
		AfterDelete:  logGreetingDeleted,
	}
}
```

Translated errors (same catalogs as the SPA):

```go
return hc.T("inventory.error.qty_positive")
// or: i18n.Error("inventory.error.qty_positive")
```

Pipeline order: app-local hooks, then global `extension.Run` on `model.<app>.<model>.<phase>`.

## Events and projections

Event store interfaces live in `packages/sdk-go/events`:

```go
type Store interface {
	Appender
	Loader
}

type Appender interface {
	Append(ctx context.Context, streamID, streamType string,
		expectedVersion int64, events ...NewEvent) ([]Event, error)
}
```

`engine.SetupEvents` uses the **shared** Postgres pool (`postgres.FromHost`) and schema `KaizenGo_<APP>_SCHEMA` (default = app name). Apps never open a private pool.

Read models are ordinary SQL tables (`{model}s_read`) applied from `apps/<name>/migrations/*.sql`. The engine does not invent DDL from YAML.

Internal-only models (`internal: true` in the spec) skip public mutations; Go writes use:

```go
_, err := registry.Create(engine.WithInternal(ctx), orgID, userID, "cost_layer", fields)
```

## Extension points

Addon apps attach without editing product apps:

```go
import "kaizengo/packages/sdk-go/extension"

extension.Register("model.*.*.afterCreate", 100, func(ctx extension.Context) error {
	slog.Info("audit", "point", ctx.Point, "recordId", ctx.RecordID)
	return nil
})
```

Or declare named handlers and wire them from YAML:

```yaml
extends:
  - point: model.*.*.afterCreate
    handler: indexDocument
    priority: 50
```

```go
extension.RegisterNamed("indexDocument", indexDocument)
// in Setup:
extension.SetupAddon(spec)
```

Wildcard handlers require `extensions: true` on the target app. See [extension-platform.md](../extension-platform.md).

## GraphQL guards

Resolvers that need a session and RBAC:

```go
import "kaizengo/packages/sdk-go/gql"

pr, err := gql.RequireAction(host, "permissions", p, "identity.users", "read")
```

`RequirePrincipal` only checks the session. Engine-registered CRUD already applies permissions via the app `resource` (default = app name).

Custom fields still register on the host:

```go
host.GQL.RegisterQuery("myField", &graphql.Field{
	Type: graphql.NewNonNull(graphql.String),
	Resolve: func(p graphql.ResolveParams) (any, error) {
		return "hi", nil
	},
})
```

## Generated types

```bash
make generate
# or: go run ./cmd/godino gen-types hellospec
```

Produces `apps/<app>/__types__/<model>.go` (system fields + declared fields) and refreshes `locale/template.pot`. Put `//go:generate` in `module.go` as hellospec does.

## Escape hatch

When YAML is not enough (aggregates, custom HTTP, non-CRUD APIs), keep `app.yaml` for nav/capabilities and implement `Setup` / `Mount` yourself. Use `events`, `pgstore`, `projection`, and `gql` directly — same building blocks the engine uses. See `apps/auth` and [SDK architecture → Custom apps](../sdk.md#custom-apps-escape-hatch).
