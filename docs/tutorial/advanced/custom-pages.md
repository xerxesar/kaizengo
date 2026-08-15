# Custom pages

`KTable` and `KForm` cover list/create. Write your own Svelte when the screen is a dashboard, a tree, or it must call GraphQL that is not generated CRUD.

Page views still live in the **core** SPA. You do not add a Vite app or a client-side router.

## Declare the view

In `apps/todo/app.yaml`, add a page and a menu leaf. Names must match the file:

```yaml
views:
  - name: Overview
    type: page
  - name: TaskList
    type: page
  - name: TaskForm
    type: page
menus:
  - id: tasks
    labelKey: todo.menu.tasks
    children:
      - id: overview
        labelKey: todo.menu.overview
        view: Overview
      - id: task_list
        labelKey: todo.menu.list
        view: TaskList
      - id: task_new
        labelKey: todo.menu.new
        view: TaskForm
```

Core glob-imports `apps/*/views/*.svelte` and keys them as `{app}.{view}` (`todo.Overview`). A new `.svelte` file is enough; do not register the component in Go.

Add strings to `apps/todo/locale/en.po`:

```po
msgid "todo.menu.overview"
msgstr "Overview"

msgid "todo.stat.open"
msgstr "Open"

msgid "todo.stat.open_hint"
msgstr "Not marked done"

msgid "todo.stat.total"
msgstr "Total"

msgid "todo.complete_all"
msgstr "Complete all"
```

## Overview with the model client

`listModelRecords` is the same helper `KTable` uses (session cookie included).

`apps/todo/views/Overview.svelte`:

```svelte
<script lang="ts">
  import { onMount } from 'svelte'
  import {
    Alert,
    Button,
    KAppStatus,
    Spinner,
    StatCard,
    listModelRecords,
    t,
  } from '@kaizengo/sdk-svelte/ui'
  import { completeTodoTasks } from '../lib/graphql'

  let loading = $state(true)
  let error = $state('')
  let total = $state(0)
  let open = $state(0)
  let working = $state(false)

  async function refresh() {
    loading = true
    error = ''
    try {
      const rows = await listModelRecords('todo', 'task', ['id', 'title', 'done'])
      total = rows.length
      open = rows.filter((r) => r.done !== true).length
    } catch (e) {
      error = e instanceof Error ? e.message : String(e)
    } finally {
      loading = false
    }
  }

  async function completeAll() {
    working = true
    error = ''
    try {
      await completeTodoTasks()
      await refresh()
    } catch (e) {
      error = e instanceof Error ? e.message : String(e)
    } finally {
      working = false
    }
  }

  onMount(() => {
    void refresh()
  })
</script>

{#if loading}
  <Spinner />
{:else}
  {#if error}
    <Alert variant="danger" dismissible ondismiss={() => (error = '')}>{error}</Alert>
  {/if}

  <div class="stats">
    <StatCard label={t('todo.stat.open')} value={open} hint={t('todo.stat.open_hint')} />
    <StatCard label={t('todo.stat.total')} value={total} />
  </div>

  <Button disabled={working || open === 0} onclick={() => void completeAll()}>
    {t('todo.complete_all')}
  </Button>
{/if}

<KAppStatus />

<style>
  .stats {
    display: grid;
    grid-template-columns: repeat(auto-fit, minmax(14rem, 1fr));
    gap: 1rem;
    margin-bottom: 1rem;
  }
</style>
```

Keep `<KAppStatus />` on every page — it is the ping / health strip the shell expects.

## Call custom GraphQL

`fetch` must send cookies. Identity does this in `apps/identity/lib/graphql.ts`; copy the same helper.

`apps/todo/lib/graphql.ts`:

```ts
async function gql<T>(query: string, variables?: Record<string, unknown>): Promise<T> {
  const res = await fetch('/graphql', {
    method: 'POST',
    credentials: 'include',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ query, variables }),
  })
  if (!res.ok) throw new Error(`GraphQL HTTP ${res.status}`)
  const body = await res.json()
  if (body.errors?.length) {
    throw new Error(body.errors.map((e: { message: string }) => e.message).join(', '))
  }
  return body.data as T
}

export function completeTodoTasks() {
  return gql<{ completeTodoTasks: number }>(`mutation { completeTodoTasks }`).then(
    (d) => d.completeTodoTasks,
  )
}

export function todoOpenCount() {
  return gql<{ todoOpenCount: number }>(`query { todoOpenCount }`).then((d) => d.todoOpenCount)
}
```

You can drive the open-count StatCard from `todoOpenCount()` instead of filtering client-side. Either is fine; the service is the source of truth for **mutations** that must run hooks.

## Layout rules

- One file per `views:` name under `apps/<app>/views/`.
- Import UI from `@kaizengo/sdk-svelte/ui` (`Alert`, `Button`, `Card`, `Page`, `Spinner`, `StatCard`, `TreeView`, …).
- Navigate with `navigateApp(menuPagePath('todo', 'task_list'))` — do not `window.location` into another app’s HTML.
- `make dev` hot-reloads Svelte. New Go (`gql.go`, `service.go`) needs a process restart.

## Larger examples

| File | What it shows |
|------|----------------|
| `apps/identity/views/Overview.svelte` | `listModelRecords` + `StatCard` |
| `apps/identity/views/Structure.svelte` | Custom query, `TreeView`, modal create |
| `apps/identity/lib/graphql.ts` | `credentials: 'include'` client |

Next: [what to learn after the tutorial](../whats-next.md).
