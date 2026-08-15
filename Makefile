.PHONY: spa-build generate build run spa-dev dev tidy cli new-app db-up db-down db-logs docs docs-build vscode-ext vscode-ext-test vscode-ext-vsix

SPA_DIR := apps/core/spa

# Load local env (KaizenGo_POSTGRES_DSN, …) when present.
ifneq (,$(wildcard .env))
include .env
export
endif

spa-build:
	cd $(SPA_DIR) && npm install --prefer-offline --no-audit --no-fund && npm run build

# Generate apps/*/__types__ and locale/template.pot from app.yaml + sources.
generate:
	go run ./cmd/godino gen-types

build: generate spa-build
	go build -o bin/server ./cmd/server

run: generate spa-build
	go run ./cmd/server

spa-dev:
	cd $(SPA_DIR) && npm run dev

# Standard dev: Go (:8080) + core Vite (:5173). App views hot-reload via Vite.
dev: generate
	@echo ""
	@echo "  Shell   → http://localhost:5173/app/"
	@echo "  API     → http://localhost:8080 (proxied via Vite)"
	@echo ""
	@trap 'kill 0' INT TERM EXIT; \
	go run ./cmd/server & \
	(cd $(SPA_DIR) && npm run dev) & \
	wait

cli:
	go build -o bin/kaizengo ./cmd/godino

# Usage: make new-app NAME=notes TYPE=svelte EXTRA='--with-graphql'
new-app: cli
	./bin/kaizengo new-app $(NAME) --type $(or $(TYPE),svelte) $(EXTRA)

tidy:
	go mod tidy

# Local PostgreSQL for event-sourced apps (notes, hello, …).
db-up:
	docker compose up -d postgres
	@if [ ! -f .env ]; then cp .env.example .env && echo "Created .env from .env.example"; fi
	@echo ""
	@echo "  Postgres → localhost:6432  (user/db: kaizengo; avoids system postgres on :5432)"
	@echo "  DSN loaded from .env when you run make dev / make run"
	@echo ""

db-down:
	docker compose down

db-logs:
	docker compose logs -f postgres

# Developer docs (Zensical). Install once: pip install -r requirements-docs.txt
ZENSICAL := $(shell if [ -x .venv/bin/zensical ]; then echo .venv/bin/zensical; else echo zensical; fi)

docs:
	$(ZENSICAL) serve

docs-build:
	$(ZENSICAL) build

# VS Code / Cursor extension (app.yaml navigation). F5 → "Run KaizenGo Extension".
vscode-ext:
	cd packages/vscode-kaizengo && npm install --prefer-offline --no-audit --no-fund && npm run compile

vscode-ext-test: vscode-ext
	cd packages/vscode-kaizengo && npm test && npm run check

# Build a .vsix, then: cursor --install-extension packages/vscode-kaizengo/kaizengo-*.vsix
vscode-ext-vsix: vscode-ext
	cd packages/vscode-kaizengo && npm run package

