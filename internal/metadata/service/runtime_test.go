package service

import (
	"context"
	"database/sql"
	"os"
	"testing"
	"time"

	"github.com/kingbenny101/kbarr/internal/cycle"
	metadb "github.com/kingbenny101/kbarr/internal/metadata/db"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/sqlitedialect"
	"github.com/uptrace/bun/driver/sqliteshim"
)

// TestAniDBSyncRecordsStartupLoad verifies that StartTitlesSync records a
// cycle_status row for anidb_sync from the initial load on startup — not just
// from a later scheduled tick. The loads are made to fail fast (invalid cache
// files, no network) so the test is deterministic and offline; syncOnce must
// still record Start/End regardless of load success.
func TestAniDBSyncRecordsStartupLoad(t *testing.T) {
	sqldb, err := sql.Open(sqliteshim.ShimName, ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer sqldb.Close()
	sqldb.SetMaxOpenConns(1)
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

	// Point DataRootDir at a temp dir pre-seeded with invalid cache files so
	// the loads fail fast without touching the network.
	tmpDir := t.TempDir()
	dataRoot := tmpDir + "/data"
	if err := os.MkdirAll(dataRoot+"/metadata", 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(dataRoot+"/metadata/anidb-titles.xml", []byte("not xml"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(dataRoot+"/metadata/anime-list-full.json", []byte("not json"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Setenv("KBARR_DATA_DIR", dataRoot); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.Unsetenv("KBARR_DATA_DIR") })

	oldDB := metadb.DB
	metadb.DB = testDB
	t.Cleanup(func() { metadb.DB = oldDB })

	svc := New(testDB)
	stop := make(chan struct{})
	go svc.StartTitlesSync(stop)

	// The startup load should be recorded as a cycle within a few seconds.
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		var row struct {
			LastFinishedAt *time.Time `bun:"last_finished_at"`
		}
		err := testDB.NewSelect().TableExpr("cycle_status").
			ColumnExpr("last_finished_at").
			Where("service = 'metadata' AND cycle = 'anidb_sync'").
			Limit(1).
			Scan(ctx, &row)
		if err == nil && row.LastFinishedAt != nil {
			return
		}
		time.Sleep(50 * time.Millisecond)
	}
	t.Fatal("anidb_sync startup load was not recorded as a cycle")
}
