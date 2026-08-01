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
