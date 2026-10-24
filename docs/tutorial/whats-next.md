# What’s next

You have a spec-driven app in the shell. If you have not yet added a service or a custom page, that is [the advanced section](advanced/index.md).

These are the usual next stops after that.

## Stay on the spec path

- **More fields** — edit `app.yaml`, add a SQL column on the read-model table, run `make generate`.
- **Enums** — `type: enum` plus `values:`. Generated Go gets a typed string and `Valid()`.
- **Search** — under the model, set `search.collection` and `search.fields` (see `hellospec`).
- **i18n** — add `locale/fa.po` (and a `locales:` entry) for RTL; `t('todo.empty')` in React and `i18n.T("todo.empty")` in Go share the same catalog.

## Platform apps you should not reimplement

| Need | App |
|------|-----|
| Login, cookies, `me` | `auth` |
| Users, orgs, memberships | `identity` |
| Roles / `MustAllow` | `permissions` |
| Locale, calendar, shell title | `settings` |
| Install / upgrade apps | `appman` |

Consume users from GraphQL ([capabilities](../capabilities.md)), not by importing another app’s Go types.

## When to write more Go

The [advanced tutorial](advanced/orm-and-services.md) covers `Setup`, `ModelRegistry`, and custom GraphQL. Use that only for things YAML cannot express: extra HTTP routes, seeding, or a host service. `apps/auth` is that hybrid. Ordinary CRUD should stay on the spec.

Addons that listen to *other* apps use `extension.Register` / `extends:` in `app.yaml` (`apps/audit` logs every model mutation).

## Where to read next

[Development](../development/index.md) covers install and day-to-day workflow. [Platform](../platform.md) covers auth, ACL, GraphQL, and the CLI. [Internals](../internals/index.md) explain boot order, the Host bag, and data-flow maps. Package notes: [Go SDK](../internals/go-sdk.md), [React SPA](../internals/solid-sdk.md).

## Reference in this repo

| Path | Why |
|------|-----|
| `apps/hellospec` | Smallest complete spec app + hooks |
| `apps/identity` | Several models, menus, and views |
| `apps/auth` | Setup/Mount for login |
| `internal/engine` | What `engine.New` actually wires |
| `apps/core/spa` | React shell: `@/k`, `@/lib`, `@/components/ui` |

## Load a subset

```bash
KaizenGo_APPS=core,identity,auth,permissions,appman,todo ./bin/server
```

Without `KaizenGo_APPS`, the process loads **installed** apps plus `autoInstall` (the shell). Install Inventory from **Apps** so its migrations run.

## Docs site

```bash
make docs
```

Preview at http://localhost:8000 — this tutorial lives there.
