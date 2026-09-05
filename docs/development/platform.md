# Platform APIs (usage)

How apps use kernel APIs: time, i18n, settings, and the fork-vs-patch model.

Part of [Development](index.md). Driver registration and catalog loading internals: [Internals](../internals/index.md) · [Go SDK](../internals/go-sdk.md).

kaizengo follows a **Linux-like** model: a small platform (“kernel”) exposes stable APIs; **apps** are products that *use* the platform. Customize a product by **forking the app**, not by patching another app’s internals.

## Settings

The **settings** app (`apps/settings`) is the UI for platform + shell knobs:

| Setting | Backed by | Effect |
|---------|-----------|--------|
| Locale | `i18n.SetLocale` | Active catalog for `i18n.T`, GraphQL `i18n`, nav titles, and `dir`/`lang` on `<html>` |
| Default calendar | `config.DefaultCalendar` | Clock uses it when no calendar arg is passed |
| Shell title | `config.ShellTitle` | Core SPA brand; live via `kaizengo:settings` event |

GraphQL: `settings`, `updateSettings(locale, defaultCalendar, shellTitle)`.
Locales are returned as `{ id, name, dir }` — Farsi (`fa`) has `dir: "rtl"`.

## Time / calendars

```go
import ptime "kaizengo/internal/platform/time"

ptime.List()
ptime.Format("persian", time.Now())
```

**Add a calendar driver** (kernel package — see [Internals → project tree](../internals/index.md#project-tree)):

1. Package under `internal/platform/time/<id>/`
2. `init()` → `ptime.Register(yourCalendar)`
3. Blank-import it from `internal/platform/drivers`

Clock already lists registered calendars via GraphQL `clockCalendars` / `clockNow`.

## Localization

**Single source of truth:** user-visible copy lives in gettext `.po` catalogs. SPAs must not hardcode parallel English strings.

### Per-app catalogs

```text
apps/<name>/locale/template.pot   # generated msgid list
apps/<name>/locale/en.po
apps/<name>/locale/fa.po
```

```po
msgid "clock.title"
msgstr "Clock"
```

`make generate` writes `locale/template.pot` (msgid list). Translate in `en.po` / `fa.po`.

Load in the app’s `Setup`:

```go
i18n.MustLoadLocaleDir("apps/clock/locale")
```

Platform/nav strings live in embedded `internal/platform/i18n/locale/*.po`.

### API

```go
import "kaizengo/packages/sdk-go/i18n"

i18n.T("clock.title")
i18n.Tf("welcome.user", name)
i18n.Error("inventory.error.qty_positive")
```

App Go should import `kaizengo/packages/sdk-go/i18n`, not `internal/platform/i18n`. The engine loads each app’s `locale/*.po` at Setup.

GraphQL:

```graphql
query {
  i18n(prefix: "clock.") {
    locale
    dir
    entries { key value }
  }
}
```

Frontend (`@kaizengo/sdk-solid/ui`):

```ts
import { t, setI18nLocale, syncDocumentLocale } from '@kaizengo/sdk-solid/ui'

// Shell boot — settings locale + <html lang dir>
const { locale } = await syncDocumentLocale()
setI18nLocale(locale)

t('clock.title') // from Vite-compiled apps/*/locale/*.po
```

`.po` files stay the source of truth (Go loads them for menus and GraphQL). Vite compiles them into the SPA at build/dev time so `t()` is synchronous. GraphQL `i18n` remains for server-rendered labels (`GET /api/apps`).

### RTL

| Locale | Direction |
|--------|-----------|
| `en` | `ltr` |
| `fa` | `rtl` |

`applyLocale` / `syncDocumentLocale` set `document.documentElement.dir` and `lang`. UI CSS prefers logical properties (`inset-inline-start`, `margin-inline-start`, …) so layouts flip correctly.

### Add a locale

1. Add `apps/<app>/locale/<id>.po` for each app that needs strings
2. Add `internal/platform/i18n/locale/<id>.po` for nav keys
3. `i18n.RegisterLocale(LocaleInfo{ID: id, Name: "…", Dir: RTL|LTR})` in platform init (or a small locale pack package)

## Apps vs platform

| Want… | Do… |
|-------|-----|
| New calendar for everyone | Platform driver + blank-import in `drivers` |
| New language for an app | Add `apps/<app>/locale/<id>.po` (+ platform `nav` .po if needed) |
| Different Clock UX for a customer | Fork `apps/clock` → ship `apps/acme_clock` |
| Patch Clock from another app | **Don’t** — not supported by design |
| Login / sessions / org users | Use `auth` + `identity` + `internal/auth` ([auth.md](../auth.md)) |
| Authorize an action | Use `permissions` / `acl.Authorizer` — see [ACL system](../acl.md) |
| Cross-app behavior (audit, search) | Install an addon app; use [extension points](../extension-platform.md) |
| Share UI for identity users | `@kaizengo/sdk-solid/identity` ([capabilities.md](../capabilities.md)) |

## Extension platform

Cross-app addons register on global extension points instead of patching product apps.

- Design: [extension-platform.md](../extension-platform.md)
- Capability catalog: [capabilities.md](../capabilities.md)
- Usage API: [Go SDK → extension points](../internals/go-sdk.md#global-extension-points)
- Dispatch internals: [Go SDK → extension points](../internals/go-sdk.md#global-extension-points)

## Try it

1. Open **Apps → Settings**: set locale **فارسی (rtl)**, default calendar **Persian** — the shell flips to RTL and labels switch to Farsi.
2. Open **Apps → Clock**: copy comes from `apps/clock/locale/*.po`; Jalali formatting via `clockNow`.
