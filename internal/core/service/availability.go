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
	"github.com/kingbenny101/kbarr/internal/parser"
	"github.com/uptrace/bun"
)

// episodePattern is the fast path for SxxExx filenames produced by the hardlinker.
// Only the episode group (2) is used — the season digit is intentionally ignored
// because anime sequels live in their own folder and the season digit guessit
// stamps is unreliable.
var episodePattern = regexp.MustCompile(`(?i)[Ss]\d+[Ee](\d+)`)

// AvailabilityChecker reconciles monitor rows with files on disk. It keeps a
// per-folder cache keyed by directory mtime so that, on cycles where nothing on
// disk changed, no directory is re-read or re-parsed.
type AvailabilityChecker struct {
	db    *bun.DB
	cache map[string]folderScan // sanitized folder name -> last scan
}

// folderScan is a cached directory listing: the mtime it was taken at and the
// episode numbers found in it (episode -> first file path).
type folderScan struct {
	mtime    time.Time
	episodes map[int64]string
}

// NewAvailabilityChecker constructs a checker with an empty cache.
func NewAvailabilityChecker(db *bun.DB) *AvailabilityChecker {
	return &AvailabilityChecker{db: db, cache: map[string]folderScan{}}
}

// PollAvailability runs CheckAvailability on a loop. Kept as a package function
// for the existing caller; it owns a single checker so the mtime cache persists.
func PollAvailability(ctx context.Context, bunDB *bun.DB) {
	NewAvailabilityChecker(bunDB).Poll(ctx)
}

// CheckAvailability runs a single reconciliation pass. Used by the manual trigger
// endpoint; it uses a throwaway checker (full scan, no cache reuse).
func CheckAvailability(ctx context.Context, bunDB *bun.DB) {
	NewAvailabilityChecker(bunDB).Check(ctx)
}

func (c *AvailabilityChecker) Poll(ctx context.Context) {
	c.Check(ctx)
	for {
		interval := config.GetSeconds(c.db, "availabilityCheckInterval", 60*time.Second, 10*time.Second)
		select {
		case <-time.After(interval):
			c.Check(ctx)
		case <-ctx.Done():
			return
		}
	}
}

type monitorRow struct {
	models.Monitor
	MediaFolder string `bun:"media_folder"`
}

// libEpKey identifies an episode monitor within a specific library.
type libEpKey struct {
	libraryID int64
	episode   int64
}

func (c *AvailabilityChecker) Check(ctx context.Context) {
	mediaPath := strings.TrimRight(config.Get(c.db, "mediaPath", ""), "/")
	if mediaPath == "" {
		slog.Debug("availability: mediaPath not configured, skipping")
		return
	}
	slog.Debug("availability: check start", "media_path", mediaPath)

	allowedExts := allowedVideoExts(c.db)

	var allMonitors []monitorRow
	if err := c.db.NewSelect().
		TableExpr("monitors mon").
		ColumnExpr("mon.*, COALESCE(m.media_folder, '') AS media_folder").
		Join("LEFT JOIN media m ON m.id = mon.library_id").
		Where("mon.is_episode = true AND mon.episode_number IS NOT NULL AND mon.deleted_at IS NULL").
		Scan(ctx, &allMonitors); err != nil {
		slog.Error("availability: failed to fetch monitors", "error", err)
		return
	}
	slog.Debug("availability: loaded monitors", "count", len(allMonitors))

	// Index episode monitors by (library_id, episode). The folder scopes the show,
	// so season is irrelevant to matching.
	monitorsByLibEp := map[libEpKey]*monitorRow{}
	for i := range allMonitors {
		m := &allMonitors[i]
		if m.EpisodeNumber == 0 {
			continue
		}
		monitorsByLibEp[libEpKey{int64(m.LibraryID), int64(m.EpisodeNumber)}] = m
	}

	folderToLib, ambiguous := c.indexMediaFolders(ctx)

	// Scan disk into folder -> episode -> path, reusing cached listings for any
	// directory whose mtime is unchanged since the last cycle.
	filesByFolder := c.scanDisk(mediaPath, allowedExts)

	// ── Phase 1: mark episodes available ────────────────────────────────────
	for folder, episodes := range filesByFolder {
		if ambiguous[folder] {
			slog.Debug("availability: ambiguous folder, skipping", "folder", folder)
			continue
		}
		libID, ok := folderToLib[folder]
		if !ok {
			slog.Debug("availability: folder maps to no library, skipping", "folder", folder)
			continue
		}
		for ep := range episodes {
			mon, ok := monitorsByLibEp[libEpKey{int64(libID), ep}]
			if !ok {
				slog.Debug("availability: no monitor for episode", "folder", folder, "episode", ep)
				continue
			}
			if mon.Available {
				continue
			}
			if _, err := c.db.NewUpdate().Model((*models.Monitor)(nil)).
				Set("available = true, updated_at = now()").
				Where("id = ?", mon.ID).Exec(ctx); err != nil {
				slog.Warn("availability: failed to update monitor", "monitor_id", mon.ID, "error", err)
			} else {
				mon.Available = true
				slog.Info("availability: episode marked available", "monitor_id", mon.ID, "title", mon.Title, "episode", ep)
			}
		}
	}

	// ── Phase 2: clear availability for monitors whose file is gone ───────────
	for i := range allMonitors {
		mon := &allMonitors[i]
		if !mon.Available || mon.EpisodeNumber == 0 {
			continue
		}
		folder := monitorFolder(mon)
		if eps, ok := filesByFolder[folder]; ok {
			if _, present := eps[int64(mon.EpisodeNumber)]; present {
				continue
			}
		}
		if mon.Status == "downloaded" {
			// kbarr's own download vanished — re-search, but throttled: 'missing'
			// is only retried after missingRetryInterval, not every cycle.
			c.db.NewUpdate().Model((*models.Monitor)(nil)).
				Set("available = false, status = 'missing', updated_at = now()").
				Where("id = ?", mon.ID).Exec(ctx)
			slog.Info("availability: episode file gone, reset to missing", "monitor_id", mon.ID, "title", mon.Title)
		} else {
			c.db.NewUpdate().Model((*models.Monitor)(nil)).
				Set("available = false, updated_at = now()").
				Where("id = ?", mon.ID).Exec(ctx)
			slog.Info("availability: episode marked unavailable (manual file removed)", "monitor_id", mon.ID, "title", mon.Title)
		}
	}

	// ── Phase 3: aggregate season monitors (per library, season-agnostic) ─────
	c.syncSeasonMonitors(ctx)

	slog.Debug("availability: check done")
}

