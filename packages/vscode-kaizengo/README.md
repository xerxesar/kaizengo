# KaizenGo VS Code extension

Developer tooling for KaizenGo apps: a visual spec view, navigation in specs, and completion for models, fields, queries, and commands.

## What it does

In any `apps/<name>/app.yaml`, the editor title has a **Vis** button (top right). It opens a panel beside the spec:

```text
Vis
----
spec tree
  view layouts (column / row)
  models, queries, commands, menus
----
```

Each block is the screen layout from `views`: nested rows and columns, and the fields or columns inside a component. Click a block to select it in the YAML. **column** / **row** on a region rewrites that layout direction.

Problems in a view (unknown component, field, model, or kind) show in the panel and as squiggles in the spec.

**F12** / **Ctrl+click** (Go to Definition) and hover also resolve:

| You click | Opens |
|-----------|--------|
| `depends` app name | that app's `app.yaml` |
| `uses` capability | the `provides` entry (or platform source for `platform.*`) |
| `menus.view` | `views/<Name>.page.tsx` |
| `models:` path | `models/<name>/spec.yaml` |
| `models` name | generated `__types__/<model>.go` |
| `relation` / `inverse` | the related model or field in YAML |
| view `model`, column `field`, `groupBy` | the model or field |
| layout `component` | that component's `id` |
| view `path` | the compiled page |
| `labelKey` | `msgid` in `locale/*.po` (and platform nav catalogs) |
| `locales` id | `locale/<id>.po` |
| `extends.handler` | `RegisterNamed("…")` in the app's Go code |
| `exports.components.module` | the Solid/TS module |
| `exports.*.app` | the target app's `app.yaml` |

**Shift+F12** (Find All References) lists the other side: `uses` of a capability, `relation` fields of a model, menus that point at a view.

## Completion

Suggestions come from every app's `app.yaml` / `models/*/spec.yaml`. Trigger them with **Ctrl+Space**, or by typing `"` / `'` / `.` inside a value. In Go and TSX, string suggestions are turned on so names complete as you type.

| Where | Completes |
|-------|-----------|
| `relation`, `search.collection`, `queries` / `commands` model keys, security `resource` | model names (`role`, `identity.user`, `hellospec.greeting`) |
| `inverse`, `search.fields`, security `fields` / `domain`, `type`, enum `default` | fields, field types, or enum values |
| `menus.view` | view names (`GreetingList`) |
| view `kind`, `defaultView`, `widget`, `align`, layout `direction` | the allowed values |
| view `model`, `field`, `groupBy` | models and fields |
| layout `component` | component ids on that view |
| Go `ModelRegistry` methods (`Create`, `FindBy`, `ListAll`, …) and `Hooks` | model name argument |
| Go `FindBy` / `ListBy` field argument, `map[string]any` keys, `rec["…"]` | fields of that model; enum values after an enum key |
| `ModelsFromHost(host, "identity")` | that app's models on the bound registry |
| TSX `query` / `command` / `model` | `app.name` refs |
| TSX `field`, `groupByField`, `xField`, `yField`, `seriesField`, `keyField` | fields of the model bound by the nearest `query`, `command`, or `model` |
| TSX `card` / `keyOf` / `actions` / `columnOf` callbacks | `item.field` on the callback argument (`model.message`) |

Built-in record columns (`id`, `orgId`, `authorId`, `createdAt`, `updatedAt`, `deleted`) are included anywhere a stored field can appear.

## Run it against this repo

From the KaizenGo workspace:

1. `make vscode-ext` (installs deps and compiles)
2. Run **Run KaizenGo Extension** from the Run and Debug view (F5)

That opens an Extension Development Host with this repository as the workspace. Open `apps/inventory/app.yaml` and Ctrl+click `core`, `Dashboard`, or `models/product`.

## Package and install

Build a `.vsix` and install it into Cursor or VS Code:

```bash
make vscode-ext-vsix

cursor --install-extension packages/vscode-kaizengo/kaizengo-0.3.0.vsix
# or: code --install-extension packages/vscode-kaizengo/kaizengo-0.3.0.vsix
```

Reload the window after install. To remove: `cursor --uninstall-extension kaizengo.kaizengo`.

From the Command Palette you can also use **Extensions: Install from VSIX…** and pick the file.

For a one-off debug session without installing, use **Run KaizenGo Extension** (F5).

## Develop

```bash
cd packages/vscode-kaizengo
npm install
npm test
npm run compile
```
