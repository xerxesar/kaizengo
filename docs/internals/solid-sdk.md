# React SPA platform

How to write app views against the core shell. Platform UI lives in `apps/core/spa` (React + Vite CSR). Primitives come from shadcn/Radix under `@/components/ui`; Motion is used for light shell transitions.

Shell registry, Vite aliases, and GraphQL naming: [Internals](index.md#frontend-data-flow). Go contracts: [Go SDK](go-sdk.md). Workflow: [Development](../development/index.md).

## Idea

```text
apps/core/spa                 →  single React Vite SPA (/app/)
apps/myapp/views/*.page.tsx   →  compiled into the central bundle
```

## Directory

```text
apps/core/spa/src/
  components/ui/      # shadcn-style primitives
  components/shell/   # Layout, menus, search, layout-bits
  components/k/       # CQRS widgets (useKQuery, KTable, KForm, KCollection, …)
  lib/                # auth, gql, i18n, menus, model-client
  pages/              # Login
  styles/             # tokens + themes
```

## Imports

| Import | Role |
|--------|------|
| `@/lib` | Platform: Layout, `t()`, model client, menus, theme, keymap |
| `@/k` | Spec-driven CQRS UI: `KTable`, `KKanban`, `KCollection`, `KForm`, `useKQuery`, … |
| `@/components/ui` | Button, Input, Alert, Select, Dialog, … |
| `lucide-react` | Icons |
| `@/styles/index.css` | Theme tokens (imported once from `main.tsx`) |

Vite aliases `@` → `apps/core/spa/src`. App views are included in the SPA `tsconfig` / build.

```tsx
import { t, cn } from '@/lib'
import { KTable, KAppStatus } from '@/k'
import { Button } from '@/components/ui/button'

export default function GreetingList() {
  return (
    <>
      <KTable query="hellospec.greetings" paginated searchable emptyMessage={t('hellospec.empty')} />
      <Button variant="primary">OK</Button>
      <KAppStatus />
    </>
  )
}
```

### Pagination

`KPagination` provides prev/next, manual page entry, and page size. It syncs to `?page=` / `?pageSize=` by default.

- **Flag:** `<KTable paginated />`, `<KKanban paginated />`, or `<KCollection paginated />` — wired through **`useKQuery`** (server `page` / `pageSize`)
- **Composable:** `<KPagination total={n} />`, or `usePaginationParams()` when you own chrome yourself
- **Server:** list queries accept optional `page` / `pageSize`; companion `{listField}Count` returns the total
- **Hotkeys** (core keymap): `Alt+,` previous page, `Alt+.` next page

### Search / filter / groupBy

`KSearch` is an Odoo-inspired toolbar (multi-field search, AND/OR filters, nested groupBy, saved templates). Syncs to `?q=` / `?searchIn=` / `?domain=` / `?groupBy=`.

```tsx
<KTable query="hellospec.greetings" paginated searchable />
```

- **Flag:** `searchable` on `KTable` / `KKanban` / `KCollection` — `useKQuery` passes filters into `listModelRecordsPage`
- **Composable:** `<KSearch model="hellospec.greeting" fields={…} />` + `useKSearchParams()`
- **Domain:** JSON Odoo-style (`|` / `&` / `!`, leaves `[field, op, value]`)
- **Templates:** saved per model in localStorage (`listSearchTemplates` / `saveSearchTemplate`)
- **Groups:** `{listField}Groups(groupBy: …)` → `{ values, count }[]` (kanban columns)
- **Hotkeys** (core keymap): `Alt+F` focus search, `Alt+Shift+F` filter, `Alt+G` group by

### Shared query hook

```text
useKQuery → KQueryShell (KSearch + KPagination) → KTableView | KKanbanBoard
```

- Prefer `query` / `model` props; do not hand-fetch in normal app pages.
- Presentational escape hatches: `KTableView`, `KKanbanBoard`, or `KCollection` **`items`** mode for ad-hoc client lists. Prefer a **virtual model** + `query=` (see appman) when the data is a platform registry.
- Agent conventions: root [`AGENTS.md`](../../AGENTS.md) and `.cursor/rules/k-components.mdc`.

```tsx
import { KPagination, usePaginationParams } from '@/k'
import { listModelRecordsPage } from '@/lib'

const { page, pageSize, setPage, setPageSize } = usePaginationParams()
const data = await listModelRecordsPage('hellospec', 'greeting', ['message'], { page, pageSize }, 'hellospecGreetings')
// Prefer useKQuery / KTable for normal pages; this is the low-level escape hatch.
```

## Spec-driven pages

```tsx
import { useState } from 'react'
import { t } from '@/lib'
import { KForm, KFormField, KTable } from '@/k'

export default function GreetingForm() {
  const [refreshToken, setRefreshToken] = useState(0)
  return (
    <>
      <KForm command="hellospec.postGreeting" onsuccess={() => setRefreshToken((n) => n + 1)}>
        <KFormField field="message" label={t('hellospec.create')} />
      </KForm>
      <KTable query="hellospec.greetings" refreshToken={refreshToken} />
    </>
  )
}
```

## Auth and routing

- Go owns sessions (`/auth/*`, GraphQL `RequireAuth`).
- The SPA checks `/auth/me` client-side and mounts Login vs shell.
- `react-router` basename `/app`; menu-driven views resolve via ViewHost + `*.page.tsx` registry (not one Next/file route per Go view).

## Dependencies

Owned by `apps/core/spa/package.json`: `react`, `react-dom`, `react-router`, Radix, `lucide-react`, `motion`, Tailwind v4, Vite + `@vitejs/plugin-react`.

## Dev

```bash
cd apps/core/spa && npm run dev
# or from repo root: make dev
```

Open **http://localhost:5173/app/**. Production: `make spa-build` → Go serves `apps/core/spa/dist` at `/app/*`.
