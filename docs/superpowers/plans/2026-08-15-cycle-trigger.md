# Cycle Trigger Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add a "Run now" button per cycle row on the System page that wakes that cycle's poll loop so it executes immediately and records the run in `cycle_status`.

**Architecture:** A single secured core endpoint `POST /api/cycles/{service}/{cycle}/trigger` routes to the right loop: core cycles (`availability`, `metadata_refresh`) are woken in-process via a non-blocking channel send; indexer/downloader/metadata cycles are woken over HTTP — each service exposes `POST /trigger` that non-blocking-sends into its own loop's select. The loop owns execution and records start/end via the existing `cycle.Recorder`, so triggered runs appear in the System page like scheduled ones. Frontend gets a labeled "Run now" button per row (desktop table, last column) and per card (mobile).

**Tech Stack:** Go (core/indexer/downloader/metadata services, huma v2 API), React + TypeScript + Mantine (web), vitest + testing-library, bun sqlite in tests.

## Global Constraints

- Commit messages: plain sentence-case, NO prefixes (user mandate).
- All new code must pass: `go test ./...`, `npx vitest run`, `npx tsc --noEmit`, `npm run lint` (run inside `web/`).
- Trigger is a NON-BLOCKING signal: buffered channel of size 1, `select { case ch <- struct{}{}: default: }` — never block the HTTP handler, drop redundant signals.
- The loop always owns execution — no goroutines spawning work from handlers.
- Triggered passes must record via the loop's normal `rec.Start`/`rec.End` path so the System page shows them.
- Keep `POST /api/availability/check` and the Downloads page's `POST /api/downloads/trigger` behavior working (the downloader one now semantically wakes the loop instead of running a goroutine).
- Env keys: `INDEXER_HEALTH_ADDR` (fallback `http://localhost:8082`), `DOWNLOADER_HEALTH_ADDR` (fallback `http://localhost:8083`), `METADATA_ADDR` (fallback `http://localhost:8081`) — use `handlers.SvcAddr` like the existing routes.

---

### Task 1: Indexer trigger channel

**Files:**
- Modify: `internal/indexer/service/service.go` (struct, `New`, add `Trigger`, `PollAndQueue` select)
- Modify: `cmd/indexer/main.go` (add `POST /trigger` handler)
- Create: `internal/indexer/service/trigger_test.go`

**Interfaces:**
- Consumes: `cycle.NewRecorder`, `cycle.Schema` (existing)
- Produces: `(*IndexerService).Trigger()` — non-blocking send; `POST /trigger` on indexer HTTP (port 8082)

- [ ] **Step 1: Write the failing tests**

```go
// internal/indexer/service/trigger_test.go
package service

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/kingbenny101/kbarr/internal/cycle"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/sqlitedialect"
	"github.com/uptrace/bun/driver/sqliteshim"
)

// testLoopDB returns an in-memory DB with the cycle_status and settings tables.
func testLoopDB(t *testing.T) *bun.DB {
	t.Helper()
	sqldb, err := sql.Open(sqliteshim.ShimName, ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { sqldb.Close() })
	db := bun.NewDB(sqldb, sqlitedialect.New())
	ctx := context.Background()
	if _, err := db.ExecContext(ctx, cycle.Schema); err != nil {
		t.Fatal(err)
	}
	if _, err := db.ExecContext(ctx, `CREATE TABLE settings (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
		updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
		deleted_at TIMESTAMP,
		key TEXT NOT NULL UNIQUE,
		value TEXT
	)`); err != nil {
		t.Fatal(err)
	}
	return db
}

// waitForFinish polls cycle_status until the cycle's last_finished_at is set.
func waitForFinish(t *testing.T, db *bun.DB, service, cycle string, timeout time.Duration) time.Time {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		var row struct {
			LastFinishedAt *time.Time `bun:"last_finished_at"`
		}
		err := db.NewSelect().TableExpr("cycle_status").
			ColumnExpr("last_finished_at").
			Where("service = ? AND cycle = ?", service, cycle).
			Scan(context.Background(), &row)
		if err == nil && row.LastFinishedAt != nil {
			return *row.LastFinishedAt
		}
		time.Sleep(50 * time.Millisecond)
	}
	t.Fatalf("cycle %s/%s never finished within %v", service, cycle, timeout)
	return time.Time{}
}

// waitForNewFinish polls until last_finished_at is after the given time.
func waitForNewFinish(t *testing.T, db *bun.DB, service, cycle string, after time.Time, timeout time.Duration) time.Time {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		var row struct {
			LastFinishedAt *time.Time `bun:"last_finished_at"`
		}
		err := db.NewSelect().TableExpr("cycle_status").
			ColumnExpr("last_finished_at").
			Where("service = ? AND cycle = ?", service, cycle).
			Scan(context.Background(), &row)
		if err == nil && row.LastFinishedAt != nil && row.LastFinishedAt.After(after) {
			return *row.LastFinishedAt
		}
		time.Sleep(50 * time.Millisecond)
	}
	t.Fatalf("cycle %s/%s did not run again within %v", service, cycle, timeout)
	return time.Time{}
}

func TestPollAndQueueWakesOnTrigger(t *testing.T) {
	db := testLoopDB(t)
	// A 3600s interval means the loop only passes again when triggered.
	if _, err := db.ExecContext(context.Background(),
		`INSERT INTO settings (key, value) VALUES ('prowlarrInterval', '3600')`); err != nil {
		t.Fatal(err)
	}

	svc := New(db)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go svc.PollAndQueue(ctx)

	first := waitForFinish(t, db, "indexer", "monitor_poll", 5*time.Second)
	svc.Trigger()
	waitForNewFinish(t, db, "indexer", "monitor_poll", first, 5*time.Second)
}
```

