# Solid apps

## Idea

There is **one central Solid SPA** at `apps/core/spa`. KaizenGo apps contribute individual `.tsx` view files under `apps/<name>/views/`. The core shell resolves menu selections to these views at build time.

```text
apps/core/spa          →  single Vite SPA (/app/)
apps/myapp/views/*.page.tsx →  compiled into the central bundle
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
| `apps/<name>/views/*.tsx` | Other view modules (components, not pages) |
| `apps/<name>/lib/` | Optional shared TS/helpers for that app's views |
| `apps/<name>/app.yaml` | Declarative nav, models, menus |
| `apps/<name>/module.go` | Go setup (nav, GraphQL, services) — no SPA asset routes |

Apps without menus use `apps/<name>/views/Index.page.tsx` as the default page.

## View registry

`apps/core/spa/src/lib/views/registry.ts` discovers all `apps/*/views/**/*.page.tsx` files via Vite's `import.meta.glob` and maps them by `{app}.{view}` (e.g. `identity.Overview`).

Cross-app component exports (`exports.components` in `app.yaml`) are registered in the same file.

## Build

```bash
cd apps/core/spa
npm install
npm run build
```

Or from the repo root:

```bash
make spa-build
```

## Dev

```bash
cd apps/core/spa
npm run dev
```

Open **http://localhost:5173/app/**. Edits to the shell, SDK UI, or any `apps/*/views/*.page.tsx` file hot-reload via Vite.

## UI (`@kaizengo/sdk-solid/ui`)

Shared components, model client, i18n, and layout primitives:

```bash
# packages/sdk-solid/ui is consumed as source (bundled into the central SPA at build time)
```

Import once in the shell entry:

```tsx
import '@kaizengo/sdk-solid/ui/styles.css'
```

Example page:

```tsx
import { KTable, KAppStatus, t } from '@kaizengo/sdk-solid/ui'

export default function GreetingList() {
  return (
    <>
      <KTable model="hellospec.greeting" />
      <KAppStatus />
    </>
  )
}
```

Themes live in `packages/sdk-solid/ui/src/styles/themes/`. Switch at runtime with `setTheme()`.

See [Solid SDK](internals/solid-sdk.md) for the full component list.
