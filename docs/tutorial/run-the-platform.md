# Run the platform

You need a running host before you can see an app in the shell.

## Install dependencies

From the repository root:

```bash
cd apps/core/spa && npm install && cd -
```

## Start Postgres

Event-sourced apps (including the one you will build) store events and read models in Postgres:

```bash
make db-up
```

That starts Postgres on port **6432** and copies `.env.example` to `.env` if needed. `make dev` loads `KaizenGo_POSTGRES_DSN` from `.env`.

## Dev server

```bash
make dev
```

| URL | Role |
|-----|------|
| http://localhost:5173/app/ | Shell with hot reload (use this while developing) |
| http://localhost:8080 | API, GraphQL, production SPA |

Sign in with:

- Email: `admin@kaizengo.local`
- Password: `changeme`

Open **Apps** in the shell and click around. **HelloSpec** is the reference spec-driven app; **Identity** is a larger one with several views.

!!! tip "Default password"
    The server logs a warning when the default admin password is used. Change it in production with `KaizenGo_ADMIN_PASSWORD`.

## What just started

```text
Browser  →  core SPA (login, apps menu, view host)
                │
                ├─ GET /auth/me          auth app
                ├─ GET /api/apps         nav catalog
                └─ POST /graphql         engine CRUD + app fields
                       │
Postgres ←─────────────┴── per-app schema (events + *_read tables)
```

Apps register in `init()` via a blank import in `apps/apps.go`. `module.Load` runs **Setup** (services, GraphQL, nav) then **Mount** (HTTP routes). You will tap that same lifecycle when you add Todo.

## Useful commands

| Command | Use |
|---------|-----|
| `make dev` | Go API + Vite HMR |
| `make generate` | Regenerate `apps/*/__types__` and `locale/template.pot` |
| `make spa-build` | Production shell bundle |
| `make db-logs` | Follow Postgres |

Go changes (new `module.go`) need a restart of `make dev`. Svelte files under `apps/*/views/` hot-reload.

When the shell loads and you can open HelloSpec, continue to [app anatomy](app-anatomy.md).
