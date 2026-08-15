# App anatomy

A Kaizen app is a directory under `apps/` plus a one-line Go registration. The engine reads `app.yaml` and wires the rest.

## Layout

```text
apps/todo/
  app.yaml              # contract: models, views, menus, locales
  module.go             # engine.New(...)
  hooks.go              # optional BeforeCreate / AfterUpdate / …
  views/
    TaskList.svelte     # one file per page view
    TaskForm.svelte
  locale/template.pot   # generated — msgid list; do not edit
  locale/en.po          # gettext strings
  migrations/
    001_events.sql      # streams + events
    002_tasks_read.sql  # read model for task
  __types__/            # generated — do not edit
```

Compare with `apps/hellospec/` — same shape, different names.

## Spec first

`app.yaml` is the source of truth. From it the engine provides:

| Spec | Runtime |
|------|---------|
| `models` | Event-sourced CRUD, GraphQL, `{app}Views` list/form metadata |
| `views` (type `page`) | Svelte files at `apps/<app>/views/<Name>.svelte` |
| `nav` | Entry in the shell Apps menu |
| `menus` | In-app menu tree (`todoMenus`) |
| `locales` | Loaded `.po` catalogs |
| `depends` / `uses` | Load order and capability checks |

GraphQL names follow the spec. For app `todo` and model `task`:

| Operation | Field |
|-----------|--------|
| List | `todoTasks` |
| Get | `todoTask` |
| Create | `createTodoTask` |
| Update | `updateTodoTask` |
| Delete | `deleteTodoTask` |

You rarely call these by hand. `KTable` and `KForm` take `model="todo.task"` and issue the queries for you.

## Registration

```go
func init() {
	module.Register(engine.New(engine.Options{
		AppName: "todo",
		Version: "0.1.0",
	}))
}
```

`AppName` must match the directory and `app.yaml` `name`. Blank-import the package from `apps/apps.go` so `init()` runs:

```go
import (
	_ "kaizengo/apps/todo"
)
```

## What stays out of the app

- **Login and sessions** — `apps/auth`
- **Users and orgs** — `apps/identity`
- **RBAC** — `apps/permissions` (engine CRUD checks the app `resource`, defaulting to the app name)
- **Shell chrome** — `apps/core`

Your app declares `depends: [core, identity, auth, permissions]` so those pieces are loaded first. It should not import their service packages for ordinary CRUD.

Next: [build the Todo app](build-an-app.md).
