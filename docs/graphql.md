# GraphQL

## Design

GraphQL is a **runtime registry** on `host.GQL`. Apps register fields in `Setup`. Core mounts a single `/graphql` handler after all apps have set up — **core does not import other apps**.

```go
// in your app Setup — do not edit apps/core for this
host.GQL.RegisterQuery("myField", &graphql.Field{
	Type: graphql.NewNonNull(graphql.String),
	Resolve: func(graphql.ResolveParams) (any, error) {
		return "hi", nil
	},
})

host.GQL.RegisterMutation("doThing", &graphql.Field{ … })
```

Library: [`graphql-go/graphql`](https://github.com/graphql-go/graphql) + handler (GraphiQL on the same endpoint).

## Auth

`/graphql` is behind `auth.RequireAuth`. Unauthenticated requests get **401**. Browser and SPA clients must send the session cookie (`credentials: 'include'`) or `Authorization: Bearer <sessionID>`. See [auth.md](auth.md).

Some fields also enforce **permissions** (e.g. identity mutations, counter). Failures surface as GraphQL errors / forbidden from the resolver.

## Endpoint

- `POST /graphql` — queries and mutations  
- Browser GraphiQL UI is enabled on `/graphql` (GET) — sign in first (cookie), or pass a Bearer session

Example (after login):

```bash
curl -c cookies.txt -s http://localhost:8080/auth/login \
  -H 'content-type: application/json' \
  -d '{"email":"admin@kaizengo.local","password":"changeme"}'

curl -b cookies.txt -s http://localhost:8080/graphql \
  -H 'content-type: application/json' \
  -d '{"query":"{ hellospecPing }"}'
```

### Built-in fields (when apps are loaded)

**Platform** (registered by core)

```graphql
query { i18n(prefix: "shell.") { locale dir entries { key value } } }
query { search(q: "hello") { id title snippet } }
```

**Auth** (`apps/auth`)

```graphql
query { me { id email name roles orgId } }
```

**Identity** (`apps/identity`)

```graphql
query { identityOrganizations { id name slug } }
query { identityOrgUnits { id name type parentId } }
query { identityUsers { id email name status } }
query { identityMemberships { id userId orgUnitId role } }
```

**Counter** (`apps/counter` — also requires permissions)

```graphql
query { counter }
mutation { addCounter(by: 1) }
mutation { resetCounter }
```

**Settings**

```graphql
query { settings { locale defaultCalendar shellTitle locales { id name dir } } }
mutation { updateSettings(locale: "fa", defaultCalendar: "persian", shellTitle: "KaizenGo") { locale } }
```

**App Manager** (`apps/appman`)

```graphql
query { apps { name title version installed upgrade autoInstall depends } }
mutation { installApp(name: "inventory") { name installed version } }
mutation { upgradeApp(name: "inventory") { name version installedVersion } }
```

**Notes** (`apps/notes` — ownership + sharing; requires permissions)

```graphql
query { notes { id title body access ownerId updatedAt } }
mutation { createNote(title: "Hello", body: "…") { id access } }
mutation { shareNote(noteId: "…", userId: "…", permission: "read") { userId permission } }
```

## Client usage (SPA)

Always include credentials:

```ts
const res = await fetch('/graphql', {
  method: 'POST',
  credentials: 'include',
  headers: { 'Content-Type': 'application/json' },
  body: JSON.stringify({ query: '{ counter }' }),
})
const { data, errors } = await res.json()
```

In Vite spa-dev, `/graphql` and `/auth` are proxied to `:8080`.

`@kaizengo/sdk-svelte/ui` helpers (`syncDocumentLocale`, `fetchI18n`, etc.) already send credentials. SPA UI strings use compiled `.po` catalogs; `fetchI18n` is optional for GraphQL catalog access.

## Server state

Keep mutable state in your app package, `Provide` it on the host if other apps need it, and close over it in GraphQL resolvers (see `apps/auth`).

## Without an app

```bash
KaizenGo_APPS=core,identity ./bin/server
# { counter } → Cannot query field "counter"
```
