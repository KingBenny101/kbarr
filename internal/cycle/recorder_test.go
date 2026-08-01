package cycle_test

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

func newTestDB(t *testing.T) *bun.DB {
	t.Helper()
	sqldb, err := sql.Open(sqliteshim.ShimName, ":memory:")
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
