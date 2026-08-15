package service

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/kingbenny101/kbarr/internal/cycle"
	"github.com/kingbenny101/kbarr/internal/models"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/sqlitedialect"
	"github.com/uptrace/bun/driver/sqliteshim"
)

func newIndexerTestDB(t *testing.T) *bun.DB {
	t.Helper()
	sqldb, err := sql.Open(sqliteshim.ShimName, ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { sqldb.Close() })
	sqldb.SetMaxOpenConns(1)
	db := bun.NewDB(sqldb, sqlitedialect.New())
	ctx := context.Background()
	if _, err := db.ExecContext(ctx, `CREATE TABLE monitors (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
		updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
		deleted_at TIMESTAMP,
		library_id INTEGER,
		title TEXT,
		episode_title TEXT,
		season INTEGER,
		episode_number INTEGER,
		is_episode BOOLEAN,
		is_season BOOLEAN,
		source TEXT,
		external_id TEXT,
		status TEXT,
		available BOOLEAN NOT NULL DEFAULT false,
		monitored BOOLEAN NOT NULL DEFAULT false,
		quality TEXT NOT NULL DEFAULT '',
		subtitles TEXT NOT NULL DEFAULT ''
	)`); err != nil {
		t.Fatal(err)
	}
	if _, err := db.ExecContext(ctx, cycle.Schema); err != nil {
		t.Fatal(err)
	}
	return db
}

func seedMonitor(t *testing.T, db *bun.DB, title, status string, monitored bool, updatedAt time.Time) int64 {
	t.Helper()
	mon := &models.Monitor{
		LibraryID:     7,
		Title:         title,
		Season:        1,
		EpisodeNumber: 1,
		IsEpisode:     true,
		Status:        status,
		Monitored:     monitored,
	}
	ctx := context.Background()
	if _, err := db.NewInsert().Model(mon).Exec(ctx); err != nil {
		t.Fatal(err)
	}
	if _, err := db.NewUpdate().Model((*models.Monitor)(nil)).
		Set("updated_at = ?", updatedAt).
		Where("id = ?", mon.ID).Exec(ctx); err != nil {
		t.Fatal(err)
	}
	return int64(mon.ID)
}

// A triggered pass must re-search monitored missing monitors even when the
// configured retry interval has not elapsed — otherwise "Run now" does
// nothing for recently-marked-missing items.
func TestRetryMissingBypassesTheIntervalGate(t *testing.T) {
	ctx := context.Background()
	db := newIndexerTestDB(t)
	s := New(db)

	old := time.Now().Add(-2 * time.Hour)
	a := seedMonitor(t, db, "Show A", "missing", true, old)
	b := seedMonitor(t, db, "Show B", "missing", true, time.Now().Add(-30*time.Minute))
	c := seedMonitor(t, db, "Show C", "pending", true, old)
	d := seedMonitor(t, db, "Show D", "missing", false, old)

	if didWork := s.retryMissing(ctx); !didWork {
		t.Fatal("retryMissing = false; want true")
	}

	for _, id := range []int64{a, b} {
		var mon models.Monitor
		if err := db.NewSelect().Model(&mon).Where("id = ?", id).Scan(ctx); err != nil {
			t.Fatal(err)
		}
		if mon.Status != "missing" {
			t.Errorf("monitor %d status = %q; want missing (no indexer providers in test)", id, mon.Status)
		}
		if !mon.UpdatedAt.After(old) {
			t.Errorf("monitor %d was not re-searched: updated_at %v not after seed %v", id, mon.UpdatedAt, old)
		}
	}

	var pending models.Monitor
	if err := db.NewSelect().Model(&pending).Where("id = ?", c).Scan(ctx); err != nil {
		t.Fatal(err)
	}
	if pending.UpdatedAt.Sub(old).Abs() > time.Second {
		t.Errorf("pending monitor was touched: updated_at = %v, want %v", pending.UpdatedAt, old)
	}

	var unmonitored models.Monitor
	if err := db.NewSelect().Model(&unmonitored).Where("id = ?", d).Scan(ctx); err != nil {
		t.Fatal(err)
	}
	if unmonitored.UpdatedAt.Sub(old).Abs() > time.Second {
		t.Errorf("unmonitored monitor was touched: updated_at = %v, want %v", unmonitored.UpdatedAt, old)
	}

	var count int
	if err := db.NewRaw(`SELECT COUNT(*) FROM cycle_status WHERE service = 'indexer' AND cycle = 'process_missing' AND state = 'idle' AND last_finished_at IS NOT NULL`).Scan(ctx, &count); err != nil {
		t.Fatal(err)
	}
	if count == 0 {
		t.Error("process_missing cycle was not recorded")
	}
}

func TestRetryMissingNothingToRetry(t *testing.T) {
	ctx := context.Background()
	db := newIndexerTestDB(t)
	s := New(db)
	seedMonitor(t, db, "Show C", "pending", true, time.Now().Add(-2*time.Hour))
	seedMonitor(t, db, "Show D", "missing", false, time.Now().Add(-2*time.Hour))

	if didWork := s.retryMissing(ctx); didWork {
		t.Fatal("retryMissing = true with no monitored missing monitors; want false")
	}
	var count int
	if err := db.NewRaw(`SELECT COUNT(*) FROM cycle_status WHERE service = 'indexer' AND cycle = 'process_missing'`).Scan(ctx, &count); err != nil {
		t.Fatal(err)
	}
	if count != 0 {
		t.Errorf("process_missing recorded %d times with nothing to retry", count)
	}
}

// Waking the poll loop through Trigger() must force a missing retry even when
// the monitor is not yet due for its scheduled retry.
func TestTriggerForcesMissingRetry(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	db := newIndexerTestDB(t)
	s := New(db)
	seededAt := time.Now().Add(-30 * time.Minute)
	id := seedMonitor(t, db, "Show A", "missing", true, seededAt)

	done := make(chan struct{})
	go func() {
		s.PollAndQueue(ctx)
		close(done)
	}()
	defer func() { cancel(); <-done }()

	time.Sleep(250 * time.Millisecond)
	s.Trigger()

	deadline := time.Now().Add(3 * time.Second)
	for {
		var mon models.Monitor
		if err := db.NewSelect().Model(&mon).Where("id = ?", id).Scan(ctx); err != nil {
			t.Fatal(err)
		}
		if mon.UpdatedAt.After(seededAt) {
			return
		}
		if time.Now().After(deadline) {
			t.Fatal("trigger did not force a missing retry within 3s")
		}
		time.Sleep(25 * time.Millisecond)
	}
}
