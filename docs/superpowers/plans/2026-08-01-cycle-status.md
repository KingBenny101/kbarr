# Cycle Status Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Show every background cycle's last-run and next-run times on a new System page.

**Architecture:** All four services share one Postgres DB. Each cycle records state (idle/running, started/finished timestamps, duration, next-run time) into a `cycle_status` table through a tiny shared `internal/cycle` recorder. Core's API exposes `GET /api/cycles` (pure DB read); the web UI polls it plus the existing `/api/workers` endpoint and renders live countdowns on a new `/system` page.

**Tech Stack:** Go + bun ORM (Postgres; sqlite dialect for tests), huma/chi API, React + Mantine + vitest + jsdom + @testing-library/react.

## Global Constraints

- Table: `cycle_status`; columns exactly as in Task 1 `Schema` const; `PRIMARY KEY (service, cycle)`.
- Cycle identifiers: `availability`, `monitor_poll`, `metadata_refresh`, `downloader_poll`, `anidb_sync`. Service identifiers: `core`, `indexer`, `downloader`, `metadata`.
- State values: `'idle'` | `'running'`. `missingRetryInterval` gets NO row (it is a per-item throttle).
- Recorder failures are logged (`slog.Warn`) and must never break or slow the loop.
- `GET /api/cycles` is a pure DB read (no sidecar probing) and secured like other routes.
- UI: route `/system`, sidebar entry in the "System" group, poll every 15s, 1s client clock for countdowns, relative + wall-clock (tooltip) display.
- TDD: no production code without a failing test first; commits after each task.

---

### Task 1: Cycle recorder package

**Files:**
- Create: `internal/cycle/recorder.go`
- Create: `internal/cycle/recorder_test.go`
- Modify: `internal/core/db/migrations.go` (append `cycle.Schema` to the `stmts` list)
- Modify: `go.mod`, `go.sum` (via `go get`, test-only deps)

**Interfaces:**
- Produces (used by Tasks 2–4):
  - `const Schema string` — idempotent `CREATE TABLE IF NOT EXISTS cycle_status (...)` DDL
  - `type Cycle struct { Service string; Cycle string; DisplayName string }`
  - `type Recorder struct{ ... }`
  - `func NewRecorder(db *bun.DB) *Recorder`
  - `func (r *Recorder) Start(ctx context.Context, c Cycle) error` — upsert `state='running'`, `last_started_at=now`; remembers start time internally for duration
  - `func (r *Recorder) End(ctx context.Context, c Cycle, nextAt time.Time) error` — upsert `state='idle'`, `last_finished_at=now`, `last_duration_ms` (from remembered start), `next_run_at=nextAt`

- [ ] **Step 1: Add test-only Go deps and write the failing test**

```bash
go get github.com/uptrace/bun/dialect/sqlitedialect@v1.2.18 github.com/uptrace/bun/driver/sqliteshim@v1.2.18
```

Create `internal/cycle/recorder_test.go`:

```go
package cycle_test

import (
	"context"
	"testing"
	"time"

	"github.com/kingbenny101/kbarr/internal/cycle"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/sqlitedialect"
	"github.com/uptrace/bun/driver/sqliteshim"
)

func newTestDB(t *testing.T) *bun.DB {
	t.Helper()
	sqldb, err := sqliteshim.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { sqldb.Close() })
	db := bun.NewDB(sqldb, sqlitedialect.New())
	if _, err := db.ExecContext(context.Background(), cycle.Schema); err != nil {
		t.Fatal(err)
	}
	return db
}

func TestStartThenEnd(t *testing.T) {
	ctx := context.Background()
	db := newTestDB(t)
	rec := cycle.NewRecorder(db)
	c := cycle.Cycle{Service: "core", Cycle: "availability", DisplayName: "Availability check"}

	if err := rec.Start(ctx, c); err != nil {
		t.Fatalf("Start: %v", err)
	}
	time.Sleep(2 * time.Millisecond)
	next := time.Now().Add(time.Minute)
	if err := rec.End(ctx, c, next); err != nil {
		t.Fatalf("End: %v", err)
	}

	var row struct {
		State          string     `bun:"state"`
		LastStartedAt  time.Time  `bun:"last_started_at"`
		LastFinishedAt time.Time  `bun:"last_finished_at"`
		LastDurationMs int64      `bun:"last_duration_ms"`
		NextRunAt      *time.Time `bun:"next_run_at"`
	}
	err := db.NewRaw("SELECT state, last_started_at, last_finished_at, last_duration_ms, next_run_at FROM cycle_status WHERE service = ? AND cycle = ?", "core", "availability").Scan(ctx, &row)
	if err != nil {
		t.Fatalf("scan: %v", err)
	}
	if row.State != "idle" {
		t.Errorf("state = %q, want idle", row.State)
	}
	if row.LastDurationMs < 1 {
		t.Errorf("duration = %d, want >= 1", row.LastDurationMs)
	}
	if row.NextRunAt == nil {
		t.Fatal("next_run_at is nil")
	}
	if row.NextRunAt.Sub(next) > time.Second {
		t.Errorf("next_run_at = %v, want ~%v", row.NextRunAt, next)
	}
}

func TestStartTwiceEndOnce(t *testing.T) {
	ctx := context.Background()
	db := newTestDB(t)
	rec := cycle.NewRecorder(db)
	c := cycle.Cycle{Service: "core", Cycle: "availability", DisplayName: "Availability check"}

	if err := rec.Start(ctx, c); err != nil {
		t.Fatal(err)
	}
	time.Sleep(2 * time.Millisecond)
	if err := rec.Start(ctx, c); err != nil {
		t.Fatal(err)
	}
	time.Sleep(2 * time.Millisecond)
	if err := rec.End(ctx, c, time.Now().Add(time.Minute)); err != nil {
		t.Fatal(err)
	}

	var state string
	var duration int64
	err := db.NewRaw("SELECT state, last_duration_ms FROM cycle_status WHERE service = ? AND cycle = ?", "core", "availability").Scan(ctx, &state, &duration)
	if err != nil {
		t.Fatal(err)
	}
	if state != "idle" {
		t.Errorf("state = %q, want idle", state)
	}
	if duration < 1 {
		t.Errorf("duration = %d, want >= 1", duration)
	}
}

func TestEndWithoutStart(t *testing.T) {
	ctx := context.Background()
	db := newTestDB(t)
	rec := cycle.NewRecorder(db)
	c := cycle.Cycle{Service: "indexer", Cycle: "monitor_poll", DisplayName: "Monitor poll"}

	// Must not panic; duration degrades to 0 and the row is still recorded.
	if err := rec.End(ctx, c, time.Now().Add(time.Second)); err != nil {
		t.Fatal(err)
	}
	var duration int64
	if err := db.NewRaw("SELECT last_duration_ms FROM cycle_status WHERE service = ? AND cycle = ?", "indexer", "monitor_poll").Scan(ctx, &duration); err != nil {
		t.Fatal(err)
	}
	if duration != 0 {
		t.Errorf("duration = %d, want 0", duration)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

```bash
go test ./internal/cycle/
```
Expected: compile error — `./internal/cycle: package not found` (or `undefined: cycle.NewRecorder`).

- [ ] **Step 3: Write the recorder**

Create `internal/cycle/recorder.go`:

```go
package cycle

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"github.com/uptrace/bun"
)

