<p align="center">
    <img src="static/icon.png" alt="KaizenGo logo" width="160" />
    <h1 align="center">KaizenGo</h1>
</p>


<p align="center">
  <strong>Enterprise Operations Platform</strong> — modular Go host with an apps registry, SPA shell, and GraphQL field plugins.
</p>

<p align="center">
  <a href="whitepaper_v1.md">Whitepaper</a> ·
  <a href="https://xerxesar.github.io/kaizengo/">Documentation</a>
</p>

---

KaizenGo is a platform for building integrated operational software: a single host that registers apps, serves a shared React shell with login, and composes GraphQL from a spec. The architecture favors **continuous improvement** — add apps without forking the core, wire services through a registry, and localize with `.po` catalogs. See [Architecture](docs/architecture.md).

## Features

| Area | Stack |
|------|--------|
| HTTP routing | [chi](https://github.com/go-chi/chi) |
| App lifecycle | Registry, manifests, dependency injection |
| Auth | Session cookies (`kg_session`), identity + auth apps, RBAC permissions |
| Frontend shell | React SPA in `apps/core/spa` (`@/k`, `@/lib`, `@/components/ui`) |
| Localization | gettext `.po` catalogs, GraphQL `i18n`, RTL (`fa`) |
| API | Composable GraphQL field registry (auth-protected) |
| Tooling | `kaizengo` CLI for scaffolding new apps; VS Code extension for `app.yaml` navigation |

Built-in apps include **Identity**, **Clock**, **Counter**, **Settings**, **Status**, **Oracle**, and a **Notes** example, plus a backend **Permissions** service. Industry products (e.g. KMiner for mining operations) can be built on the same foundation.


## Requirements

- **Go** 1.22+ (module targets 1.26)
- **Node.js** 20+ and **npm** (core SPA and app builds)

See [Installation](docs/installation.md) for full setup.

## Quick start

Build and serve:
```bash
cd apps/core/spa && npm install && cd -
make cli build
./bin/server
# http://localhost:8080/app/
```
Or run with hot-reload during development:

```bash
cd apps/core/spa && npm install && cd -
make dev APP=clock   # Go :8080 + Vite :5173/app/ + watch one app
```

Open **http://localhost:5173/app/** (Vite `base` is `/app/`). Sign in with `admin@kaizengo.local` / `changeme`. Use `make spa-build` once to browse every module without watchers. See [Development](docs/development/index.md) and [Auth](docs/auth.md).


## Documentation

| Guide | Description |
|-------|-------------|
| [Docs index](docs/index.md) | Overview and site map |
| [Architecture](docs/architecture.md) | 3+1 layers, GraphQL vs gRPC, CLI |
| [Tutorial](docs/tutorial/index.md) | Build a spec-driven Kaizen app |
| [Development](docs/development/index.md) | Install, workflow, platform APIs |
| [Platform](docs/platform.md) | Apps, auth, ACL, GraphQL, CLI |
| [Reference](docs/internals/index.md) | Internals, Go SDK, React SPA |
| [Whitepaper (v1)](whitepaper_v1.md) | Vision, problem space, and platform design |

## Create a new app

```bash
./bin/kaizengo new-app notes
make spa-build
```

Details: [CLI](docs/cli.md) · [Apps system](docs/apps.md) · [React SPA](docs/internals/solid-sdk.md)

## Project layout

```text
apps/           # Registered apps (core shell, identity, clock, …)
packages/
  sdk-go/       # Go app SDK contracts (appspec, acl, i18n, views, plugin)
cmd/            # server, kaizen runtime CLI, kaizengo scaffolding CLI
docs/           # Developer guides
internal/       # Platform kernel (module host, auth, config, i18n, GraphQL)
static/         # Shared assets (favicon, CSS)
whitepaper_v1.md
```

## Learn more

The [whitepaper](whitepaper_v1.md) describes KaizenGo as an **Enterprise Operations Platform (EOP)**: connecting processes, people, data, and continuous improvement rather than replacing ERP point-by-point. The codebase here is the reference implementation of that modular host.
