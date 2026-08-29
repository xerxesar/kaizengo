# CLI (`kaizengo`)

## Build

```bash
make cli
# → bin/kaizengo
```

## `kaizengo new-app`

Bootstraps `apps/<name>/`, writes UI stubs, and registers a blank import in `apps/apps.go`.

```bash
kaizengo new-app <name> [flags]
```

| Flag | Default | Description |
|------|---------|-------------|
| `--type` | `vanilla` | `vanilla` or `svelte` |
| `--title` | Title-cased name | Apps menu label |
| `--summary` | `"<title> app"` | Manifest summary |
| `--with-graphql` | off | Sample Query field `<pkg>Ping` + client call |
| `--event-sourced` | `true` | Scaffold PostgreSQL event-sourced module pattern |

### Examples

Vanilla module (no build step):

```bash
./bin/kaizengo new-app notes
```

Svelte single-component app:

```bash
./bin/kaizengo new-app notes --type solid
cd apps/notes/spa && npm install && npm run build
```

Svelte + GraphQL sample:

```bash
./bin/kaizengo new-app notes --type solid --with-graphql --title "Notes"
cd apps/notes/spa && npm install && npm run build
# Append apps/notes/spa to APP_SPAS in the Makefile
go run ./cmd/server
# Sign in → Apps → Notes, or http://localhost:8080/app/notes
```

### What gets created

**Vanilla**

```text
apps/<name>/
  module.go
  spa/spa.js          # served as /app-assets/<name>/spa.js
```

**Svelte**

```text
apps/<name>/
  module.go           # TitleKey: nav.<name> for i18n menu labels
  spa/
    App.tsx
    main.ts           # mount/unmount adapter
    vite.config.ts    # library build → dist/spa.js
    package.json
    …
```

### Recommended follow-ups for Solid apps

Scaffolded templates are a starting point. Event-sourced mode writes `app.yaml` plus an `packages/sdk-go/engine` one-liner module — GraphQL CRUD and projections come from the spec.

Align new apps with the rest of the monorepo:

1. Depend on `@kaizengo/sdk-solid/ui` (`file:../../../packages/sdk-solid`) and import `@kaizengo/sdk-solid/ui/styles.css`
2. Prefer `createAppViteConfig` from `packages/sdk-solid/spa-config/app-vite.ts` instead of a one-off Vite config
3. Edit `apps/<name>/app.yaml` models/views — avoid hand-written service/gql unless you need custom domain rules
4. Use `credentials: 'include'` on every `/graphql` fetch (required under session auth)
5. Append `apps/<name>/spa` to `APP_SPAS` in the Makefile
6. Use the `Layout` contract from [solid.md](solid.md) for page chrome
7. Set `KaizenGo_POSTGRES_DSN` (see `make db-up`); grant the app `resource` in permissions for non-admin roles

See [sdk.md](sdk.md), [auth.md](auth.md), and [development.md](development.md).

### Name rules

- Lowercase letter start: `^[a-z][a-z0-9_]*$`
- Reserved: `core`, `apps`

## `kaizengo version`

Prints the CLI version.