// Schema creates the cycle_status table. Idempotent; shared by the migrations
// runner and the unit tests.
const Schema = `CREATE TABLE IF NOT EXISTS cycle_status (
	service          TEXT NOT NULL,
	cycle            TEXT NOT NULL,
	display_name     TEXT NOT NULL,
	state            TEXT NOT NULL,
	last_started_at  TIMESTAMPTZ,
	last_finished_at TIMESTAMPTZ,
	last_duration_ms BIGINT NOT NULL DEFAULT 0,
	next_run_at      TIMESTAMPTZ,
	PRIMARY KEY (service, cycle)
)`

// Cycle identifies one background loop within one service.
type Cycle struct {
	Service     string // "core"
	Cycle       string // "availability"
	DisplayName string // "Availability check"
}

const startSQL = `INSERT INTO cycle_status (service, cycle, display_name, state, last_started_at)
VALUES (?, ?, ?, 'running', ?)
ON CONFLICT (service, cycle) DO UPDATE SET
	display_name = excluded.display_name,
	state = 'running',
	last_started_at = excluded.last_started_at`

const endSQL = `INSERT INTO cycle_status (service, cycle, display_name, state, last_finished_at, last_duration_ms, next_run_at)
VALUES (?, ?, ?, 'idle', ?, ?, ?)
ON CONFLICT (service, cycle) DO UPDATE SET
	display_name = excluded.display_name,
	state = 'idle',
	last_finished_at = excluded.last_finished_at,
	last_duration_ms = excluded.last_duration_ms,
	next_run_at = excluded.next_run_at`

// Recorder persists a cycle's state to the shared database. It is safe for
// concurrent use; one instance per service.
type Recorder struct {
	db     *bun.DB
	mu     sync.Mutex
	starts map[string]time.Time
}

func NewRecorder(db *bun.DB) *Recorder {
	return &Recorder{db: db, starts: map[string]time.Time{}}
}

func key(c Cycle) string { return c.Service + "/" + c.Cycle }

// Start marks the cycle as running and records when it began.
func (r *Recorder) Start(ctx context.Context, c Cycle) error {
	r.mu.Lock()
	r.starts[key(c)] = time.Now()
	r.mu.Unlock()
	if _, err := r.db.ExecContext(ctx, startSQL, c.Service, c.Cycle, c.DisplayName, time.Now()); err != nil {
		slog.Warn("cycle: failed to record start", "service", c.Service, "cycle", c.Cycle, "error", err)
		return err
	}
	return nil
}

// End marks the cycle idle and records its finish time, duration (computed
// from the most recent Start), and the time of the next run.
func (r *Recorder) End(ctx context.Context, c Cycle, nextAt time.Time) error {
	r.mu.Lock()
	started := r.starts[key(c)]
	r.mu.Unlock()
	var durationMs int64
	if !started.IsZero() {
		durationMs = time.Since(started).Milliseconds()
	}
	now := time.Now()
	if _, err := r.db.ExecContext(ctx, endSQL, c.Service, c.Cycle, c.DisplayName, now, durationMs, nextAt); err != nil {
		slog.Warn("cycle: failed to record end", "service", c.Service, "cycle", c.Cycle, "error", err)
		return err
	}
	return nil
}
```

- [ ] **Step 4: Run test to verify it passes**

```bash
go test ./internal/cycle/ -v
```
Expected: 3 PASS.

- [ ] **Step 5: Wire the migration**

In `internal/core/db/migrations.go`, add `cycle.Schema` to the `stmts` slice (append after the `sessions` entry, before the closing `}` of the slice):

```go
		// sessions: DB-persisted auth tokens surviving restarts
		`CREATE TABLE IF NOT EXISTS sessions (
			token_hash   TEXT PRIMARY KEY,
			username     TEXT NOT NULL,
			created_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
			last_seen_at TIMESTAMPTZ NOT NULL DEFAULT now()
		)`,

		// cycle_status: last/next run times for the background loops
		cycle.Schema,
	}
