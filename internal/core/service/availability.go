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

	"github.com/kingbenny101/kbarr/internal/config"
	"github.com/kingbenny101/kbarr/internal/models"
	"github.com/kingbenny101/kbarr/internal/naming"
	"github.com/uptrace/bun"
)

var episodePattern = regexp.MustCompile(`(?i)[Ss](\d+)[Ee](\d+)`)

var (
	seasonInTitleRe = regexp.MustCompile(`(?i)\bseason\s*(\d+)\b`)
	ordinalSeasonRe = regexp.MustCompile(`(?i)\b(\d+)(?:st|nd|rd|th)\s+season\b`)
)

func effectiveSeason(title string, dbSeason int64) int64 {
	if dbSeason > 1 {
		return dbSeason
	}
	if m := ordinalSeasonRe.FindStringSubmatch(title); m != nil {
		if n, err := strconv.ParseInt(m[1], 10, 64); err == nil && n > 1 {
			return n
		}
	}
	if m := seasonInTitleRe.FindStringSubmatch(title); m != nil {
		if n, err := strconv.ParseInt(m[1], 10, 64); err == nil && n > 1 {
			return n
		}
	}
	return dbSeason
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

	type monitorRow struct {
		models.Monitor
		MediaFolder string `bun:"media_folder"`
	}
	var allMonitors []monitorRow
	if err := bunDB.NewSelect().
		TableExpr("monitors mon").
		ColumnExpr("mon.*, COALESCE(m.media_folder, '') AS media_folder").
		Join("LEFT JOIN media m ON m.id = mon.library_id").
		Where("mon.is_episode = true AND mon.episode_number IS NOT NULL AND mon.deleted_at IS NULL").
		Scan(ctx, &allMonitors); err != nil {
		slog.Error("availability: failed to fetch monitors", "error", err)
		return
	}
	slog.Info("availability: loaded monitors", "count", len(allMonitors))
	for i := range allMonitors {
		m := &allMonitors[i]
		title := m.Title
		ep := int64(m.EpisodeNumber)
		season := int64(1)
		if m.Season > 0 {
			season = int64(m.Season)
		}
		status := m.Status
		slog.Debug("availability: monitor loaded", "monitor_id", m.ID, "title", title, "season", season, "episode", ep, "status", status)
	}

	type seKey struct {
		title   string
		season  int64
		episode int64
	}
	type seKeyNoTitle struct{ season, episode int64 }
	monitorsByKey := map[seKey]*monitorRow{}
	monitorsByEpisode := map[seKeyNoTitle][]*monitorRow{}
	for i := range allMonitors {
		m := &allMonitors[i]
		if m.Title == "" || m.EpisodeNumber == 0 {
			continue
		}
		dbSeason := int64(1)
		if m.Season > 0 {
			dbSeason = int64(m.Season)
		}
		season := effectiveSeason(m.Title, dbSeason)
		folder := m.MediaFolder
		if folder == "" {
			folder = naming.SanitizeFilename(m.Title)
		}
		k := seKey{folder, season, int64(m.EpisodeNumber)}
		monitorsByKey[k] = m
		nk := seKeyNoTitle{season, int64(m.EpisodeNumber)}
		monitorsByEpisode[nk] = append(monitorsByEpisode[nk], m)
	}

	type fileKey struct {
		title   string
		season  int64
		episode int64
	}
	foundOnDisk := map[fileKey]string{}

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

			mk := seKey{dirName, s, e}
			mon, ok := monitorsByKey[mk]
			if !ok {
				nk := seKeyNoTitle{s, e}
				candidates := monitorsByEpisode[nk]
				slog.Info("availability: primary key miss, trying fallback", "title", dirName, "season", s, "episode", e, "fallback_candidates", len(candidates))
				for _, c := range candidates {
					slog.Info("availability: fallback candidate", "monitor_id", c.ID, "title_in_db", c.Title, "folder", c.MediaFolder)
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
			if mon.Available {
				slog.Debug("availability: already available", "monitor_id", mon.ID)
				continue
			}
			if _, err := bunDB.NewUpdate().Model((*models.Monitor)(nil)).
				Set("available = true, updated_at = now()").
				Where("id = ?", mon.ID).Exec(ctx); err != nil {
				slog.Warn("availability: failed to update monitor", "monitor_id", mon.ID, "error", err)
			} else {
				slog.Info("availability: episode marked available", "monitor_id", mon.ID, "title", dirName, "season", s, "episode", e)
			}
		}
	}

	// Phase 2: clear available=true for monitors whose file is no longer on disk.
	for i := range allMonitors {
		mon := &allMonitors[i]
		if !mon.Available || mon.Title == "" || mon.EpisodeNumber == 0 {
			continue
		}
		dbSeason := int64(1)
		if mon.Season > 0 {
			dbSeason = int64(mon.Season)
		}
		season := effectiveSeason(mon.Title, dbSeason)
		folder := mon.MediaFolder
		if folder == "" {
			folder = naming.SanitizeFilename(mon.Title)
		}
		fk := fileKey{folder, season, int64(mon.EpisodeNumber)}
		if _, exists := foundOnDisk[fk]; exists {
			continue
		}
		slog.Info("availability: file missing for available monitor", "monitor_id", mon.ID, "title", mon.Title, "season", season, "episode", int64(mon.EpisodeNumber))

		if mon.Status == "downloaded" {
			bunDB.NewUpdate().Model((*models.Monitor)(nil)).
				Set("available = false, status = 'pending', updated_at = now()").
				Where("id = ?", mon.ID).Exec(ctx)
			slog.Info("availability: episode reset to pending (kbarr download removed)", "monitor_id", mon.ID, "title", mon.Title)
		} else {
			bunDB.NewUpdate().Model((*models.Monitor)(nil)).
				Set("available = false, updated_at = now()").
				Where("id = ?", mon.ID).Exec(ctx)
			slog.Info("availability: episode marked unavailable (manual file removed)", "monitor_id", mon.ID, "title", mon.Title)
		}
	}

	// Phase 3: sync season monitors.
	var seasonMonitors []models.Monitor
	bunDB.NewSelect().Model(&seasonMonitors).
		Where("is_season = true AND status != 'unmonitored' AND deleted_at IS NULL AND library_id != 0").
		Scan(ctx)

	for _, sm := range seasonMonitors {
		if sm.Status == "" {
			continue
		}
		var total, available int
		bunDB.NewSelect().TableExpr("monitors").
			ColumnExpr("COUNT(*) AS total, SUM(CASE WHEN available THEN 1 ELSE 0 END) AS available").
			Where("library_id = ? AND is_episode = true AND deleted_at IS NULL", sm.LibraryID).
			Scan(ctx, &total, &available)
		slog.Info("availability: season monitor check", "monitor_id", sm.ID, "library_id", sm.LibraryID, "total_episodes", total, "available_episodes", available, "current_status", sm.Status)

		allAvailable := total > 0 && available == total
		if allAvailable {
			// All episodes present — reconcile both the flag and the status, so a
			// season left in 'missing'/'searching' by the indexer reflects reality.
			if !sm.Available || sm.Status != "downloaded" {
				bunDB.NewUpdate().Model((*models.Monitor)(nil)).
					Set("available = true, status = 'downloaded', updated_at = now()").
					Where("id = ?", sm.ID).Exec(ctx)
				slog.Info("availability: season marked available", "monitor_id", sm.ID)
			}
		} else {
			// Incomplete — clear availability and, if it had been marked complete,
			// reset to pending so the indexer searches a season pack again.
			if sm.Available {
				bunDB.NewUpdate().Model((*models.Monitor)(nil)).
					Set("available = false, status = 'pending', updated_at = now()").
					Where("id = ?", sm.ID).Exec(ctx)
				slog.Info("availability: season marked unavailable", "monitor_id", sm.ID)
			} else if sm.Status == "downloaded" {
				bunDB.NewUpdate().Model((*models.Monitor)(nil)).
					Set("status = 'pending', updated_at = now()").
					Where("id = ?", sm.ID).Exec(ctx)
				slog.Info("availability: season status reset to pending", "monitor_id", sm.ID)
			}
		}
	}

	slog.Info("availability: check done")
}