- [ ] **Step 2: Run the test to verify it fails**

Run: `go test ./internal/indexer/service/ -run TestPollAndQueueWakesOnTrigger -v`
Expected: FAIL — `svc.Trigger` undefined (and `IndexerService` has no trigger field yet).

- [ ] **Step 3: Add the trigger channel to IndexerService**

In `internal/indexer/service/service.go`, change the struct and constructor:

```go
type IndexerService struct {
	db      *bun.DB
	trigger chan struct{}
}

func New(db *bun.DB) *IndexerService {
	return &IndexerService{db: db, trigger: make(chan struct{}, 1)}
}

// Trigger wakes the poll loop immediately. Non-blocking: if the loop is already
// running a pass or a wake is already pending, the signal is dropped.
func (s *IndexerService) Trigger() {
	select {
	case s.trigger <- struct{}{}:
	default:
	}
}
```

- [ ] **Step 4: Wake the loop in PollAndQueue**

In `PollAndQueue` (around line 102), add the trigger case to the sleep select:

```go
		interval := s.currentMonitorInterval()
		select {
		case <-time.After(interval):
		case <-s.trigger:
		case <-ctx.Done():
			return
		}
```

- [ ] **Step 5: Add the HTTP trigger to cmd/indexer/main.go**

In `cmd/indexer/main.go`, after the `POST /test/kbdex` registration (line 103), add:

```go
	mux.HandleFunc("POST /trigger", func(w http.ResponseWriter, _ *http.Request) {
		svc.Trigger()
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"ok": "true"})
	})
```

- [ ] **Step 6: Run the test to verify it passes**

Run: `go test ./internal/indexer/service/ -run TestPollAndQueueWakesOnTrigger -v`
Expected: PASS (first pass recorded, trigger wakes a second pass within 5s).

- [ ] **Step 7: Run the full Go suite**

Run: `go test ./...`
Expected: all pass.

- [ ] **Step 8: Commit**

```bash
git add internal/indexer/service/service.go internal/indexer/service/trigger_test.go cmd/indexer/main.go
git commit -m "Wake the indexer poll loop on a trigger signal"
```

---

### Task 2: Downloader trigger channel

**Files:**
- Modify: `internal/downloader/service/downloader.go` (struct, `NewDownloaderService`, add `Trigger`, `PollAndDownload` select)
- Modify: `cmd/downloader/main.go` (change `POST /trigger` handler to wake the loop; drop unused `context` import)
- Create: `internal/downloader/service/trigger_test.go`

**Interfaces:**
- Consumes: `cycle.NewRecorder`, `cycle.Schema` (existing)
- Produces: `(*DownloaderService).Trigger()` — non-blocking send; `POST /trigger` on downloader HTTP (port 8083) now wakes the loop

- [ ] **Step 1: Write the failing test**

```go
// internal/downloader/service/trigger_test.go
package service

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/kingbenny101/kbarr/internal/cycle"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/sqlitedialect"
	"github.com/uptrace/bun/driver/sqliteshim"
)

func TestPollAndDownloadWakesOnTrigger(t *testing.T) {
	sqldb, err := sql.Open(sqliteshim.ShimName, ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer sqldb.Close()
	db := bun.NewDB(sqldb, sqlitedialect.New())
	ctx := context.Background()
	if _, err := db.ExecContext(ctx, cycle.Schema); err != nil {
		t.Fatal(err)
	}
	if _, err := db.ExecContext(ctx, `CREATE TABLE settings (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
		updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
		deleted_at TIMESTAMP,
		key TEXT NOT NULL UNIQUE,
		value TEXT
	)`); err != nil {
		t.Fatal(err)
	}
	if _, err := db.ExecContext(ctx,
		`INSERT INTO settings (key, value) VALUES ('downloaderInterval', '3600')`); err != nil {
		t.Fatal(err)
	}

	svc := NewDownloaderService(db)
	loopCtx, cancel := context.WithCancel(ctx)
	defer cancel()
	go svc.PollAndDownload(loopCtx)

	deadline := time.Now().Add(5 * time.Second)
	var first time.Time
	for time.Now().Before(deadline) {
		var row struct {
			LastFinishedAt *time.Time `bun:"last_finished_at"`
		}
		err := db.NewSelect().TableExpr("cycle_status").
			ColumnExpr("last_finished_at").
			Where("service = 'downloader' AND cycle = 'downloader_poll'").
			Scan(ctx, &row)
		if err == nil && row.LastFinishedAt != nil {
			first = *row.LastFinishedAt
			break
		}
		time.Sleep(50 * time.Millisecond)
	}
	if first.IsZero() {
		t.Fatal("first downloader pass never finished")
	}

	svc.Trigger()

	deadline = time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		var row struct {
			LastFinishedAt *time.Time `bun:"last_finished_at"`
		}
		err := db.NewSelect().TableExpr("cycle_status").
			ColumnExpr("last_finished_at").
			Where("service = 'downloader' AND cycle = 'downloader_poll'").
			Scan(ctx, &row)
		if err == nil && row.LastFinishedAt != nil && row.LastFinishedAt.After(first) {
			return
		}
		time.Sleep(50 * time.Millisecond)
	}
	t.Fatal("downloader poll did not wake on trigger")
}
```

