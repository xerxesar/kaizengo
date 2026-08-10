# Platform (kernel)

kaizengo follows a **Linux-like** model: a small platform (“kernel”) exposes stable APIs and extension points; **apps** are products that *use* the platform. Customize a product by **forking the app**, not by patching another app’s internals (unlike Odoo `_inherit`).

## Layout

```text
internal/platform/
  time/           # calendar registry (extension point)
    gregorian.go  # built-in driver
    persian/      # optional Jalali driver (blank-import)
  i18n/           # locale catalog stub
  config/         # process-wide defaults (locale calendar, shell title)
  drivers/        # composition: blank-imports optional drivers
```

[`cmd/server/main.go`](../cmd/server/main.go) loads drivers:

```go
_ "kaizengo/internal/platform/drivers"
```

Apps never import each other to “extend” UI or models.

## Settings

The **settings** app (`apps/settings`) is the UI for platform + shell knobs:

| Setting | Backed by | Effect |
|---------|-----------|--------|
| Locale | `i18n.SetLocale` | `i18n.T` strings (Clock, Settings labels, …) |
| Default calendar | `config.DefaultCalendar` | Clock uses it when no calendar arg is passed |
| Shell title | `config.ShellTitle` | Core SPA brand; live via `kaizengo:settings` event |

GraphQL: `settings`, `updateSettings(locale, defaultCalendar, shellTitle)`.

## Time / calendars

```go
import ptime "kaizengo/internal/platform/time"

ptime.List()
ptime.Format("persian", time.Now())
```

**Add a calendar driver** (kernel module):

1. Package under `internal/platform/time/<id>/`
2. `init()` → `ptime.Register(yourCalendar)`
3. Blank-import it from `internal/platform/drivers`

Clock does **not** change. It already lists whatever calendars are registered and formats via GraphQL `clockCalendars` / `clockNow`.

## i18n stub

```go
i18n.T("clock.title")
i18n.Register("fa", map[string]string{"clock.title": "ساعت"})
i18n.SetLocale("fa")
```

Locale packs are platform drivers too — register messages in `init`, don’t monkey-patch apps.

## Apps vs platform

| Want… | Do… |
|-------|-----|
| New calendar for everyone | Platform driver + blank-import in `drivers` |
| Different Clock UX for a customer | Fork `apps/clock` → ship `apps/acme_clock` |
| Patch Clock from another app | **Don’t** — not supported by design |

## Try it

1. Open **Apps → Settings**: set locale `fa`, default calendar **Persian**, change shell title — brand updates immediately.
2. Open **Apps → Clock**: calendar select starts from the default; pick **Persian (Jalali)** for Jalali formatting via `clockNow`.
