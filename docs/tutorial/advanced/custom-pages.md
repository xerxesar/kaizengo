# Custom pages

`KTable` and `KForm` cover list/create. Write your own React page when the screen is a dashboard, a tree, or it must call GraphQL that is not generated CRUD.

Page views still live in the **core** SPA. You do not add a Vite app or a client-side router.

## Add a page

Create `apps/todo/views/Overview.page.tsx` and point a menu leaf at `Overview`. Names must match the file (without `.page.tsx`):

```yaml
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

Core glob-imports `apps/*/views/**/*.tsx` and keys pages as `{app}.{Name}` (`todo.Overview`). A new `.page.tsx` file is enough; do not register the component in Go.

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

`apps/todo/views/Overview.page.tsx`:

```tsx
import { useEffect, useState } from 'react'
import { Alert } from '@/components/ui/alert'
import { Button } from '@/components/ui/button'
import { listModelRecords, t, StatCard } from '@/lib'
import { KAppStatus } from '@/k'
import { completeTodoTasks } from '../lib/graphql'

export default function Overview() {
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState('')
  const [total, setTotal] = useState(0)
  const [open, setOpen] = useState(0)
  const [working, setWorking] = useState(false)

  async function refresh() {
    setLoading(true)
    setError('')
    try {
      const rows = await listModelRecords('todo', 'task', ['id', 'title', 'done'])
      setTotal(rows.length)
      setOpen(rows.filter((r) => r.done !== true).length)
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e))
    } finally {
      setLoading(false)
    }
  }

  async function completeAll() {
    setWorking(true)
    setError('')
    try {
      await completeTodoTasks()
      await refresh()
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e))
    } finally {
      setWorking(false)
    }
  }

  useEffect(() => {
    void refresh()
  }, [])

  return (
    <>
      {loading ? <p>{t('todo.empty')}</p> : null}
      {error ? <Alert variant="danger">{error}</Alert> : null}
      {!loading ? (
        <>
          <div className="mb-4 grid grid-cols-[repeat(auto-fit,minmax(14rem,1fr))] gap-4">
            <StatCard label={t('todo.stat.open')} value={open} hint={t('todo.stat.open_hint')} />
            <StatCard label={t('todo.stat.total')} value={total} />
          </div>
          <Button disabled={working || open === 0} onClick={() => void completeAll()}>
            {t('todo.complete_all')}
          </Button>
        </>
      ) : null}
      <KAppStatus />
    </>
  )
}
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

- One `.page.tsx` file per menu view under `apps/<app>/views/`.
- Import `t`, `StatCard`, `listModelRecords`, `navigateApp`, and `menuPagePath` from `@/lib`. Import `KAppStatus` from `@/k`. Import `Alert` and `Button` from `@/components/ui`.
- Navigate with `navigateApp(menuPagePath('todo', 'task_list'))`.
- `make dev` hot-reloads React views. New Go (`gql.go`, `service.go`) needs a process restart.

## Larger examples

| File | What it shows |
|------|----------------|
| `apps/identity/views/Overview.page.tsx` | `listModelRecords` + `StatCard` |
| `apps/identity/views/Structure.page.tsx` | Custom query and a tree of org units |
| `apps/identity/lib/graphql.ts` | `credentials: 'include'` client |

Next: [what to learn after the tutorial](../whats-next.md).
