# Cycle Status Design

**Date:** 2026-08-01
**Status:** Approved

## Overview

The app runs several background cycles across four services (availability check,
monitor poll, metadata refresh, downloader poll, AniDB title sync) with no way to
see when each last ran or when it will run next. This feature adds a new
**System page** in the web UI showing every cycle with its last-run and next-run
times, state, and duration, plus per-service health.

## Architecture

All four services share one Postgres database. Each cycle records its state into
a central `cycle_status` table; the core API reads it back; the web UI polls the
API and renders live countdowns.

```
loop (any service) ── Record via recorder ──▶ cycle_status table (Postgres)
                                                        ▲
web UI ◀── GET /api/cycles ◀── core API ◀──────────────┘
web UI ◀── GET /api/workers (existing) ── for offline detection
```

## Storage

New table in `internal/core/db/migrations.go` (bun, `CREATE TABLE IF NOT EXISTS`):

```
service            TEXT NOT NULL   -- 'core' | 'indexer' | 'downloader' | 'metadata'
cycle              TEXT NOT NULL   -- 'availability' | 'monitor_poll' | 'metadata_refresh'
                                   -- | 'downloader_poll' | 'anidb_sync'
display_name       TEXT NOT NULL   -- human label, e.g. "Availability check"
state              TEXT NOT NULL   -- 'idle' | 'running'
last_started_at    TIMESTAMPTZ     -- start of current/last run
last_finished_at   TIMESTAMPTZ     -- end of last run
last_duration_ms   BIGINT
next_run_at        TIMESTAMPTZ
PRIMARY KEY (service, cycle)
```

## Recorder

New shared package `internal/cycle`, built on the existing bun `*bun.DB`:

```go
type Cycle struct {
    Service     string // "core"
    Cycle       string // "availability"
    DisplayName string // "Availability check"
}

type Recorder struct{ db *bun.DB }

func (r *Recorder) Start(ctx context.Context, c Cycle) error
func (r *Recorder) End(ctx context.Context, c Cycle, nextAt time.Time) error
```

- `Start` upserts `state='running'`, `last_started_at=now`.
- `End` upserts `state='idle'`, `last_finished_at=now`, `last_duration_ms`,
  `next_run_at=nextAt`.
- Failures are logged with `slog.Warn` and never break the loop.
- Each service constructs one `Recorder` and one `Cycle` descriptor per loop.

## Loop wiring

Each loop wraps its existing per-tick body: `Start` before the pass, `End` with
`time.Now().Add(interval)` after, where `interval` is the value already re-read
from config that tick. Rows stay `running` during long passes.

| Loop | Service | Existing function | Interval setting |
|---|---|---|---|
| Availability check | core | `PollAvailability` | `availabilityCheckInterval` |
| Metadata refresh | core | `PollMetadataRefresh` / `runRefreshPass` | `metadataRefreshInterval` |
| Monitor poll | indexer | `PollAndQueue` | `prowlarrInterval` |
| Downloader poll | downloader | `PollAndDownload` | `downloaderInterval` |
| AniDB title sync | metadata | runtime.go ticker body | `anidbSyncInterval` |

`missingRetryInterval` is a per-item throttle inside the monitor poll, not a
cycle — it gets no row.

## API

New huma route in `internal/core/api/routes.go`:

- `GET /api/cycles` (secured) → `SELECT * FROM cycle_status ORDER BY next_run_at`

Response body: `{ "cycles": [{ service, cycle, display_name, state,
last_started_at, last_finished_at, last_duration_ms, next_run_at }] }`

Pure DB read; no sidecar probing. Service health stays on the existing
`GET /api/workers`.

## Frontend

- New `web/src/pages/SystemPage.tsx`, route `/system`, sidebar entry under
  "System" (label "System", gauge-style icon).
- On mount fetch `/api/cycles` + `/api/workers`; re-poll every 15s.
- A client-side 1-second timer drives relative-time updates so countdowns tick
  live without re-polling.
- One row per cycle: name + service badge, state pill
  (`Running now` / `Idle` / `Offline` when the service is not in the healthy
  workers list), `last ran 23s ago`, `next in 12s`, and last-run duration.
  Tooltips show exact wall-clock times.
- Never-run cycles show `never`; offline services keep last known times with the
  `Offline` pill.
- Poll failure: keep last data and show a subtle stale indicator; the 1s clock
  keeps ticking.
- Pure helpers `formatRelative(ts)` and wall-clock formatter isolated for unit
  tests.

## Testing

- **Go:** recorder `Start`/`End` upsert and state transitions against bun's
  in-memory SQLite (bun is dialect-agnostic; adds a dev-only sqlite driver).
  Handler test for the `GET /api/cycles` response shape.
- **Web:** vitest + jsdom (existing stack) for `formatRelative`/wall-clock
  helpers and a `SystemPage` render test with mocked fetch covering rows, state
  pills, and offline handling.
