.PHONY: db db-down dev dev-down run-metadata run-indexer run-downloader run-core run-frontend up down build-frontend generate release

# Start only Postgres (for local dev)
db:
	docker compose up -d postgres

db-down:
	docker compose stop postgres

# Run individual services locally with air
run-metadata:
	cd services/metadata && set -a && source ../../.env && set +a && air

run-indexer:
	cd services/indexer && set -a && source ../../.env && set +a && air

run-downloader:
	cd services/downloader && set -a && source ../../.env && set +a && air

run-core:
	cd services/core && set -a && source ../../.env && set +a && air

run-frontend:
	cd frontend && npm run dev

# Run all services with tmux (for local dev)
dev:
	./scripts/dev.sh

dev-down:
	tmux kill-session -t kbarr

# Production stack (uses pre-built images from ghcr.io)
up:
	docker compose up -d

down:
	docker compose down -v

# Build and release
build-frontend:
	cd frontend && npm ci && npm run build
	cp -r frontend/dist/. services/core/internal/api/frontend/

release:
	./scripts/release.sh
