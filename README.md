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

KaizenGo is a platform for building integrated operational software: a single host that registers apps, serves a shared Solid shell with login, and composes GraphQL from plugins. The architecture favors **continuous improvement** — add apps without forking the core, wire services through a registry, localize with `.po` catalogs, and ship UI as lazy-loaded modules on `@kaizengo/sdk-solid`.

## Features

| Area | Stack |
|------|--------|
| HTTP routing | [chi](https://github.com/go-chi/chi) |
| App lifecycle | Registry, manifests, dependency injection |
| Auth | Session cookies (`kg_session`), identity + auth apps, RBAC permissions |
| Frontend shell | SolidJS SPA with dynamic `import()` and login gate |
| Solid SDK | `@kaizengo/sdk-solid` — ui, identity/search clients, spa-config |
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

Open **http://localhost:5173/app/** (Vite `base` is `/app/`). Sign in with `admin@kaizengo.local` / `changeme`. Use `make spa-build` once to browse every module without watchers. See [Development](docs/development.md) and [Auth](docs/auth.md).


## Documentation

| Guide | Description |
|-------|-------------|
| [Docs index](docs/index.md) | Overview and tutorial |
| [Tutorial](docs/tutorial/index.md) | Build a spec-driven Kaizen app |
| [Installation](docs/installation.md) | Prerequisites, clone, build |
| [Development](docs/development.md) | `make` targets, ports, workflow |
| [Auth & identity](docs/auth.md) | Sessions, identity app, permissions |
| [Apps system](docs/apps.md) | Architecture and app lifecycle |
| [Platform](docs/platform.md) | Kernel APIs — time, i18n, config |
| [CLI](docs/cli.md) | `kaizengo new-app` bootstrapper |
| [Solid apps](docs/solid.md) | ESM modules and `@kaizengo/sdk-solid/ui` |
| [GraphQL](docs/graphql.md) | Runtime field registry and queries |
| [Whitepaper (v1)](whitepaper_v1.md) | Vision, problem space, and platform design |

## Create a new app

```bash
./bin/kaizengo new-app notes --type solid --with-graphql
cd apps/notes/spa && npm install && npm run build
```

Details: [CLI](docs/cli.md) · [Apps system](docs/apps.md) · [Solid apps](docs/solid.md)

## Project layout

```text
apps/           # Registered apps (core shell, identity, clock, …)
packages/
  sdk-go/       # Go app SDK (engine, appspec, extension, events)
  sdk-solid/   # Solid SDK (@kaizengo/sdk-solid) — ui, search, spa-config
cmd/            # server and kaizengo CLI entrypoints
docs/           # Developer guides
internal/       # Platform kernel (module host, auth, config, i18n, GraphQL)
static/         # Shared assets (favicon, CSS)
whitepaper_v1.md
```

## Learn more

The [whitepaper](whitepaper_v1.md) describes KaizenGo as an **Enterprise Operations Platform (EOP)**: connecting processes, people, data, and continuous improvement rather than replacing ERP point-by-point. The codebase here is the reference implementation of that modular host.