- [ ] **Step 2: Run the test to verify it fails**

Run: `go test ./internal/downloader/service/ -run TestPollAndDownloadWakesOnTrigger -v`
Expected: FAIL — `svc.Trigger` undefined.

- [ ] **Step 3: Add the trigger channel to DownloaderService**

In `internal/downloader/service/downloader.go`, change the struct and constructor:

```go
type DownloaderService struct {
	db      *bun.DB
	client  dlprovider.TorrentClient
	trigger chan struct{}
}

func NewDownloaderService(db *bun.DB) *DownloaderService {
	return &DownloaderService{
		db:      db,
		client:  dlprovider.Get(db),
		trigger: make(chan struct{}, 1),
	}
}

// Trigger wakes the poll loop immediately. Non-blocking: if the loop is already
// running a pass or a wake is already pending, the signal is dropped.
func (s *DownloaderService) Trigger() {
	select {
	case s.trigger <- struct{}{}:
	default:
	}
}
```

- [ ] **Step 4: Wake the loop in PollAndDownload**

In `PollAndDownload` (around line 120), add the trigger case:

```go
	for {
		interval := s.pollInterval()
		select {
		case <-time.After(interval):
			s.recordedPoll(ctx, rec, dlCycle)
		case <-s.trigger:
			s.recordedPoll(ctx, rec, dlCycle)
		case <-ctx.Done():
			return
		}
	}
```

- [ ] **Step 5: Change the /trigger handler in cmd/downloader/main.go**

Replace the existing `POST /trigger` handler (lines 94-101) with:

```go
	mux.HandleFunc("POST /trigger", func(w http.ResponseWriter, _ *http.Request) {
		svc.Trigger()
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"ok": "true", "message": "Triggered"})
	})
```

Then remove the now-unused `"context"` import from the import block at the top of the file (it was only used by the old goroutine's `context.Background()`).

- [ ] **Step 6: Run the test to verify it passes**

Run: `go test ./internal/downloader/service/ -run TestPollAndDownloadWakesOnTrigger -v`
Expected: PASS.

- [ ] **Step 7: Run the full Go suite**

Run: `go test ./...`
Expected: all pass.

- [ ] **Step 8: Commit**

```bash
git add internal/downloader/service/downloader.go internal/downloader/service/trigger_test.go cmd/downloader/main.go
git commit -m "Wake the downloader poll loop on a trigger signal"
```

---

### Task 3: Metadata trigger channel

**Files:**
- Modify: `internal/metadata/service/anidb.go` (struct, `New`, add `Trigger`)
- Modify: `internal/metadata/service/runtime.go` (extract sync body, add trigger case to `StartTitlesSync` select)
- Modify: `cmd/metadata/main.go` (add `POST /trigger` handler)
- Create: `internal/metadata/service/trigger_test.go`

**Interfaces:**
- Consumes: `cycle.NewRecorder`, `cycle.Cycle` (existing)
- Produces: `(*AniDBService).Trigger()` — non-blocking send; `POST /trigger` on metadata HTTP (port 8081)

- [ ] **Step 1: Write the failing test**

```go
// internal/metadata/service/trigger_test.go
package service

import "testing"

func TestTriggerNonBlocking(t *testing.T) {
	svc := New(nil)

	// First trigger fills the buffered channel.
	svc.Trigger()
	select {
	case <-svc.trigger:
	default:
		t.Fatal("trigger not delivered")
	}

	// Channel is now empty again; a second trigger must not block.
	done := make(chan struct{})
	go func() {
		svc.Trigger()
		close(done)
	}()
	select {
	case <-done:
	default:
		t.Fatal("Trigger blocked on an empty channel")
	}

	// A trigger while the channel is full must be dropped, not block.
	svc.Trigger()
	svc.Trigger()
	select {
	case <-svc.trigger:
	default:
		t.Fatal("expected a pending trigger")
	}
}
```

- [ ] **Step 2: Run the test to verify it fails**

Run: `go test ./internal/metadata/service/ -run TestTriggerNonBlocking -v`
Expected: FAIL — `svc.trigger` does not exist (compile error).

- [ ] **Step 3: Add the trigger channel to AniDBService**

In `internal/metadata/service/anidb.go`, add the field to the struct (after `loadMu sync.Mutex`):

```go
	// trigger wakes the titles sync loop immediately (see Trigger).
	trigger chan struct{}
```

Initialize it in `New`:

```go
func New(db *bun.DB) *AniDBService {
	return &AniDBService{
		db:         db,
		httpClient: &http.Client{Timeout: 45 * time.Second},
		trigger:    make(chan struct{}, 1),
	}
}
```

Add the method at the end of the file:

```go
// Trigger wakes the titles sync loop immediately. Non-blocking: if a sync is
// already running or a wake is already pending, the signal is dropped.
func (s *AniDBService) Trigger() {
	select {
	case s.trigger <- struct{}{}:
	default:
	}
}
```

- [ ] **Step 4: Extract the sync body and wake the loop in StartTitlesSync**

In `internal/metadata/service/runtime.go`, replace `StartTitlesSync` with:

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
			nextInterval := s.syncOnce(rec, syncCycle)
			if nextInterval != interval {
				interval = nextInterval
				ticker.Reset(interval)
			}
		case <-s.trigger:
			nextInterval := s.syncOnce(rec, syncCycle)
			if nextInterval != interval {
				interval = nextInterval
				ticker.Reset(interval)
			}
		case <-stop:
			slog.Info("Titles sync loop stopped")
			return
		}
	}
}

