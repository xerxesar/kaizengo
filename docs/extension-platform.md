# Extension platform plan

**Goal:** Odoo-like composability (install an addon → behavior changes across apps) **without** Odoo-style `_inherit` (no app patches another app’s code or generated artifacts).

**Strategy:** Apps **publish capabilities**. Addons **register on extension points**. The **SDK** ships reusable components and clients. The **platform** owns cross-cutting drivers (search, audit, …).

---

## Problem statement

| Today | Limit |
|-------|-------|
| `app.yaml` + `packages/sdk-go/engine` | Spec-first CRUD within one app |
| Per-app `Hooks` in `module.go` | Only the owning app can register |
| `host.Provide` / `Lookup` | Ad hoc Go coupling |
| GraphQL flat schema | Public API, but no formal `uses` / `provides` |
| `@kaizengo/sdk-solid/ui` | Shared widgets, no pluggable backends (search) |
| Notes → `users(orgId)` | Works, but types/clients duplicated per SPA |

We cannot install `typesense` and have every list view upgrade automatically. We cannot extend `hellospec.greeting.beforeCreate` from a separate addon module.

---

## Design principles

1. **Contracts over internals** — publish GraphQL fields, UI components, lifecycle point names; never export `service/store` packages to other apps.
2. **Composition over inheritance** — ordered hook chains and slot injection, not class/view xpath patching.
3. **Platform for horizontal concerns** — search, audit, notifications, metrics → `internal/platform/*` drivers + blank-import.
4. **Declare intent in yaml** — `provides`, `uses`, `search`, `extensionPoints` for validation and docs.
5. **Fail at startup** — missing `uses` dependency or unknown extension point → clear error during `Setup`.
6. **Escape hatch preserved** — custom apps (Notes) opt into extension points manually; no forced engine.

---

## Architecture

```text
┌─────────────────────────────────────────────────────────────┐
│  Addon apps (typesense, audit, …)                           │
│  extension.Register(point, priority, handler)               │
└──────────────────────────┬──────────────────────────────────┘
                           │
┌──────────────────────────▼──────────────────────────────────┐
│  sdk/extension — global registry, wildcard match, ordering  │
└──────────────────────────┬──────────────────────────────────┘
                           │
        ┌──────────────────┼──────────────────┐
        ▼                  ▼                  ▼
  sdk/engine          apps/notes         apps/identity
  emits points        manual emit        provides capabilities
        │                  │                  │
        └──────────────────┴──────────────────┘
                           │
┌──────────────────────────▼──────────────────────────────────┐
│  packages/sdk-solid — components (UserPicker), clients (identity) │
│  packages/sdk-solid/ui — SearchInput → platform.search                │
└──────────────────────────┬──────────────────────────────────┘
                           │
┌──────────────────────────▼──────────────────────────────────┐
│  internal/platform — search, (future: audit, notify)        │
└─────────────────────────────────────────────────────────────┘
```

---

## Core concepts

### Capabilities (`provides` / `uses`)

Stable named surfaces other modules may depend on.

```yaml
# apps/identity/app.yaml
provides:
  - identity.users
  - identity.organizations
  - identity.memberships

# apps/auth/app.yaml
provides:
  - auth.sessions

# apps/notes/app.yaml
uses:
  - identity.users
  - permissions.rbac
```

Startup validates: every `uses` entry is satisfied by a loaded app’s `provides` (or platform built-in).

**Built-in platform capabilities** (no app required): `platform.i18n`, `platform.time`, `platform.config`.

### Extension points

Named lifecycle slots with **ordered, multi-handler** chains.

Naming convention:

```text
model.<app>.<model>.beforeCreate
model.<app>.<model>.afterCreate
model.*.afterCreate                    # wildcard — all models
graphql.<fieldName>.beforeResolve      # future
view.<app>.<view>.slot.<name>          # future UI
platform.search.query                  # driver slot
```

Handlers receive a context struct (same shape as today’s `engine.HookContext`, extended with `Point`, `Priority`).

### SDK components

Reusable SPA building blocks backed by published GraphQL:

