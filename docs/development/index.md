# Development

How to work in this repository: install it, run it, and call platform services (time, i18n, settings).

Product systems (apps, auth, ACL, GraphQL, the CLI) live under [Platform](../platform.md). Host mechanics and UI contracts live under **Reference**: [Internals](../internals/index.md) · [Go SDK](../internals/go-sdk.md) · [React SPA](../internals/solid-sdk.md).

| Guide | Use when you need to… |
|-------|------------------------|
| [Installation](../installation.md) | Clone the repo, install tooling, produce `bin/server` |
| [Workflow](workflow.md) | Run `make dev`, ports, env vars, Postgres |
| [Platform APIs](platform.md) | Time, i18n, calendars, settings, fork-vs-patch |

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
| UI | `@/k`, `@/lib`, `@/components/ui` in `apps/core/spa` |

→ [Go SDK](../internals/go-sdk.md) · [React SPA](../internals/solid-sdk.md) · [Internals](../internals/index.md)