// syncOnce runs one titles dump + anime-lists mapping sync, recording the
// cycle, and returns the next interval so the caller can reset its ticker.
func (s *AniDBService) syncOnce(rec *cycle.Recorder, syncCycle cycle.Cycle) time.Duration {
	_ = rec.Start(context.Background(), syncCycle)
	if err := s.LoadTitlesDump(); err != nil {
		slog.Warn("Scheduled titles sync failed", "error", err)
	}
	if err := s.LoadAnimeListsMapping(); err != nil {
		slog.Warn("Scheduled anime-lists sync failed", "error", err)
	}
	nextInterval := s.currentTitlesInterval()
	_ = rec.End(context.Background(), syncCycle, time.Now().Add(nextInterval))
	return nextInterval
}
```

- [ ] **Step 5: Add the HTTP trigger to cmd/metadata/main.go**

In `cmd/metadata/main.go`, after the `POST /prepare` registration (line 49), add:

```go
	mux.HandleFunc("POST /trigger", func(w http.ResponseWriter, _ *http.Request) {
		svc.Trigger()
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"ok": "true"})
	})
```

- [ ] **Step 6: Run the test to verify it passes**

Run: `go test ./internal/metadata/service/ -run TestTriggerNonBlocking -v`
Expected: PASS.

- [ ] **Step 7: Run the full Go suite**

Run: `go test ./...`
Expected: all pass.

- [ ] **Step 8: Commit**

```bash
git add internal/metadata/service/anidb.go internal/metadata/service/runtime.go internal/metadata/service/trigger_test.go cmd/metadata/main.go
git commit -m "Wake the AniDB title sync loop on a trigger signal"
```

---

### Task 4: Core trigger channels

**Files:**
- Modify: `internal/core/service/availability.go` (checker struct, `NewAvailabilityChecker`, `Trigger`, `Poll` select, `PollAvailability` signature)
- Modify: `internal/core/service/refresh.go` (`PollMetadataRefresh` returns a trigger func)
- Create: `internal/core/service/trigger_test.go`

**Interfaces:**
- Consumes: `cycle.NewRecorder`, `cycle.Schema`, `clients.MetadataClient` (existing)
- Produces: `PollAvailability(ctx, bunDB) func()` and `PollMetadataRefresh(ctx, mc) func()` — each starts its own goroutine and returns a non-blocking trigger func

- [ ] **Step 1: Write the failing tests**

```go
// internal/core/service/trigger_test.go
package service

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/kingbenny101/kbarr/internal/cycle"
	"github.com/kingbenny101/kbarr/internal/core/db"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/sqlitedialect"
	"github.com/uptrace/bun/driver/sqliteshim"
)

func TestPollAvailabilityWakesOnTrigger(t *testing.T) {
	sqldb, err := sql.Open(sqliteshim.ShimName, ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer sqldb.Close()
	testDB := bun.NewDB(sqldb, sqlitedialect.New())
	ctx := context.Background()
	if _, err := testDB.ExecContext(ctx, cycle.Schema); err != nil {
		t.Fatal(err)
	}
	if _, err := testDB.ExecContext(ctx, `CREATE TABLE settings (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
		updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
		deleted_at TIMESTAMP,
		key TEXT NOT NULL UNIQUE,
		value TEXT
	)`); err != nil {
		t.Fatal(err)
	}
	if _, err := testDB.ExecContext(ctx,
		`INSERT INTO settings (key, value) VALUES ('availabilityCheckInterval', '3600')`); err != nil {
		t.Fatal(err)
	}

	loopCtx, cancel := context.WithCancel(ctx)
	defer cancel()
	trigger := PollAvailability(loopCtx, testDB)

	// First pass runs immediately when the loop starts.
	deadline := time.Now().Add(5 * time.Second)
	var first time.Time
	for time.Now().Before(deadline) {
		var row struct {
			LastFinishedAt *time.Time `bun:"last_finished_at"`
		}
		err := testDB.NewSelect().TableExpr("cycle_status").
			ColumnExpr("last_finished_at").
			Where("service = 'core' AND cycle = 'availability'").
			Scan(ctx, &row)
		if err == nil && row.LastFinishedAt != nil {
			first = *row.LastFinishedAt
			break
		}
		time.Sleep(50 * time.Millisecond)
	}
	if first.IsZero() {
		t.Fatal("first availability pass never finished")
	}

	trigger()

	deadline = time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		var row struct {
			LastFinishedAt *time.Time `bun:"last_finished_at"`
		}
		err := testDB.NewSelect().TableExpr("cycle_status").
			ColumnExpr("last_finished_at").
			Where("service = 'core' AND cycle = 'availability'").
			Scan(ctx, &row)
		if err == nil && row.LastFinishedAt != nil && row.LastFinishedAt.After(first) {
			return
		}
		time.Sleep(50 * time.Millisecond)
	}
	t.Fatal("availability poll did not wake on trigger")
}

