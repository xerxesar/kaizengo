# ORM and services

KaizenGo has no classic ORM. Spec models are event-sourced. The type you use from Go is `engine.ModelRegistry`: it appends events, runs hooks, and projects the read-model table.

## Setup

`engine.Options.Setup` runs after locales, nav, catalog queries, migrations, and generated CRUD. That is where you get the registry.

Replace `apps/todo/module.go` with:

```go
package todo

//go:generate go run ../../cmd/godino gen-types todo

import (
	"kaizengo/internal/module"
	"kaizengo/internal/engine"
)

func init() {
	var svc *Service
	module.Register(engine.New(engine.Options{
		AppName: "todo",
		Version: "0.1.0",
		Setup: func(host *module.Host, events *engine.EventsSetup) error {
			svc = New(events.Models)
			host.Provide(Name, svc)
			registerGQL(host, svc)
			return nil
		},
	}).Hooks("task", taskHooks()))
}
```

`events` gives you:

| Field | Use |
|-------|-----|
| `Models` | This app’s spec models (`task`) |
| `Pool` | Shared Postgres pool — never open your own |
| `Schema` | App schema name (default `todo`) |

The engine also stores `Models` on the host as `todo.models`, so other apps can call `engine.ModelsFromHost(host, "todo")`.

`Mount` (not used here) runs **after every app’s Setup**. Use it for HTTP routes that must see other services. `apps/auth` mounts `/auth/login` there.

## Registry API

Records are `map[string]any` with **camelCase** keys (`title`, `done`, `orgId`, `id`).

Org-scoped (what generated GraphQL uses):

| Method | Role |
|--------|------|
| `Create(ctx, orgID, authorID, model, fields)` | Append create + project |
| `Update(ctx, orgID, model, id, fields)` | Partial update |
| `Delete(ctx, orgID, model, id)` | Soft-delete via event |
| `List(ctx, orgID, model)` | Non-deleted rows for the org |
| `Get(ctx, orgID, model, id)` | One row, org-checked |

Not org-scoped (seeds, cross-lookups):

| Method | Role |
|--------|------|
| `GetByID(ctx, model, id)` | By primary key |
| `FindBy(ctx, model, field, value)` | First match (`field` is the spec name, e.g. `"email"`) |
| `ListAll(ctx, model)` | Every non-deleted row |

`model` is the spec name (`"task"`), not the table name. Writes go through the same pipeline as GraphQL, including [hooks](../hooks.md).

## Service

`apps/todo/service.go`:

```go
package todo

import (
	"context"
	"fmt"

	"kaizengo/internal/engine"
)

const Name = "todo"

type Service struct {
	models *engine.ModelRegistry
}

func New(models *engine.ModelRegistry) *Service {
	return &Service{models: models}
}

func (s *Service) OpenCount(ctx context.Context, orgID string) (int, error) {
	tasks, err := s.models.List(ctx, orgID, "task")
	if err != nil {
		return 0, err
	}
	n := 0
	for _, rec := range tasks {
		if done, _ := rec["done"].(bool); !done {
			n++
		}
	}
	return n, nil
}

func (s *Service) CompleteAll(ctx context.Context, orgID, authorID string) (int, error) {
	tasks, err := s.models.List(ctx, orgID, "task")
	if err != nil {
		return 0, err
	}
	n := 0
	for _, rec := range tasks {
		if done, _ := rec["done"].(bool); done {
			continue
		}
		id := fmt.Sprint(rec["id"])
		if _, err := s.models.Update(ctx, orgID, "task", id, map[string]any{"done": true}); err != nil {
			return n, err
		}
		n++
	}
	return n, nil
}
```

`host.Provide(Name, svc)` puts the service on the host bag. Other apps look it up with `host.Lookup("todo")` and a type assertion, the same way auth looks up `permissions`.

## Custom GraphQL

Generated CRUD stays. Extra fields go on `host.GQL` during Setup.

`apps/todo/gql.go`:

```go
package todo

import (
	permsvc "kaizengo/apps/permissions/service"
	"kaizengo/internal/module"
	sdkgql "kaizengo/internal/gql"

	"github.com/graphql-go/graphql"
)

func registerGQL(host *module.Host, svc *Service) {
	host.GQL.RegisterQuery("todoOpenCount", &graphql.Field{
		Type: graphql.NewNonNull(graphql.Int),
		Resolve: func(p graphql.ResolveParams) (any, error) {
			pr, err := sdkgql.RequireAction(host, permsvc.Name, p, "todo.task", permsvc.ActRead)
			if err != nil {
				return nil, err
			}
			return svc.OpenCount(p.Context, pr.OrgID)
		},
	})
	host.GQL.RegisterMutation("completeTodoTasks", &graphql.Field{
		Type: graphql.NewNonNull(graphql.Int),
		Resolve: func(p graphql.ResolveParams) (any, error) {
			pr, err := sdkgql.RequireAction(host, permsvc.Name, p, "todo.task", permsvc.ActUpdate)
			if err != nil {
				return nil, err
			}
			return svc.CompleteAll(p.Context, pr.OrgID, pr.UserID)
		},
	})
}
```

`RequireAction` loads the session and checks ACL for non-model APIs. Spec model CRUD is enforced inside the engine (`{app}.{model}` resources via `acl_entry`). Duplicate GraphQL field names panic at startup.

Restart `make dev` and try in GraphiQL (`/graphql`):

```graphql
query { todoOpenCount }
mutation { completeTodoTasks }
```

## Other apps’ models

To read identity users from this app:

```go
users, err := engine.ModelsFromHost(host, "identity")
if err != nil {
	return err
}
admin, err := users.FindBy(ctx, "user", "email", "admin@kaizengo.local")
```

Load order matters: `depends:` in `app.yaml` must list `identity` so its Setup has already `Provide`d `identity.models`. Auth does this for login (`apps/auth/module.go`).

## Side tables (not spec models)

Sessions and password hashes are not event-sourced models. Auth keeps them as ordinary SQL on `events.Pool` + `events.Schema` (`apps/auth/store.go`). Quote schema-qualified names; do not create a second pool.

Never `UPDATE todo.tasks_read` from a service. Use `Models.Update` so hooks and the event log stay consistent.

## Generated types

`make generate` writes `apps/todo/__types__/`. Enums become typed strings with `Valid()`. The registry still returns `map[string]any`; map into a struct only if a service needs it. Identity seed uses `__types__` for org-unit enums (`apps/identity/seed.go`).

Next: [a custom Overview page](custom-pages.md) that calls `todoOpenCount` and `completeTodoTasks`.
