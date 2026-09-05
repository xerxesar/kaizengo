# Capability contracts

Apps declare what they **provide** (stable surfaces other modules may use) and what they **require** (`uses`). The server validates every `uses` entry at startup against loaded `provides` plus platform built-ins.

See also [extension-platform.md](extension-platform.md) for how capabilities connect to extension points and SDK clients.

## Declaring capabilities

In `app.yaml`:

```yaml
# apps/identity/app.yaml
provides:
  - identity.users
  - identity.organizations
  - identity.memberships
uses:
  - platform.i18n

# apps/auth/app.yaml
provides:
  - auth.sessions
uses:
  - identity.users

# apps/notes/app.yaml
depends:
  - core
  - identity
  - permissions
uses:
  - identity.users
  - permissions.rbac
```

| Field | Meaning |
|-------|---------|
| `provides` | Named capability surfaces this app publishes |
| `uses` | Capabilities required at runtime (validated at startup) |
| `depends` | Load order only — does **not** imply capability satisfaction |

If `notes` declares `uses: [identity.users]` but `identity` is excluded from `KaizenGo_APPS`, startup fails with a clear error.

## Platform built-ins

Always available without an app provider:

| Capability | Provided by |
|------------|-------------|
| `platform.i18n` | `internal/platform/i18n` |
| `platform.time` | `internal/platform/time` |
| `platform.config` | `internal/platform/config` |

## Current catalog

| Capability | Provider app | Contract |
|------------|--------------|----------|
| `core.shell` | `core` | Shell SPA, nav catalog, `/graphql` host |
| `identity.users` | `identity` | GraphQL `identityUsers`, `identityUser` |
| `identity.organizations` | `identity` | Orgs, org units, memberships |
| `auth.sessions` | `auth` | `/auth/login`, sessions, `me` query |
| `permissions.rbac` | `permissions` | Unified `acl_entry` RBAC (`Can` / `MustAllow` / `ListDomain`) |
| `appman.apps` | `appman` | GraphQL `apps`, `installApp`, `upgradeApp` |
| `platform.search.backend` | `typesense` | GraphQL `search`, index sync on CRUD |

Naming: lowercase segments separated by dots (`identity.users`). Version suffixes (e.g. `/v2`) are deferred until a breaking change.

## Platform search

GraphQL query (requires authenticated session):

```graphql
query ($q: String!, $collections: [String!]) {
  search(q: $q, collections: $collections) {
    id
    collection
    title
    snippet
    score
  }
  searchBackend
}
```

Engine apps with `search:` on a model auto-index on create/update/delete. Custom apps (Notes) call `search.Upsert`/`Delete` from their service layer.

Default backend is in-memory (dev-friendly). Set `KaizenGo_TYPESENSE_URL` and load `apps/typesense` for remote indexing.

The Typesense addon:

- Registers `platform.search.query` middleware so GraphQL `search` / SearchBar hits Typesense for indexed fields (falls back to memory on error or when unset)
- Owns the search settings UI (`apps/typesense/spa`) and injects a **Search** tab into the Settings app via `exports.views` (`slot: tabs`)
- Exposes GraphQL `searchConfig`, `updateSearchModelConfig`, `reindexSearchModel`

## Identity users — GraphQL contract

Apps that `uses: [identity.users]` should consume users via GraphQL or `@kaizengo/sdk-solid/identity`, not by importing another app's Go types.

### Query: list org members

```graphql
query ($orgId: ID!) {
  users(orgId: $orgId) {
    id
    orgId
    email
    name
    status
    createdAt
  }
}
```

Requires an authenticated session with access to the org.

### Query: single user

```graphql
query ($id: ID!) {
  user(id: $id) {
    id
    orgId
    email
    name
    status
    createdAt
    memberships { id orgUnitId role }
  }
}
```

### TypeScript client

```ts
import { fetchUsers, fetchActiveUsers, UserPicker } from '@kaizengo/sdk-solid/identity'

const users = await fetchActiveUsers(orgId)
```

`UserPicker` is a Solid component that loads active users for an org and renders a `@kaizengo/sdk-solid/ui` `Select`. Used in Notes share dialog.

Add `@kaizengo/sdk-solid` to your SPA `package.json` (same pattern as `@kaizengo/sdk-solid/ui`). Vite resolves it via `packages/sdk-solid/spa-config/app-vite.ts`.

## Validation

After dependency resolution in `internal/module/load.go`:

1. Collect `provides` from every loaded app's `app.yaml`
2. Merge with `platform.*` built-ins
3. Fail if any app's `uses` is missing from the merged set

Implementation: `sdk/appspec.ValidateLoadedCapabilities`.

## Related

- [auth.md](auth.md) — sessions and identity
- [acl.md](acl.md) — unified ACL policies and enforcement
- [Go SDK](internals/go-sdk.md) — extension points and engine hooks
- [extension-platform.md](extension-platform.md) — full extension platform design
- [Go SDK](internals/go-sdk.md) — extension dispatch
