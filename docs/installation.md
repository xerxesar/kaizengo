# Installation

## Prerequisites

| Tool | Version | Notes |
|------|---------|--------|
| Go | 1.22+ (repo uses 1.26) | [go.dev/dl](https://go.dev/dl/) |
| Node.js | 20+ | for the core SPA and Svelte app builds |
| npm | comes with Node | |

## Get the code

```bash
git clone <your-repo-url> kaizengo
cd kaizengo
```

## Install JS deps (core shell)

```bash
cd apps/core/spa
npm install
cd -
```

App SPAs install their own deps the first time you build them (`make spa-build` or `make build`).

## Build the server + SPAs

```bash
make build
```

This:

1. Runs GraphQL Code Generator for the core shell (optional typed client)
2. Builds `apps/core/spa` → `apps/core/spa/dist`
3. Builds every app listed in `APP_SPAS` (counter, identity, notes, clock, status, settings, oracle, …)
4. Compiles `./cmd/server` → `bin/server`

## Install the CLI

```bash
make cli
# binary: bin/kaizengo
# optional: export PATH="$PWD/bin:$PATH"
```

Or:

```bash
go build -o bin/kaizengo ./cmd/kaizengo
```

## Verify

```bash
./bin/server
# open http://localhost:8080/app/
# sign in: admin@kaizengo.local / changeme
```

```bash
curl -s http://localhost:8080/health
# ok

curl -c cookies.txt -s http://localhost:8080/auth/login \
  -H 'content-type: application/json' \
  -d '{"email":"admin@kaizengo.local","password":"changeme"}'

curl -b cookies.txt -s http://localhost:8080/api/apps
```

`/api/apps` and `/graphql` require a session. See [auth.md](auth.md).

## Select which apps load

```bash
KaizenGo_APPS=core,identity,auth,permissions,appman ./bin/server
# empty = installed apps + autoInstall (core, identity, auth, permissions, appman, settings)
```

For a usable shell with login, always include at least `core`, `identity`, and `auth` (add `permissions` for RBAC-gated fields).
