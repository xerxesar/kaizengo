# Solid SDK

How to write app views and call shared UI / GraphQL clients.

Shell registry, Vite aliases, and GraphQL naming plumbing are covered below and in [Internals](index.md#frontend-data-flow). Go contracts and engine: [Go SDK](go-sdk.md). Workflow: [Development](../development/index.md).

## Idea

There is **one central Solid SPA** at `apps/core/spa`. Apps contribute `.tsx` views under `apps/<name>/views/`. The shell resolves menu selections to those views at build time.

```text
apps/core/spa                 →  single Vite SPA (/app/)
apps/myapp/views/*.page.tsx   →  compiled into the central bundle
```

## Bootstrap

```bash
./bin/kaizengo new-app myapp --type solid
make spa-build
```

## Layout

| Path | Role |
|------|------|
| `apps/<name>/views/<View>.page.tsx` | Menu page (matches `menus.view`) |
| `apps/<name>/views/*.tsx` | Components / partials (not menu targets) |
| `apps/<name>/lib/` | Optional shared TS for that app’s views |
| `apps/<name>/app.yaml` | Nav, models, menus |
| `apps/<name>/module.go` | Go setup — no SPA asset routes |

Apps without menus use `views/Index.page.tsx` as the default page.

## Imports

| Import | Role |
|--------|------|
| `@kaizengo/sdk-solid/ui` | Layout, `KTable` / `KForm`, `t()`, model client, menus |
| `@kaizengo/sdk-solid/ui/styles.css` | Theme tokens (once in core `main.ts`) |
| `@kaizengo/sdk-solid/identity` | `fetchUsers`, `UserPicker` |
| `@kaizengo/sdk-solid/search` | `SearchBar`, `searchQuery` |

Core already aliases `@kaizengo/sdk-solid` to `packages/sdk-solid`. Optional per-app Vite entries use `file:../../../packages/sdk-solid`.

## Spec-driven pages

```tsx
import { KTable, KAppStatus, t } from '@kaizengo/sdk-solid/ui'

export default function GreetingList() {
  return (
    <>
      <KTable model="hellospec.greeting" emptyMessage={t('hellospec.empty')} />
      <KAppStatus />
    </>
  )
}
```

Form + refresh:

```tsx
import { KForm, KFormField, KTable, t } from '@kaizengo/sdk-solid/ui'

export default function GreetingForm() {
  let table: { refresh: () => Promise<void> } | undefined
  return (
    <>
      <KForm model="hellospec.greeting" onsuccess={() => void table?.refresh()}>
        <KFormField field="message" label={t('hellospec.create')} />
      </KForm>
      <KTable ref={(el) => (table = el)} model="hellospec.greeting" />
    </>
  )
}
```

`KTable` / `KForm` load `{app}Views` and call list/create/update/delete automatically. Imperative helpers when you need custom UI:

```ts
import {
  listModelRecords,
  createModelRecord,
  fetchModelViews,
  listViewForModel,
} from '@kaizengo/sdk-solid/ui'

const views = await fetchModelViews('hellospec')
const list = listViewForModel(views, 'greeting')
const rows = await listModelRecords('hellospec', 'greeting')
await createModelRecord('hellospec', 'greeting', { message: 'Hello, world' })
```

Requests go to `/graphql` with `credentials: 'include'`. Field names for app `hellospec`, model `greeting`:

| Operation | Field |
|-----------|-------|
| List | `hellospecGreetings` |
| Create | `createHellospecGreeting` |
| Views / Menus | `hellospecViews` / `hellospecMenus` |

→ GraphQL registration internals: [Go SDK](go-sdk.md) · naming conventions in [Go SDK → GraphQL naming](go-sdk.md#graphql-naming-convention).

## i18n

```tsx
import { t } from '@kaizengo/sdk-solid/ui'

<h1>{t('hellospec.title')}</h1>
```

`make generate` harvests static `t('…')` keys into `locale/template.pot`. Vite compiles `.po` files into the shell. Go uses the same catalogs via `packages/sdk-go/i18n` — see [Platform APIs → Localization](../development/platform.md#localization).

## Identity & search

```ts
import { fetchActiveUsers, UserPicker } from '@kaizengo/sdk-solid/identity'
import { SearchBar, searchQuery } from '@kaizengo/sdk-solid/search'

const users = await fetchActiveUsers()
const hits = await searchQuery('hello', { collections: ['hellospec.greeting'] })
```

Identity needs `uses: [identity.users]`. Search needs model `search:` in YAML and the Typesense (or memory) backend. Contracts: [capabilities.md](../capabilities.md).

## Menus and routing helpers

```ts
import {
  fetchAppMenus,
  navigateApp,
  contentAppForMenu,
  fetchViewSlots,
  KViewSlots,
} from '@kaizengo/sdk-solid/ui'
```

- `fetchAppMenus(app)` → `{app}Menus` (local + `exports.menus`)
- Shell URL: `/app/{hostApp}/{page}`
- Declaring menus / contributions: [Go SDK → navigation](go-sdk.md#navigation-and-menus)

## Theming

Once in core `main.ts`:

```ts
import '@kaizengo/sdk-solid/ui/styles.css'
import { initTheme } from '@kaizengo/sdk-solid/ui'

initTheme('carbon')
```

Themes live under `packages/sdk-solid/ui/src/styles/themes/`. Switch at runtime with `setTheme()`.

## Dev

```bash
cd apps/core/spa && npm run dev
# or from repo root: make dev
```

Open **http://localhost:5173/app/**. Edits to the shell, SDK UI, or any `apps/*/views/*.page.tsx` hot-reload. Full command table: [Workflow](../development/workflow.md).
