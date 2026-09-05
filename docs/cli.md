# CLI (`kaizengo`)

## Build

```bash
make cli
# → bin/kaizengo
```

## `kaizengo new-app`

Bootstraps `apps/<name>/`, writes Solid view stubs, and registers a blank import in `apps/apps.go`.

```bash
kaizengo new-app <name> [flags]
```

| Flag | Default | Description |
|------|---------|-------------|
| `--type` | `solid` | UI type (`solid` only) |
| `--title` | Title-cased name | Apps menu label |
| `--summary` | `"<title> app"` | Manifest summary |
| `--with-graphql` | off | Sample Query field `<pkg>Ping` + client call |
| `--event-sourced` | `true` | Scaffold PostgreSQL event-sourced module pattern |
| `--addon` | off | Cross-app addon (extension points, no views) |

### Examples

Solid event-sourced app (default):

```bash
./bin/kaizengo new-app notes --title Notes
make spa-build
```

With a sample GraphQL client helper:

```bash
./bin/kaizengo new-app notes --type solid --with-graphql --title Notes
make spa-build
go run ./cmd/server
# Sign in → Apps → Notes, or http://localhost:8080/app/notes
```

### What gets created

**Solid (event-sourced, default)**

```text
apps/<name>/
  app.yaml
  module.go              # engine.New one-liner
  hooks.go
  migrations/
  locale/
  views/*.page.tsx       # list/form pages for the core shell
```

Views compile into the central SPA (`apps/core/spa`); there is no per-app Vite bundle.

### Recommended follow-ups

Scaffolded templates are a starting point. Event-sourced mode writes `app.yaml` plus an `internal/engine` one-liner — GraphQL CRUD and projections come from the spec.

1. Import from `@kaizengo/sdk-solid/ui` in views (already aliased in the core SPA)
2. Edit `apps/<name>/app.yaml` models/views — avoid hand-written services unless you need custom domain rules
3. Use `credentials: 'include'` on every `/graphql` fetch (session auth)
4. Set `KaizenGo_POSTGRES_DSN` (see `make db-up`); grant the app `resource` in permissions for non-admin roles

See [Go SDK](internals/go-sdk.md), [Solid SDK](internals/solid-sdk.md), [auth.md](auth.md), and [Workflow](development/workflow.md).

### Name rules

- Lowercase letter start: `^[a-z][a-z0-9_]*$`
- Reserved: `core`, `apps`

## `kaizengo version`

Prints the CLI version.
