# KaizenGo VS Code extension

Developer tooling for KaizenGo apps. The first feature is **navigation in `app.yaml`**.

## What it does

In any `apps/<name>/app.yaml`, **F12** / **Ctrl+click** (Go to Definition) and hover resolve:

| You click | Opens |
|-----------|--------|
| `depends` app name | that app's `app.yaml` |
| `uses` capability | the `provides` entry (or platform source for `platform.*`) |
| `menus.view` | `views/<Name>.page.tsx` |
| `models:` path | `models/<name>/spec.yaml` |
| `models` name | generated `__types__/<model>.go` |
| `relation` / `inverse` | the related model or field in YAML |
| `labelKey` | `msgid` in `locale/*.po` (and platform nav catalogs) |
| `locales` id | `locale/<id>.po` |
| `extends.handler` | `RegisterNamed("…")` in the app's Go code |
| `exports.components.module` | the Svelte/TS module |
| `exports.*.app` | the target app's `app.yaml` |

**Shift+F12** (Find All References) lists the other side: `uses` of a capability, `relation` fields of a model, menus that point at a view.

## Run it against this repo

From the KaizenGo workspace:

1. `make vscode-ext` (installs deps and compiles)
2. Run **Run KaizenGo Extension** from the Run and Debug view (F5)

That opens an Extension Development Host with this repository as the workspace. Open `apps/inventory/app.yaml` and Ctrl+click `core`, `Dashboard`, or `models/product`.

## Package and install

Build a `.vsix` and install it into Cursor or VS Code:

```bash
make vscode-ext-vsix

cursor --install-extension packages/vscode-kaizengo/kaizengo-0.1.0.vsix
# or: code --install-extension packages/vscode-kaizengo/kaizengo-0.1.0.vsix
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
