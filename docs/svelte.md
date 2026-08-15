# Svelte apps

## Idea

There is **one central Svelte app** at `apps/core/spa`. KaizenGo apps contribute individual `.svelte` view files under `apps/<name>/views/`. The core shell resolves menu selections to these views at build time.

```text
apps/core/spa          →  single Vite SPA (/app/)
apps/myapp/views/*.page.svelte →  compiled into the central bundle
```

## Bootstrap

```bash
./bin/kaizengo new-app myapp --type svelte
make spa-build
```

## Layout

| Path | Role |
|------|------|
| `apps/<name>/views/<View>.page.svelte` | Menu page (matches `menus.view`) |
| `apps/<name>/views/*.svelte` | Other view modules (components, not pages) |
| `apps/<name>/lib/` | Optional shared TS/helpers for that app's views |
| `apps/<name>/app.yaml` | Declarative nav, models, menus |
| `apps/<name>/module.go` | Go setup (nav, GraphQL, services) — no SPA asset routes |

Apps without menus use `apps/<name>/views/Index.page.svelte` as the default page.

## View registry

`apps/core/spa/src/lib/views/registry.ts` discovers all `apps/*/views/**/*.svelte` files via Vite's `import.meta.glob` and maps them by `{app}.{view}` (e.g. `identity.Overview`). Files ending in `.page.svelte` are pages; other `.svelte` files under `views/` are components.

Cross-app component exports (`exports.components` in `app.yaml`) are registered in the same file.

## Build

```bash
cd apps/core/spa
npm install
npm run build
# or from repo root:
make spa-build
```

Only the core SPA is built. App views are bundled into it automatically.

## Dev

```bash
make dev
```

Open **http://localhost:5173/app/**. Edits to the shell, SDK UI, or any `apps/*/views/*.svelte` file hot-reload via Vite.

## UI (`@kaizengo/sdk-svelte/ui`)

Shared Svelte components for Odoo-style views — pages, tabs, tables, forms, modals, tree views.

```bash
# packages/sdk-svelte/ui is consumed as source (bundled into the central SPA at build time)
```

In `apps/core/spa/src/main.ts`:

```ts
import '@kaizengo/sdk-svelte/ui/styles.css'
```

In app views:

```svelte
<script lang="ts">
  import { Layout, LayoutMain, Table, Button } from '@kaizengo/sdk-svelte/ui'
</script>
```

See `apps/identity/views/` for a full enterprise example.

## i18n

Keep `apps/<name>/locale/*.po` as translations. `make generate` writes `locale/template.pot` from `app.yaml` and static `t('…')` keys. Vite compiles the `.po` files into the shell at build/dev time (`virtual:kaizengo-i18n`); `t('myapp.title')` is synchronous and updates when the locale changes.

```svelte
<script lang="ts">
  import { t } from '@kaizengo/sdk-svelte/ui'
</script>

<h1>{t('myapp.title')}</h1>
```

Go still loads the same `.po` files for menu labels and the GraphQL `i18n` query.

## Standard view structure

The **core shell** owns `<Page>`; app views use `<Layout>` only (no nested `<Page>`). See the previous layout contract in this doc's history — `Layout`, `LayoutMenu`, `LayoutMain`, etc.

App pages are **left-aligned** by default. Use `Layout variant="centered"` only for auth screens.

## Theming

Initialize at startup in `main.ts`:

```ts
import { initTheme } from '@kaizengo/sdk-svelte/ui'
initTheme('carbon')
```

Themes live in `packages/sdk-svelte/ui/src/styles/themes/`. Switch at runtime with `setTheme()`.
