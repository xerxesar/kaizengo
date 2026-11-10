# CLI

KaizenGo has two command-line surfaces with different jobs:

| Tool | Talks to | Use for |
|------|----------|---------|
| **Runtime CLI** — web terminal, command palette, native `kaizen` | A running server, over GraphQL only | Querying and changing business data, inspecting the catalog, changing [app domains](architecture.md#one-domain-pipeline-tip-revisions-git-like) |
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
| `help [command]` | Grouped command list or one command's usage |
| `man <command> [domain-sub]` | Longer manual (SYNOPSIS / DESCRIPTION / EXAMPLES) |
| `whoami` | Current user, org, and roles |
| `apps` | Apps with callable operations (compiled or dynamic) |
| `ls [app]` | Queries and commands you are allowed to call |
| `describe <app.op>` | Arguments (GraphQL types) and return fields |
| `query <app.op> [key=value …]` | Run a query |
| `run <app.op> [key=value …]` | Run a command |
| `domain list [compiled\|dynamic]` | Every app's domain with tip revision, source, and load errors |
| `domain show <app> [--rev N]` | Tip domain as YAML; `--rev N` one stored revision |
| `domain show-view\|show-menu\|show-query\|show-command\|show-model\|show-policy <app> <name>` | One tip fragment as YAML |
| `domain history <app>` | Tip revisions, newest first, with their changes |
| `domain conflicts <app>` | Pending tip-vs-code conflict paths after an upgrade |
| `domain rebase <app> [--force] [--dry-run]` | Three-way merge tip onto pending/current code; `--force` takes code on conflicts |
| `domain diff <<EOF … EOF` | Changes a document would make |
| `domain apply [--dry-run] [--force] <<EOF … EOF` | Validate, migrate, store a tip snapshot, and hot-mount |
| `domain apply <app> -ted` | Open web vim editor seeded with that app's tip |
| `domain add-field` / `set-field` / `remove-field` | Create, replace (incl. `--name` rename), or omit a model field (`--force` drops the column) |
| `domain add-index` / `set-index` / `remove-index` | Index helpers (`--force` drops the index) |
| `domain add-view` / `set-view` / `remove-view` | Path or dynamic views (including editing compiled-origin forms) |
| `domain set-view\|set-menu\|set-query\|set-command … -ted` | Web terminal: vim YAML editor prefilled with the current fragment |
| `domain add-menu` / `set-menu` / `remove-menu` | Menu helpers |
| `domain add-query` / `set-query` / `remove-query` | Query helpers |
| `domain add-command` / `set-command` / `remove-command` | Command helpers |
| `domain revert <app> <revision> [--dry-run]` | Append a tip equal to that revision |
| `domain delete <app> [--dry-run]` | Soft-clear tip: compiled remounts code, dynamic unmounts; tables and history kept |
| `domain export [app …]` | Back up runtime tips as a YAML bundle |
| `domain import [--dry-run] <<EOF … EOF` | Restore a bundle (each tip becomes a new revision) |
| `plugins` | External Go apps loaded over gRPC |
| `gql <document> [variables-json]` | Raw GraphQL escape hatch |

Operations are addressed as `app.name` (`crm.leads`) or by GraphQL field (`crmLeads`). Values after `key=` are coerced by the argument's GraphQL type: `Int`, `Float`, `Boolean`, lists as JSON (`["a","b"]`) or comma-separated (`a,b`), everything else as a string. Quote values with spaces: `title="Big deal"`.

| Flag | Effect |
|------|--------|
| `--fields a,b` | Select only these return fields |
| `--all` | Include audit fields (`orgId`, `authorId`, `createdAt`, …) |
| `--json` | Print the GraphQL result as JSON instead of a table |

Primary multi-line bodies use a heredoc: the first line ends in `<<TAG`, the body follows, and a line containing only `TAG` ends it. Flag values use `=` (bash-like quotes); they may span lines inside open quotes or with a trailing `\`, but when a heredoc is open on the first line, later newlines belong to that body — so `--migrate=` must be closed on the first line if you also pass a tip heredoc.

In the **web terminal**, `-ted` / `--ted` on `set-view`, `set-menu`, `set-query`, `set-command`, or `apply <app>` opens a CodeMirror vim editor seeded with the current YAML. Save with `:wq` (or Apply); cancel with `:q`. Native `kaizen` prints the seed and expects a heredoc instead.

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

#### Editing any app's domain (including compiled)

Compiled and dynamic apps work the same way: helpers read the tip, patch it, and apply a new full snapshot. Soft removes omit entities from the API but keep Postgres objects; pass `--force` to `DROP` columns/indexes/junctions.

```text
domain add-field hellospec.greeting okok string --indexed
domain set-field hellospec.greeting okok string --name ok   # tip + default migrate-pre RENAME
domain set-view hellospec GreetingForm <<EOF
layout: column
components:
  - id: form
    kind: form
    model: greeting
    groups:
      - labelKey: hellospec.menu.group
        columns: 2
        fields:
          - { field: message, span: 2, widget: textarea, order: 1 }
          - { field: mood, span: 1, order: 2 }
          - { field: internalNote, span: 1, order: 3 }
          - { field: ok, span: 1, order: 4 }
EOF
domain remove-field hellospec.greeting ok          # soft: column kept
domain remove-field hellospec.greeting ok --force  # DROP COLUMN
domain add-index crm.lead stage,title --unique --name lead_stage
domain add-view crm LeadBoard collection --model lead <<EOF
defaultView: table
views: [table, kanban]
columns:
  - field: title
    width: 2fr
    order: 1
EOF
```

| Flag | Effect |
|------|--------|
| `--required` / `--indexed` / `--unique` / `--readonly` | Same as the field keys in `app.yaml` |
| `--on-delete restrict\|setnull\|cascade` | `many2one` foreign-key action (`setnull` is stored as `set null`) |
| `--values a,b` | Enum values |
| `--relation app.model` | Target of a relation field |
| `--compute expr` / `--related path` / `--depends a,b` | Computed field, related path, or dependency list |
| `--store` / `--no-store` | Persist a computed field, or fill it on read |
| `--name` | On `set-field`: rename the field (rewrites view/search/index refs; default `--migrate-pre` RENAME). On indexes: index name |
| `--method btree\|hash\|gin` | Index method |
| `--include a,b` | Covering columns on a btree index |
| `--where "col = 'v'"` | Partial index predicate (`column = value` or `column <> value`, joined with `AND`) |
| `--model` / `--path` | Dynamic view model or hardcoded page path |
| `--dry-run` | Print the changes without applying |
| `--force` | Real DDL drops / allow type rewrites on apply |
| `--migrate-pre=SQL` | Data migration SQL **before** additive DDL (e.g. `RENAME COLUMN`) |
| `--migrate=SQL` / `--migrate-post=SQL` | Data migration SQL **after** additive DDL (backfill); `--migrate` is an alias of `--migrate-post` |
| `--migrate-go <name>` | Compiled Go migration (post phase) via `Domains.RegisterDomainMigration` |
| `--no-migrate` | Skip the data-migration prompt (accept empty/default columns / soft omit) |

Apply order: **migrate-pre → additive DDL → migrate-post / migrate-go → force drops**. Field adds/removes/type changes require an explicit migration choice unless you use `set-field --name` (which defaults to a pre `RENAME COLUMN`). Prefer `--migrate="…"` for post backfills (open quotes or `\` for multi-line SQL).

```text
# rename field in tip + DB (default migrate-pre)
domain set-field hellospec.greeting okok string --name ok

# backfill after adding a column (post)
domain add-field hellospec.greeting ok string --migrate="UPDATE greetings SET ok = okok || '$' WHERE ok = '';"

# multi-line post SQL
domain add-field hellospec.greeting ok string --migrate="
UPDATE greetings SET ok = okok || '$' WHERE ok = '';
"

# or accept the empty/default column
domain add-field hellospec.greeting ok string --no-migrate

# tip heredoc + explicit pre rename
domain apply --migrate-pre="ALTER TABLE greetings RENAME COLUMN okok TO ok;" <<EOF
name: hellospec
…
EOF
```

View/menu-only tip edits do not ask for migrate. For larger edits, run `domain show hellospec`, edit the YAML, then `domain diff` and `domain apply` (with `--migrate-pre` / `--migrate` / `--no-migrate` when models change). Go hook `extends` and component exports stay in code.

#### Revisions (git-like)

Each apply appends a **revision**: a full tip snapshot. `domain history` is `git log`; `domain show --rev N` is `git show`; `domain revert N` appends a new tip equal to that snapshot.

When you upgrade KaizenGo and compiled `app.yaml` changed under a tip that already has runtime edits, boot three-way-merges against the last compiled/rebase baseline. Clean paths apply automatically; real conflicts quarantine the app on its old tip. Incoming code is **not** tip until rebase:

```text
domain list                          # rebaseStatus / conflictCount
domain conflicts hellospec
domain rebase hellospec --dry-run
domain rebase hellospec              # fails while conflicts remain
domain rebase hellospec --force      # take code on every conflict path
domain revert hellospec 1            # always available escape hatch
```

Visual resolution: **Settings → Domains**. The web terminal navigates there when a domain command needs migrate or hits rebase conflicts. A migrate gate also stashes the proposed tip server-side (`platformDomainMigratePending`), so opening Domains from the menu (or after native CLI) still fills **Data migrate**. Conflicts are three-way (ancestor / tip / code) with per-path Keep tip or Take code. Data migrate offers guided actions (rename / backfill / skip / custom) plus advanced pre/post SQL. Step-by-step test scenarios: [Domains UI](development/domains-ui.md).

```text
domain history hellospec
domain show hellospec --rev 2
domain revert hellospec 1
kaizen domain export > domains.yaml
kaizen -f domains.yaml domain import --dry-run
```

### Web terminal and command palette

Both are part of the core shell (`apps/core/spa/src/components/shell/`) and use `@/lib/platform-shell`.

| Key | Binding id | Opens |
|-----|------------|-------|
| `Alt+K` | `command_palette` | Command palette: apps, catalog operations, and terminal commands, filtered as you type |
| `Alt+T` | `toggle_terminal` | Bottom terminal panel |
| `Alt+Shift+T` | `terminal_fullscreen` | Toggle terminal fullscreen (also a maximize button in the panel) |

Both bindings are declared in `apps/core/keymap.yaml` and can be remapped in Settings → Keymap.

- In the palette, Enter on a query without required arguments runs it in the terminal. Any other operation opens the terminal prefilled (`run crm.addLead `). Enter on an unmatched filter runs the text as a shell line.
- Terminal UI state (open, height, fullscreen, scrollback, draft input) persists in `localStorage` (`kaizengo:terminal.state`) across refresh. Drag the top edge to resize.
- Tab completes commands, domain subcommands (`show-view`, `set-view`, …), `app.op` targets, tip fragment names, and common flags (`-ted`, migrate flags). Up/Down walk command history (`kaizengo:terminal.history`). Ctrl+L or `clear` clears scrollback. Esc leaves fullscreen first, then closes. Enter inserts a newline while input is pending (open heredoc, unclosed quote, or trailing `\`).
- `help` groups commands; `man domain` / `man domain set-view` print longer docs.
- After `domain apply`, `add-field`, `set-view`, `revert`, `delete`, or `import`, the terminal refreshes its completion catalog and the Apps menu.

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
kaizen                                                  # interactive prompt; heredoc / quotes / \ span lines
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
