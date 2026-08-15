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