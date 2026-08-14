package handlers

import (
	"context"
	"database/sql"
	"testing"

	"github.com/kingbenny101/kbarr/internal/core/db"
	"github.com/kingbenny101/kbarr/internal/models"
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
	ctx := context.Background()
	_, err = db.ExecContext(ctx, `CREATE TABLE monitors (
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
	)`)
	if err != nil {
		t.Fatal(err)
	}
	return db
}

func TestTriggerMonitorSearch(t *testing.T) {
	ctx := context.Background()
	testDB := newTestDB(t)

	// Save and restore the global DB
	oldDB := db.DB
	db.DB = testDB
	t.Cleanup(func() { db.DB = oldDB })

	// Seed a monitored item
	mon := &models.Monitor{
		LibraryID:     9,
		Title:         "Test Show",
		EpisodeTitle:  "Episode 1",
		Season:        1,
		EpisodeNumber: 1,
		IsEpisode:     true,
		IsSeason:      false,
		Source:        "anidb",
		ExternalID:    "12345",
		Status:        "monitored",
		Monitored:     true,
	}
	if _, err := testDB.NewInsert().Model(mon).Exec(ctx); err != nil {
		t.Fatal(err)
	}

	// Call handler - this will fail because handler doesn't exist yet
	handler := TriggerMonitorSearch()
	input := &TriggerMonitorSearchInput{ID: mon.ID}
	_, err := handler(ctx, input)
	if err != nil {
		t.Fatalf("TriggerMonitorSearch failed: %v", err)
	}

	// Verify status becomes "pending"
	var updated models.Monitor
	if err := testDB.NewSelect().Model(&updated).Where("id = ?", mon.ID).Scan(ctx); err != nil {
		t.Fatal(err)
	}
	if updated.Status != "pending" {
		t.Errorf("expected status 'pending', got %q", updated.Status)
	}
}