```

Add the import `"github.com/kingbenny101/kbarr/internal/cycle"` to `migrations.go`.

- [ ] **Step 6: Verify build and all tests**

```bash
go build ./... && go vet ./... && go test ./...
```
Expected: build/vet clean, all tests pass.

- [ ] **Step 7: Commit**

```bash
git add internal/cycle/ go.mod go.sum internal/core/db/migrations.go
git commit -m "Add cycle recorder package and cycle_status table"
```

---

### Task 2: GET /api/cycles endpoint

**Files:**
- Create: `internal/core/api/handlers/cycles.go`
- Create: `internal/core/api/handlers/cycles_test.go`
- Modify: `internal/core/api/routes.go` (register route + import stays `handlers`)

**Interfaces:**
- Consumes: Task 1 `cycle.Schema`, `cycle.Cycle`, `cycle.Recorder`; global `db.DB *bun.DB` (package `internal/core/db`).
- Produces (used by Task 6): `GET /api/cycles` → `200` JSON:
  `{ "cycles": [ { "service", "cycle", "display_name", "state", "last_started_at", "last_finished_at", "last_duration_ms", "next_run_at" } ] }`
  with nullable timestamps omitted when null (`omitempty`), ordered by `next_run_at`.

- [ ] **Step 1: Write the failing test**

Create `internal/core/api/handlers/cycles_test.go`:

```go
package handlers

import (
	"context"
	"testing"
	"time"

	"github.com/kingbenny101/kbarr/internal/cycle"
	"github.com/kingbenny101/kbarr/internal/core/db"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/sqlitedialect"
	"github.com/uptrace/bun/driver/sqliteshim"
)

func TestGetCycles(t *testing.T) {
	ctx := context.Background()
	sqldb, err := sqliteshim.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer sqldb.Close()

	oldDB := db.DB
	db.DB = bun.NewDB(sqldb, sqlitedialect.New())
	t.Cleanup(func() { db.DB = oldDB })

	if _, err := db.DB.ExecContext(ctx, cycle.Schema); err != nil {
		t.Fatal(err)
	}
	rec := cycle.NewRecorder(db.DB)
	c := cycle.Cycle{Service: "core", Cycle: "availability", DisplayName: "Availability check"}
	if err := rec.Start(ctx, c); err != nil {
		t.Fatal(err)
	}
	if err := rec.End(ctx, c, time.Now().Add(30*time.Second)); err != nil {
		t.Fatal(err)
	}

	out, err := GetCycles()(ctx, &struct{}{})
	if err != nil {
		t.Fatal(err)
	}
	rows := out.Body.Cycles
	if len(rows) != 1 {
		t.Fatalf("len(cycles) = %d, want 1", len(rows))
	}
	row := rows[0]
	if row.Service != "core" || row.Cycle != "availability" || row.DisplayName != "Availability check" {
		t.Errorf("identity = %s/%s %q", row.Service, row.Cycle, row.DisplayName)
	}
	if row.State != "idle" {
		t.Errorf("state = %q, want idle", row.State)
	}
	if row.NextRunAt == nil {
		t.Error("next_run_at is nil")
	}
	if row.LastFinishedAt == nil {
		t.Error("last_finished_at is nil")
	}
	if row.LastDurationMs < 0 {
		t.Errorf("last_duration_ms = %d", row.LastDurationMs)
	}
}

