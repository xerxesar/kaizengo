# kaizengo docs

Guides for running the project and building apps.

The searchable site is built with [Zensical](https://zensical.org/docs/):

```bash
pip install -r requirements-docs.txt
make docs          # http://localhost:8000
```

Fonts and Mermaid are **vendored under `docs/`** (no Google Fonts / unpkg at runtime). See [assets/README.md](assets/README.md). Mermaid is injected early via [`overrides/main.html`](../overrides/main.html).

Start with **Tutorial → Develop a Kaizen app**.

| Section | Contents |
|---------|----------|
| [Development](development/index.md) | Workflow, platform APIs, auth, ACL, GraphQL, apps, CLI |
| [Reference](internals/index.md) | Internals, [Go SDK](internals/go-sdk.md), [Solid SDK](internals/solid-sdk.md) |
| [Installation](installation.md) | Prerequisites, clone, install tooling |
| [Auth & identity](auth.md) | Sessions, identity app, permissions service |
| [ACL system](acl.md) | Resource ids, policies, `security.yaml`, enforcement |
| [Apps system](apps.md) | Architecture, lifecycle, registering apps |
| [Capabilities](capabilities.md) | `provides` / `uses` contracts and identity SDK |
| [Extension platform plan](extension-platform.md) | Capabilities, global hooks, SDK components — roadmap |
| [CLI](cli.md) | `kaizengo new-app` bootstrapper |
| [GraphQL](graphql.md) | Runtime field registry, auth-aware clients |

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
