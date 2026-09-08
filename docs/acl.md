# ACL system

KaizenGo uses a **unified ACL** (access control list): one policy row shape covers models, fields, record rules, menus, GraphQL queries, and custom APIs. Policies are stored in `apps/permissions` and evaluated by `packages/sdk-go/acl`. The engine enforces them at the ORM and GraphQL layers.

For sessions, login, and identity, see [Auth & identity](auth.md).

## Architecture

```text
app.yaml + security.yaml
        |
        v
  engine (modelService, catalog queries, custom resolvers)
        |
        v
  permissions service  implements  acl.Authorizer
        |
        v
  acl_entry rows  (role, effect, kind, resource, actions?, fields, domain, priority)
```

| Piece | Location | Role |
|-------|----------|------|
| Policy store | `apps/permissions` models `role`, `user_role`, `acl_entry` | Persist and seed policies |
| Evaluator | `packages/sdk-go/acl` | Match resources, domains, field masks, priority |
| Authorizer API | `permissions` host service (`acl.ServiceName`) | `Can`, `CanCatalog`, `MustAllow`, `ListDomain`, `DeniedFields` |
| Enforcement | `internal/engine` | Model CRUD, menu catalog, `{app}Ping`, custom GQL |
| Declarative seed | `security.yaml` in each app | Upsert roles, entries, demo users on boot |
| Admin UI | Identity → **Access** tab | Inspect roles, rules, create overrides |

GraphQL `resources` and `aclActions` (permissions app) list every registered securable surface for policy authoring.

## Resource identifiers

Every policy `resource` field uses a **single convention**: `{app}.{kind}.{name…}` where `kind` is a fixed segment and `name` is a short slug (never another dotted path). The same kind is stored on the `acl_entry.kind` column.

| Kind | Pattern | Example | Declared in | Controls |
|------|---------|---------|-------------|----------|
| **View** | `{app}.view.{name}` | `identity.view.Users` | views / menus / auto catalog | Page surface; also drives menu + Apps nav |
| **Query** | `{app}.query.{name}` | `hellospec.query.hellospecGreetings` | `queries:` / engine catalog | GraphQL query (call) |
| **Command** | `{app}.command.{name}` | `hellospec.command.hellospecPostGreeting` | `commands:` | GraphQL mutation (call) |
| **Mutation** | `{app}.mutation.{name}` | `appman.mutation.installApp` | Legacy custom GQL | GraphQL mutation |
| **App** | `{app}` | `inventory` | App registration | Coarse app-wide surface |

**Not cataloged**

| Surface | Why |
|---------|-----|
| **Model** `{app}.{model}` | Persistence-only; internal field/domain ACL when commands write |
| **Menu** / **Nav** | Not ACL targets — visibility is **implied from view access** |

Wildcards: `*` (everything) or `{app}.*` (prefix match).

### Menus and Apps nav (implied from views)

```text
app.yaml nav:     → Apps dropdown shows the app if any of its views is allowed
app.yaml menus:   → menubar item shows if its linked view is allowed
                   → folder items show if any child remains
```

Deny (or omit allow, when using default-deny policies) on `identity.view.Users` hides the Users tab. If every view in Identity is denied, Identity disappears from the Apps dropdown.

### Model ACL (internal only)

Models use `{app}.{model}` (no `.model.` infix) for **internal** field/domain checks when commands write through `ModelRegistry`. They are **not** registered in the `resources` catalog — grant `query` / `command` / `view` for client access.

Standard GraphQL catalog queries per app:

| Query | Resource |
|-------|----------|
| `{app}Ping` | `{app}.query.{app}Ping` (camelCase app name) |
| `{app}Menus` | `{app}.query.{app}Menus` |
| `{app}Views` | `{app}.query.{app}Views` |
| `{app}ViewSlots` | `{app}.query.{app}ViewSlots` |

Use **raw resource ids** in policies and in the Access UI combobox — not translated labels.

