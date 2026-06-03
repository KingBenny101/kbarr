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
		slog.Debug("availability: mediaPath not configured, skipping")
		return
	}
	slog.Info("availability: check start", "media_path", mediaPath)

	// Load all episode monitors once, indexed by sanitized title for O(1) lookup.
	var allMonitors []db.Monitor
	if err := bunDB.NewSelect().
		Model(&allMonitors).
		Where("is_episode = true AND episode_number IS NOT NULL AND deleted_at IS NULL").
		Scan(ctx); err != nil {
		slog.Error("availability: failed to fetch monitors", "error", err)
		return
	}
	slog.Info("availability: loaded monitors", "count", len(allMonitors))
	for i := range allMonitors {
		m := &allMonitors[i]
		title := ""
		if m.Title != nil {
			title = *m.Title
		}
		ep := int64(0)
		if m.EpisodeNumber != nil {
			ep = *m.EpisodeNumber
		}
		season := int64(1)
		if m.Season != nil && *m.Season > 0 {
			season = *m.Season
		}
		status := ""
		if m.Status != nil {
			status = *m.Status
		}
		slog.Debug("availability: monitor loaded", "monitor_id", m.ID, "title", title, "sanitized_title", sanitizeFilename(title), "season", season, "episode", ep, "status", status)
	}

	type seKey struct {
		title   string
		season  int64
		episode int64
	}
	// Primary index: sanitizedTitle + season + episode
	monitorsByKey := map[seKey]*db.Monitor{}
	// Fallback index: season + episode only (used when title doesn't match directory name)
	type seKeyNoTitle struct{ season, episode int64 }
	monitorsByEpisode := map[seKeyNoTitle][]*db.Monitor{}
	for i := range allMonitors {
		m := &allMonitors[i]
		if m.Title == nil || m.EpisodeNumber == nil {
			continue
		}
		season := int64(1)
		if m.Season != nil && *m.Season > 0 {
			season = *m.Season
		}
		k := seKey{sanitizeFilename(*m.Title), season, *m.EpisodeNumber}
		monitorsByKey[k] = m
		nk := seKeyNoTitle{season, *m.EpisodeNumber}
		monitorsByEpisode[nk] = append(monitorsByEpisode[nk], m)
	}

	// Phase 1: walk media directory — filesystem is the source of truth.
	type fileKey struct {
		title   string
		season  int64
		episode int64
	}
	foundOnDisk := map[fileKey]string{} // key → file path

	titleDirs, err := os.ReadDir(mediaPath)
	if err != nil {
		slog.Warn("availability: cannot read media path", "path", mediaPath, "error", err)
		return
	}
	slog.Info("availability: scanning title dirs", "count", len(titleDirs))

	for _, titleDir := range titleDirs {
		if !titleDir.IsDir() {
			continue
		}
		dirName := titleDir.Name()
		dirPath := filepath.Join(mediaPath, dirName)
		files, err := os.ReadDir(dirPath)
		if err != nil {
			slog.Warn("availability: cannot read title dir", "path", dirPath, "error", err)
			continue
		}
		slog.Info("availability: scanning dir", "title", dirName, "files", len(files))

		for _, f := range files {
			if f.IsDir() {
				continue
			}
			m := episodePattern.FindStringSubmatch(f.Name())
			if m == nil {
				slog.Debug("availability: no S__E__ pattern in filename", "file", f.Name())
				continue
			}
			s, _ := strconv.ParseInt(m[1], 10, 64)
			e, _ := strconv.ParseInt(m[2], 10, 64)
			fk := fileKey{dirName, s, e}
			if _, seen := foundOnDisk[fk]; !seen {
				foundOnDisk[fk] = filepath.Join(dirPath, f.Name())
			}
			slog.Info("availability: file found", "title", dirName, "season", s, "episode", e, "file", f.Name())

			// Mark matching monitor available — try title+season+episode first,
			// fall back to season+episode alone in case title formats differ.
			mk := seKey{dirName, s, e}
			mon, ok := monitorsByKey[mk]
			if !ok {
				nk := seKeyNoTitle{s, e}
				candidates := monitorsByEpisode[nk]
				slog.Info("availability: primary key miss, trying fallback", "title", dirName, "season", s, "episode", e, "fallback_candidates", len(candidates))
				for _, c := range candidates {
					slog.Info("availability: fallback candidate", "monitor_id", c.ID, "title_in_db", *c.Title, "sanitized", sanitizeFilename(*c.Title))
				}
				if len(candidates) == 1 {
					mon = candidates[0]
					ok = true
				} else if len(candidates) > 1 {
					slog.Info("availability: multiple fallback candidates, skipping to avoid wrong match", "season", s, "episode", e)
					continue
				}
			}
			if !ok {
				slog.Info("availability: no monitor found for file", "title", dirName, "season", s, "episode", e)
				continue
			}
			status := ""
			if mon.Status != nil {
				status = *mon.Status
			}
			if status == "available" {
				slog.Debug("availability: already available", "monitor_id", mon.ID)
				continue
			}
			if _, err := bunDB.NewUpdate().Model((*db.Monitor)(nil)).
				Set("status = 'available', updated_at = now()").
				Where("id = ?", mon.ID).Exec(ctx); err != nil {
				slog.Warn("availability: failed to update monitor", "monitor_id", mon.ID, "error", err)
			} else {
				slog.Info("availability: episode marked available", "monitor_id", mon.ID, "title", dirName, "season", s, "episode", e)
			}
		}
	}

	// Phase 2: revert available monitors whose file is no longer on disk.
	for i := range allMonitors {
		mon := &allMonitors[i]
		if mon.Status == nil || *mon.Status != "available" || mon.Title == nil {
			continue
		}
		season := int64(1)
		if mon.Season != nil && *mon.Season > 0 {
			season = *mon.Season
		}
		fk := fileKey{sanitizeFilename(*mon.Title), season, *mon.EpisodeNumber}
		if _, exists := foundOnDisk[fk]; exists {
			continue
		}
		slog.Info("availability: file missing for available monitor", "monitor_id", mon.ID, "title", *mon.Title, "season", season, "episode", *mon.EpisodeNumber)
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
		slog.Info("availability: episode reverted", "monitor_id", mon.ID, "title", *mon.Title, "season", season, "episode", *mon.EpisodeNumber, "new_status", revertStatus)
	}

	// Phase 3: sync season monitors.
	var seasonMonitors []db.Monitor
	bunDB.NewSelect().Model(&seasonMonitors).
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
		slog.Info("availability: season monitor check", "monitor_id", sm.ID, "library_id", *sm.LibraryID, "total_episodes", total, "available_episodes", available, "current_status", *sm.Status)

		allAvailable := total > 0 && available == total
		isAvailable := *sm.Status == "available"
		if allAvailable && !isAvailable {
			bunDB.NewUpdate().Model((*db.Monitor)(nil)).
				Set("status = 'available', updated_at = now()").Where("id = ?", sm.ID).Exec(ctx)
			slog.Info("availability: season marked available", "monitor_id", sm.ID)
		} else if !allAvailable && isAvailable {
			bunDB.NewUpdate().Model((*db.Monitor)(nil)).
				Set("status = 'monitored', updated_at = now()").Where("id = ?", sm.ID).Exec(ctx)
			slog.Info("availability: season reverted to monitored", "monitor_id", sm.ID)
		}
	}

	slog.Info("availability: check done")
}