// scanDisk returns folder -> (episode -> path) for every show directory under
// mediaPath. Directories whose mtime is unchanged since the last cycle are served
// from the cache without a ReadDir. Folder keys are sanitized for matching.
func (c *AvailabilityChecker) scanDisk(mediaPath string, allowedExts map[string]bool) map[string]map[int64]string {
	titleDirs, err := os.ReadDir(mediaPath)
	if err != nil {
		slog.Warn("availability: cannot read media path", "path", mediaPath, "error", err)
		return nil
	}

	result := map[string]map[int64]string{}
	live := map[string]bool{}

	for _, titleDir := range titleDirs {
		if !titleDir.IsDir() {
			continue
		}
		folder := naming.SanitizeFilename(titleDir.Name())
		if folder == "" {
			continue
		}
		live[folder] = true
		dirPath := filepath.Join(mediaPath, titleDir.Name())

		info, err := titleDir.Info()
		if err != nil {
			slog.Warn("availability: cannot stat dir", "path", dirPath, "error", err)
			continue
		}
		mtime := info.ModTime()

		if cached, ok := c.cache[folder]; ok && cached.mtime.Equal(mtime) {
			result[folder] = cached.episodes
			continue
		}

		episodes := c.readFolderEpisodes(dirPath, allowedExts)
		c.cache[folder] = folderScan{mtime: mtime, episodes: episodes}
		result[folder] = episodes
		slog.Debug("availability: scanned dir", "folder", folder, "episodes", len(episodes))
	}

	// Forget cache entries for directories that no longer exist.
	for folder := range c.cache {
		if !live[folder] {
			delete(c.cache, folder)
		}
	}

	return result
}

// readFolderEpisodes reads one directory and returns episode -> first file path.
func (c *AvailabilityChecker) readFolderEpisodes(dirPath string, allowedExts map[string]bool) map[int64]string {
	episodes := map[int64]string{}
	files, err := os.ReadDir(dirPath)
	if err != nil {
		slog.Warn("availability: cannot read title dir", "path", dirPath, "error", err)
		return episodes
	}
	for _, f := range files {
		if f.IsDir() {
			continue
		}
		ep, ok := parseEpisodeNumber(f.Name(), allowedExts)
		if !ok {
			continue
		}
		if _, seen := episodes[ep]; !seen {
			episodes[ep] = filepath.Join(dirPath, f.Name())
		}
	}
	return episodes
}

