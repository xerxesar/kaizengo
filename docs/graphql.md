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

## Endpoint

- `POST /graphql` — queries and mutations  
- Browser GraphiQL UI is enabled on `/graphql` (GET)

Example:

```bash
curl -s http://localhost:8080/graphql \
  -H 'content-type: application/json' \
  -d '{"query":"{ hello(name:\"Go\") }"}'
```

Counter app (when loaded):

```graphql
query { counter }
mutation { addCounter(by: 1) }
mutation { resetCounter }
```

## Client usage (SPA)

From any app module:

```ts
const res = await fetch('/graphql', {
  method: 'POST',
  headers: { 'Content-Type': 'application/json' },
  body: JSON.stringify({ query: '{ counter }' }),
})
const { data, errors } = await res.json()
```

In Vite spa-dev, `/graphql` is proxied to `:8080`.

## Server state

Keep mutable state in your app package (`apps/myapp/service`), `Provide` it on the host if other apps need it, and close over it in GraphQL resolvers (see `apps/counter`).

## Schema docs / codegen

`apps/core/graphql/schema.graphqls` documents **core-owned** fields for optional client codegen in the shell. App-owned fields are **not** listed there — they appear only when that app is loaded (`KaizenGo_APPS`).

## Without an app

```bash
KaizenGo_APPS=core ./bin/server
# { counter } → Cannot query field "counter"
```
