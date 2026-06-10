SHELL := /bin/bash

.PHONY: db db-down dev dev-down run up down up-local down-local build-web gen-spec release

# Start Postgres + pgadmin (for local dev, port 5432 exposed)
db:
	docker compose -f docker-compose.yml -f docker-compose.dev.yml --env-file dev.env up -d postgres pgadmin

db-down:
	docker compose -f docker-compose.yml -f docker-compose.dev.yml down postgres pgadmin

# Run all services (requires overmind)
dev:
	set -a && source dev.env && set +a && overmind start

dev-down:
	overmind quit

# Run a single service: make run s=core
run:
	set -a && source dev.env && set +a && overmind start -l $(s)

# Production stack (uses pre-built images from ghcr.io)
up:
	docker compose up -d

down:
	docker compose down -v

# Production stack built locally (no ghcr pull required)
up-local:
	docker compose -f docker-compose.yml -f docker-compose.local.yml up -d --build

down-local:
	docker compose -f docker-compose.yml -f docker-compose.local.yml down -v

# Generate OpenAPI spec into VitePress public dir
gen-spec:
	go run ./cmd/genspec > docs/openapi.json

# Build and release
build-web:
	cd web && npm ci && npm run build
	cp -r web/dist/. internal/core/api/frontend/

release:
	./scripts/release.sh
