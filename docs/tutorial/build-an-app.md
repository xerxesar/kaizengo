# Build the Todo app

You will add `apps/todo` by hand so each file is obvious. The CLI (`kaizengo new-app`) scaffolds the same layout if you want a shortcut later.

## 1. Spec

Create `apps/todo/app.yaml`:

```yaml
name: todo
title: Todo
summary: Spec-driven task list
depends:
  - core
  - identity
  - auth
  - permissions
uses:
  - identity.users
  - permissions.rbac
nav:
  labelKey: nav.todo
  route: todo
models:
  - name: task
    fields:
      - name: title
        type: string
        required: true
      - name: done
        type: bool
        default: false
menus:
  - id: tasks
    labelKey: todo.menu.tasks
    children:
      - id: task_list
        labelKey: todo.menu.list
        view: TaskList
      - id: task_new
        labelKey: todo.menu.new
        view: TaskForm
locales:
  - id: en
    name: English
    dir: ltr
```

Field types: `string`, `text`, `int`, `number`, `bool`, `enum`, `date`, `datetime`, `json`, plus relations `many2one`, `one2many`, `many2many` (`relation: app.model` or `model`; `one2many` also needs `inverse:`). Aliases like `float`, `integer`, and `fk` work. Enums need `values:`. Validation keys include `minLength`, `maxLength`, `pattern`, `min`, `max`.

## 2. Module

`apps/todo/module.go`:

```go
package todo

//go:generate go run ../../cmd/godino gen-types todo

import (
	"kaizengo/internal/module"
	"kaizengo/packages/sdk-go/engine"
)

func init() {
	module.Register(engine.New(engine.Options{
		AppName: "todo",
		Version: "0.1.0",
	}))
}
```

Hooks come in the [next page](hooks.md). For now this one-liner is enough: locales, nav, catalog queries, event store, and GraphQL CRUD all come from the spec.

## 3. Migrations

Every app gets its own Postgres schema (default: the app name). The engine does **not** generate tables from YAML — you write SQL.

`apps/todo/migrations/001_events.sql` — copy the event store from `apps/hellospec/migrations/001_events.sql` (`streams` and `events`).

`apps/todo/migrations/002_tasks_read.sql`:

```sql
-- Read model: todo.task  (table name = {model}s_read)

CREATE TABLE IF NOT EXISTS tasks_read (
    id         TEXT PRIMARY KEY,
    org_id     TEXT NOT NULL,
    author_id  TEXT NOT NULL,
    deleted    BOOLEAN NOT NULL DEFAULT false,
    created_at TIMESTAMPTZ NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL,
    title      TEXT NOT NULL DEFAULT '',
    done       BOOLEAN NOT NULL DEFAULT false
);

CREATE INDEX IF NOT EXISTS idx_tasks_read_org ON tasks_read(org_id, updated_at DESC);
```

System columns (`id`, `org_id`, `author_id`, `deleted`, `created_at`, `updated_at`) are required. Add one column per model field, snake_case (`done` stays `done`).

## 4. Views

Views are SolidJS files compiled into the **core** SPA. Pages are `views/<Name>.page.tsx`.

`apps/todo/views/TaskList.page.tsx`:

```svelte
<script lang="ts">
  import { KTable, KAppStatus, t } from '@kaizengo/sdk-solid/ui'
</script>

<KTable model="todo.task" emptyMessage={t('todo.empty')} />

<KAppStatus />
```

`apps/todo/views/TaskForm.page.tsx`:

```svelte
<script lang="ts">
  import { KForm, KFormField, KTable, KAppStatus, t } from '@kaizengo/sdk-solid/ui'

  let table = $state<{ refresh: () => Promise<void> }>()
</script>

<KForm model="todo.task" onsuccess={() => void table?.refresh()}>
  <KFormField field="title" label={t('todo.create')} placeholder={t('todo.new_placeholder')} />
</KForm>

<KTable bind:this={table} model="todo.task" emptyMessage={t('todo.empty')} class="mt-4" />

<KAppStatus />
```

`model="todo.task"` is `{app}.{model}`. The SDK maps that to `todoTasks` / `createTodoTask` and sends the session cookie.

## 5. Locale

`apps/todo/locale/en.po`:

```po
msgid ""
msgstr ""
"Language: en\n"
"Content-Type: text/plain; charset=UTF-8\n"

msgid "nav.todo"
msgstr "Todo"

msgid "todo.empty"
msgstr "No tasks yet."

msgid "todo.create"
msgstr "Add task"

msgid "todo.new_placeholder"
msgstr "What needs doing?"

msgid "todo.menu.tasks"
msgstr "Tasks"

msgid "todo.menu.list"
msgstr "All tasks"

msgid "todo.menu.new"
msgstr "New task"
```

`nav.todo` is the Apps menu label (`labelKey` in the spec). Menu keys must match `menus[].labelKey`.

## 6. Register and generate

Blank-import the package in `apps/apps.go`:

```go
import (
	_ "kaizengo/apps/todo"
	// ...existing apps
)
```

Generate types from the spec:

```bash
make generate
# writes apps/todo/__types__/task.go
```

Restart `make dev` (new Go package). Open the shell, pick **Todo** from Apps, and create a task.

!!! warning "Admin only on first run"
    Engine CRUD is gated by ACL in the model layer. The seeded admin role may do everything (`*` / `*`). Other roles need an `acl_entry` for resource `todo.task` (or `todo.*`) — declare it in `security.yaml` and list the file under `security:` in `app.yaml`. See [ACL system](../acl.md).

## Shortcut: CLI

Same layout, generated for you:

```bash
make cli
./bin/kaizengo new-app todo --type svelte --title Todo
```

The scaffold uses an `item` model and `Items` / `NewItem` views. Rename in `app.yaml` and the view files if you want `task` instead.

Next: [add a hook](hooks.md) so titles are trimmed before they hit the event store.
