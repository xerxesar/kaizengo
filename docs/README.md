# kaizengo docs

Guides for running the project and building apps. The site home is [index.md](index.md).

The searchable site is built with [Zensical](https://zensical.org/docs/):

```bash
pip install -r requirements-docs.txt
make docs          # http://localhost:8000
```

Fonts and Mermaid are **vendored under `docs/`** (no Google Fonts / unpkg at runtime). See [assets/README.md](assets/README.md). Mermaid is injected early via [`overrides/main.html`](../overrides/main.html).

Start with **Tutorial → Develop a Kaizen app**.

| Section | Contents |
|---------|----------|
| [Architecture](architecture.md) | 3+1 layers, GraphQL vs gRPC, CLI, dynamic domain |
| [Tutorial](tutorial/index.md) | Run the platform, app anatomy, Todo app, hooks, advanced pages |
| [Development](development/index.md) | [Installation](installation.md), [workflow](development/workflow.md), [platform APIs](development/platform.md) |
| [Platform](platform.md) | [Apps](apps.md), [auth](auth.md), [ACL](acl.md), [capabilities](capabilities.md), [GraphQL](graphql.md), [CLI](cli.md), [external Go apps](grpc-plugins.md) |
| [Reference](internals/index.md) | Internals, [Go SDK](internals/go-sdk.md), [React SPA](internals/solid-sdk.md) |
| [Extension plan](extension-platform.md) | Capabilities, global hooks, shared UI — roadmap |

## Quick start

```bash
cd apps/core/spa && npm install && cd -
make build
./bin/server
# open http://localhost:8080/app/ — sign in with admin@kaizengo.local / changeme
```

Or for hot UI reload:

```bash
make cli            # builds bin/kaizengo
make spa-build      # once: build all app modules
make dev APP=clock  # Go :8080 + Vite :5173/app/ + watch one app
```
