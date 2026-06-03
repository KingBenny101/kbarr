package service

import (
	"context"
	"log/slog"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/kingbenny101/kbarr/services/core/internal/db"
	"github.com/kingbenny101/kbarr/shared/config"
	"github.com/uptrace/bun"
)

var invalidFilenameChars = regexp.MustCompile(`[/\\:*?"<>|]`)
var episodePattern = regexp.MustCompile(`(?i)[Ss](\d+)[Ee](\d+)`)

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

	// Phase 1: walk the media directory and mark monitors available based on files found.
	// Build a set of (season, episode) pairs that exist on disk, then update the DB.
	type seKey struct{ season, episode int64 }
	presentFiles := map[seKey]string{} // key → first matching file path

	titleDirs, err := os.ReadDir(mediaPath)
	if err != nil {
		slog.Warn("availability: cannot read media path", "path", mediaPath, "error", err)
		return
	}

	for _, titleDir := range titleDirs {
		if !titleDir.IsDir() {
			continue
		}
		files, err := os.ReadDir(filepath.Join(mediaPath, titleDir.Name()))
		if err != nil {
			continue
		}
		for _, f := range files {
			if f.IsDir() {
				continue
			}
			m := episodePattern.FindStringSubmatch(f.Name())
			if m == nil {
				continue
			}
			s, _ := strconv.ParseInt(m[1], 10, 64)
			e, _ := strconv.ParseInt(m[2], 10, 64)
			k := seKey{s, e}
			if _, seen := presentFiles[k]; !seen {
				presentFiles[k] = filepath.Join(mediaPath, titleDir.Name(), f.Name())
			}
		}
	}

	// Mark monitors available for every (season, episode) found on disk.
	for k, filePath := range presentFiles {
		res, err := bunDB.NewUpdate().
			Model((*db.Monitor)(nil)).
			Set("status = 'available', updated_at = now()").
			Where("is_episode = true AND episode_number = ? AND status != 'available' AND deleted_at IS NULL", k.episode).
			Where("season = ? OR (season IS NULL AND ? = 1)", k.season, k.season).
			Exec(ctx)
		if err != nil {
			slog.Warn("availability: failed to mark available", "error", err)
			continue
		}
		if n, _ := res.RowsAffected(); n > 0 {
			slog.Info("availability: episode(s) marked available", "season", k.season, "episode", k.episode, "file", filePath, "rows", n)
		}
	}

	// Phase 2: revert available monitors whose file is no longer present.
	var availableMonitors []db.Monitor
	bunDB.NewSelect().
		Model(&availableMonitors).
		Where("is_episode = true AND status = 'available' AND episode_number IS NOT NULL AND deleted_at IS NULL").
		Scan(ctx)

	for _, mon := range availableMonitors {
		season := int64(1)
		if mon.Season != nil && *mon.Season > 0 {
			season = *mon.Season
		}
		k := seKey{season, *mon.EpisodeNumber}
		if _, exists := presentFiles[k]; exists {
			continue
		}
		// File gone — revert. Use completed queue entry to decide target status.
		var queueCount int
		bunDB.NewSelect().TableExpr("download_queue").ColumnExpr("COUNT(*)").
			Where("monitor_id = ? AND status = 'completed' AND deleted_at IS NULL", mon.ID).
			Scan(ctx, &queueCount)
		revertStatus := "unmonitored"
		if queueCount > 0 {
			revertStatus = "monitored"
		}
		bunDB.NewUpdate().Model((*db.Monitor)(nil)).
			Set("status = ?, updated_at = now()", revertStatus).
			Where("id = ?", mon.ID).Exec(ctx)
		title := ""
		if mon.Title != nil {
			title = *mon.Title
		}
		slog.Info("availability: episode reverted (file removed)", "monitor_id", mon.ID, "title", title, "season", season, "episode", *mon.EpisodeNumber, "new_status", revertStatus)
	}

	// Phase 3: sync season monitors.
	var seasonMonitors []db.Monitor
	bunDB.NewSelect().
		Model(&seasonMonitors).
		Where("is_season = true AND status != 'unmonitored' AND deleted_at IS NULL AND library_id IS NOT NULL").
		Scan(ctx)

	for _, sm := range seasonMonitors {
		if sm.Status == nil {
			continue
		}
		var total, available int
		bunDB.NewSelect().TableExpr("monitors").
			ColumnExpr("COUNT(*) AS total, SUM(CASE WHEN status = 'available' THEN 1 ELSE 0 END) AS available").
			Where("library_id = ? AND is_episode = true AND deleted_at IS NULL", *sm.LibraryID).
			Scan(ctx, &total, &available)

		allAvailable := total > 0 && available == total
		isAvailable := *sm.Status == "available"

		if allAvailable && !isAvailable {
			bunDB.NewUpdate().Model((*db.Monitor)(nil)).
				Set("status = 'available', updated_at = now()").Where("id = ?", sm.ID).Exec(ctx)
			slog.Info("availability: season marked available", "monitor_id", sm.ID, "library_id", *sm.LibraryID)
		} else if !allAvailable && isAvailable {
			bunDB.NewUpdate().Model((*db.Monitor)(nil)).
				Set("status = 'monitored', updated_at = now()").Where("id = ?", sm.ID).Exec(ctx)
			slog.Info("availability: season reverted to monitored", "monitor_id", sm.ID, "library_id", *sm.LibraryID)
		}
	}
}
