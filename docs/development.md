# Development

## Ports

| Process | URL | Role |
|---------|-----|------|
| Go server | http://localhost:8080 | API, GraphQL, static app assets, production shell |
| Vite (dev) | http://localhost:5173/app/ | Core shell HMR; proxies `/graphql`, `/api`, `/app-assets` |

## Common commands

```bash
make cli          # build bin/kaizengo
make generate     # spa codegen
make spa-build    # build core + counter SPAs
make build        # generate + spa-build + go build
make run          # build assets + go run server
make spa-dev      # Vite only
make dev          # Go server + Vite together
make tidy         # go mod tidy
```

## Recommended loop

```bash
make cli
make dev
```

Then open **http://localhost:5173/app/** (not bare `/` — Vite `base` is `/app/`).

Keep the Go server running so `/graphql` and `/app-assets/*` resolve via the Vite proxy.

## Environment

| Variable | Meaning |
|----------|---------|
| `ADDR` | Listen address (default `:8080`) |
| `KaizenGo_APPS` | Comma-separated app names to load (default: all installable) |

## Project layout (high level)

```text
cmd/server     HTTP process
cmd/kaizengo     CLI (new-app, …)
apps/          loadable apps (core, clock, counter, …)
internal/module  registry, Host, GraphQL registry, nav
docs/          these guides
```

## Adding a Svelte app to `make spa-build`

The Makefile currently builds core + counter explicitly. After `kaizengo new-app foo --type svelte`, either:

```bash
cd apps/foo/spa && npm install && npm run build
```

or append a similar line to the `spa-build` target in the Makefile.

Vanilla apps need no build step — they serve `spa/spa.js` directly.
