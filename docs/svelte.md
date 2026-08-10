# Svelte apps

## Idea

Core is a small Svelte **shell**. Other apps are separate ESM bundles that implement `mount` / `unmount`. A Svelte app is just a component compiled as a Vite **library**.

```text
core shell  --import()-->  /app-assets/myapp/spa.js
                              └─ mount(MyComponent, { target: el })
```

## Bootstrap

```bash
./bin/kaizengo new-app myapp --type svelte
cd apps/myapp/spa && npm install && npm run build
```

## Layout

| File | Role |
|------|------|
| `spa/App.svelte` | Your UI |
| `spa/main.ts` | Adapts Svelte `mount`/`unmount` to the shell contract; injects `spa.css` |
| `spa/vite.config.ts` | `build.lib` → `dist/spa.js` + `dist/spa.css` |
| `module.go` | Registers nav + serves `spa/dist` |

## Build

Uses `file:` deps pointing at `apps/core/spa/node_modules` so you do not duplicate Vite/Svelte installs.

```bash
cd apps/myapp/spa
npm install
npm run build
```

Serve path (from `module.go`):

```go
http.FileServer(http.Dir("apps/myapp/spa/dist"))
// URL: /app-assets/myapp/spa.js
```

## Dev tip

The shell in Vite dev loads `/app-assets/...` through the proxy to Go. Rebuild the app lib after changes (`npm run build` in the app), or add a watch script later.

## Other frameworks

Same contract — compile to ESM that exports:

```ts
export default {
  async mount(el: HTMLElement) { /* ReactDOM.createRoot(el).render(...) */ },
  unmount() { /* root.unmount() */ },
}
```

Static HTML works too: set `el.innerHTML` in `mount`.
