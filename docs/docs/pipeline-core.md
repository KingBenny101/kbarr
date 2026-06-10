---
sidebar_label: Core service
sidebar_position: 1
---

# Core service pipeline

The core service is the central hub. It owns the database, exposes the REST API consumed by the frontend, and runs the availability scanner.

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

Runs continuously in the background every `availabilityCheckInterval` seconds (default 10 s). Each cycle has three phases:

**Phase 1 — mark available**

Loads all episode monitors from the DB, then walks `mediaPath` one level deep. Each subdirectory is a show folder; files inside are matched against the regex `S\d\dE\d\d`. A matched file resolves to a monitor via `{folder, season, episode}` and sets `available = true`.

**Phase 2 — clear stale available flags**

For every monitor currently marked `available = true`, checks whether its file was found in Phase 1. If not found and the monitor's status was `downloaded` (kbarr placed it there), the status resets to `pending` so it gets re-downloaded. Otherwise it is simply marked unavailable.

**Phase 3 — sync season monitors**

Counts total vs. available episode monitors per `library_id`. If all episodes are available, the season monitor is marked available. If any are missing, it is cleared.

## Key packages

| Package | Role |
|---|---|
| `internal/core/api/` | huma v2 router, route registration, handler wiring |
| `internal/core/db/` | All Bun ORM queries (media, monitors, downloads, settings) |
| `internal/core/auth/` | JWT token issuance and validation |
| `internal/core/clients/` | HTTP client that proxies to the metadata service |
| `internal/core/service/` | Availability scanner |
| `internal/config/` | Settings schema, defaults, get/set helpers |
