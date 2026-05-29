.PHONY: db db-down run-anidb run-prowlarr run-downloader run-core run-frontend up down build-frontend generate release

# Start only Postgres (for local dev)
db:
	docker compose up -d postgres

db-down:
	docker compose stop postgres

# Run individual services locally with air
run-anidb:
	cd services/anidb && set -a && source ../../.env && source ../../.env.local 2>/dev/null; set +a && air

run-prowlarr:
	cd services/prowlarr && set -a && source ../../.env && source ../../.env.local 2>/dev/null; set +a && air

run-downloader:
	cd services/downloader && set -a && source ../../.env && source ../../.env.local 2>/dev/null; set +a && air

run-core:
	cd services/core && set -a && source ../../.env && source ../../.env.local 2>/dev/null; set +a && air

run-frontend:
	cd frontend && npm run dev

# Production stack (uses pre-built images from ghcr.io)
up:
	docker compose up -d

down:
	docker compose down -v

# Build and release
build-frontend:
	cd frontend && npm ci && npm run build
	cp -r frontend/dist/. services/core/internal/api/frontend/

generate:
	cd shared/proto && go generate

release:
	./scripts/release.sh
