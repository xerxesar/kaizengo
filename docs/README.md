# kaizengo docs

Guides for running the project and building apps.

The searchable site is built with [Zensical](https://zensical.org/docs/):

```bash
pip install -r requirements-docs.txt
make docs          # http://localhost:8000
```

Start with **Tutorial → Develop a Kaizen app**.

| Guide | Contents |
|-------|----------|
| [Internals & SDKs](internals/index.md) | Boot flow, Go/Solid SDKs with code examples |
| [Installation](installation.md) | Prerequisites, clone, install tooling |
| [Development](development.md) | `make` targets, ports, env vars |
| [Auth & identity](auth.md) | Sessions, identity app, permissions service |
| [ACL system](acl.md) | Resource ids, policies, `security.yaml`, enforcement |
| [Apps system](apps.md) | Architecture, lifecycle, registering apps |
| [SDK architecture](sdk.md) | Odoo-like app pattern on sdk/internal/apps |
| [Capabilities](capabilities.md) | `provides` / `uses` contracts and identity SDK |
| [Extension platform plan](extension-platform.md) | Capabilities, global hooks, SDK components — roadmap |
| [Platform](platform.md) | Kernel APIs (time, i18n), drivers, fork-vs-patch |
| [CLI](cli.md) | `kaizengo new-app` bootstrapper |
| [Solid apps](solid.md) | ESM modules, `@kaizengo/sdk-solid/ui`, theming |
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
