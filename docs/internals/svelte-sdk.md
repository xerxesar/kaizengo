# Svelte SDK (`packages/sdk-svelte`)

Shared UI and GraphQL clients for the central shell and app views. Consumed as **source** (not a published npm build); Vite aliases resolve it from the monorepo.

## Package exports

From `packages/sdk-svelte/package.json`:

| Import | Role |
|--------|------|
| `@kaizengo/sdk-svelte/ui` | Layout, `KTable` / `KForm`, i18n `t()`, menu helpers, model client |
| `@kaizengo/sdk-svelte/ui/styles.css` | Theme tokens (imported once in core `main.ts`) |
| `@kaizengo/sdk-svelte/identity` | `fetchUsers`, `UserPicker` |
| `@kaizengo/sdk-svelte/search` | `SearchBar`, `searchQuery` |
| `@kaizengo/sdk-svelte/spa-config/app-vite.js` | Shared Vite config for optional per-app library builds |

In an app’s `package.json` (when it has its own Vite entry):

```json
"@kaizengo/sdk-svelte": "file:../../../packages/sdk-svelte"
```

Core SPA already aliases `@kaizengo/sdk-svelte` to `packages/sdk-svelte`.

## Views in the shell

App pages are plain Svelte files under `apps/<name>/views/`:

| Pattern | Meaning |
|---------|---------|
| `Foo.page.svelte` | Menu-mountable page (`view: Foo`) |
| other `*.svelte` | Components / partials (not menu targets) |

The registry glob in core maps them to `{app}.{name}` keys and resolves menu leaves (including cross-app `component` exports).

## UI kit

Import components from `@kaizengo/sdk-svelte/ui`:

```svelte
<script lang="ts">
  import { Layout, LayoutMain, Button, Table, t } from '@kaizengo/sdk-svelte/ui'
</script>
```

Core owns `<Page>`; app views use `<Layout>` / `<LayoutMain>` only. Prefer left-aligned layouts; `variant="centered"` is for auth-style screens.

### Spec-driven CRUD components

`KTable` and `KForm` bind to namespaced models (`app.model`). They load `{app}Views` metadata and call list/create/update/delete mutations automatically.

**List** (`apps/hellospec/views/GreetingList.page.svelte`):

```svelte
<script lang="ts">
  import { KTable, KAppStatus, t } from '@kaizengo/sdk-svelte/ui'
</script>

<KTable model="hellospec.greeting" emptyMessage={t('hellospec.empty')} />
<KAppStatus />
```

**Form + refresh** (`GreetingForm.page.svelte`):

```svelte
<script lang="ts">
  import { KForm, KFormField, KTable, KAppStatus, t } from '@kaizengo/sdk-svelte/ui'

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
} from '@kaizengo/sdk-svelte/ui'

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
  import { t } from '@kaizengo/sdk-svelte/ui'
</script>

<h1>{t('hellospec.title')}</h1>
```

`make generate` harvests static `t('…')` keys into `locale/template.pot`. Vite’s `poCatalogPlugin` (via spa-config) compiles `.po` files into the shell; locale switches update the UI without a reload. Go uses the same catalogs through `packages/sdk-go/i18n`.

## Identity & search clients

Capability-facing packages wrap GraphQL so product apps do not hardcode identity queries:

```ts
import { fetchActiveUsers, UserPicker } from '@kaizengo/sdk-svelte/identity'

const users = await fetchActiveUsers()
```

```ts
import { SearchBar, searchQuery } from '@kaizengo/sdk-svelte/search'

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
} from '@kaizengo/sdk-svelte/ui'
```

- `fetchAppMenus(app)` → `{app}Menus` tree (local menus + `exports.menus` contributions)
- `contentAppForMenu(item, hostApp)` → which app bundle/view owns a contributed leaf
- Shell URL shape: `/app/{hostApp}/{page}`

View slots from addons:

```ts
import { fetchViewSlots, KViewSlots } from '@kaizengo/sdk-svelte/ui'

const slots = await fetchViewSlots('hellospec', 'GreetingList')
```

## Theming

Once in core `main.ts`:

```ts
import '@kaizengo/sdk-svelte/ui/styles.css'
import { initTheme } from '@kaizengo/sdk-svelte/ui'

initTheme('carbon')
```

Themes live under `packages/sdk-svelte/ui/src/styles/themes/`. Switch at runtime with `setTheme()`.

## Related

- [Svelte apps](../svelte.md) — central SPA layout and build
- [Go SDK](go-sdk.md) — what those GraphQL fields come from
- [Capabilities](../capabilities.md) — `identity.users`, search contracts