func TestPollMetadataRefreshWakesOnTrigger(t *testing.T) {
	sqldb, err := sql.Open(sqliteshim.ShimName, ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer sqldb.Close()
	testDB := bun.NewDB(sqldb, sqlitedialect.New())
	ctx := context.Background()
	if _, err := testDB.ExecContext(ctx, cycle.Schema); err != nil {
		t.Fatal(err)
	}

	oldDB := db.DB
	db.DB = testDB
	t.Cleanup(func() { db.DB = oldDB })

	loopCtx, cancel := context.WithCancel(ctx)
	defer cancel()
	trigger := PollMetadataRefresh(loopCtx, nil)

	// The first scheduled pass is delayed a minute; only a trigger can produce
	// a recorded pass within the timeout.
	trigger()

	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		var row struct {
			LastFinishedAt *time.Time `bun:"last_finished_at"`
		}
		err := testDB.NewSelect().TableExpr("cycle_status").
			ColumnExpr("last_finished_at").
			Where("service = 'core' AND cycle = 'metadata_refresh'").
			Scan(ctx, &row)
		if err == nil && row.LastFinishedAt != nil {
			return
		}
		time.Sleep(50 * time.Millisecond)
	}
	t.Fatal("metadata refresh did not wake on trigger")
}
```

- [ ] **Step 2: Run the tests to verify they fail**

Run: `go test ./internal/core/service/ -run 'TestPollAvailabilityWakesOnTrigger|TestPollMetadataRefreshWakesOnTrigger' -v`
Expected: FAIL — `PollAvailability` and `PollMetadataRefresh` don't return `func()` yet (compile error).

- [ ] **Step 3: Add the trigger channel to AvailabilityChecker**

In `internal/core/service/availability.go`, change the struct:

```go
type AvailabilityChecker struct {
	db        *bun.DB
	cache     map[string]folderScan // sanitized folder name -> last scan
	scanCount int                   // incremented each cycle; every 10th forces a full re-scan
	trigger   chan struct{}
}
```

Initialize it in `NewAvailabilityChecker`:

```go
func NewAvailabilityChecker(db *bun.DB) *AvailabilityChecker {
	return &AvailabilityChecker{db: db, cache: map[string]folderScan{}, trigger: make(chan struct{}, 1)}
}

// Trigger wakes the poll loop immediately. Non-blocking: if the loop is already
// running a pass or a wake is already pending, the signal is dropped.
func (c *AvailabilityChecker) Trigger() {
	select {
	case c.trigger <- struct{}{}:
	default:
	}
}
```

- [ ] **Step 4: Wake the loop in Poll and change PollAvailability's signature**

In `Poll` (around line 166), add the trigger case:

```go
		select {
		case <-time.After(interval):
			c.recordedCheck(ctx, rec, avCycle, interval)
		case <-c.trigger:
			c.recordedCheck(ctx, rec, avCycle, interval)
		case <-ctx.Done():
			return
		}
```

Change `PollAvailability` to own its goroutine and return a trigger func:

```go
// PollAvailability runs CheckAvailability on a loop, returning a non-blocking
// trigger that wakes the loop for an immediate pass. Kept as a package function
// for the existing caller; it owns a single checker so the mtime cache persists.
func PollAvailability(ctx context.Context, bunDB *bun.DB) func() {
	checker := NewAvailabilityChecker(bunDB)
	go checker.Poll(ctx)
	return checker.Trigger
}
```

- [ ] **Step 5: Change PollMetadataRefresh to return a trigger func**

In `internal/core/service/refresh.go`, replace `PollMetadataRefresh` with:

```go
// PollMetadataRefresh runs the background refresh loop: on each tick it
// re-fetches every still-airing library show and upserts any new episodes.
// Finished shows (those with an end date) are never re-fetched, so the
// per-tick AniDB load is bounded by the handful of shows currently airing —
// and each request still goes through the metadata service's 2s throttle and
// ban cooldown. Returns a non-blocking trigger that wakes the loop for an
// immediate pass.
func PollMetadataRefresh(ctx context.Context, mc *clients.MetadataClient) func() {
	trigger := make(chan struct{}, 1)
	go func() {
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
			case <-trigger:
			}
			_ = rec.Start(ctx, refreshCycle)
			runRefreshPass(ctx, mc)
			interval := config.GetMinutes(db.DB, "metadataRefreshInterval", 1440*time.Minute, 60*time.Minute)
			_ = rec.End(ctx, refreshCycle, time.Now().Add(interval))
			timer.Reset(interval)
		}
	}()
	return func() {
		select {
		case trigger <- struct{}{}:
		default:
		}
	}
}
```

- [ ] **Step 6: Run the tests to verify they pass**

Run: `go test ./internal/core/service/ -run 'TestPollAvailabilityWakesOnTrigger|TestPollMetadataRefreshWakesOnTrigger' -v`
Expected: PASS.

- [ ] **Step 7: Run the full Go suite**

Run: `go test ./...`
Expected: all pass (note: `cmd/core/main.go` still compiles because the old signatures were only used by `go coreservice.PollAvailability(...)` — the return value is simply discarded in a `go` statement, so this task stays green; the call site is updated in Task 5).

- [ ] **Step 8: Commit**

```bash
git add internal/core/service/availability.go internal/core/service/refresh.go internal/core/service/trigger_test.go
git commit -m "Expose trigger functions for the core poll loops"
```

---

### Task 5: Core API trigger endpoint

**Files:**
- Create: `internal/core/api/handlers/cycle_trigger.go`
- Modify: `internal/core/api/routes.go` (register route)
- Modify: `internal/core/api/server.go` (NewRouter signature)
- Modify: `cmd/core/main.go` (wire trigger funcs, update call sites)
- Create: `internal/core/api/handlers/cycle_trigger_test.go`

**Interfaces:**
- Consumes: `PollAvailability(ctx, db) func()`, `PollMetadataRefresh(ctx, mc) func()` (from Task 4); `handlers.SvcAddr` (existing)
- Produces: `handlers.TriggerCycle(coreTriggers map[string]func(), indexerAddr, downloaderAddr, metadataAddr string) func(context.Context, *TriggerCycleInput) (*struct{}, error)`; route `POST /api/cycles/{service}/{cycle}/trigger` (204, 404, 502)

- [ ] **Step 1: Write the failing tests**

```go
// internal/core/api/handlers/cycle_trigger_test.go
package handlers

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/danielgtaylor/huma/v2"
)

