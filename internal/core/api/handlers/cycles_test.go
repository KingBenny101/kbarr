package handlers

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
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
	sqldb, err := sql.Open(sqliteshim.ShimName, ":memory:")
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
	if out.Body.ServerTime.IsZero() {
		t.Error("server_time is zero")
	}
}

func TestGetCyclesEmpty(t *testing.T) {
	ctx := context.Background()
	sqldb, err := sql.Open(sqliteshim.ShimName, ":memory:")
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
	raw, err := json.Marshal(out.Body)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(raw, []byte(`"cycles":[]`)) {
		t.Errorf("body = %s, want cycles to marshal as [] not null", raw)
	}
}
