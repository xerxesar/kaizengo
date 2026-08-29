# Development

## Ports

| Process | URL | Role |
|---------|-----|------|
| Go server | http://localhost:8080 | API, GraphQL, production SPA at `/app/` |
| Vite (dev) | http://localhost:5173/app/ | Central SPA HMR; proxies API |

## Standard workflow

```bash
cd apps/core/spa && npm install && cd -
make dev
```

Open **http://localhost:5173/app/** → sign in → open any app from the Apps menu.

| What you edit | What happens |
|---------------|--------------|
| `apps/core/spa/src/` (shell, login) | Vite HMR — instant |
| `apps/*/views/*.tsx` (app views) | Vite HMR — instant |
| `packages/sdk-solid/ui/` | Vite HMR — instant |
| Go code (`apps/*/module.go`, services) | Restart `make dev` (or run Go in a separate terminal) |

### Commands

```bash
make dev              # Go + central Vite SPA
make spa-build        # production SPA build
make run              # production build + Go (no HMR)
```

### First-time / switching modules

- **`make dev APP=clock`** builds and watches clock automatically.
- To **click around other apps** without watchers, run **`make spa-build`** once. Their prebuilt `dist/` bundles are served from Go.
- Default login: `admin@kaizengo.local` / `changeme`

### Split terminals (optional)

Same as `make dev APP=clock`, but easier to read logs:

```bash
# Terminal 1
make run

# Terminal 2
cd apps/core/spa && npm run dev

# Terminal 3
cd apps/clock/spa && npm run dev
```

## Debugging

| Layer | Where to look |
|-------|----------------|
| Shell UI | Browser DevTools on `:5173/app/` |
| Mounted app | DevTools Network → `spa.js?dev=…` should be 200 |
| GraphQL | http://localhost:8080/playground |
| Go API | Terminal running `go run ./cmd/server` |
| Auth | Network tab → `/auth/me`, `/auth/login` |

After editing a mounted app, wait for the terminal line `built in …ms` from that app's watcher before expecting UI changes.

## PostgreSQL (event-sourced apps)

Notes, Hello, and other event-sourced modules need PostgreSQL. Start a local instance with Docker:

```bash
make db-up          # starts Postgres and creates .env if missing
make dev APP=hello  # loads DSN from .env automatically
```

| Command | Action |
|---------|--------|
| `make db-up` | Start Postgres on `:6432` and create `.env` from `.env.example` |
| `make db-down` | Stop container |
| `make db-logs` | Follow Postgres logs |

Copy `.env.example` → `.env` if you prefer manual setup. `make dev` and `make run` load `.env` automatically.

If you set `KaizenGo_POSTGRES_DSN` manually in fish, use:

```fish
set -x KaizenGo_POSTGRES_DSN 'postgres://kaizengo:kaizengo@localhost:6432/kaizengo?sslmode=disable'
```

A bad or unreachable DSN prevents the server from starting. The process opens one shared pool (`internal/platform/postgres`) before loading apps; apps obtain it via `postgres.FromHost(host)` or `postgres.FromContext(ctx)`.

Data persists in the `kaizengo_pgdata` Docker volume. App schemas are created when an app is **installed** or **upgraded** (App Manager), and again on boot for already-installed apps (pending SQL only).

Docker maps Postgres to host port **6432** (not 5432) so it does not clash with a system PostgreSQL install.

## Environment

| Variable | Meaning |
|----------|---------|
| `ADDR` | Listen address (default `:8080`) |
| `KaizenGo_APPS` | Comma-separated app names to load (overrides the install table; default: installed + `autoInstall`) |
| `KaizenGo_POSTGRES_DSN` | Shared PostgreSQL DSN (connected once at server start; apps use `platform.postgres`) |
| `KaizenGo_IDENTITY_SCHEMA` | Identity schema (default `identity`) |
| `KaizenGo_AUTH_SCHEMA` | Auth schema (default `auth`) |
| `KaizenGo_PERMISSIONS_SCHEMA` | Permissions schema (default `permissions`) |
| `KaizenGo_NOTES_SCHEMA` | Notes schema in PostgreSQL (default `notes`) |
| `KaizenGo_MONGO_URI` | Optional Mongo URI for projection sinks |
| `KaizenGo_ADMIN_EMAIL` | Seed admin email (default `admin@kaizengo.local`) |
| `KaizenGo_ADMIN_PASSWORD` | Seed admin password (default `changeme`) |

Auth details: [auth.md](auth.md).

## Project layout

```text
cmd/server       HTTP process
cmd/kaizengo     CLI (new-app, …)
apps/            loadable apps (core, identity, clock, …)
packages/sdk-solid/ui  shared Solid UI SDK (@kaizengo/sdk-solid/ui)
packages/sdk-solid/spa-config  shared Vite config for app SPAs
packages/vscode-kaizengo  VS Code / Cursor extension (app.yaml navigation)
internal/auth    session middleware + Principal
internal/module  registry, Host, GraphQL, nav
internal/platform  time, i18n, config, drivers
```

## VS Code / Cursor extension

`packages/vscode-kaizengo` adds Go to Definition in `app.yaml` and `models/*/spec.yaml` (pages, models, `depends` / `uses`, locales, handlers).

```bash
make vscode-ext          # compile
make vscode-ext-test     # parse/resolve tests against this repo
```

Then run **Run KaizenGo Extension** from the debug view (F5). See `packages/vscode-kaizengo/README.md`.

## Adding a Svelte app

1. `./bin/kaizengo new-app foo --type svelte`
2. Append `apps/foo/spa` to `APP_SPAS` in the Makefile
3. Use shared config: `packages/sdk-solid/spa-config/app-vite.ts`
4. Dev: `make dev APP=foo`
