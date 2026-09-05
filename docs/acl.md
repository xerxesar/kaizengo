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
  acl_entry rows  (role, effect, resource, actions, fields, domain, priority)
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

Every policy `resource` field uses a **single convention**: `{app}.{kind}.{name…}` where `kind` is a fixed segment and `name` is a short slug (never another dotted path).

| Kind | Pattern | Example | Declared in | Controls |
|------|---------|---------|-------------|----------|
| **Nav** | `{app}.nav` | `identity.nav` | `app.yaml` `nav:` | Shell **Apps** dropdown entry |
| **Menu** | `{app}.menu.{id}` | `identity.menu.users` | `app.yaml` `menus:` | In-app menubar tab |
| **Model** | `{app}.{model}` | `identity.user` | `app.yaml` `models:` | CRUD / field / record ACL |
| **View** | `{app}.view.{name}` | `identity.view.Users` | views / auto catalog | Page surface (catalog) |
| **Query** | `{app}.query.{name}` | `identity.query.identityPing` | Engine catalog GQL | GraphQL query |
| **Mutation** | `{app}.mutation.{name}` | `appman.mutation.installApp` | Custom GQL | GraphQL mutation |
| **Event** | `{app}.event.{model}` | `identity.event.membership` | Per model (implicit) | Event-stream ACL |
| **App** | `{app}` | `inventory` | App registration | Coarse app-wide surface |

Wildcards: `*` (everything) or `{app}.*` (prefix match).

### Two different “menus”

```text
app.yaml nav:          →  identity.nav           →  core shell Apps dropdown
app.yaml menus:        →  identity.menu.users    →  tabs inside /app/identity/
```

Hiding `identity.nav` removes Identity from the Apps menu. Hiding `identity.menu.users` removes only the Users tab inside Identity.

### Event stream vs event resource

Models get an internal **event-store stream type** (defaults to `{app}.{model}`, e.g. `identity.membership`) used by Postgres event sourcing. That is **not** the ACL id.

| Concept | Example | Purpose |
|---------|---------|---------|
| Stream type (internal) | `identity.membership` | `events` table `stream_type` column |
| Event ACL resource | `identity.event.membership` | Policy target (`{app}.event.{model}`) |

Previously the catalog incorrectly registered `identity.event.identity.membership` by reusing the stream type as the ACL name. ACL ids always use the **model name** as the final segment.

### Model resource (no kind segment)

Models are the exception: `{app}.{model}` with no `.model.` infix — historical convention, matches GraphQL `identityUsers` / search collections like `identity.user`.

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
| `resource` | See table above; `*` matches everything |
| `actions` | JSON array, e.g. `["read","update"]` or `["*"]` |
| `fields` | `"*"` (all fields) or JSON array of field names |
| `domain` | `[]` = all records; else ANDed triples, e.g. `[["authorId","=","$user.id"]]` |
| `priority` | Higher wins; at equal priority **deny** beats **allow** |
| `active` | Inactive rows are ignored |

### Actions

| Action | Typical use |
|--------|-------------|
| `read` | List, get, menu visibility, catalog queries |
| `create` | Insert |
| `update` | Patch |
| `delete` | Remove |
| `execute` | Side-effect / operational APIs |
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

`acl.EvaluateCatalog` is used for **navigation catalogs** (menus today):

- No matching policy → **allowed** (menus visible by default)
- Matching policies → same priority/deny rules as `Evaluate`

This lets apps hide specific menu items with a deny rule without granting every other menu explicitly.

## Where enforcement runs

| Surface | Mechanism | Default when no policy |
|---------|-----------|------------------------|
| Shell Apps dropdown (`GET /api/apps`) | `FilterShellNav` per `{app}.nav` (`CanCatalog`) | Allow |
| Model List/Get/Create/Update/Delete | `modelService` + `acl.Authorizer` | Deny |
| `{app}Ping` | `gql.RequireAction` on query resolver | Deny |
| `{app}Menus` | `FilterMenuCatalog` per `{app}.menu.{id}` (`CanCatalog`) | Allow per item |
| `{app}Views`, `{app}ViewSlots` | Session required; view ACL not yet filtered in catalog | — |
| Custom GraphQL (appman, permissions catalog) | `gql.RequireAction` in resolver | Deny |
| Internal seed / migrations | `engine.WithInternal` | ACL skipped |

Spec CRUD GraphQL resolvers only require a session; **the ORM enforces ACL**, not the resolver wrapper alone.

### Hiding a shell app (Apps dropdown)

```yaml
# apps/identity/security.yaml
entries:
  - name: hide-identity-app
    role: member
    effect: deny
    resource: identity.nav
    actions: [read]
    priority: 2000
    fields: "*"
```

Members no longer see **Identity** in the core Apps menu. Admins still do.

### Hiding an in-app menu tab

```yaml
# apps/identity/security.yaml
entries:
  - name: hide-users-tab
    role: member
    effect: deny
    resource: identity.menu.users
    actions: [read]
    priority: 2000
    fields: "*"
```

Restart the server after changing `security.yaml`. The **Users** tab disappears from the Identity menubar for members; other tabs stay visible.

Direct URL navigation to a view may still load the page if the view route is not separately guarded — menu ACL controls catalog visibility.

### Blocking a query

```yaml
entries:
  - name: deny-identity-ping
    role: member
    effect: deny
    resource: identity.query.identityPing
    actions: [read]
    priority: 2000
    fields: "*"
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
  - name: myapp-member-read
    role: member
    effect: allow
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

`apps/hellospec/security.yaml` demonstrates field ACL and own-record rules:

| Policy | Meaning |
|--------|---------|
| allow read `message`, `mood` | Field-limited read |
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