## Policy row (`acl_entry`)

| Field | Meaning |
|-------|---------|
| `roleId` | Role slug (`admin`, `member`, …) |
| `effect` | `allow` or `deny` |
| `kind` | Resource type: `query`, `command`, `view`, `app`, … |
| `resource` | See table above; `*` matches everything |
| `actions` | **Model / app / api only.** JSON array such as `["read","update"]` or `["*"]`. Ignored for call-style kinds |
| `fields` | `"*"` (all fields) or JSON array of field names (models) |
| `domain` | `[]` = all records; else ANDed triples, e.g. `[["authorId","=","$user.id"]]` |
| `priority` | Higher wins; at equal priority **deny** beats **allow** |
| `active` | Inactive rows are ignored |

### Call-style vs model ACL

**Call-style** kinds (`query`, `command`, `view`) are binary for invoke: omit `actions`. Queries and commands may still set `fields` / `domain` for record-level scoping. Views are invoke-only (menus/nav follow view access).

**Model** (and coarse `app` / `api`) policies still use `actions` plus optional `fields` / `domain` for CRUD and field masking.

### Actions (model / app surfaces)

| Action | Typical use |
|--------|-------------|
| `read` | List, get |
| `create` | Insert |
| `update` | Patch |
| `delete` | Remove |
| `execute` | Side-effect / operational APIs (legacy non-CQRS) |
| `*` | All actions on the resource |

### Domain variables

`$user.id`, `$user.orgId`, `$user.roles`.

Operators: `=`, `!=`, `in`, `not in`, `>`, `<`, `>=`, `<=`, `like`, `is set`, `is not set`.

## Evaluation

`acl.Evaluate` is used for **data and API access** (models, queries, custom resolvers):

- No matching policy → **denied**
- Matching policies sorted by `priority` (desc), then deny before allow at same priority
- Field rules apply per record on List/Get (domain on field allows/denies is respected per row)
- `ListDomain` derives SQL filters from allow/deny domains for list queries

`acl.EvaluateCatalog` is used for **view visibility** (and thus menus / Apps nav):

- No matching policy → **allowed** (visible by default)
- Matching policies → same priority/deny rules as `Evaluate`

Deny a view to hide its menu tab; if every view in an app is denied, the app drops out of the Apps dropdown.

## Where enforcement runs

| Surface | Mechanism | Default when no policy |
|---------|-----------|------------------------|
| Shell Apps dropdown (`GET /api/apps`) | `FilterShellNav` — any allowed view in the app | Allow if any view allowed (or no tracked views) |
| In-app menubar (`{app}Menus`) | `FilterMenuCatalog` — linked view allowed | Allow |
| Model List/Get/Create/Update/Delete | `modelService` + `acl.Authorizer` | Deny |
| `{app}Ping` / CQRS query·command | `gql.RequireAction` | Deny |
| `{app}Views`, `{app}ViewSlots` | Session required; list not filtered yet | — |
| Custom GraphQL (appman, permissions catalog) | `gql.RequireAction` in resolver | Deny |
| Internal seed / migrations | `engine.WithInternal` | ACL skipped |

### Hiding a page (and its menu / app nav)

```yaml
# apps/identity/security.yaml
entries:
  - name: hide-users-view
    role: member
    effect: deny
    kind: view
    resource: identity.view.Users
    priority: 2000
```

Members no longer see the **Users** tab. Deny every Identity view to hide Identity from the Apps dropdown.

Direct URL navigation to a denied view may still load the page if the route is not separately guarded — catalog filtering controls menu/nav visibility.

### Blocking a query

```yaml
entries:
  - name: deny-identity-ping
    role: member
    effect: deny
    kind: query
    resource: identity.query.identityPing
    priority: 2000
```

Members receive **permission denied** on `identityPing`. Admins still pass via the seeded `allow *` rule.

## Declarative policies (`security.yaml`)

List files under `security:` in `app.yaml`:

