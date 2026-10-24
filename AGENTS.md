# KaizenGo — agent guide

Enterprise Operations Platform (EOP). Spec-driven apps under `apps/`, React SPA shell in `apps/core/spa`, Go engine in `internal/` + `packages/sdk-go`.

## Architecture (3+1)

Canonical map: [`docs/architecture.md`](docs/architecture.md).

| Layer | Code |
|-------|------|
| 1. Platform engine | `internal/engine`, `internal/module`, `internal/platform`, `internal/auth` |
| 2. Domain definition | `apps/<name>/app.yaml` → `packages/sdk-go/appspec` |
| 3. Business data | Postgres rows via the engine (`ModelRegistry`, CQRS fields) |
| +1 Dynamic domain | Layers stacked on any app at runtime, stored as revisions: `internal/engine/domains.go`, `kaizengo.domain_revisions`, `platformApplyDomain` |

- Apps categorize domain + data; compiled-in and dynamic apps go through one pipeline (base layer from code or empty, plus stored layer revisions, hot-mounted).
- Clients (React, in-web CLI, native `kaizen` CLI) use **GraphQL only**. CLI grammar lives server-side in `internal/shell` behind `platformShell`.
- **gRPC** (hashicorp/go-plugin, `internal/plugin`, `packages/sdk-go/plugin`) is only for external Go apps: domain defs, Go hooks/handlers, platform services for hooks.
- Everything except Go hooks/handlers must be reachable via GraphQL. New root fields show up in `platformCatalog` automatically (attributed by app prefix, else under `graphql`), so the CLI and palette can call them.
- Build: `make kaizen` (native CLI), `make plugins` (`plugins/*` → `bin/plugins/`), `make proto` (regenerate `packages/sdk-go/plugin/proto`).
- Web terminal / palette: `apps/core/spa/src/components/shell/{Terminal,CommandPalette}.tsx` + `@/lib/platform-shell`; hotkeys `command_palette` (Alt+K) / `toggle_terminal` (Alt+T) in `apps/core/keymap.yaml`.

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
packages/sdk-go/    →  Go app SDK (+ plugin/ for external Go apps)
internal/engine/    →  GraphQL, list page, mount, keymap catalog, domain layers, platform meta GQL
internal/shell/     →  CLI grammar (platformShell)
internal/plugin/    →  go-plugin host for external Go apps
cmd/kaizen/         →  native runtime CLI (GraphQL client)
cmd/kaizengo/       →  dev scaffolding CLI (new-app, gen-types)
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
