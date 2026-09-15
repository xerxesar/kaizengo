# KaizenGo — agent guide

Enterprise Operations Platform (EOP). Spec-driven apps under `apps/`, React SPA shell in `apps/core/spa`, Go engine in `internal/` + `packages/sdk-go`.

## Before changing code

1. Read matching docs under `docs/` (especially `docs/internals/` for platform work).
2. Prefer existing patterns in `apps/hellospec` (CQRS UI) and `apps/core/spa/src/components/k` (list widgets).
3. Keep docs in sync when you change internals, packages, or public K APIs.

## Cursor rules

| Rule | When |
|------|------|
| `.cursor/rules/project.mdc` | Always — philosophy & workflow |
| `.cursor/rules/k-components.mdc` | K* widgets, `useKQuery`, list/search UI |
| `.cursor/rules/keymap.mdc` | Hotkeys: `data-keymap-id`, `keymap.yaml` |
| `.cursor/rules/spa-views.mdc` | App `views/*.page.tsx`, SPA imports |
| `.cursor/rules/spec-apps.mdc` | `app.yaml`, modules, Go types |

## Stack map

```text
apps/<name>/
  app.yaml + views/*.page.tsx + module.go (+ keymap.yaml)  →  product apps
apps/core/spa/src/
  components/k/     →  CQRS UI (prefer @/k)
  components/ui/    →  shadcn primitives
  components/shell/ →  layout, menus
  lib/              →  model-client, i18n, auth, keymap
packages/sdk-go/    →  Go app SDK
internal/engine/    →  GraphQL, list page, mount, keymap catalog
```

## Frontend imports

| Import | Use for |
|--------|---------|
| `@/k` | `KTable`, `KForm`, `KCollection`, `useKQuery`, … |
| `@/lib` | `t()`, model client, layout helpers, keymap helpers |
| `@/components/ui/*` | Button, Input, Alert, … |

Do **not** reintroduce Solid or `packages/sdk-solid` — the shell is React.

## Canonical list UI

Query-backed lists always fetch through **`useKQuery`** (server pagination + search). See `.cursor/rules/k-components.mdc`.

```tsx
<KTable query="hellospec.greetings" paginated searchable />
```

List chrome hotkeys (`KSearch` / `KPagination` / `KForm`) are declared in `apps/core/keymap.yaml` — see `.cursor/rules/keymap.mdc`.