```yaml
# app.yaml
security:
  - security.yaml
```

Each file can define:

```yaml
roles:
  - name: member
    label: Member

disable:
  - old-broad-grant          # deactivate acl_entry by name

entries:
  - name: myapp-member-query
    role: member
    effect: allow
    kind: query
    resource: myapp.query.myappItems

  - name: myapp-member-read
    role: member
    effect: allow
    kind: model
    resource: myapp.item
    actions: [read]
    fields: [title]          # or "*" for all fields
    domain: []               # or [[authorId, "=", "$user.id"]]
    priority: 0

users:
  - email: demo@example.com
    name: Demo User
    password: secret
    roles: [member]
```

**Apply timing:** `ApplySecurity` runs in a host **startup hook** after every app's `Setup` finishes. That way `identity` (which loads before `permissions` in the dependency graph) can still seed policies that need the permissions service.

Order: ensure roles → disable named entries → upsert entries → find-or-create users, set passwords, assign roles. Requires `permissions` loaded (and `identity` / `auth` when seeding `users:` with passwords).

## Seeded defaults

`permissions.SeedDefaults` (on permissions app boot) creates:

| Role | Policies |
|------|----------|
| `admin` | `allow` `*` / `*` / fields `*` |
| `member` | read `identity.user`, `identity.organization`, `identity.org_unit`, `identity.membership`; CRUD `hellospec.greeting` (see hellospec `security.yaml`); read `inventory.*`; read `appman`; read `settings.*` |

App-specific rules belong in each app's `security.yaml`, not hand-written Go seed.

## HelloSpec example

`apps/hellospec/security.yaml` demonstrates CQRS call grants plus model field ACL:

| Policy | Meaning |
|--------|---------|
| allow `kind: query/command` on greeting CQRS fields | Binary invoke grants |
| allow read `message`, `mood` | Field-limited model read |
| allow create `message`, `mood`, `internalNote` | Field-limited create |
| allow update/delete with `authorId = $user.id` | Record rule |
| deny `internalNote` @ priority 1000 | Field deny for others |
| allow read `internalNote` own @ priority 1001 | Higher-priority carve-out |

Demo login: `jahan.doran@acme.example` / `member`.

## Managing policies

**UI:** Identity → **Access** (permissions app contributes the tab). Pick a role, inspect users and ordered rules, add overrides, deactivate entries.

**GraphQL** (admin session):

```graphql
query {
  resources { app kind name resource label actions }
  aclActions
}
```

Model rows: `permissionsAclEntrys`, `permissionsRoles`, `permissionsUserRoles`.

## Custom apps

### Non-model GraphQL

```go
import (
  "kaizengo/packages/sdk-go/acl"
  sdkgql "kaizengo/internal/gql"
)

// Inside a resolver:
if _, err := sdkgql.RequireAction(host, acl.ServiceName, p, "appman", acl.ActRead); err != nil {
  return nil, err
}
```

Register the resource in app Setup if it is not auto-registered:

```go
acl.Register(acl.ResourceDescriptor{
  App: "myapp", Kind: acl.KindAPI, Name: "export",
  Resource: "myapp.export",
  Actions:  acl.ReadActions(),
  Surface:  "graphql",
})
```

### Programmatic seed (Setup hooks)

```go
perm, _ := host.Lookup(acl.ServiceName)
svc := perm.(interface {
  EnsureEntry(ctx context.Context, orgID, roleName string, spec service.EntrySpec) error
})
// or EnsureRole / EnsureACLEntry via engine.ApplySecurity patterns
```

Prefer `security.yaml` for declarative, reviewable policies.

## Related docs

- [Auth & identity](auth.md) — sessions, login, identity app
- [Go SDK → security](internals/go-sdk.md) — `security:` in `app.yaml`
- [ORM and services](tutorial/advanced/orm-and-services.md) — `RequireAction` for hybrid resolvers
- [Go SDK](internals/go-sdk.md) — `acl` and `gql` packages
