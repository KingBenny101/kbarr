package db

import (
	"context"
	"database/sql"
	"testing"

	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/sqlitedialect"
	"github.com/uptrace/bun/driver/sqliteshim"

	"github.com/kingbenny101/kbarr/internal/models"
)

func newMonitorTestDB(t *testing.T) *bun.DB {
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

// Credits (OP/ED) and specials map to episode_number 0 in the UI. Upserting two
// of them for the same season must keep them as distinct monitor rows, each
// carrying its own external_id, so per-episode monitoring is accurate.
func TestUpsertMonitorKeepsSameNumberedCreditsSeparate(t *testing.T) {
	ctx := context.Background()
	db := newMonitorTestDB(t)

	opening := models.Monitor{LibraryID: 9, Title: "show", EpisodeTitle: "Opening", Season: 1, EpisodeNumber: 0, IsEpisode: true, Source: "anidb", ExternalID: "313664", Monitored: true}
	ending := models.Monitor{LibraryID: 9, Title: "show", EpisodeTitle: "Ending", Season: 1, EpisodeNumber: 0, IsEpisode: true, Source: "anidb", ExternalID: "313665", Monitored: true}

	if err := upsertMonitor(ctx, db, opening); err != nil {
		t.Fatalf("upsert opening: %v", err)
	}
	if err := upsertMonitor(ctx, db, ending); err != nil {
		t.Fatalf("upsert ending: %v", err)
	}

	var rows []models.Monitor
	if err := db.NewSelect().Model(&rows).Scan(ctx); err != nil {
		t.Fatalf("select: %v", err)
	}
	if len(rows) != 2 {
		t.Fatalf("got %d monitor row(s), want 2 distinct credit rows", len(rows))
	}
	byTitle := map[string]models.Monitor{}
	for _, r := range rows {
		byTitle[r.EpisodeTitle] = r
	}
	if got := byTitle["Opening"].ExternalID; got != "313664" {
		t.Errorf("Opening external_id = %q, want 313664", got)
	}
	if got := byTitle["Ending"].ExternalID; got != "313665" {
		t.Errorf("Ending external_id = %q, want 313665 (was overwritten by the Opening row)", got)
	}
}

// Re-monitoring the same credit after an unmonitor must update the original row
// (matched by external_id), not silently create or clobber a different one.
func TestUpsertMonitorRemonitorKeepsExternalID(t *testing.T) {
	ctx := context.Background()
	testDB := newMonitorTestDB(t)
	DB = testDB

	opening := models.Monitor{LibraryID: 9, Title: "show", EpisodeTitle: "Opening", Season: 1, EpisodeNumber: 0, IsEpisode: true, Source: "anidb", ExternalID: "313664", Monitored: false}
	if err := upsertMonitor(ctx, testDB, opening); err != nil {
		t.Fatalf("upsert: %v", err)
	}
	if err := UnmonitorByDetails(9, "313664"); err != nil {
		t.Fatalf("unmonitor: %v", err)
	}
	opening.Monitored = true
	if err := upsertMonitor(ctx, testDB, opening); err != nil {
		t.Fatalf("re-upsert: %v", err)
	}

	var rows []models.Monitor
	if err := testDB.NewSelect().Model(&rows).Scan(ctx); err != nil {
		t.Fatalf("select: %v", err)
	}
	if len(rows) != 1 {
		t.Fatalf("got %d rows, want 1", len(rows))
	}
	if rows[0].ExternalID != "313664" || !rows[0].Monitored {
		t.Errorf("row = external_id %q monitored %v, want 313664/true", rows[0].ExternalID, rows[0].Monitored)
	}
}

// Turning a whole season off must only affect regular episodes and the season
// row itself — credits (OP/ED, episode_number 0) are monitored independently.
func TestDeleteMonitorsBySeasonKeepsCredits(t *testing.T) {
	ctx := context.Background()
	testDB := newMonitorTestDB(t)
	DB = testDB

	seed := []models.Monitor{
		{LibraryID: 9, EpisodeTitle: "Episode 1", Season: 1, EpisodeNumber: 1, IsEpisode: true, Source: "anidb", ExternalID: "313608", Monitored: true},
		{LibraryID: 9, EpisodeTitle: "Episode 2", Season: 1, EpisodeNumber: 2, IsEpisode: true, Source: "anidb", ExternalID: "313609", Monitored: true},
		{LibraryID: 9, EpisodeTitle: "Opening", Season: 1, EpisodeNumber: 0, IsEpisode: true, Source: "anidb", ExternalID: "313664", Monitored: true},
		{LibraryID: 9, EpisodeTitle: "Ending", Season: 1, EpisodeNumber: 0, IsEpisode: true, Source: "anidb", ExternalID: "313665", Monitored: true},
		{LibraryID: 9, EpisodeTitle: "", Season: 1, EpisodeNumber: 0, IsEpisode: false, IsSeason: true, Source: "anidb", ExternalID: "19600", Monitored: true},
	}
	for _, m := range seed {
		if err := upsertMonitor(ctx, testDB, m); err != nil {
			t.Fatalf("seed %s: %v", m.EpisodeTitle, err)
		}
	}

	if err := DeleteMonitorsBySeason(9, 1); err != nil {
		t.Fatalf("delete by season: %v", err)
	}

	var rows []models.Monitor
	if err := testDB.NewSelect().Model(&rows).Scan(ctx); err != nil {
		t.Fatalf("select: %v", err)
	}
	byExternal := map[string]bool{}
	for _, r := range rows {
		byExternal[r.ExternalID] = r.Monitored
	}
	if byExternal["313608"] || byExternal["313609"] {
		t.Errorf("regular episodes should be unmonitored after season-off, got %v", byExternal)
	}
	if byExternal["19600"] {
		t.Error("season row should be unmonitored after season-off")
	}
	if !byExternal["313664"] || !byExternal["313665"] {
		t.Errorf("credits must keep monitoring after season-off, got %v", byExternal)
	}
}