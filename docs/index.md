# KaizenGo

KaizenGo is a modular host for operational software: apps register themselves, share a Svelte shell with login, and compose GraphQL from a spec.

This site is the developer documentation. Start with the tutorial if you want to ship an app.

[Get started: develop a Kaizen app](tutorial/index.md){ .md-button .md-button--primary }

## How apps fit together

```text
apps/<name>/
  app.yaml       # models, menus, locales
  module.go      # engine.New(...) — one-liner for spec apps
  models/        # optional per-model spec.yaml + hooks
  views/         # <Name>.page.svelte in the core shell
  migrations/    # Postgres event store + read models
```

The engine turns `app.yaml` into event-sourced CRUD, GraphQL, nav, and permissions checks. You write YAML first; Go and Svelte only where the spec is not enough ([advanced tutorial](tutorial/advanced/index.md)).

## Preview this site

```bash
python3 -m venv .venv
source .venv/bin/activate
pip install -r requirements-docs.txt
make docs
```

Then open [http://localhost:8000](http://localhost:8000). Pushes to `main` publish the site to [GitHub Pages](https://xerxesar.github.io/kaizengo/).
