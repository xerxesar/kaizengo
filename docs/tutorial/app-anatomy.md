# App anatomy

A Kaizen app is a directory under `apps/` plus a one-line Go registration. The engine reads `app.yaml` and wires the rest.

## Layout

```text
apps/todo/
  app.yaml              # contract: models, menus, locales, security:
  security.yaml         # optional — roles, ACL entries, demo users
  module.go             # engine.New(...)
  models/task/
    spec.yaml           # or keep the model inline in app.yaml
    hooks.go            # optional BeforeCreate / AfterUpdate / …
  views/
    TaskList.page.tsx
    TaskForm.page.tsx
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
| `models` | Event-sourced CRUD, GraphQL, `{app}Views` list/form metadata (`internal: true` keeps writes in-process) |
| pages (`views/*.page.tsx`) | Svelte screens menus mount |
| `nav` | Entry in the shell Apps menu |
| `menus` | In-app menu tree (`todoMenus`) |
| `locales` | Loaded `.po` catalogs |
| `depends` / `uses` | Load order and capability checks |
| `security:` | Merged YAML files → roles, `acl_entry` policies, demo users on Setup |

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
- **ACL** — `apps/permissions` + `packages/sdk-go/acl` (engine enforces models, menus, queries, and custom APIs). App-specific grants go in `security.yaml` — see [ACL system](../acl.md), not hand-written seed Go.
- **Shell chrome** — `apps/core`

Your app declares `depends: [core, identity, auth, permissions]` so those pieces are loaded first. It should not import their service packages for ordinary CRUD.

Next: [build the Todo app](build-an-app.md).
