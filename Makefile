.PHONY: dev dev-down up down generate build-frontend docker-build

COMPOSE_DEV = docker compose -f docker-compose.yml -f docker-compose.dev.yml
COMPOSE = docker compose

dev:
	$(COMPOSE_DEV) watch

dev-down:
	$(COMPOSE_DEV) down --remove-orphans

up:
	$(COMPOSE) up --build

down:
	$(COMPOSE) down

build-frontend:
	cd frontend && npm ci && npm run build
	cp -r frontend/dist/. services/core/internal/api/frontend/
	cp -r frontend/dist/. services/core/frontend/

docker-build: build-frontend
	$(COMPOSE) build core

generate:
	cd shared/proto && go generate
	cd services/core && sqlc generate
