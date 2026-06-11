SHELL := /bin/bash

.PHONY: db db-down dev dev-down prod-up prod-down local local-down gen-spec release

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

# Production stack (uses pre-built images from ghcr.io)
prod-up:
	docker compose up -d

prod-down:
	docker compose down

# Production stack built locally (no ghcr pull required)
local:
	docker compose -f docker-compose.yml -f docker-compose.local.yml up -d --build

local-down:
	docker compose -f docker-compose.yml -f docker-compose.local.yml down -v

# Generate OpenAPI spec into VitePress public dir
gen-spec:
	go run ./cmd/genspec > docs/openapi.json

# Release
release:
	./scripts/release.sh
