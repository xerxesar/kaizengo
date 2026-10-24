# KaizenGo

KaizenGo is a modular host for operational software: apps register themselves, share a React shell with login, and compose GraphQL from a spec. Domains can also be defined at runtime from the CLI or by external Go apps — see [Architecture](architecture.md).

This site is the developer documentation. Start with the tutorial if you want to ship an app.

[Get started: develop a Kaizen app](tutorial/index.md){ .md-button .md-button--primary }

## How apps fit together

```text
apps/<name>/
  app.yaml       # models, menus, locales
  module.go      # engine.New(...) — one-liner for spec apps
  models/        # optional per-model spec.yaml + hooks
  views/         # <Name>.page.tsx in the core shell
  migrations/    # Postgres tables for the app's models
```

The engine turns `app.yaml` into model CRUD, GraphQL, nav, and [ACL checks](acl.md). You write YAML first; Go and React only where the spec is not enough ([advanced tutorial](tutorial/advanced/index.md)).

## Docs map

| Section | Purpose |
|---------|---------|
| [Architecture](architecture.md) | 3+1 layers, GraphQL vs gRPC, CLI, dynamic domain |
| [Tutorial](tutorial/index.md) | Build a first app on the spec path |
| [Development](development/index.md) | Install the repo, `make dev`, platform APIs (time, i18n, settings) |
| [Platform](platform.md) | Apps, auth, ACL, capabilities, GraphQL, CLI, external Go apps |
| [Reference](internals/index.md) | Internals, Go SDK, React SPA |
| [Extension plan](extension-platform.md) | Roadmap for cross-app addons |

## Preview this site

```bash
python3 -m venv .venv
source .venv/bin/activate
pip install -r requirements-docs.txt
make docs
```

Then open [http://localhost:8000](http://localhost:8000). Pushes to `main` publish the site to [GitHub Pages](https://xerxesar.github.io/kaizengo/).
