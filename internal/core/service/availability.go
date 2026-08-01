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
// because anime sequels live in their own folder and the season digit the parser
// stamps is unreliable.
var episodePattern = regexp.MustCompile(`(?i)[Ss]\d+[Ee](\d+)`)

// AvailabilityChecker reconciles monitor rows with files on disk. It keeps a
// per-folder cache keyed by directory mtime so that, on cycles where nothing on
// disk changed, no directory is re-read or re-parsed.
type AvailabilityChecker struct {
	db        *bun.DB
	cache     map[string]folderScan // sanitized folder name -> last scan
	scanCount int                   // incremented each cycle; every 10th forces a full re-scan
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

// ReconcileLibraryAvailability marks any episode monitors of a single library
// whose files already exist on disk as available + downloaded (recording quality
// and subtitles), then re-aggregates that library's season monitors. It is meant
// to run synchronously right after monitors are created, so monitoring an anime
// that is already on disk neither triggers a redundant search/download (the
// indexer only claims monitors with available = false) nor leaves the episode
// stuck at 'pending'. It scans only the one show folder, not the whole media path.
func ReconcileLibraryAvailability(ctx context.Context, bunDB *bun.DB, libraryID uint) {
	NewAvailabilityChecker(bunDB).reconcileLibrary(ctx, libraryID)
}

func (c *AvailabilityChecker) reconcileLibrary(ctx context.Context, libraryID uint) {
	mediaPath := strings.TrimRight(config.Get(c.db, "mediaPath", ""), "/")
	if mediaPath == "" || libraryID == 0 {
		return
	}

	var row struct {
		Title       string `bun:"title"`
		MediaFolder string `bun:"media_folder"`
	}
	if err := c.db.NewSelect().TableExpr("media").
		ColumnExpr("title, COALESCE(media_folder, '') AS media_folder").
		Where("id = ? AND deleted_at IS NULL", libraryID).
		Scan(ctx, &row); err != nil {
		slog.Warn("reconcile: media not found", "library_id", libraryID, "error", err)
		return
	}

	dirPath := resolveShowDir(mediaPath, row.MediaFolder, row.Title)
	if dirPath == "" {
		// Nothing on disk yet — leave the monitors pending for the indexer.
		return
	}
	episodes := c.readFolderEpisodes(dirPath, allowedVideoExts(c.db))
	if len(episodes) == 0 {
		return
	}

	var mons []models.Monitor
	if err := c.db.NewSelect().Model(&mons).
		Where("library_id = ? AND is_episode = true AND deleted_at IS NULL", libraryID).
		Scan(ctx); err != nil {
		slog.Warn("reconcile: failed to load monitors", "library_id", libraryID, "error", err)
		return
	}

	for _, mon := range mons {
		path, ok := episodes[int64(mon.EpisodeNumber)]
		if !ok {
			continue
		}
		if mon.Available && mon.Status == "downloaded" {
			continue
		}
		q, subs := c.detectQualityAndSubs(path)
		if _, err := c.db.NewUpdate().Model((*models.Monitor)(nil)).
			Set("available = true, status = 'downloaded', quality = ?, subtitles = ?, updated_at = now()", q, subs).
			Where("id = ?", mon.ID).Exec(ctx); err != nil {
			slog.Warn("reconcile: failed to update monitor", "monitor_id", mon.ID, "error", err)
			continue
		}
		slog.Info("reconcile: episode already on disk, marked downloaded", "monitor_id", mon.ID, "title", mon.Title, "episode", mon.EpisodeNumber, "quality", q, "subtitles", subs)
	}

	c.syncSeasonMonitors(ctx)
}

// resolveShowDir returns the on-disk directory for a show, trying the media
// folder and title verbatim first, then falling back to a sanitized-name match
// against the directories under mediaPath. Returns "" when nothing matches.
func resolveShowDir(mediaPath, mediaFolder, title string) string {
	for _, cand := range []string{mediaFolder, title} {
		if cand == "" {
			continue
		}
		p := filepath.Join(mediaPath, cand)
		if fi, err := os.Stat(p); err == nil && fi.IsDir() {
			return p
		}
	}
	want := map[string]bool{}
	for _, n := range []string{mediaFolder, title} {
		if k := naming.SanitizeFilename(n); k != "" {
			want[k] = true
		}
	}
	ents, err := os.ReadDir(mediaPath)
	if err != nil {
		return ""
	}
	for _, e := range ents {
		if want[naming.SanitizeFilename(e.Name())] {
			return filepath.Join(mediaPath, e.Name())
		}
	}
	return ""
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
	// Every 10th cycle we force a full re-scan regardless of mtime, to guard
	// against network mounts (NFS/SMB) where directory mtime is not reliably
	// updated when files inside the directory change.
	c.scanCount++
	filesByFolder, changedFolders := c.scanDisk(mediaPath, allowedExts, c.scanCount%10 == 0)

	// foundByLibEp records every (library, episode) whose file was located on disk
	// this cycle, using the same folder→library resolution Phase 1 marks available
	// with. Phase 2 consults it instead of re-deriving a folder string per monitor,
	// so the two phases can never disagree (a media_folder that doesn't match the
	// on-disk directory name previously made Phase 1 mark available while Phase 2
	// marked the same episode missing, flip-flopping every cycle).
	foundByLibEp := map[libEpKey]bool{}

	// ── Phase 1: mark episodes available ────────────────────────────────────
	for folder, episodes := range filesByFolder {
		if ambiguous[folder] {
			slog.Debug("availability: ambiguous folder, skipping", "folder", folder)
			continue
		}
		libID, ok := folderToLib[folder]
		if !ok {
			slog.Warn("availability: folder maps to no library", "folder", folder)
			continue
		}
		for ep, path := range episodes {
			foundByLibEp[libEpKey{int64(libID), ep}] = true
			mon, ok := monitorsByLibEp[libEpKey{int64(libID), ep}]
			if !ok {
				slog.Warn("availability: no monitor for parsed episode", "folder", folder, "episode", ep)
				continue
			}

			// Probe the file for quality + subtitles when the monitor has neither
			// recorded yet, or when this folder changed on disk this cycle (a file
			// was added/removed — e.g. a subtitle dropped in by hand). The folder
			// mtime cache keeps this off the hot path for unchanged directories, so
			// re-probing only happens on real changes.
			setQuality, setSubs, doProbe := "", "", mon.Quality == "" && mon.Subtitles == "" || changedFolders[folder]
			if doProbe {
				setQuality, setSubs = c.detectQualityAndSubs(path)
			}

			// Only skip the update when the row is already fully healed: file
			// recorded and status promoted. A stale status (e.g. 'missing' left
			// by the indexer) must be repaired even when nothing changed on
			// disk, so episodes never show missing once their file is present.
			if mon.Available && mon.Status == "downloaded" && !doProbe {
				continue
			}

			newStatus := statusForFoundFile(mon.Status)
			q := c.db.NewUpdate().Model((*models.Monitor)(nil)).
				Set("available = true, updated_at = now()")
			if newStatus != mon.Status {
				q = q.Set("status = ?", newStatus)
			}
			if doProbe {
				q = q.Set("quality = ?, subtitles = ?", setQuality, setSubs)
			}
			if _, err := q.Where("id = ?", mon.ID).Exec(ctx); err != nil {
				slog.Warn("availability: failed to update monitor", "monitor_id", mon.ID, "error", err)
			} else {
				mon.Available = true
				mon.Quality, mon.Subtitles = setQuality, setSubs
				slog.Info("availability: episode marked available", "monitor_id", mon.ID, "title", mon.Title, "episode", ep, "quality", setQuality, "subtitles", setSubs)
			}
		}
	}

	// ── Phase 2: clear availability for monitors whose file is gone ───────────
	for i := range allMonitors {
		mon := &allMonitors[i]
		if !mon.Available || mon.EpisodeNumber == 0 {
			continue
		}
		if foundByLibEp[libEpKey{int64(mon.LibraryID), int64(mon.EpisodeNumber)}] {
			continue
		}
		if mon.Status == "downloaded" {
			// kbarr's own download vanished — re-search, but throttled: 'missing'
			// is only retried after missingRetryInterval, not every cycle.
			c.db.NewUpdate().Model((*models.Monitor)(nil)).
				Set("available = false, status = 'missing', quality = '', subtitles = '', updated_at = now()").
				Where("id = ?", mon.ID).Exec(ctx)
			slog.Info("availability: episode file gone, reset to missing", "monitor_id", mon.ID, "library_id", mon.LibraryID, "episode_number", mon.EpisodeNumber, "title", mon.Title)
		} else {
			c.db.NewUpdate().Model((*models.Monitor)(nil)).
				Set("available = false, quality = '', subtitles = '', updated_at = now()").
				Where("id = ?", mon.ID).Exec(ctx)
			slog.Info("availability: episode marked unavailable (manual file removed)", "monitor_id", mon.ID, "library_id", mon.LibraryID, "episode_number", mon.EpisodeNumber, "title", mon.Title)
		}
	}

	// ── Phase 3: aggregate season monitors (per library, season-agnostic) ─────
	c.syncSeasonMonitors(ctx)

	slog.Debug("availability: check done")
}

// scanDisk returns folder -> (episode -> path) for every show directory under
// mediaPath. Directories whose mtime is unchanged since the last cycle are served
// from the cache without a ReadDir. Folder keys are sanitized for matching.
// scanDisk additionally returns the set of folders that were freshly read this
// cycle (mtime changed or forced), so the caller can re-probe their episodes —
// a subtitle dropped in by hand bumps the directory mtime, and re-probing is how
// its language gets noticed for an episode whose quality was already recorded.
func (c *AvailabilityChecker) scanDisk(mediaPath string, allowedExts map[string]bool, forceRescan bool) (map[string]map[int64]string, map[string]bool) {
	titleDirs, err := os.ReadDir(mediaPath)
	if err != nil {
		slog.Warn("availability: cannot read media path", "path", mediaPath, "error", err)
		return nil, nil
	}

	result := map[string]map[int64]string{}
	changed := map[string]bool{}
	live := map[string]bool{}

	for _, titleDir := range titleDirs {
		dirPath := filepath.Join(mediaPath, titleDir.Name())
		// DirEntry.IsDir() returns false for symlinks even when they point to a
		// directory. Fall back to os.Stat (which follows symlinks) so that
		// symlinked show folders are treated the same as real ones.
		isDir := titleDir.IsDir()
		if !isDir {
			if fi, err := os.Stat(dirPath); err == nil {
				isDir = fi.IsDir()
			}
		}
		if !isDir {
			continue
		}
		folder := naming.SanitizeFilename(titleDir.Name())
		if folder == "" {
			continue
		}
		live[folder] = true

		info, err := os.Stat(dirPath)
		if err != nil {
			slog.Warn("availability: cannot stat dir", "path", dirPath, "error", err)
			continue
		}
		mtime := info.ModTime()

		if !forceRescan {
			if cached, ok := c.cache[folder]; ok && cached.mtime.Equal(mtime) {
				result[folder] = cached.episodes
				continue
			}
		}

		episodes := c.readFolderEpisodes(dirPath, allowedExts)
		c.cache[folder] = folderScan{mtime: mtime, episodes: episodes}
		result[folder] = episodes
		changed[folder] = true
		slog.Debug("availability: scanned dir", "folder", folder, "episodes", len(episodes))
	}

	// Forget cache entries for directories that no longer exist.
	for folder := range c.cache {
		if !live[folder] {
			delete(c.cache, folder)
		}
	}

	return result, changed
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

// statusForFoundFile returns the status a monitor should move to when its file
// is found on disk. Stale statuses left by the indexer or by monitor creation
// are promoted to downloaded; statuses owned by other actors (an in-flight
// download, an explicit unmonitor) are left untouched.
func statusForFoundFile(status string) string {
	switch status {
	case "pending", "searching", "missing", "monitored":
		return "downloaded"
	default:
		return status
	}
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
