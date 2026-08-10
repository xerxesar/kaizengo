# Installation

## Prerequisites

| Tool | Version | Notes |
|------|---------|--------|
| Go | 1.22+ (repo uses 1.26) | [go.dev/dl](https://go.dev/dl/) |
| Node.js | 20+ | for the core SPA and Svelte app builds |
| npm | comes with Node | |

Optional:

- `templ` — only if you add templ SSR later (`go install github.com/a-h/templ/cmd/templ@latest`)

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

## Build the server + SPAs

```bash
make build
```

This:

1. Runs GraphQL Code Generator for the core shell (optional typed client)
2. Builds `apps/core/spa` → `apps/core/spa/dist`
3. Builds the counter Svelte app → `apps/counter/spa/dist`
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
curl -s http://localhost:8080/health          # ok
curl -s http://localhost:8080/api/apps        # nav catalog JSON
open http://localhost:8080/app/               # SPA shell
```

## Select which apps load

```bash
KaizenGo_APPS=core,counter ./bin/server
# empty / * = all installable apps
```
