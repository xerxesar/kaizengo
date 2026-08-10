# kaizengo docs

Guides for running the project and building apps.

| Guide | Contents |
|-------|----------|
| [Installation](installation.md) | Prerequisites, clone, install tooling |
| [Development](development.md) | `make` targets, ports, env vars |
| [Apps system](apps.md) | Architecture, lifecycle, registering apps |
| [Platform](platform.md) | Kernel APIs (time, i18n), drivers, fork-vs-patch |
| [CLI](cli.md) | `kaizengo new-app` bootstrapper |
| [Svelte apps](svelte.md) | Single-component ESM apps in the core shell |
| [GraphQL](graphql.md) | Runtime field registry, queries from apps |

## Quick start

```bash
go install github.com/a-h/templ/cmd/templ@latest   # optional historical tool; not required for current SPA shell
cd apps/core/spa && npm install && cd -
make build
./bin/server
# open http://localhost:8080/app/
```

Or for hot UI reload:

```bash
make cli          # builds bin/kaizengo
make dev          # Go :8080 + Vite :5173/app/
```
