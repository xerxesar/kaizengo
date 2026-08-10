.PHONY: generate spa-codegen spa-build build run spa-dev dev tidy cli new-app

SPA_DIR := apps/core/spa
COUNTER_SPA := apps/counter/spa

generate: spa-codegen

spa-codegen:
	cd $(SPA_DIR) && npm run codegen

spa-build: spa-codegen
	cd $(SPA_DIR) && npm run build
	cd $(COUNTER_SPA) && npm install --prefer-offline --no-audit --no-fund && npm run build

build: generate spa-build
	go build -o bin/server ./cmd/server

run: generate spa-build
	go run ./cmd/server

spa-dev:
	cd $(SPA_DIR) && npm run dev

dev: generate
	@trap 'kill 0' INT TERM EXIT; \
	go run ./cmd/server & \
	(cd $(SPA_DIR) && npm run dev) & \
	wait

cli:
	go build -o bin/kaizengo ./cmd/kaizengo

# Usage: make new-app NAME=notes TYPE=svelte EXTRA='--with-graphql'
new-app: cli
	./bin/kaizengo new-app $(NAME) --type $(or $(TYPE),vanilla) $(EXTRA)

tidy:
	go mod tidy
