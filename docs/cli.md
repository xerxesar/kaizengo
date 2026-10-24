# CLI

KaizenGo has two command-line surfaces with different jobs:

| Tool | Talks to | Use for |
|------|----------|---------|
| **Runtime CLI** — web terminal, command palette, native `kaizen` | A running server, over GraphQL only | Querying and changing business data, inspecting the catalog, changing [app domains](architecture.md#one-domain-pipeline-base-plus-layers) |
| **`kaizengo`** | The source tree | Scaffolding apps, generating `__types__` and `template.pot` |

Everything except Go hooks can be done from the runtime CLI. Hooks live in compiled apps or [external Go apps](grpc-plugins.md).

## Runtime CLI

The grammar lives on the server (`internal/shell`). A client sends one line to the `platformShell(line: String!)` mutation and prints `output`. Each command compiles to GraphQL documents that run against the live schema with the caller's session, so ACL applies exactly as it does for the SPA.

```graphql
mutation { platformShell(line: "query crm.leads q=acme") { ok output data } }
query { platformShellCommands { name usage summary } }
```

`data` is the raw GraphQL result as a JSON string, for clients that want structured output.

### Commands

| Command | Description |
|---------|-------------|
| `help [command]` | Command list or one command's usage |
| `whoami` | Current user, org, and roles |
| `apps` | Apps with callable operations (compiled or dynamic) |
| `ls [app]` | Queries and commands you are allowed to call |
| `describe <app.op>` | Arguments (GraphQL types) and return fields |
| `query <app.op> [key=value …]` | Run a query |
| `run <app.op> [key=value …]` | Run a command |
| `domain list [compiled\|dynamic]` | Every app's domain with its revision, layer revision, source, and load errors |
| `domain show <app> [--layer \| --rev N]` | The effective domain as YAML (base plus layer); `--layer` prints only the stored layer, `--rev N` one stored revision |
| `domain history <app>` | Base and layer revisions, newest first, with their changes |
| `domain diff <<EOF … EOF` | Changes a document would make |
| `domain apply [--dry-run] <<EOF … EOF` | Validate, migrate (additive), store a layer revision, and hot-mount |
| `domain add-field <app>.<model> <field> <type> [flags]` | Add one field to any app's model (see below) |
| `domain revert <app> <revision> [--dry-run]` | Re-apply the layer as it was at a revision |
| `domain delete <app> [--dry-run]` | Drop the layer: a compiled app returns to its base, a dynamic app is unmounted; tables and history are kept |
| `domain export [app …]` | Back up the current layers as a YAML bundle |
| `domain import [--dry-run] <<EOF … EOF` | Restore a bundle (each app's layer becomes a new revision) |
| `plugins` | External Go apps loaded over gRPC |
| `gql <document> [variables-json]` | Raw GraphQL escape hatch |

Operations are addressed as `app.name` (`crm.leads`) or by GraphQL field (`crmLeads`). Values after `key=` are coerced by the argument's GraphQL type: `Int`, `Float`, `Boolean`, lists as JSON (`["a","b"]`) or comma-separated (`a,b`), everything else as a string. Quote values with spaces: `title="Big deal"`.

| Flag | Effect |
|------|--------|
| `--fields a,b` | Select only these return fields |
| `--all` | Include audit fields (`orgId`, `authorId`, `createdAt`, …) |
| `--json` | Print the GraphQL result as JSON instead of a table |

Multi-line input is only valid as a heredoc: the first line ends in `<<TAG`, the body follows, and a line containing only `TAG` ends it.

```text
domain apply <<EOF
name: crm
title: CRM
summary: Leads defined at runtime
models:
  - name: lead
    fields:
      - { name: title, type: string, required: true }
      - { name: amount, type: number }
queries:
  - { name: leads, list: lead }
commands:
  - { name: addLead, create: lead }
EOF

run crm.addLead title="Acme deal" amount=1200
query crm.leads --fields title,amount
```

#### Adding a field to an existing app

Compiled and dynamic apps work the same way. `domain add-field` reads the effective domain, adds the field, and applies it: the column is created in place (existing rows get the type's zero value), the GraphQL types gain the field, and the change is stored as a new layer revision. No restart is needed.

```text
domain add-field identity.user nickname string --indexed --dry-run
domain add-field identity.user nickname string --indexed
domain add-field crm.lead stage enum --values new,won --required
domain add-field crm.lead owner many2one --relation identity.user
```

| Flag | Effect |
|------|--------|
| `--required` / `--indexed` / `--readonly` | Same as the field keys in `app.yaml` |
| `--values a,b` | Enum values |
| `--relation app.model` | Target of a relation field |
| `--dry-run` | Print the changes without applying |

For larger edits, run `domain show identity`, edit the YAML, then `domain diff` and `domain apply` it. From a shell: `kaizen domain show identity > identity.yaml`, edit, then `kaizen -f identity.yaml domain apply` (the `#` header line is a YAML comment). For a compiled app, everything its code defines must stay as it is: a layer can add models, fields, queries, commands, roles, and policy entries, and override the title and summary. `domain show <app>` leaves out UI wiring (nav, menus, views, locales, keymap) and seeded users; those always come from code.

#### Layers and revisions

An app's domain is its **base** (the compiled `app.yaml`, or nothing for a dynamic app) plus one stored **layer**. Every apply, revert, delete, or import appends a revision, and the engine records a new base revision whenever the compiled spec changes. When code changes, the stored layer is re-applied on top of the new base at boot; if it no longer fits (for example the code now defines the same field differently), the app mounts its base and `domain list` shows the error.

```text
domain history identity
domain show identity --layer
domain revert identity 1            # back to the base: the column is kept, the field leaves the API
domain revert identity 2            # field is back with its data
kaizen domain export > domains.yaml
kaizen -f domains.yaml domain import --dry-run
```

### Web terminal and command palette

Both are part of the core shell (`apps/core/spa/src/components/shell/`) and use `@/lib/platform-shell`.

| Key | Binding id | Opens |
|-----|------------|-------|
| `Alt+K` | `command_palette` | Command palette: apps, catalog operations, and terminal commands, filtered as you type |
| `Alt+T` | `toggle_terminal` | Bottom terminal panel |

Both bindings are declared in `apps/core/keymap.yaml` and can be remapped in Settings → Keymap.

- In the palette, Enter on a query without required arguments runs it in the terminal. Any other operation opens the terminal prefilled (`run crm.addLead `). Enter on an unmatched filter runs the text as a shell line.
- In the terminal, Tab completes command names, `app.op` targets, and argument names from `platformCatalog`. Up and Down walk history (stored in `localStorage`), Ctrl+L or `clear` clears the screen, and Esc closes the panel. Enter inserts a newline while a heredoc is still open.
- After `domain apply`, `add-field`, `revert`, `delete`, or `import`, the terminal refreshes its completion catalog and the Apps menu.

### Native CLI (`kaizen`)

`kaizen` is a thin client over the same mutation. It signs in with email and password, stores the session cookie, and sends lines to `platformShell`.

```bash
make kaizen
# → bin/kaizen

kaizen --db main login --email admin@kaizengo.local     # prompts for the password
kaizen query crm.leads --fields title,amount
kaizen run crm.addLead 'title=Big deal' amount=1200
kaizen -f crm.yaml domain apply --dry-run               # file becomes the heredoc body
kaizen --json query crm.leadsCount
kaizen                                                  # interactive prompt; heredocs span lines
kaizen logout
```

Global flags go before the command:

| Flag / env | Default | Description |
|------------|---------|-------------|
| `--url` / `KAIZEN_URL` | `http://localhost:8080` | Server base URL |
| `--db` / `KAIZEN_DB` | (remembered from login) | Database to select when signing in |
| `-f <file>` | — | Append a file as the heredoc body (`-` for stdin) |
| `--json` | off | Print the GraphQL `data` payload instead of the text output |

`login` also reads `KAIZEN_EMAIL` / `KAIZEN_PASSWORD`. The session (session and database cookies) is saved in the user config directory (`~/.config/kaizen/session.json` on Linux), keyed by URL. Each argument after the command is one token, so shell quoting works as expected. The exit code is 1 when the server reports an error. Piped stdin runs line by line, which makes scripts possible: `kaizen < setup.kz`.

The catalog covers every GraphQL root field. Besides each app's spec operations, helper fields are attributed to their app (`crm.leadsCount`, `crm.leadsGroups groupBy=stage`), and hand-registered fields live under `graphql` (`query graphql.settings`, `run graphql.updateSettings locale=fa`). Input-object arguments take JSON (`input='{"model":"crm.lead","label":"Won"}'`).

## Scaffolding (`kaizengo`)

```bash
make cli
# → bin/kaizengo
```

### `kaizengo new-app`

Bootstraps `apps/<name>/`, writes React view stubs, and registers a blank import in `apps/apps.go`.

```bash
kaizengo new-app <name> [flags]
```

| Flag | Default | Description |
|------|---------|-------------|
| `--type` | `react` | UI type (`react` only; `solid` is accepted as an alias) |
| `--title` | Title-cased name | Apps menu label |
| `--summary` | `"<title> app"` | Manifest summary |
| `--with-graphql` | off | Sample Query field `<pkg>Ping` plus a client call |
| `--event-sourced` | `true` | Scaffold an appspec app with models and migrations (legacy flag name) |
| `--addon` | off | Cross-app addon (extension points, no views) |

```bash
./bin/kaizengo new-app notes --title Notes
make spa-build
go run ./cmd/server
# Sign in → Apps → Notes, or http://localhost:8080/app/notes
```

What gets created:

```text
apps/<name>/
  app.yaml
  module.go              # engine.New one-liner
  hooks.go
  migrations/
  locale/
  views/*.page.tsx       # list/form pages for the core shell
```

Views compile into the central SPA (`apps/core/spa`); there is no per-app Vite bundle.

Follow-ups:

1. Use `@/k` (`KTable`, `KForm`, `useKQuery`) and `@/lib` in views.
2. Edit `apps/<name>/app.yaml` models and views; avoid hand-written services unless you need custom domain rules.
3. Grant the app's resources in permissions for non-admin roles.

To try a domain without scaffolding or restarting, use `domain apply` from the runtime CLI instead.

Name rules: `^[a-z][a-z0-9_]*$`; `core` and `apps` are reserved.

### `kaizengo gen-types [app]`

Generates `apps/<app>/__types__` (Go and TypeScript) and `locale/template.pot` from `app.yaml` and sources. `make generate` runs it for every app.

### `kaizengo version`

Prints the CLI version.