| Component | Capability | Package |
|-----------|------------|---------|
| `UserPicker`, `UserTable` | `identity.users` | `packages/sdk-solid/identity` |
| `SearchInput` (backend-aware) | `platform.search` | `packages/sdk-solid/ui` + `packages/sdk-solid/search` |
| `OrgUnitTree` | `identity.organizations` | `packages/sdk-solid/identity` |

Apps import from `@kaizengo/sdk-solid`, not from `apps/identity/spa`.

---

## Implementation phases

### Phase 0 — Document & align (1–2 days)

**Deliverables**

- [ ] This plan reviewed and linked from [sdk.md](sdk.md) and [platform.md](platform.md)
- [ ] Glossary in [apps.md](apps.md): *capability*, *extension point*, *addon*, *provider*
- [ ] Inventory of existing cross-app calls (Notes→users, counter→permissions, etc.)

**Exit criteria:** team agrees on naming conventions and non-goals (no xpath view inherit).

---

### Phase 1 — Contracts in app.yaml (3–5 days)

**Deliverables**

- [ ] `packages/sdk-go/appspec`: `Provides []string`, `Uses []string` on `AppSpec`
- [ ] Parse + validate in `appspec.Validate()`:
  - duplicate provides
  - unknown uses at `module.Load` time (registry pass after all manifests known)
- [ ] Add `provides` / `uses` to existing `app.yaml` files:
  - `identity` → provides users, orgs, auth
  - `permissions` → provides rbac
  - `notes`, `hellospec`, `counter` → uses identity + permissions as needed
- [ ] `host` helper: `ValidateCapabilities(loaded []Manifest)` or run in `module.Load` after resolve

**Exit criteria:** server refuses to start if `notes` declares `uses: [identity.users]` but identity not loaded.

---

### Phase 2 — Global extension registry (5–8 days)

**Deliverables**

- [ ] New package `packages/sdk-go/extension`:
  ```go
  func Register(point string, priority int, fn func(ext.Context) error)
  func Run(point string, hc ext.Context) error  // sorted by priority, short-circuit on error
  func Match(point string) []Handler            // exact + wildcard patterns
  ```
- [ ] `ext.Context` embeds / mirrors `engine.HookContext` + `Point`, `App`, `Model`
- [ ] Wire `sdk/engine/model.go` pipeline:
  ```text
  normalize → app hooks → extension.Run(beforeCreate) → validate → append → project → app hooks → extension.Run(afterCreate)
  ```
- [ ] Keep existing per-app `engine.Hooks` (app-local) — run **before** global extensions or document order
- [ ] Proof addon: `apps/audit` (or test module) registers `model.*.afterCreate` → structured log
- [ ] Document extension point catalog in [sdk.md](sdk.md)

**Exit criteria:** audit addon logs greeting create/delete without editing `hellospec/hooks.go`.

---

### Phase 3 — Identity SDK client + components (4–6 days)

**Deliverables**

- [ ] `packages/sdk-solid/identity/`:
  - `client.ts` — `fetchUsers`, `fetchUser`, types (move from Notes duplication)
  - `UserPicker.tsx`, `UserTable.tsx` (minimal)
- [ ] Export from `packages/sdk-solid/package.json`
- [ ] Refactor `apps/notes/spa` to use SDK client + `UserPicker` for share dialog
- [ ] GraphQL documented as stable under [auth.md](auth.md) or new `docs/capabilities.md`
- [ ] `identity` app.yaml lists `provides: [identity.users, …]`

**Exit criteria:** new engine app can add assignee field with `<UserPicker />` and zero custom GraphQL.

---

### Phase 4 — Platform search + extension-ready lists (8–12 days)

**Deliverables**

- [ ] `internal/platform/search`:
  - `Backend` interface: `Query`, `Upsert`, `Delete`
  - Noop default (client-side filter fallback)
