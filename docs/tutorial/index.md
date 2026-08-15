# Develop a Kaizen app

This tutorial takes you from a running platform to a working app: a **Todo** module with a `task` model, list and form views, and a small Go hook. An optional advanced section then adds a service on the model registry and a custom Svelte page.

You will:

1. Run the host locally
2. Learn the files every app has
3. Declare a spec, migrations, module, and views
4. Add a lifecycle hook
5. Load the app and create a record in the shell
6. (Optional) Access models from Go, wrap a service, and build a custom page

Expected time: about 30 minutes for the spec path, plus 20 minutes for advanced.

## Prerequisites

| Tool | Why |
|------|-----|
| Go 1.22+ | Server and `kaizengo` CLI |
| Node.js 20+ | Core Svelte shell |
| Docker | Local Postgres (`make db-up`) |

You should be comfortable editing Go, YAML, and a little Svelte. You do not need to know GraphQL by heart — the engine generates CRUD from the spec.

## What you will not do (on the spec path)

- Fork `apps/core` or patch the kernel
- Hand-write GraphQL resolvers for ordinary CRUD
- Stand up a private database pool

Those are the usual ways to fight the platform. Pages 1–4 stay on the spec/engine path used by `hellospec` and `identity`. [Advanced](advanced/index.md) is where you *do* write extra Go and Svelte — still using the shared pool and `ModelRegistry`, not a private ORM.

## Path

1. [Run the platform](run-the-platform.md)
2. [App anatomy](app-anatomy.md)
3. [Build the Todo app](build-an-app.md)
4. [Hooks](hooks.md)
5. [Advanced](advanced/index.md) — ORM, services, custom pages
6. [What’s next](whats-next.md)

The bundled demo `apps/hellospec` is the same pattern if you prefer reading real code alongside the steps.
