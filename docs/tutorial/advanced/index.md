# Advanced

The first half of this tutorial stays on the spec: YAML models, generated GraphQL, `KTable` / `KForm`, and a small hook. That is enough for ordinary CRUD.

This section is for the rest: **read and write models from Go**, wrap that in a **service** with extra GraphQL, and ship **Solid pages** that are not a table or a form.

You will extend the Todo app with:

1. A `todo` service that uses the model registry (open-task count, complete-all)
2. An Overview page that loads records itself and calls those operations

Work through the pages in order. Keep `make dev` running; restart it when Go files change. Solid under `apps/todo/views/` hot-reloads.

## When to leave the spec

| Need | Stay on YAML | Write Go / custom Solid |
|------|----------------|--------------------------|
| Extra field, enum, validation | Edit `app.yaml` + a migration | — |
| Normalize or reject a write | [Hooks](../hooks.md) | — |
| Seed rows, batch update, look up another app | — | `events.Models` in `Setup` |
| Login, cookies, side tables | — | Service + `events.Pool` (`apps/auth`) |
| Dashboard, tree, custom layout | — | Page view in `views/` |

Do not open a private database pool. Do not `UPDATE` `*_read` tables by hand — that skips hooks, events, and search.

## Path

1. [ORM and services](orm-and-services.md) — `ModelRegistry`, `Options.Setup`, GraphQL
2. [Custom pages](custom-pages.md) — Solid views beyond `KTable` / `KForm`

Reference apps in this repo: `apps/identity` (seed + custom views), `apps/auth` (service, HTTP, GraphQL).