func TestGetCyclesEmpty(t *testing.T) {
	ctx := context.Background()
	sqldb, err := sqliteshim.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer sqldb.Close()

	oldDB := db.DB
	db.DB = bun.NewDB(sqldb, sqlitedialect.New())
	t.Cleanup(func() { db.DB = oldDB })

	if _, err := db.DB.ExecContext(ctx, cycle.Schema); err != nil {
		t.Fatal(err)
	}
	out, err := GetCycles()(ctx, &struct{}{})
	if err != nil {
		t.Fatal(err)
	}
	if len(out.Body.Cycles) != 0 {
		t.Errorf("len(cycles) = %d, want 0", len(out.Body.Cycles))
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

```bash
go test ./internal/core/api/handlers/ -run TestGetCycles -v
```
Expected: FAIL — `undefined: GetCycles`.

- [ ] **Step 3: Write the handler**

Create `internal/core/api/handlers/cycles.go`:

```go
package handlers

import (
	"context"
	"time"

	"github.com/danielgtaylor/huma/v2"
	"github.com/kingbenny101/kbarr/internal/core/db"
)

type CycleStatus struct {
	Service        string     `json:"service"`
	Cycle          string     `json:"cycle"`
	DisplayName    string     `json:"display_name"`
	State          string     `json:"state"`
	LastStartedAt  *time.Time `json:"last_started_at,omitempty"`
	LastFinishedAt *time.Time `json:"last_finished_at,omitempty"`
	LastDurationMs int64      `json:"last_duration_ms"`
	NextRunAt      *time.Time `json:"next_run_at,omitempty"`
}

type CyclesOutput struct {
	Body struct {
		Cycles []CycleStatus `json:"cycles"`
	}
}

// GetCycles reports the last/next run times of every background cycle.
// Pure DB read: service liveness is reported separately by GET /api/workers.
func GetCycles() func(context.Context, *struct{}) (*CyclesOutput, error) {
	return func(ctx context.Context, _ *struct{}) (*CyclesOutput, error) {
		var rows []CycleStatus
		if err := db.DB.NewRaw(
			"SELECT service, cycle, display_name, state, last_started_at, last_finished_at, last_duration_ms, next_run_at FROM cycle_status ORDER BY next_run_at",
		).Scan(ctx, &rows); err != nil {
			return nil, huma.Error500InternalServerError("failed to read cycle status", err)
		}
		return &CyclesOutput{Body: struct {
			Cycles []CycleStatus `json:"cycles"`
		}{Cycles: rows}}, nil
	}
}
```

- [ ] **Step 4: Register the route**

In `internal/core/api/routes.go`, add after the `get-workers` line (line 25):

```go
	huma.Register(api, huma.Operation{OperationID: "get-cycles", Method: "GET", Path: "/api/cycles", Security: secured, Tags: []string{"system"}, Summary: "Get cycle status"}, handlers.GetCycles())
```

- [ ] **Step 5: Run tests to verify they pass**

```bash
go test ./internal/core/api/handlers/ -run TestGetCycles -v
```
Expected: 2 PASS.

- [ ] **Step 6: Verify build**

```bash
go build ./... && go vet ./...
```
Expected: clean.

- [ ] **Step 7: Commit**

```bash
git add internal/core/api/handlers/cycles.go internal/core/api/handlers/cycles_test.go internal/core/api/routes.go
git commit -m "Add GET /api/cycles endpoint"
```

---

### Task 3: Wire core service loops

**Files:**
- Modify: `internal/core/service/availability.go` (function `Poll`, lines 158-169)
- Modify: `internal/core/service/refresh.go` (function `PollMetadataRefresh`, lines 42-55)

**Interfaces:**
- Consumes: Task 1 `cycle.NewRecorder`, `cycle.Cycle`, `Recorder.Start/End`.

- [ ] **Step 1: Baseline verification**

```bash
go build ./... && go test ./internal/core/...
```
Expected: PASS (baseline before wiring). The loops are infinite by design and the wiring is thin recorder calls around already-tested functions, so verification for this task is the build gate plus the existing suite.

- [ ] **Step 2: Wire the availability loop**

In `internal/core/service/availability.go`, change `Poll` (lines 158-169) to:

```go
func (c *AvailabilityChecker) Poll(ctx context.Context) {
	rec := cycle.NewRecorder(c.db)
	avCycle := cycle.Cycle{Service: "core", Cycle: "availability", DisplayName: "Availability check"}
	c.recordedCheck(ctx, rec, avCycle, config.GetSeconds(c.db, "availabilityCheckInterval", 60*time.Second, 10*time.Second))
	for {
		// Interval is re-read every tick so config changes apply immediately.
		interval := config.GetSeconds(c.db, "availabilityCheckInterval", 60*time.Second, 10*time.Second)
		select {
		case <-time.After(interval):
			c.recordedCheck(ctx, rec, avCycle, interval)
		case <-ctx.Done():
			return
		}
	}
}

// recordedCheck runs one availability pass, recording its start/end and the
// next scheduled run. Status-write failures are logged by the recorder and
// never interrupt the scan itself.
func (c *AvailabilityChecker) recordedCheck(ctx context.Context, rec *cycle.Recorder, avCycle cycle.Cycle, interval time.Duration) {
	_ = rec.Start(ctx, avCycle)
	c.Check(ctx)
	_ = rec.End(ctx, avCycle, time.Now().Add(interval))
}
```

Add the import `"github.com/kingbenny101/kbarr/internal/cycle"` to `availability.go`.

- [ ] **Step 3: Wire the metadata refresh loop**

In `internal/core/service/refresh.go`, change `PollMetadataRefresh` (lines 42-55) to:

```go
func PollMetadataRefresh(ctx context.Context, mc *clients.MetadataClient) {
	rec := cycle.NewRecorder(db.DB)
	refreshCycle := cycle.Cycle{Service: "core", Cycle: "metadata_refresh", DisplayName: "Metadata refresh"}
	// First pass is delayed a little so it doesn't pile onto startup work.
	timer := time.NewTimer(time.Minute)
	defer timer.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-timer.C:
		}
		_ = rec.Start(ctx, refreshCycle)
		runRefreshPass(ctx, mc)
		interval := config.GetMinutes(db.DB, "metadataRefreshInterval", 1440*time.Minute, 60*time.Minute)
		_ = rec.End(ctx, refreshCycle, time.Now().Add(interval))
		timer.Reset(interval)
	}
}
```

Add the import `"github.com/kingbenny101/kbarr/internal/cycle"` to `refresh.go`.

- [ ] **Step 4: Verify build and tests**

```bash
go build ./... && go vet ./... && go test ./internal/core/...
```
Expected: clean, all pass.

- [ ] **Step 5: Commit**

```bash
git add internal/core/service/availability.go internal/core/service/refresh.go
git commit -m "Record availability and metadata refresh cycles"
```

---

### Task 4: Wire sidecar loops

**Files:**
- Modify: `internal/indexer/service/service.go` (function `PollAndQueue`, lines 77-93)
- Modify: `internal/downloader/service/downloader.go` (function `PollAndDownload`, lines 113-126)
- Modify: `internal/metadata/service/runtime.go` (function `StartTitlesSync`, lines 20-53)

**Interfaces:**
- Consumes: Task 1 `cycle.NewRecorder`, `cycle.Cycle`, `Recorder.Start/End`. Each service already has a `db *bun.DB` field (`s.db`).
- Notes: indexer cycle name `monitor_poll` (records each `processMonitors` pass; when a pass did work the next pass is immediate, so `nextAt = now`, otherwise `now + interval`). Downloader records one cycle covering both `ProcessPending` + `UpdateDownloading`. Metadata records only ticker ticks (the initial loads at startup are not a scheduled cycle).

- [ ] **Step 1: Baseline verification**

```bash
go build ./... && go test ./internal/indexer/... ./internal/downloader/... ./internal/metadata/...
```
Expected: PASS (baseline before wiring).

- [ ] **Step 2: Wire the indexer monitor poll**

In `internal/indexer/service/service.go`, change `PollAndQueue` (lines 77-93) to:

```go
func (s *IndexerService) PollAndQueue(ctx context.Context) {
	rec := cycle.NewRecorder(s.db)
	pollCycle := cycle.Cycle{Service: "indexer", Cycle: "monitor_poll", DisplayName: "Monitor poll"}
	for {
		_ = rec.Start(ctx, pollCycle)
		didWork := s.processMonitors(ctx)
		next := time.Now().Add(s.currentMonitorInterval())
		if didWork {
			// Work is available immediately; the next pass starts right away.
			next = time.Now()
		}
		_ = rec.End(ctx, pollCycle, next)
		if didWork {
			select {
			case <-ctx.Done():
				return
			default:
			}
			continue
		}

		interval := s.currentMonitorInterval()
		select {
		case <-time.After(interval):
		case <-ctx.Done():
			return
		}
	}
}
```

Add the import `"github.com/kingbenny101/kbarr/internal/cycle"` to `service.go`.

- [ ] **Step 3: Wire the downloader poll**

In `internal/downloader/service/downloader.go`, change `PollAndDownload` (lines 113-126) to:

```go
func (s *DownloaderService) PollAndDownload(ctx context.Context) {
	rec := cycle.NewRecorder(s.db)
	dlCycle := cycle.Cycle{Service: "downloader", Cycle: "downloader_poll", DisplayName: "Downloader poll"}
	s.recordedPoll(ctx, rec, dlCycle)

	for {
		interval := s.pollInterval()
		select {
		case <-time.After(interval):
			s.recordedPoll(ctx, rec, dlCycle)
		case <-ctx.Done():
			return
		}
	}
}

func (s *DownloaderService) recordedPoll(ctx context.Context, rec *cycle.Recorder, dlCycle cycle.Cycle) {
	_ = rec.Start(ctx, dlCycle)
	s.ProcessPending(ctx)
	s.UpdateDownloading(ctx)
	_ = rec.End(ctx, dlCycle, time.Now().Add(s.pollInterval()))
}
```

Add the import `"github.com/kingbenny101/kbarr/internal/cycle"` to `downloader.go`.

- [ ] **Step 4: Wire the AniDB title sync**

In `internal/metadata/service/runtime.go`, change `StartTitlesSync` (lines 20-53) to:

```go
func (s *AniDBService) StartTitlesSync(stop <-chan struct{}) {
	interval := s.currentTitlesInterval()

	if err := s.LoadTitlesDump(); err != nil {
		slog.Warn("Initial titles sync failed", "error", err)
	}
	if err := s.LoadAnimeListsMapping(); err != nil {
		slog.Warn("Initial anime-lists sync failed", "error", err)
	}

	rec := cycle.NewRecorder(s.db)
	syncCycle := cycle.Cycle{Service: "metadata", Cycle: "anidb_sync", DisplayName: "AniDB title sync"}

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			_ = rec.Start(context.Background(), syncCycle)
			if err := s.LoadTitlesDump(); err != nil {
				slog.Warn("Scheduled titles sync failed", "error", err)
			}
			if err := s.LoadAnimeListsMapping(); err != nil {
				slog.Warn("Scheduled anime-lists sync failed", "error", err)
			}

			nextInterval := s.currentTitlesInterval()
			if nextInterval != interval {
				interval = nextInterval
				ticker.Reset(interval)
			}
			_ = rec.End(context.Background(), syncCycle, time.Now().Add(interval))
		case <-stop:
			slog.Info("Titles sync loop stopped")
			return
		}
	}
}
```

Add the imports `"context"` and `"github.com/kingbenny101/kbarr/internal/cycle"` to `runtime.go`.

- [ ] **Step 5: Verify build and tests**

```bash
go build ./... && go vet ./... && go test ./...
```
Expected: clean, all pass.

- [ ] **Step 6: Commit**

```bash
git add internal/indexer/service/service.go internal/downloader/service/downloader.go internal/metadata/service/runtime.go
git commit -m "Record indexer, downloader, and AniDB sync cycles"
```

---

### Task 5: Frontend format helpers

**Files:**
- Create: `web/src/lib/format.ts`
- Create: `web/src/lib/format.test.ts`

**Interfaces:**
- Produces (used by Task 6):
  - `formatRelative(ts: Date | null, now: Date): string` — `"never"` for null; past: `"23s ago"`, `"5m ago"`, `"3h ago"`, `"2d ago"`; future: `"in 12s"`, `"in 5m"`, `"in 3h"`, `"in 2d"`. Seconds only for `< 1 minute`.
  - `formatWallClock(ts: Date): string` — `"14:32:05"` (24-hour HH:MM:SS).
  - `formatDuration(ms: number): string` — `"0s"`, `"1m 02s"`, `"1h 05m"`.

- [ ] **Step 1: Write the failing test**

Create `web/src/lib/format.test.ts`:

```ts
import { describe, expect, it } from "vitest"
import { formatDuration, formatRelative, formatWallClock } from "./format"

const NOW = new Date("2026-08-01T12:00:00Z")

describe("formatRelative", () => {
    it("returns never for null", () => {
        expect(formatRelative(null, NOW)).toBe("never")
    })

    it("formats past seconds", () => {
        expect(formatRelative(new Date("2026-08-01T11:59:37Z"), NOW)).toBe("23s ago")
    })

    it("formats future seconds", () => {
        expect(formatRelative(new Date("2026-08-01T12:00:12Z"), NOW)).toBe("in 12s")
    })

    it("formats minutes", () => {
        expect(formatRelative(new Date("2026-08-01T11:55:00Z"), NOW)).toBe("5m ago")
        expect(formatRelative(new Date("2026-08-01T12:05:00Z"), NOW)).toBe("in 5m")
    })

    it("formats hours", () => {
        expect(formatRelative(new Date("2026-08-01T09:00:00Z"), NOW)).toBe("3h ago")
        expect(formatRelative(new Date("2026-08-01T15:00:00Z"), NOW)).toBe("in 3h")
    })

    it("formats days", () => {
        expect(formatRelative(new Date("2026-07-30T12:00:00Z"), NOW)).toBe("2d ago")
        expect(formatRelative(new Date("2026-08-03T12:00:00Z"), NOW)).toBe("in 2d")
    })

    it("uses whole units", () => {
        expect(formatRelative(new Date("2026-08-01T11:59:00Z"), NOW)).toBe("1m ago")
        expect(formatRelative(new Date("2026-08-01T11:30:30Z"), NOW)).toBe("29m ago")
    })
})

describe("formatWallClock", () => {
    it("formats 24-hour time with seconds", () => {
        expect(formatWallClock(new Date("2026-08-01T14:32:05Z"))).toBe("14:32:05")
    })
})

describe("formatDuration", () => {
    it("formats sub-minute durations", () => {
        expect(formatDuration(0)).toBe("0s")
        expect(formatDuration(5_000)).toBe("5s")
    })

    it("formats minute durations with seconds", () => {
        expect(formatDuration(62_000)).toBe("1m 02s")
    })

    it("formats hour durations", () => {
        expect(formatDuration(3_900_000)).toBe("1h 05m")
    })
})
```

- [ ] **Step 2: Run test to verify it fails**

```bash
npm test -- src/lib/format.test.ts
```
Expected: FAIL — `Cannot find module './format'`.

- [ ] **Step 3: Write the helpers**

Create `web/src/lib/format.ts`:

```ts
function unit(ms: number): { value: number; unit: string } {
    const abs = Math.abs(ms)
    if (abs < 60_000) return { value: Math.floor(abs / 1000), unit: "s" }
    if (abs < 3_600_000) return { value: Math.floor(abs / 60_000), unit: "m" }
    if (abs < 86_400_000) return { value: Math.floor(abs / 3_600_000), unit: "h" }
    return { value: Math.floor(abs / 86_400_000), unit: "d" }
}

export function formatRelative(ts: Date | null, now: Date): string {
    if (!ts) return "never"
    const diffMs = ts.getTime() - now.getTime()
    const { value, unit: u } = unit(diffMs)
    return diffMs < 0 ? `${value}${u} ago` : `in ${value}${u}`
}

export function formatWallClock(ts: Date): string {
    return ts.toLocaleTimeString([], { hour12: false })
}

export function formatDuration(ms: number): string {
    const totalSec = Math.round(ms / 1000)
    const hours = Math.floor(totalSec / 3600)
    const minutes = Math.floor((totalSec % 3600) / 60)
    const seconds = totalSec % 60
    if (hours > 0) return `${hours}h ${String(minutes).padStart(2, "0")}m`
    if (minutes > 0) return `${minutes}m ${String(seconds).padStart(2, "0")}s`
    return `${seconds}s`
}
```

- [ ] **Step 4: Run test to verify it passes**

```bash
npm test -- src/lib/format.test.ts
```
Expected: PASS (11 tests).

- [ ] **Step 5: Lint check**

```bash
npm run lint
```
Expected: no new errors (only the pre-existing LibraryPage warning).

- [ ] **Step 6: Commit**

```bash
git add web/src/lib/format.ts web/src/lib/format.test.ts
git commit -m "Add time formatting helpers for the system page"
```

---

### Task 6: System page

**Files:**
- Create: `web/src/pages/SystemPage.tsx`
- Create: `web/src/pages/SystemPage.test.tsx`
- Create: `web/src/test/setup.ts`
- Modify: `web/vitest.config.ts` (add `setupFiles`)
- Modify: `web/src/App.tsx` (route + sidebar entry)
- Modify: `web/package.json`, `web/package-lock.json` (via `npm i -D`)

**Interfaces:**
- Consumes: Task 5 `formatRelative`, `formatWallClock`, `formatDuration`; existing `usePolling` (`web/src/hooks.ts`), `apiFetch`/`API_URL` (`web/src/utils.ts`); Task 2 `GET /api/cycles` shape; existing `GET /api/workers` → `[{ name, running }]`.

- [ ] **Step 1: Add test deps and write the failing test**

```bash
npm i -D @testing-library/react @testing-library/jest-dom
```

Create `web/src/test/setup.ts` (jsdom lacks `window.matchMedia`, which Mantine requires; vitest runs without `globals`, so React Testing Library cleanup must be registered explicitly):

```ts
import "@testing-library/jest-dom/vitest"
import { cleanup } from "@testing-library/react"
import { afterEach } from "vitest"

afterEach(cleanup)

Object.defineProperty(window, "matchMedia", {
    writable: true,
    value: (query: string) => ({
        matches: false,
        media: query,
        onchange: null,
        addListener: () => {},
        removeListener: () => {},
        addEventListener: () => {},
        removeEventListener: () => {},
        dispatchEvent: () => false,
    }),
})
```

Update `web/vitest.config.ts`:

```ts
import { defineConfig } from "vitest/config"

export default defineConfig({
    test: {
        environment: "jsdom",
        include: ["src/**/*.test.ts", "src/**/*.test.tsx"],
        setupFiles: ["src/test/setup.ts"],
    },
})
```

Create `web/src/pages/SystemPage.test.tsx`:

```tsx
import { render, screen } from "@testing-library/react"
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest"
import SystemPage from "./SystemPage"

function jsonResponse(data: unknown): Response {
    return new Response(JSON.stringify(data), {
        status: 200,
        headers: { "Content-Type": "application/json" },
    })
}

const CYCLES = [
    {
        service: "core",
        cycle: "availability",
        display_name: "Availability check",
        state: "idle",
        last_started_at: "2026-08-01T11:59:00Z",
        last_finished_at: "2026-08-01T11:59:03Z",
        last_duration_ms: 3000,
        next_run_at: "2026-08-01T12:00:03Z",
    },
    {
        service: "core",
        cycle: "metadata_refresh",
        display_name: "Metadata refresh",
        state: "running",
        last_started_at: "2026-08-01T11:58:00Z",
        last_finished_at: null,
        last_duration_ms: 0,
        next_run_at: null,
    },
]

const WORKERS = [
    { name: "core", display_name: "Core", running: true },
    { name: "metadata", display_name: "Metadata", running: false },
    { name: "indexer", display_name: "Indexer", running: true },
    { name: "downloader", display_name: "Downloader", running: true },
]

beforeEach(() => {
    const fetchMock = vi.fn(async (input: RequestInfo | URL) => {
        const url = String(input)
        if (url.includes("/api/cycles")) return jsonResponse({ cycles: CYCLES })
        if (url.includes("/api/workers")) return jsonResponse(WORKERS)
        return jsonResponse({})
    })
    vi.stubGlobal("fetch", fetchMock)
})

afterEach(() => {
    vi.unstubAllGlobals()
})

describe("SystemPage", () => {
    it("renders a row per cycle with state and times", async () => {
        render(<SystemPage />)

        expect(await screen.findByText("Availability check")).toBeInTheDocument()
        expect(screen.getByText("Metadata refresh")).toBeInTheDocument()

        expect(screen.getByText("Idle")).toBeInTheDocument()
        expect(screen.getByText("Running now")).toBeInTheDocument()
        // Offline: availability row is core (healthy) — the offline pill comes
        // from a cycle whose service is missing from the healthy workers.
    })

    it("marks cycles of offline services as Offline", async () => {
        const fetchMock = vi.fn(async (input: RequestInfo | URL) => {
            const url = String(input)
            if (url.includes("/api/cycles")) {
                return jsonResponse({
                    cycles: [
                        {
                            ...CYCLES[0],
                            service: "metadata",
                            cycle: "anidb_sync",
                            display_name: "AniDB title sync",
                        },
                    ],
                })
            }
            if (url.includes("/api/workers")) return jsonResponse(WORKERS)
            return jsonResponse({})
        })
        vi.stubGlobal("fetch", fetchMock)

        render(<SystemPage />)

        expect(await screen.findByText("AniDB title sync")).toBeInTheDocument()
        expect(screen.getByText("Offline")).toBeInTheDocument()
    })
})
```

- [ ] **Step 2: Run test to verify it fails**

```bash
npm test -- src/pages/SystemPage.test.tsx
```
Expected: FAIL — `Cannot find module './SystemPage'`.

- [ ] **Step 3: Write the System page**

Create `web/src/pages/SystemPage.tsx`:

```tsx
import { useEffect, useMemo, useState } from "react"
import { Badge, Group, Paper, Stack, Table, Text, Title, Tooltip } from "@mantine/core"
import { API_URL, apiFetch } from "@/utils"
import { usePolling } from "@/hooks"
import { formatDuration, formatRelative, formatWallClock } from "@/lib/format"

interface CycleStatus {
    service: string
    cycle: string
    display_name: string
    state: "idle" | "running"
    last_started_at: string | null
    last_finished_at: string | null
    last_duration_ms: number
    next_run_at: string | null
}

interface ServiceHealth {
    name: string
    running: boolean
}

function statePill(state: string, offline: boolean): React.ReactNode {
    if (offline) return <Badge color="red" variant="light">Offline</Badge>
    if (state === "running") return <Badge color="yellow" variant="light">Running now</Badge>
    return <Badge color="gray" variant="light">Idle</Badge>
}

function timeCell(ts: string | null, running: boolean, now: Date): React.ReactNode {
    if (!ts) return <Text c="dimmed">never</Text>
    const date = new Date(ts)
    return (
        <Tooltip label={formatWallClock(date)}>
            <Text>{running ? "started " + formatRelative(date, now) : formatRelative(date, now)}</Text>
        </Tooltip>
    )
}

export default function SystemPage() {
    const [cycles, setCycles] = useState<CycleStatus[]>([])
    const [workers, setWorkers] = useState<ServiceHealth[]>([])
    const [now, setNow] = useState(() => new Date())
    const [stale, setStale] = useState(false)

    useEffect(() => {
        const timer = setInterval(() => setNow(new Date()), 1000)
        return () => clearInterval(timer)
    }, [])

    const fetchCycles = async (): Promise<boolean> => {
        try {
            const res = await apiFetch(`${API_URL}/api/cycles`)
            if (!res.ok) throw new Error()
            const data: { cycles: CycleStatus[] } = await res.json()
            setCycles(data.cycles)
            setStale(false)
            return true
        } catch {
            setStale(true)
            return false
        }
    }

    const fetchWorkers = async (): Promise<boolean> => {
        try {
            const res = await apiFetch(`${API_URL}/api/workers`)
            if (!res.ok) throw new Error()
            const data: ServiceHealth[] = await res.json()
            setWorkers(data)
            return true
        } catch {
            return false
        }
    }

    usePolling(fetchCycles, { interval: 15_000 }, [])
    usePolling(fetchWorkers, { interval: 15_000 }, [])

    const offlineServices = useMemo(
        () => new Set(workers.filter((w) => !w.running).map((w) => w.name)),
        [workers],
    )

    return (
        <Stack gap="md">
            <Group justify="space-between">
                <Title order={2}>System</Title>
                {stale && <Text c="orange" size="sm">Status data is stale — retrying…</Text>}
            </Group>
            <Paper withBorder p="md">
                <Table striped highlightOnHover>
                    <Table.Thead>
                        <Table.Tr>
                            <Table.Th>Cycle</Table.Th>
                            <Table.Th>State</Table.Th>
                            <Table.Th>Last run</Table.Th>
                            <Table.Th>Next run</Table.Th>
                            <Table.Th>Duration</Table.Th>
                        </Table.Tr>
                    </Table.Thead>
                    <Table.Tbody>
                        {cycles.map((c) => {
                            const offline = offlineServices.has(c.service)
                            const running = !offline && c.state === "running"
                            const lastTs = running ? c.last_started_at : c.last_finished_at ?? c.last_started_at
                            return (
                                <Table.Tr key={`${c.service}/${c.cycle}`}>
                                    <Table.Td>
                                        <Group gap="xs">
                                            <Text fw={500}>{c.display_name}</Text>
                                            <Badge size="xs" variant="outline" color="gray">{c.service}</Badge>
                                        </Group>
                                    </Table.Td>
                                    <Table.Td>{statePill(c.state, offline)}</Table.Td>
                                    <Table.Td>{timeCell(lastTs, running, now)}</Table.Td>
                                    <Table.Td>
                                        {running ? (
                                            <Text c="dimmed">—</Text>
                                        ) : (
                                            timeCell(c.next_run_at, false, now)
                                        )}
                                    </Table.Td>
                                    <Table.Td>{formatDuration(c.last_duration_ms)}</Table.Td>
                                </Table.Tr>
                            )
                        })}
                        {cycles.length === 0 && (
                            <Table.Tr>
                                <Table.Td colSpan={5}><Text c="dimmed" ta="center">No cycles recorded yet</Text></Table.Td>
                            </Table.Tr>
                        )}
                    </Table.Tbody>
                </Table>
            </Paper>
        </Stack>
    )
}
```

- [ ] **Step 4: Add the route and sidebar entry**

In `web/src/App.tsx`:
- Add import: `import { SystemPage } from "@/pages/SystemPage"` (after the LogsPage import, line 15).
- Add to the `navigation` "System" group items (after the Logs entry, line 29):

```tsx
            { label: "System", to: "/system", icon: <IconGauge size={18} /> },
```

- Extend the tabler icons import (line 5) with `IconGauge`:

```tsx
import { IconLibraryPhoto, IconSearch, IconCompass, IconListCheck, IconActivity, IconGauge, IconSettings, IconDatabase, IconPlug, IconDownload, IconCloudDownload, IconMoonFilled, IconSunFilled } from "@tabler/icons-react"
```

- Add the route after the `/logs` route (line 193):

```tsx
                            <Route path="/system" element={<SystemPage />} />
```

- [ ] **Step 5: Run tests to verify they pass**

```bash
npm test -- src/pages/SystemPage.test.tsx
```
Expected: 2 PASS.

- [ ] **Step 6: Full verification**

```bash
npm run lint && npm run build && npm test
```
Expected: lint 0 errors (1 pre-existing LibraryPage warning), build clean, all tests pass.

- [ ] **Step 7: Commit**

```bash
git add web/src/pages/SystemPage.tsx web/src/pages/SystemPage.test.tsx web/src/test/setup.ts web/vitest.config.ts web/src/App.tsx web/package.json web/package-lock.json
git commit -m "Add system page with cycle status"
```