// indexMediaFolders maps sanitized folder names to library (media) IDs. Each
// media row contributes its media_folder and its sanitized title as keys. Keys
// that resolve to more than one library are flagged ambiguous and skipped at
// match time so two shows can never be conflated.
func (c *AvailabilityChecker) indexMediaFolders(ctx context.Context) (map[string]uint, map[string]bool) {
	var rows []struct {
		ID          uint   `bun:"id"`
		Title       string `bun:"title"`
		MediaFolder string `bun:"media_folder"`
	}
	folderToLib := map[string]uint{}
	ambiguous := map[string]bool{}
	if err := c.db.NewSelect().
		TableExpr("media").
		ColumnExpr("id, title, COALESCE(media_folder, '') AS media_folder").
		Where("deleted_at IS NULL").
		Scan(ctx, &rows); err != nil {
		slog.Error("availability: failed to index media folders", "error", err)
		return folderToLib, ambiguous
	}

	add := func(name string, id uint) {
		key := naming.SanitizeFilename(name)
		if key == "" {
			return
		}
		if existing, ok := folderToLib[key]; ok && existing != id {
			ambiguous[key] = true
			return
		}
		folderToLib[key] = id
	}
	for _, r := range rows {
		add(r.MediaFolder, r.ID)
		add(r.Title, r.ID)
	}
	return folderToLib, ambiguous
}

// syncSeasonMonitors reconciles season monitors with the availability of their
// episode monitors. It is purely DB-driven and runs every cycle.
func (c *AvailabilityChecker) syncSeasonMonitors(ctx context.Context) {
	var seasonMonitors []models.Monitor
	if err := c.db.NewSelect().Model(&seasonMonitors).
		Where("is_season = true AND status != 'unmonitored' AND deleted_at IS NULL AND library_id != 0").
		Scan(ctx); err != nil {
		slog.Error("availability: failed to fetch season monitors", "error", err)
		return
	}

	for _, sm := range seasonMonitors {
		if sm.Status == "" {
			continue
		}
		var total, available int
		c.db.NewSelect().TableExpr("monitors").
			ColumnExpr("COUNT(*) AS total, SUM(CASE WHEN available THEN 1 ELSE 0 END) AS available").
			Where("library_id = ? AND is_episode = true AND deleted_at IS NULL", sm.LibraryID).
			Scan(ctx, &total, &available)

		allAvailable := total > 0 && available == total
		if allAvailable {
			if !sm.Available || sm.Status != "downloaded" {
				c.db.NewUpdate().Model((*models.Monitor)(nil)).
					Set("available = true, status = 'downloaded', updated_at = now()").
					Where("id = ?", sm.ID).Exec(ctx)
				slog.Info("availability: season marked available", "monitor_id", sm.ID, "title", sm.Title)
			}
			continue
		}

		// Incomplete. Reset to 'missing' (not 'pending') so the indexer's
		// missingRetryInterval throttles re-search instead of re-queuing every cycle.
		if sm.Available {
			c.db.NewUpdate().Model((*models.Monitor)(nil)).
				Set("available = false, status = 'missing', updated_at = now()").
				Where("id = ?", sm.ID).Exec(ctx)
			slog.Info("availability: season incomplete, reset to missing", "monitor_id", sm.ID, "title", sm.Title, "available", available, "total", total)
		} else if sm.Status == "downloaded" {
			c.db.NewUpdate().Model((*models.Monitor)(nil)).
				Set("status = 'missing', updated_at = now()").
				Where("id = ?", sm.ID).Exec(ctx)
			slog.Info("availability: season incomplete, reset to missing", "monitor_id", sm.ID, "title", sm.Title, "available", available, "total", total)
		}
	}
}

// monitorFolder returns the sanitized folder key for an episode monitor.
func monitorFolder(m *monitorRow) string {
	folder := m.MediaFolder
	if folder == "" {
		folder = m.Title
	}
	return naming.SanitizeFilename(folder)
}

// parseEpisodeNumber extracts an episode number from a filename, returning false
// for non-video files or names with no detectable episode. The SxxExx fast path
// is tried first, then the anitogo-based parser (which also handles absolute
// numbering and Japanese/hash notation).
func parseEpisodeNumber(filename string, allowedExts map[string]bool) (int64, bool) {
	ext := strings.ToLower(filepath.Ext(filename))
	if len(allowedExts) > 0 && !allowedExts[ext] {
		return 0, false
	}
	if m := episodePattern.FindStringSubmatch(filename); m != nil {
		if n, err := strconv.ParseInt(m[1], 10, 64); err == nil && n > 0 {
			return n, true
		}
	}
	if ep := parser.Parse(filename).Episode; ep > 0 {
		return int64(ep), true
	}
	return 0, false
}

// allowedVideoExts returns the configured allowed video extensions as a set.
func allowedVideoExts(db *bun.DB) map[string]bool {
	raw := config.Get(db, "allowedVideoExtensions", ".mkv,.mp4,.avi,.mov,.wmv,.m4v")
	exts := map[string]bool{}
	for _, e := range strings.Split(raw, ",") {
		if t := strings.TrimSpace(strings.ToLower(e)); t != "" {
			exts[t] = true
		}
	}
	return exts
}
