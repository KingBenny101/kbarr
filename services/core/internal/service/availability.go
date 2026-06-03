package service

import (
	"context"
	"fmt"
	"log/slog"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/kingbenny101/kbarr/services/core/internal/db"
	"github.com/kingbenny101/kbarr/shared/config"
	"github.com/uptrace/bun"
)

var invalidFilenameChars = regexp.MustCompile(`[/\\:*?"<>|]`)

func sanitizeFilename(name string) string {
	return strings.TrimSpace(invalidFilenameChars.ReplaceAllString(name, "_"))
}

func PollAvailability(ctx context.Context, bunDB *bun.DB) {
	CheckAvailability(ctx, bunDB)

	for {
		interval := config.GetSeconds(bunDB, "availabilityCheckInterval", 10*time.Second, 10*time.Second)
		select {
		case <-time.After(interval):
			CheckAvailability(ctx, bunDB)
		case <-ctx.Done():
			return
		}
	}
}

func CheckAvailability(ctx context.Context, bunDB *bun.DB) {
	mediaPath := strings.TrimRight(config.Get(bunDB, "mediaPath", ""), "/")
	if mediaPath == "" {
		return
	}

	// Check episode monitors
	var monitors []db.Monitor
	err := bunDB.NewSelect().
		Model(&monitors).
		Where("is_episode = true AND status NOT IN ('available', 'unmonitored') AND deleted_at IS NULL").
		Scan(ctx)
	if err != nil {
		slog.Error("availability: failed to fetch episode monitors", "error", err)
		return
	}

	for _, mon := range monitors {
		if mon.EpisodeNumber == nil || mon.Title == nil {
			continue
		}
		title := sanitizeFilename(*mon.Title)
		season := int64(1)
		if mon.Season != nil && *mon.Season > 0 {
			season = *mon.Season
		}
		episode := *mon.EpisodeNumber

		pattern := filepath.Join(mediaPath, title, fmt.Sprintf("*S%02dE%02d*", season, episode))
		matches, err := filepath.Glob(pattern)
		if err != nil || len(matches) == 0 {
			continue
		}

		_, err = bunDB.NewUpdate().
			Model((*db.Monitor)(nil)).
			Set("status = 'available', updated_at = now()").
			Where("id = ?", mon.ID).
			Exec(ctx)
		if err != nil {
			slog.Warn("availability: failed to mark episode available", "monitor_id", mon.ID, "error", err)
		} else {
			slog.Info("availability: episode marked available", "monitor_id", mon.ID, "title", *mon.Title, "season", season, "episode", episode, "file", matches[0])
		}
	}

	// Promote season monitors whose every episode is now available
	var seasonMonitors []db.Monitor
	err = bunDB.NewSelect().
		Model(&seasonMonitors).
		Where("is_season = true AND status != 'available' AND status != 'unmonitored' AND deleted_at IS NULL AND library_id IS NOT NULL").
		Scan(ctx)
	if err != nil {
		slog.Error("availability: failed to fetch season monitors", "error", err)
		return
	}

	for _, sm := range seasonMonitors {
		var total, available int
		bunDB.NewSelect().
			TableExpr("monitors").
			ColumnExpr("COUNT(*) AS total, SUM(CASE WHEN status = 'available' THEN 1 ELSE 0 END) AS available").
			Where("library_id = ? AND is_episode = true AND deleted_at IS NULL", *sm.LibraryID).
			Scan(ctx, &total, &available)

		if total > 0 && available == total {
			bunDB.NewUpdate().
				Model((*db.Monitor)(nil)).
				Set("status = 'available', updated_at = now()").
				Where("id = ?", sm.ID).
				Exec(ctx)
			slog.Info("availability: season marked available", "monitor_id", sm.ID, "library_id", *sm.LibraryID)
		}
	}
}
