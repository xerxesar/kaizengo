# CLI (`kaizengo`)

## Build

```bash
make cli
# → bin/kaizengo
```

## `kaizengo new-app`

Bootstraps `apps/<name>/`, writes UI stubs, and registers a blank import in `apps/apps.go`.

```bash
kaizengo new-app <name> [flags]
```

| Flag | Default | Description |
|------|---------|-------------|
| `--type` | `vanilla` | `vanilla` or `svelte` |
| `--title` | Title-cased name | Apps menu label |
| `--summary` | `"<title> app"` | Manifest summary |
| `--with-graphql` | off | Sample Query field `<pkg>Ping` + client call |

### Examples

Vanilla module (no build step):

```bash
./bin/kaizengo new-app notes
```

Svelte single-component app:

```bash
./bin/kaizengo new-app notes --type svelte
cd apps/notes/spa && npm install && npm run build
```

Svelte + GraphQL sample:

```bash
./bin/kaizengo new-app notes --type svelte --with-graphql --title "Notes"
cd apps/notes/spa && npm install && npm run build
go run ./cmd/server
# Apps → Notes, or http://localhost:8080/app/notes
```

### What gets created

**Vanilla**

```text
apps/<name>/
  module.go
  spa/spa.js          # served as /app-assets/<name>/spa.js
```

**Svelte**

```text
apps/<name>/
  module.go
  spa/
    App.svelte
    main.ts           # mount/unmount adapter
    vite.config.ts    # library build → dist/spa.js
    package.json
    …
```

### Name rules

- Lowercase letter start: `^[a-z][a-z0-9_]*$`
- Reserved: `core`, `apps`

## `kaizengo version`

Prints the CLI version.
