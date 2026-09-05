# KaizenGo

KaizenGo is a modular host for operational software: apps register themselves, share a Solid shell with login, and compose GraphQL from a spec.

This site is the developer documentation. Start with the tutorial if you want to ship an app.

[Get started: develop a Kaizen app](tutorial/index.md){ .md-button .md-button--primary }

## How apps fit together

```text
apps/<name>/
  app.yaml       # models, menus, locales
  module.go      # engine.New(...) — one-liner for spec apps
  models/        # optional per-model spec.yaml + hooks
  views/         # <Name>.page.tsx in the core shell
  migrations/    # Postgres event store + read models
```

The engine turns `app.yaml` into event-sourced CRUD, GraphQL, nav, and [ACL checks](acl.md). You write YAML first; Go and Solid only where the spec is not enough ([advanced tutorial](tutorial/advanced/index.md)).

## Docs map

| Section | Purpose |
|---------|---------|
| [Tutorial](tutorial/index.md) | Build your first app step by step |
| [Development](development/index.md) | Workflow, platform APIs, auth, ACL, GraphQL, apps |
| [Reference](internals/index.md) | Internals, Go SDK, Solid SDK |

## Preview this site

```bash
python3 -m venv .venv
source .venv/bin/activate
pip install -r requirements-docs.txt
make docs
```

Then open [http://localhost:8000](http://localhost:8000). Pushes to `main` publish the site to [GitHub Pages](https://xerxesar.github.io/kaizengo/).
