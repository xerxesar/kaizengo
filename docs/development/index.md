# Development

Practical guides for building apps on KaizenGo: workflow, platform APIs, and product-facing systems (auth, ACL, GraphQL, apps).

SDK package maps and host mechanics live under **Reference**: [Internals](../internals/index.md) · [Go SDK](../internals/go-sdk.md) · [Solid SDK](../internals/solid-sdk.md).

| Guide | Use when you need to… |
|-------|------------------------|
| [Workflow](workflow.md) | Run `make dev`, ports, env vars, Postgres |
| [Platform APIs](platform.md) | Time, i18n, calendars, settings, fork-vs-patch |
| [Auth](../auth.md) | Sessions, login, identity |
| [ACL](../acl.md) | Policies, `security.yaml`, enforcement |
| [Apps](../apps.md) | Install / load / Host bag |
| [CLI](../cli.md) | `kaizengo new-app` |

## Quick orientation

```text
apps/<name>/
  app.yaml + module.go     →  internal/engine wires CRUD / GQL / nav
  models/*/hooks.go        →  Before*/After* on mutations
  views/*.page.tsx         →  shell mounts pages from menus
  locale/*.po              →  shared with Go i18n.T / SPA t()
```

| Layer you call | Import |
|----------------|--------|
| Spec contracts | `kaizengo/packages/sdk-go/appspec`, `…/acl`, `…/i18n` |
| Host APIs (in-tree) | `kaizengo/internal/engine`, `…/extension`, `…/module` |
| UI | `@kaizengo/sdk-solid/ui`, `…/identity`, `…/search` |

→ [Go SDK](../internals/go-sdk.md) · [Solid SDK](../internals/solid-sdk.md) · [Internals](../internals/index.md)
