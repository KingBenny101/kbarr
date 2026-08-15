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