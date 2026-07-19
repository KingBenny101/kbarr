---
sidebar_label: Core service
sidebar_position: 1
---

# Core service pipeline

The core service owns the database, exposes the REST API consumed by the frontend, and runs the availability scanner.

## Startup

1. Database is initialised and migrations run.
2. Auth defaults (username / password hash) are seeded if absent.
3. The availability poller starts in a background goroutine.
4. The HTTP server starts on port `8080` (configurable via `PORT`).

## Request flow

```
Browser / frontend
      │
      ▼
  HTTP server (chi router + huma v2)
      │  Bearer token validated on every secured route
      ▼
  Route handler (internal/core/api/handlers/)
      │
      ├── Library routes   → internal/core/db/  (Bun ORM, PostgreSQL)
      ├── Monitor routes   → internal/core/db/
      ├── Download routes  → internal/core/db/
      │                      + proxied calls to downloader service (port 8083)
      ├── Search route     → proxied call to metadata service (port 8081)
      └── Settings routes  → internal/config/  (key-value in DB)
```

The core service does not talk to qBittorrent or torrent indexers directly — those concerns are delegated to the downloader and indexer services.

## Availability scanner

Runs every `availabilityCheckInterval` seconds (default 60 s). Caches each show folder's directory listing keyed by mtime, so a cycle with no disk changes reads no directories. Each cycle has three phases:

**Phase 1 — mark available**

Indexes `media` rows by sanitized `media_folder` (and title, as a fallback) to a `library_id`, then walks `mediaPath` one level deep. Each subdirectory is resolved to a single library; files inside are parsed for an episode number (the `SxxExx` pattern, then the anitogo parser for absolute numbering). Matching is **season-agnostic**: the folder already scopes the show, so a monitor resolves via `{library_id, episode}` and sets `available = true`. Because each anime season is its own `media` row and folder, the season digit in the filename (often an unreliable `S01` for sequels) is intentionally ignored. Folder names mapping to two libraries are flagged ambiguous and skipped.

**Phase 2 — clear stale available flags**

For every monitor marked `available = true`, checks whether its file is still present. If gone and the status was `downloaded` (kbarr placed it there), the status resets to `missing`, not `pending` — so the indexer's `missingRetryInterval` throttles re-search instead of re-queuing every cycle. Otherwise it is marked unavailable.

**Phase 3 — sync season monitors**

Counts total vs. available episode monitors per `library_id`. If all episodes are available, the season monitor marks `downloaded`. If incomplete, it resets to `missing` (avoiding an immediate re-queue loop).

## Key packages

| Package | Role |
|---|---|
| `internal/core/api/` | huma v2 router, route registration, handler wiring |
| `internal/core/db/` | All Bun ORM queries (media, monitors, downloads, settings) |
| `internal/core/auth/` | JWT token issuance and validation |
| `internal/core/clients/` | HTTP client that proxies to the metadata service |
| `internal/core/service/` | Availability scanner |
| `internal/config/` | Settings schema, defaults, get/set helpers |
