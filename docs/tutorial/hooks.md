# Hooks

YAML covers fields, validation, and CRUD. Use Go hooks when a rule cannot be declared — normalize input, reject a value, or block a delete.

To hide a model’s write API from GraphQL (and any other external caller) without a protect hook, set `internal: true` on the model spec. In-process posting still writes with `engine.WithInternal(ctx)`. See [Go SDK → models](../internals/go-sdk.md).

## Register

In `apps/todo/module.go`:

```go
func init() {
	app := engine.New(engine.Options{
		AppName: "todo",
		Version: "0.1.0",
	})
	module.Register(app.Hooks("task", taskHooks()))
}
```

`Hooks` is keyed by **model name** (`task`), not the app name.

For larger apps, put hooks next to the model spec and register from `init()`:

```go
// apps/todo/models/task/hooks.go
package task

func init() {
	engine.RegisterModelHooks("todo", "task", engine.Hooks{
		BeforeCreate: trimTitle,
		BeforeUpdate: trimTitle,
	})
}
```

Blank-import that package from the app (see `apps/inventory/models.go`). `engine.New` picks the hooks up automatically.

## Trim titles

`apps/todo/hooks.go`:

```go
package todo

import (
	"strings"

	"kaizengo/internal/engine"
)

func taskHooks() engine.Hooks {
	return engine.Hooks{
		BeforeCreate: trimTitle,
		BeforeUpdate: trimTitle,
	}
}

func trimTitle(hc engine.HookContext) error {
	title, ok := hc.Fields["title"].(string)
	if !ok {
		return nil
	}
	hc.Fields["title"] = strings.TrimSpace(title)
	return nil
}
```

`Before*` hooks run **before** spec validation and the event append. Mutate `hc.Fields` in place. Return an error to abort the mutation (the client sees a GraphQL error).

## Pipeline

Each create/update/delete runs:

```text
normalize fields
→ app Before* hook
→ extension.Run(model.<app>.<model>.before*)
→ stored compute / related / Compute hooks
→ spec validation
→ append event + project read model
→ app After* hook
→ extension.Run(model.<app>.<model>.after*)
```

`After*` sees `hc.Record` (the projected row). Do not use After hooks for invariants — the write already happened. Use them for logging or kicking off work that may fail independently.

`hellospec` is a fuller example: it prefixes messages, rejects a blocked word, prevents deleting greetings marked `[protected]`, and uppercases `shout` from a compute hook. `length`, `label`, and `topicName` are declared formulas in `app.yaml` and do not need Go.

## Computed fields

Put formulas in `app.yaml` when the value is a pure function of the row:

```yaml
- name: length
  type: int
  compute: len(message)
- name: shout
  type: string
  compute: hook
  depends: [message]
```

`compute: hook` calls `Hooks.Compute["shout"]`. See `apps/hellospec/hooks.go`.

## Restart and try

Restart `make dev`, create a task with leading spaces, and confirm the stored title is trimmed.

Next: [advanced — ORM, services, and custom pages](advanced/index.md).
