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