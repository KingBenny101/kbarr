# KBArr - Copilot Instructions

## Project Overview
KBArr is a self-hosted anime management application similar to Sonarr/Radarr but for anime.
It is a hobby project built to learn Go.

## Tech Stack
- **Backend**: Go with Chi router
- **Frontend**: React + Vite
- **Database**: SQLite via go-sqlite3 (CGO required)
- **Logging**: Uber Zap
- **Metadata**: AniDB (titles dump for search, HTTP API for details)
- **Container**: Docker with multi-stage builds

## Project Structure
- `main.go` — entry point
- `internal/api/` — HTTP route handlers
- `internal/db/` — database queries and helpers
- `internal/models/` — data structs
- `internal/anidb/` — AniDB client and titles dump
- `internal/config/` — environment variable based config
- `internal/logger/` — zap logger setup
- `frontend/` — React + Vite app

## Conventions
- Logging format: `[ServiceName] message` e.g. `[API]`, `[DB]`, `[AniDB]`, `[Main]`
- Errors are returned as values, not panicked
- Config comes from environment variables with sensible defaults
- All background/async tasks should not block the HTTP server
- Database is SQLite for now, designed to migrate to PostgreSQL later

## Current Features
- Search anime via local AniDB titles dump
- Fetch anime details from AniDB HTTP API
- Add anime to a watchlist stored in SQLite
- React frontend served as embedded static files from Go binary
- Docker + docker-compose setup

## Future Plans
- Async task queue for torrent/download management
- Prowlarr/Torznab integration for torrent search
- Torrent client integration (qBittorrent etc.)
- Microservice-ready architecture