- [ ] `packages/sdk-go/appspec`: optional `search.collections[]` on models
- [ ] Engine: `AfterCreate`/`AfterUpdate`/`AfterDelete` → `search.Upsert/Delete` when collection declared
- [ ] GraphQL: `search(q: String!, collections: [String!]): [SearchHit!]!`
- [ ] `@kaizengo/sdk-solid/ui/SearchInput` — debounced query → GraphQL `search` → `onResults` callback
- [ ] Migrate Notes list filter to `SearchInput` (Typesense-ready)
- [ ] Stub app `apps/search` or `apps/typesense` (driver only, env-gated) registering `platform.search.backend`

**Exit criteria:** with noop backend, behavior unchanged; with typesense driver, Notes search hits remote index without Notes code changes.

---

### Phase 5 — Extension declarations in yaml (optional, 5–7 days)

**Deliverables**

- [ ] `app.yaml` syntax for addon extensions:
  ```yaml
  extends:
    - point: model.*.afterCreate
      handler: indexDocument   # resolved via RegisterHook name catalog
  ```
- [ ] `extension.RegisterNamed("indexDocument", fn)` + engine resolves at Setup
- [ ] CLI template for addon app scaffold

**Exit criteria:** typesense addon wires handlers from yaml + small Go register file.

---

### Phase 6 — View slots & component catalog (future, 2–3 weeks)

**Deliverables**

- [ ] `{app}Components` GraphQL or static manifest from `exports.components` in yaml
- [ ] Engine-generated list views accept `toolbar` slot
- [ ] Spec:
  ```yaml
  extends:
    views:
      - match: "*.list"
        slot: toolbar
        component: platform.SearchBar
  ```
- [ ] SPA shell or engine SPA reads slot manifest and dynamic-imports SDK components

**Exit criteria:** typesense addon injects search bar into hellospec list without editing hellospec SPA.

---

## Non-goals (explicit)

- Odoo `_inherit` on models or views
- Runtime patching of Go types or Svelte AST
- Cross-app direct database joins
- Automatic GraphQL schema merging / federation per app
- Extension handlers in interpreted scripts (WASM/JS) — Go only for v1

---

## Migration path for existing apps

| App | Phase 1 | Phase 2 | Phase 3 | Phase 4 |
|-----|---------|---------|---------|---------|
| hellospec | uses/provides | extension proof target | — | search yaml |
| notes | uses/provides | manual extension emit in service | UserPicker | SearchInput |
| identity | provides | — | publish SDK | — |
| counter | uses/provides | optional audit point | — | — |
| clock, status, … | app.yaml only | — | — | — |

---

## Risks & mitigations

| Risk | Mitigation |
|------|------------|
| Hook order bugs | Priority int + documented default; dev mode logs chain |
| Wildcard too broad | Require explicit `optIn: extensions` on app.yaml for `model.*` |
| Performance (many handlers) | Benchmark; allow skip if noop backend |
| Circular extensions | Extensions may not call mutations that re-enter same point without guard |
| SDK drift from GraphQL | CI check: SDK client queries match identity gql schema |

---

## Success metrics

1. **Audit addon** logs all engine model mutations with zero per-app code.
2. **Notes** uses `@kaizengo/sdk-solid/identity` — no local `IdentityUser` type copy.
3. **Typesense addon** indexes hellospec + notes via wildcard hook + yaml `search:` — no edits to those apps’ hooks.go.
4. **New app** `uses: [identity.users]` fails fast if identity not in `KaizenGo_APPS`.
5. Docs: a third-party developer can build an addon from [sdk.md](sdk.md) + this plan alone.

---

## Recommended first sprint (2 weeks)

Week 1: **Phase 1 + Phase 2** (capabilities validation + global extension registry + audit proof).

Week 2: **Phase 3** (identity SDK package + Notes refactor).

Defer Phase 4 (search platform) until extension registry is proven — search is the first real addon consumer of wildcards.

---

## Open questions

1. **Hook order:** app-local hooks vs global extensions — local first or global first?
2. **After* errors:** ignore (today) vs collect vs fail — align for extensions?
3. **Capability versioning:** `identity.users/v2` when GraphQL breaking change?
4. **Opt-in vs opt-out** for wildcard extensions on sensitive apps (identity, permissions)?

Track decisions in this doc as ADR-style bullets when resolved.
