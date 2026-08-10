<p align="center">
    <img src="static/icon.png" alt="KaizenGo logo" width="160" />
    <h1 align="center">KaizenGo</h1>
</p>


<p align="center">
  <strong>Enterprise Operations Platform</strong> — modular Go host with an apps registry, SPA shell, and GraphQL field plugins.
</p>

<p align="center">
  <a href="whitepaper_v1.md">Whitepaper</a> ·
  <a href="docs/README.md">Documentation</a>
</p>

---

KaizenGo is a platform for building integrated operational software: a single host that registers apps, serves a shared Svelte shell, and composes GraphQL from plugins. The architecture favors **continuous improvement** — add apps without forking the core, wire services through a registry, and ship UI as lazy-loaded modules.

## Features

| Area | Stack |
|------|--------|
| HTTP routing | [chi](https://github.com/go-chi/chi) |
| App lifecycle | Registry, manifests, dependency injection |
| Frontend shell | Svelte 5 SPA with dynamic `import()` |
| API | Composable GraphQL field registry |
| Tooling | `kaizengo` CLI for scaffolding new apps |

Built-in apps include **Clock**, **Counter**, **Settings**, and a **Notes** example. Industry products (e.g. KMiner for mining operations) can be built on the same foundation.


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
Or Run with hot-reload during development:

```bash
cd apps/core/spa && npm install && cd -
make dev   # Go :8080 + Vite :5173/app/
```

Open **http://localhost:5173/app/** when using `make dev` (Vite base path is `/app/`).


## Documentation

| Guide | Description |
|-------|-------------|
| [Docs index](docs/README.md) | Overview and quick links |
| [Installation](docs/installation.md) | Prerequisites, clone, build |
| [Development](docs/development.md) | `make` targets, ports, workflow |
| [Apps system](docs/apps.md) | Architecture and app lifecycle |
| [Platform](docs/platform.md) | Kernel APIs — time, i18n, config |
| [CLI](docs/cli.md) | `kaizengo new-app` bootstrapper |
| [Svelte apps](docs/svelte.md) | Single-component ESM apps in the shell |
| [GraphQL](docs/graphql.md) | Runtime field registry and queries |
| [Whitepaper (v1)](whitepaper_v1.md) | Vision, problem space, and platform design |

## Create a new app

```bash
./bin/kaizengo new-app notes --type svelte --with-graphql
cd apps/notes/spa && npm install && npm run build
```

Details: [CLI](docs/cli.md) · [Apps system](docs/apps.md) · [Svelte apps](docs/svelte.md)

## Project layout

```text
apps/           # Registered apps (core shell, clock, counter, settings, …)
cmd/            # server and kaizengo CLI entrypoints
docs/           # Developer guides
internal/       # Platform kernel (module host, config, i18n, GraphQL)
static/         # Shared assets (favicon, CSS)
whitepaper_v1.md
```

## Learn more

The [whitepaper](whitepaper_v1.md) describes KaizenGo as an **Enterprise Operations Platform (EOP)**: connecting processes, people, data, and continuous improvement rather than replacing ERP point-by-point. The codebase here is the reference implementation of that modular host.
