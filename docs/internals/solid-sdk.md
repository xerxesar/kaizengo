# Solid SDK (`packages/sdk-solid`)

Shared UI and GraphQL clients for the central shell and app views. Consumed as **source** (not a published npm build); Vite aliases resolve it from the monorepo.

## Package exports

From `packages/sdk-solid/package.json`:

| Import | Role |
|--------|------|
| `@kaizengo/sdk-solid/ui` | Layout, `KTable` / `KForm`, i18n `t()`, menu helpers, model client |
| `@kaizengo/sdk-solid/ui/styles.css` | Theme tokens (imported once in core `main.ts`) |
| `@kaizengo/sdk-solid/identity` | `fetchUsers`, `UserPicker` |
| `@kaizengo/sdk-solid/search` | `SearchBar`, `searchQuery` |
| `@kaizengo/sdk-solid/spa-config/app-vite.js` | Shared Vite config for optional per-app library builds |

In an app’s `package.json` (when it has its own Vite entry):

```json
"@kaizengo/sdk-solid": "file:../../../packages/sdk-solid"
```

Core SPA already aliases `@kaizengo/sdk-solid` to `packages/sdk-solid`.

## Views in the shell

App pages are plain Svelte files under `apps/<name>/views/`:

| Pattern | Meaning |
|---------|---------|
| `Foo.page.tsx` | Menu-mountable page (`view: Foo`) |
| other `*.tsx` | Components / partials (not menu targets) |

The registry glob in core maps them to `{app}.{name}` keys and resolves menu leaves (including cross-app `component` exports).

## UI kit

Import components from `@kaizengo/sdk-solid/ui`:

```svelte
<script lang="ts">
  import { Layout, LayoutMain, Button, Table, t } from '@kaizengo/sdk-solid/ui'
</script>
```

Core owns `<Page>`; app views use `<Layout>` / `<LayoutMain>` only. Prefer left-aligned layouts; `variant="centered"` is for auth-style screens.

### Table cells

`Table` columns take plain text via `render`, or custom markup via a `cell` snippet (same idea as the `actions` snippet):

```svelte
<script lang="ts">
  import { Badge, Table, type Column } from '@kaizengo/sdk-solid/ui'

  const columns: Column<Row>[] = $derived([
    { key: 'name', label: 'Name', render: (r) => r.name },
    { key: 'status', label: 'Status', cell: statusCell },
  ])
</script>

{#snippet statusCell(row: Row)}
  <Badge variant={row.ok ? 'success' : 'muted'}>{row.ok ? 'on' : 'off'}</Badge>
{/snippet}

<Table {columns} {rows} />
```

### Spec-driven CRUD components

`KTable` and `KForm` bind to namespaced models (`app.model`). They load `{app}Views` metadata and call list/create/update/delete mutations automatically.

**List** (`apps/hellospec/views/GreetingList.page.tsx`):

```svelte
<script lang="ts">
  import { KTable, KAppStatus, t } from '@kaizengo/sdk-solid/ui'
</script>

<KTable model="hellospec.greeting" emptyMessage={t('hellospec.empty')} />
<KAppStatus />
```

**Form + refresh** (`GreetingForm.page.tsx`):

```svelte
<script lang="ts">
  import { KForm, KFormField, KTable, KAppStatus, t } from '@kaizengo/sdk-solid/ui'

  let table = $state<{ refresh: () => Promise<void> }>()
  function onFormSuccess() {
    void table?.refresh()
  }
</script>

<KForm model="hellospec.greeting" onsuccess={onFormSuccess}>
  <KFormField
    field="message"
    label={t('hellospec.create')}
    placeholder={t('hellospec.new_placeholder')}
  />
</KForm>

<KTable bind:this={table} model="hellospec.greeting" class="mt-4" />
```

### Model client (imperative GraphQL)

When you need custom UI, use the same naming helpers the smart components use:

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

Requests go to `/graphql` with `credentials: 'include'` (session cookie).

Naming for app `hellospec`, model `greeting`:

| Operation | Field |
|-----------|-------|
| List | `hellospecGreetings` |
| Get | `hellospecGreeting` |
| Create | `createHellospecGreeting` |
| Update | `updateHellospecGreeting` |
| Delete | `deleteHellospecGreeting` |
| Views | `hellospecViews` |
| Menus | `hellospecMenus` |

## i18n

```svelte
<script lang="ts">
  import { t } from '@kaizengo/sdk-solid/ui'
</script>

<h1>{t('hellospec.title')}</h1>
```

`make generate` harvests static `t('…')` keys into `locale/template.pot`. Vite’s `poCatalogPlugin` (via spa-config) compiles `.po` files into the shell; locale switches update the UI without a reload. Go uses the same catalogs through `packages/sdk-go/i18n`.

## Identity & search clients

Capability-facing packages wrap GraphQL so product apps do not hardcode identity queries:

```ts
import { fetchActiveUsers, UserPicker } from '@kaizengo/sdk-solid/identity'

const users = await fetchActiveUsers()
```

```ts
import { SearchBar, searchQuery } from '@kaizengo/sdk-solid/search'

const hits = await searchQuery('hello', { collections: ['hellospec.greeting'] })
```

Search requires the Typesense app (or memory backend) and model `search:` config in YAML. Identity requires the identity app and `uses: [identity.users]`.

## Menus and routing helpers

```ts
import {
  fetchAppMenus,
  navigateApp,
  resolveMenuSelection,
  contentAppForMenu,
} from '@kaizengo/sdk-solid/ui'
```

- `fetchAppMenus(app)` → `{app}Menus` tree (local menus + `exports.menus` contributions)
- `contentAppForMenu(item, hostApp)` → which app bundle/view owns a contributed leaf
- Shell URL shape: `/app/{hostApp}/{page}`

View slots from addons:

```ts
import { fetchViewSlots, KViewSlots } from '@kaizengo/sdk-solid/ui'

const slots = await fetchViewSlots('hellospec', 'GreetingList')
```

## Theming

Once in core `main.ts`:

```ts
import '@kaizengo/sdk-solid/ui/styles.css'
import { initTheme } from '@kaizengo/sdk-solid/ui'

initTheme('carbon')
```

Themes live under `packages/sdk-solid/ui/src/styles/themes/`. Switch at runtime with `setTheme()`.

## Related

- [Solid apps](../solid.md) — central SPA layout and build
- [Go SDK](go-sdk.md) — what those GraphQL fields come from
- [Capabilities](../capabilities.md) — `identity.users`, search contracts