func TestTriggerCycleCore(t *testing.T) {
	called := false
	h := TriggerCycle(map[string]func(){"availability": func() { called = true }},
		"http://localhost:1", "http://localhost:1", "http://localhost:1")

	_, err := h(context.Background(), &TriggerCycleInput{Service: "core", Cycle: "availability"})
	if err != nil {
		t.Fatal(err)
	}
	if !called {
		t.Fatal("core trigger func was not called")
	}
}

func TestTriggerCycleUnknown(t *testing.T) {
	h := TriggerCycle(nil, "http://localhost:1", "http://localhost:1", "http://localhost:1")

	cases := []TriggerCycleInput{
		{Service: "core", Cycle: "bogus"},
		{Service: "bogus", Cycle: "availability"},
		{Service: "indexer", Cycle: "bogus"},
	}
	for _, in := range cases {
		_, err := h(context.Background(), &in)
		if err == nil {
			t.Fatalf("want error for %+v", in)
		}
		if em, ok := err.(*huma.ErrorModel); !ok || em.Status != http.StatusNotFound {
			t.Fatalf("want 404 for %+v, got %v", in, err)
		}
	}
}

func TestTriggerCycleProxy(t *testing.T) {
	var hits int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits++
		if r.Method != http.MethodPost || r.URL.Path != "/trigger" {
			t.Errorf("unexpected request %s %s", r.Method, r.URL.Path)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	h := TriggerCycle(nil, srv.URL, srv.URL, srv.URL)
	for _, in := range []TriggerCycleInput{
		{Service: "indexer", Cycle: "monitor_poll"},
		{Service: "downloader", Cycle: "downloader_poll"},
		{Service: "metadata", Cycle: "anidb_sync"},
	} {
		if _, err := h(context.Background(), &in); err != nil {
			t.Fatalf("proxy trigger %+v failed: %v", in, err)
		}
	}
	if hits != 3 {
		t.Fatalf("hits = %d, want 3", hits)
	}
}

func TestTriggerCycleUnreachable(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	addr := srv.URL
	srv.Close()

	h := TriggerCycle(nil, addr, addr, addr)
	_, err := h(context.Background(), &TriggerCycleInput{Service: "indexer", Cycle: "monitor_poll"})
	if err == nil {
		t.Fatal("want error for unreachable service")
	}
	if em, ok := err.(*huma.ErrorModel); !ok || em.Status != http.StatusBadGateway {
		t.Fatalf("want 502, got %v", err)
	}
}
```

- [ ] **Step 2: Run the tests to verify they fail**

Run: `go test ./internal/core/api/handlers/ -run TestTriggerCycle -v`
Expected: FAIL — `TriggerCycle` and `TriggerCycleInput` undefined (compile error).

- [ ] **Step 3: Create the handler**

```go
// internal/core/api/handlers/cycle_trigger.go
package handlers

import (
	"context"
	"net/http"
	"time"

	"github.com/danielgtaylor/huma/v2"
)

type TriggerCycleInput struct {
	Service string `path:"service"`
	Cycle   string `path:"cycle"`
}

// TriggerCycle wakes the given cycle's poll loop so it runs immediately.
// Core cycles are triggered in-process; worker-service cycles are proxied to
// the service's /trigger endpoint. Unknown service/cycle → 404; an unreachable
// service → 502.
func TriggerCycle(coreTriggers map[string]func(), indexerAddr, downloaderAddr, metadataAddr string) func(context.Context, *TriggerCycleInput) (*struct{}, error) {
	return func(ctx context.Context, in *TriggerCycleInput) (*struct{}, error) {
		switch in.Service {
		case "core":
			fn, ok := coreTriggers[in.Cycle]
			if !ok {
				return nil, huma.Error404NotFound("unknown core cycle %q", in.Cycle)
			}
			fn()
			return nil, nil
		case "indexer":
			if in.Cycle != "monitor_poll" && in.Cycle != "process_missing" {
				return nil, huma.Error404NotFound("unknown indexer cycle %q", in.Cycle)
			}
			return proxyTrigger(ctx, indexerAddr)
		case "downloader":
			if in.Cycle != "downloader_poll" {
				return nil, huma.Error404NotFound("unknown downloader cycle %q", in.Cycle)
			}
			return proxyTrigger(ctx, downloaderAddr)
		case "metadata":
			if in.Cycle != "anidb_sync" {
				return nil, huma.Error404NotFound("unknown metadata cycle %q", in.Cycle)
			}
			return proxyTrigger(ctx, metadataAddr)
		default:
			return nil, huma.Error404NotFound("unknown service %q", in.Service)
		}
	}
}

func proxyTrigger(ctx context.Context, addr string) (*struct{}, error) {
	client := &http.Client{Timeout: 5 * time.Second}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, addr+"/trigger", nil)
	if err != nil {
		return nil, huma.Error500InternalServerError("failed to build trigger request", err)
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, huma.Error502BadGateway("service unreachable: "+err.Error(), err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, huma.Error502BadGateway("service returned status %d", resp.StatusCode)
	}
	return nil, nil
}
```

- [ ] **Step 4: Register the route**

In `internal/core/api/routes.go`, change `RegisterRoutes` to accept the triggers and add the route after the `get-cycles` registration (line 26):

```go
func RegisterRoutes(api huma.API, mc *clients.MetadataClient, authStore *auth.Store, version string, coreTriggers map[string]func()) {
	...
	huma.Register(api, huma.Operation{OperationID: "trigger-cycle", Method: "POST", Path: "/api/cycles/{service}/{cycle}/trigger", Security: secured, Tags: []string{"system"}, Summary: "Run a cycle now", DefaultStatus: 204}, handlers.TriggerCycle(
		coreTriggers,
		handlers.SvcAddr("INDEXER_HEALTH_ADDR", "http://localhost:8082"),
		handlers.SvcAddr("DOWNLOADER_HEALTH_ADDR", "http://localhost:8083"),
		handlers.SvcAddr("METADATA_ADDR", "http://localhost:8081"),
	))
```

- [ ] **Step 5: Update NewRouter in internal/core/api/server.go**

Change the signature and pass through:

```go
func NewRouter(metadataClient *clients.MetadataClient, version string, authStore *auth.Store, coreTriggers map[string]func()) http.Handler {
	...
	RegisterRoutes(api, metadataClient, authStore, version, coreTriggers)
```

- [ ] **Step 6: Update cmd/core/main.go**

Replace the two poll starts and the router construction (lines 62-63 and 49):

```go
	avTrigger := coreservice.PollAvailability(bgCtx, db.DB)
	refreshTrigger := coreservice.PollMetadataRefresh(bgCtx, metadataClient)

	router := api.NewRouter(metadataClient, appVersion, authStore, map[string]func(){
		"availability":     avTrigger,
		"metadata_refresh": refreshTrigger,
	})
```

(Delete the two old `go coreservice.PollAvailability(...)` / `go coreservice.PollMetadataRefresh(...)` lines.)

- [ ] **Step 7: Run the handler tests**

Run: `go test ./internal/core/api/handlers/ -run TestTriggerCycle -v`
Expected: PASS.

- [ ] **Step 8: Run the full Go suite**

Run: `go test ./...`
Expected: all pass.

- [ ] **Step 9: Commit**

```bash
git add internal/core/api/handlers/cycle_trigger.go internal/core/api/handlers/cycle_trigger_test.go internal/core/api/routes.go internal/core/api/server.go cmd/core/main.go
git commit -m "Route cycle triggers through a single core endpoint"
```

---

### Task 6: System page run now buttons

**Files:**
- Modify: `web/src/pages/SystemPage.tsx` (Run column + button on table and cards, trigger state + handler)
- Modify: `web/src/pages/SystemPage.test.tsx` (new tests)

**Interfaces:**
- Consumes: `API_URL`, `apiFetch`, `showToast` from `@/utils`; `fetchCycles` (existing); `POST /api/cycles/{service}/{cycle}/trigger` (from Task 5)
- Produces: `runCycle(c: CycleStatus)` on `SystemPage`; `runningKey` prop threaded into `CycleTable` and `CycleCards`

- [ ] **Step 1: Write the failing tests**

Append inside the `describe("SystemPage", ...)` block in `web/src/pages/SystemPage.test.tsx` (after the "Missing search retry" test), and add the missing imports at the top of the file:

```tsx
import { fireEvent, render, screen, within } from "@testing-library/react"
```

New tests:

```tsx
    it("triggers a cycle via the run now button on desktop", async () => {
        const original = window.matchMedia
        Object.defineProperty(window, "matchMedia", {
            writable: true,
            value: (query: string) => ({
                matches: query.includes("min-width: 62em"),
                media: query,
                onchange: null,
                addListener: () => {},
                removeListener: () => {},
                addEventListener: () => {},
                removeEventListener: () => {},
                dispatchEvent: () => false,
            }),
        })

        try {
            const fetchMock = vi.fn(async (input: RequestInfo | URL, init?: RequestInit) => {
                const url = String(input)
                if (url.includes("/api/cycles")) return jsonResponse({ cycles: CYCLES })
                if (url.includes("/api/workers")) return jsonResponse(WORKERS)
                return jsonResponse({})
            })
            vi.stubGlobal("fetch", fetchMock)

            renderPage()

            const rows = await screen.findAllByRole("row")
            const target = rows.find((r) => r.textContent?.includes("Availability check"))!
            const button = within(target).getByRole("button", { name: "Run now" })
            fireEvent.click(button)

            expect(fetchMock).toHaveBeenCalledWith(
                expect.stringContaining("/api/cycles/core/availability/trigger"),
                expect.objectContaining({ method: "POST" }),
            )
        } finally {
            Object.defineProperty(window, "matchMedia", { writable: true, value: original })
        }
    })

    it("shows an error toast when a trigger fails", async () => {
        const fetchMock = vi.fn(async (input: RequestInfo | URL, init?: RequestInit) => {
            const url = String(input)
            if (url.includes("/api/cycles")) return jsonResponse({ cycles: CYCLES })
            if (url.includes("/api/workers")) return jsonResponse(WORKERS)
            return new Response(JSON.stringify({ error: { message: "service unreachable" } }), {
                status: 502,
                headers: { "Content-Type": "application/json" },
            })
        })
        vi.stubGlobal("fetch", fetchMock)

        renderPage()

        const buttons = await screen.findAllByRole("button", { name: "Run now" })
        fireEvent.click(buttons[0])

        expect(fetchMock).toHaveBeenCalledWith(
            expect.stringContaining("/api/cycles/"),
            expect.objectContaining({ method: "POST" }),
        )
    })
```

- [ ] **Step 2: Run the tests to verify they fail**

Run: `npx vitest run web/src/pages/SystemPage.test.tsx` (from `web/`)
Expected: FAIL — no button named "Run now" exists yet.

- [ ] **Step 3: Add the trigger state and handler to SystemPage**

In `web/src/pages/SystemPage.tsx`:
- Change the imports line to add `Button` (Mantine) and `showToast`:

```tsx
import { Badge, Box, Button, Group, Paper, SimpleGrid, Stack, Table, Text, Title } from "@mantine/core"
import { API_URL, apiFetch, showToast } from "@/utils"
```

- Add the `runningKey` state in `SystemPage` (next to `stale`):

```tsx
    const [runningKey, setRunningKey] = useState<string | null>(null)
```

- Add the handler after `fetchWorkers`:

```tsx
    const runCycle = async (c: CycleStatus) => {
        const key = `${c.service}/${c.cycle}`
        setRunningKey(key)
        try {
            const res = await apiFetch(`${API_URL}/api/cycles/${c.service}/${c.cycle}/trigger`, { method: "POST" })
            if (!res.ok) {
                let msg = `Failed to trigger ${c.display_name}`
                try {
                    const body = await res.json()
                    if (body?.error?.message) msg = body.error.message
                } catch {
                    /* keep the fallback message */
                }
                showToast(msg, "error")
                return
            }
            showToast(`${c.display_name} triggered`, "success")
            void fetchCycles()
        } finally {
            setRunningKey(null)
        }
    }
```

- [ ] **Step 4: Add the Run column to CycleTable**

Change `CycleTable`'s signature and body:

```tsx
function CycleTable({ cycles, offlineServices, now, runningKey, onRun }: {
    cycles: CycleStatus[]
    offlineServices: Set<string>
    now: Date
    runningKey: string | null
    onRun: (c: CycleStatus) => void
}) {
    return (
        <Table striped highlightOnHover>
            <Table.Thead>
                <Table.Tr>
                    <Table.Th style={{ width: "200px", minWidth: "160px" }}>Cycle</Table.Th>
                    <Table.Th style={{ width: "140px" }}>Last run</Table.Th>
                    <Table.Th style={{ width: "140px" }}>Next run</Table.Th>
                    <Table.Th style={{ width: "120px" }}>Duration</Table.Th>
                    <Table.Th style={{ width: "110px" }}>Run</Table.Th>
                </Table.Tr>
            </Table.Thead>
```

In the row body, after the Duration cell (line 137), add:

```tsx
                            <Table.Td>
                                <Button
                                    size="xs"
                                    variant="light"
                                    loading={runningKey === `${c.service}/${c.cycle}`}
                                    onClick={() => onRun(c)}
                                >
                                    Run now
                                </Button>
                            </Table.Td>
```

And change the empty-state `colSpan={4}` to `colSpan={5}`.

- [ ] **Step 5: Add the Run button to CycleCards**

Change `CycleCards`'s signature to add `runningKey` and `onRun`, and in the header group (around line 174), after the service badge, add the button:

```tsx
                            <Button
                                size="compact-xs"
                                variant="light"
                                loading={runningKey === `${c.service}/${c.cycle}`}
                                onClick={() => onRun(c)}
                            >
                                Run now
                            </Button>
```

- [ ] **Step 6: Thread the props in SystemPage's render**

Pass to both components (lines 272-274):

```tsx
                    {isDesktop ? (
                        <CycleTable cycles={allCycles} offlineServices={offlineServices} now={now} runningKey={runningKey} onRun={runCycle} />
                    ) : (
                        <CycleCards cycles={allCycles} offlineServices={offlineServices} now={now} runningKey={runningKey} onRun={runCycle} />
                    )}
```

- [ ] **Step 7: Run the tests to verify they pass**

Run: `npx vitest run web/src/pages/SystemPage.test.tsx`
Expected: PASS (all existing + 2 new).

- [ ] **Step 8: Run typecheck and lint**

Run: `npx tsc --noEmit && npm run lint`
Expected: clean.

- [ ] **Step 9: Run the full web suite**

Run: `npx vitest run`
Expected: all pass (baseline 62 + 2 new).

- [ ] **Step 10: Commit**

```bash
git add web/src/pages/SystemPage.tsx web/src/pages/SystemPage.test.tsx
git commit -m "Add run now buttons to the system page"
```

---

## Verification checklist

- [ ] `go test ./...` passes after every Go task
- [ ] `npx vitest run`, `npx tsc --noEmit`, `npm run lint` pass (in `web/`)
- [ ] Manual smoke test with overmind: open System page → click "Run now" on Availability check → the row's ring flips to running, then the last run timestamp updates; a fresh cycle row's next run time moves forward
- [ ] Manual smoke test: stop the indexer service, click "Run now" on Monitor poll → red error toast with "service unreachable"