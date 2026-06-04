package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"mime/multipart"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/kingbenny101/kbarr/services/downloader/internal/models"
	"github.com/kingbenny101/kbarr/shared/config"
	"github.com/uptrace/bun"
)

type DownloaderService struct {
	db         *bun.DB
	qbtClient  *http.Client // maintains qBittorrent session (cookie jar)
	httpClient *http.Client // plain client for downloading .torrent files
}

func NewDownloaderService(db *bun.DB) *DownloaderService {
	jar, _ := cookiejar.New(nil)
	return &DownloaderService{
		db:         db,
		qbtClient: &http.Client{Timeout: 30 * time.Second, Jar: jar},
		// Stop following redirects when the destination is a magnet link
		httpClient: &http.Client{
			Timeout: 60 * time.Second,
			CheckRedirect: func(req *http.Request, _ []*http.Request) error {
				if req.URL.Scheme == "magnet" {
					return http.ErrUseLastResponse
				}
				return nil
			},
		},
	}
}

type torrentInfo struct {
	Hash        string  `json:"hash"`
	Name        string  `json:"name"`
	ContentPath string  `json:"content_path"`
	State       string  `json:"state"`
	Size        int64   `json:"size"`
	Progress    float64 `json:"progress"`
	ETA         int64   `json:"eta"`
	SavePath    string  `json:"save_path"`
	Category    string  `json:"category"`
}

// pathBase extracts the final path component handling both '/' (POSIX) and '\'
// (Windows) separators. filepath.Base on Linux does not split on '\'.
func pathBase(p string) string {
	if i := strings.LastIndexAny(p, "/\\"); i >= 0 {
		return p[i+1:]
	}
	return p
}


// contentName returns the actual on-disk name of the torrent content.
// content_path is the real filesystem path; its base name is the folder or file
// that qBittorrent created, which may differ from the display Name.
// pathBase is used instead of filepath.Base to correctly handle Windows paths
// returned by a natively-running qBittorrent instance.
func (t torrentInfo) contentName() string {
	if t.ContentPath != "" {
		return pathBase(t.ContentPath)
	}
	return pathBase(t.Name)
}

// contentRelPath returns the path of the torrent content relative to qBittorrent's
// save directory. Unlike contentName, this preserves subdirectories — so for a
// torrent whose root folder contains a file, it returns "FolderName/file.mkv"
// instead of just "file.mkv".
func (t torrentInfo) contentRelPath() string {
	if t.ContentPath == "" {
		return pathBase(t.Name)
	}
	if t.SavePath != "" {
		cp := strings.ReplaceAll(t.ContentPath, "\\", "/")
		sp := strings.TrimRight(strings.ReplaceAll(t.SavePath, "\\", "/"), "/")
		if strings.HasPrefix(cp, sp+"/") {
			return filepath.FromSlash(cp[len(sp)+1:])
		}
	}
	return pathBase(t.ContentPath)
}

func (s *DownloaderService) qbtConfig() (qbtURL, username, password string) {
	qbtURL = strings.TrimRight(config.Get(s.db, "qbittorrentUrl", "http://localhost:8080"), "/")
	username = config.Get(s.db, "qbittorrentUsername", "")
	password = config.Get(s.db, "qbittorrentPassword", "")
	return
}

func (s *DownloaderService) login(ctx context.Context, qbtURL, username, password string) error {
	if username == "" {
		return nil
	}
	resp, err := s.qbtClient.PostForm(qbtURL+"/api/v2/auth/login", url.Values{
		"username": {username},
		"password": {password},
	})
	if err != nil {
		return fmt.Errorf("qBittorrent login failed: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		return fmt.Errorf("qBittorrent login returned %s", resp.Status)
	}
	return nil
}

func (s *DownloaderService) pollInterval() time.Duration {
	return config.GetSeconds(s.db, "downloaderInterval", 1*time.Second, 1*time.Second)
}

func (s *DownloaderService) PollAndDownload(ctx context.Context) {
	s.ProcessPending(ctx)
	s.UpdateDownloading(ctx)

	for {
		interval := s.pollInterval()
		select {
		case <-time.After(interval):
			s.ProcessPending(ctx)
			s.UpdateDownloading(ctx)
		case <-ctx.Done():
			return
		}
	}
}

func (s *DownloaderService) ProcessPending(ctx context.Context) {
	var entries []models.DownloadQueue
	err := s.db.NewSelect().
		Model(&entries).
		Where("status = 'pending' AND deleted_at IS NULL").
		Scan(ctx)
	if err != nil {
		slog.Error("Failed to fetch pending download queue entries", "error", err)
		return
	}
	if len(entries) == 0 {
		return
	}

	qbtURL, username, password := s.qbtConfig()
	if err := s.login(ctx, qbtURL, username, password); err != nil {
		slog.Error("qBittorrent login failed", "error", err)
		return
	}

	for _, entry := range entries {
		if entry.TorrentURL == nil || *entry.TorrentURL == "" {
			continue
		}

		hash, err := s.addTorrent(ctx, qbtURL, *entry.TorrentURL)
		if err != nil {
			slog.Error("Failed to add torrent — blacklisting and re-queuing", "id", entry.ID, "error", err)
			s.blacklistTorrent(ctx, entry)
			s.db.NewUpdate().Model((*models.DownloadQueue)(nil)).
				Set("deleted_at = now(), updated_at = now()").
				Where("id = ?", entry.ID).Exec(ctx)
			if entry.MonitorID != nil {
				s.db.NewUpdate().Model((*models.Monitor)(nil)).
					Set("status = 'pending', updated_at = now()").
					Where("id = ?", *entry.MonitorID).Exec(ctx)
				s.releaseEpisodeMonitors(ctx, *entry.MonitorID)
			}
			continue
		}

		// Wait briefly then query qBittorrent to resolve the hash (for .torrent uploads)
		// and the actual torrent name (which determines the subdirectory).
		downloadPath := strings.TrimRight(config.Get(s.db, "downloadPath", ""), "/")
		var savePath string
		time.Sleep(2 * time.Second)
		if items, err := s.fetchTorrents(ctx, qbtURL, hash, ""); err == nil && len(items) > 0 {
			t := items[0]
			if hash == "" {
				hash = t.Hash
			}
			if downloadPath != "" {
				savePath = filepath.Join(downloadPath, t.contentName())
			}
		} else if hash == "" && entry.TorrentName != nil && *entry.TorrentName != "" {
			// Fallback: search by name if hash still unknown
			if items, err := s.fetchTorrents(ctx, qbtURL, "", ""); err == nil {
				for _, t := range items {
					if t.Name == *entry.TorrentName {
						hash = t.Hash
						if downloadPath != "" {
							savePath = filepath.Join(downloadPath, t.contentName())
						}
						break
					}
				}
			}
		}

		_, err = s.db.NewUpdate().
			Model((*models.DownloadQueue)(nil)).
			Set("status = 'downloading', torrent_hash = ?, save_path = ?, progress_updated_at = now(), updated_at = now()", hash, savePath).
			Where("id = ?", entry.ID).
			Exec(ctx)
		if err != nil {
			slog.Error("Failed to update download queue status", "id", entry.ID, "error", err)
			continue
		}

		if entry.MonitorID != nil {
			s.db.NewUpdate().
				Model((*models.Monitor)(nil)).
				Set("status = 'downloading', updated_at = now()").
				Where("id = ?", *entry.MonitorID).
				Exec(ctx)
		}

		slog.Info("Torrent added", "id", entry.ID, "hash", hash, "save_path", savePath)
	}
}

func (s *DownloaderService) UpdateDownloading(ctx context.Context) {
	var entries []models.DownloadQueue
	err := s.db.NewSelect().
		Model(&entries).
		Where("status = 'downloading' AND torrent_hash IS NOT NULL AND deleted_at IS NULL").
		Scan(ctx)
	if err != nil {
		slog.Error("Failed to fetch downloading entries", "error", err)
		return
	}
	if len(entries) == 0 {
		return
	}

	qbtURL, username, password := s.qbtConfig()
	if err := s.login(ctx, qbtURL, username, password); err != nil {
		slog.Error("qBittorrent login failed", "error", err)
		return
	}

	for _, entry := range entries {
		if entry.TorrentHash == nil {
			continue
		}

		items, err := s.fetchTorrents(ctx, qbtURL, *entry.TorrentHash, "")
		if err != nil || len(items) == 0 {
			continue
		}

		t := items[0]

		prevProgress := float64(0)
		if entry.Progress != nil {
			prevProgress = *entry.Progress
		}

		if t.Progress != prevProgress {
			// Progress changed — update both progress and the timestamp
			s.db.NewUpdate().
				Model((*models.DownloadQueue)(nil)).
				Set("progress = ?, progress_updated_at = now(), updated_at = now()", t.Progress).
				Where("id = ?", entry.ID).
				Exec(ctx)
		}

		if t.Progress >= 1.0 {
			if t.State == "moving" {
				slog.Info("Torrent is moving files — deferring completion", "id", entry.ID, "hash", *entry.TorrentHash)
				continue
			}
			// Rebuild save_path from the torrent's actual content path + local downloadPath.
			// contentRelPath preserves subdirectories (e.g. TorrentFolder/file.mkv) so
			// we land at the right location when the torrent has a root folder.
			// We use downloadPath (not t.SavePath) because qBittorrent's path is from its
			// own container and is not accessible to us.
			if rel := t.contentRelPath(); rel != "" {
				downloadPath := strings.TrimRight(config.Get(s.db, "downloadPath", ""), "/")
				if downloadPath != "" {
					actualPath := filepath.Join(downloadPath, rel)
					entry.SavePath = &actualPath
				}
			}
			s.onComplete(ctx, entry)
			continue
		}

		// Stall detection: if progress hasn't moved for stallTimeout seconds, remove the torrent
		stallTimeout := config.GetSeconds(s.db, "stallTimeout", 300*time.Second, 0)
		if entry.ProgressUpdatedAt == nil {
			slog.Debug("Stall check skipped — progress_updated_at is nil", "id", entry.ID)
		} else {
			slog.Debug("Stall check", "id", entry.ID, "stalled_for", time.Since(*entry.ProgressUpdatedAt).Round(time.Second), "timeout", stallTimeout, "progress", t.Progress)
		}
		if stallTimeout > 0 && entry.ProgressUpdatedAt != nil && time.Since(*entry.ProgressUpdatedAt) > stallTimeout {
			slog.Warn("Torrent stalled — blacklisting and removing", "id", entry.ID, "hash", *entry.TorrentHash, "stalled_for", time.Since(*entry.ProgressUpdatedAt).Round(time.Second))
			s.blacklistTorrent(ctx, entry)
			_ = s.deleteTorrent(ctx, qbtURL, *entry.TorrentHash, true)
			s.db.NewUpdate().
				Model((*models.DownloadQueue)(nil)).
				Set("deleted_at = now(), updated_at = now()").
				Where("id = ?", entry.ID).
				Exec(ctx)
			if entry.MonitorID != nil {
				s.db.NewUpdate().
					Model((*models.Monitor)(nil)).
					Set("status = 'pending', updated_at = now()").
					Where("id = ?", *entry.MonitorID).
					Exec(ctx)
				s.releaseEpisodeMonitors(ctx, *entry.MonitorID)
			}
		}
	}
}

func (s *DownloaderService) LoginAndDelete(ctx context.Context, qbtURL, username, password, hash string, deleteFiles bool) error {
	if err := s.login(ctx, qbtURL, username, password); err != nil {
		return err
	}
	return s.deleteTorrent(ctx, qbtURL, hash, deleteFiles)
}

func (s *DownloaderService) blacklistTorrent(ctx context.Context, entry models.DownloadQueue) {
	bl := models.TorrentBlacklist{
		TorrentHash: entry.TorrentHash,
		TorrentName: entry.TorrentName,
		Title:       entry.Title,
	}
	if _, err := s.db.NewInsert().Model(&bl).Exec(ctx); err != nil {
		slog.Warn("Failed to blacklist stalled torrent", "id", entry.ID, "error", err)
	}
}

func (s *DownloaderService) deleteTorrent(ctx context.Context, qbtURL, hash string, deleteFiles bool) error {
	deleteFilesStr := "false"
	if deleteFiles {
		deleteFilesStr = "true"
	}
	form := url.Values{"hashes": {hash}, "deleteFiles": {deleteFilesStr}}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, qbtURL+"/api/v2/torrents/delete", strings.NewReader(form.Encode()))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	resp, err := s.qbtClient.Do(req)
	if err != nil {
		return err
	}
	resp.Body.Close()
	return nil
}

func (s *DownloaderService) onComplete(ctx context.Context, entry models.DownloadQueue) {
	// Resolve monitor upfront — needed for media_folder lookup and season pack handling.
	var mon *models.Monitor
	if entry.MonitorID != nil {
		mon = s.resolveMonitor(ctx, *entry.MonitorID)
	}
	mediaFolder := s.resolveMediaFolder(ctx, mon)
	episodeHint := 0
	if mon != nil && mon.IsEpisode != nil && *mon.IsEpisode && mon.EpisodeNumber != nil && *mon.EpisodeNumber > 0 {
		episodeHint = int(*mon.EpisodeNumber)
	}

	var walked, hasVideo bool
	if entry.SavePath != nil && *entry.SavePath != "" {
		walked, hasVideo = s.createSymlinks(ctx, *entry.SavePath, entry.Title, mediaFolder, episodeHint)
	}

	// If we successfully walked the directory but found no supported video files,
	// blacklist this torrent and re-queue the monitor so a different release is tried.
	// Guard on mediaPath being set: if it's unset that's a config issue, not a bad torrent.
	mediaPath := strings.TrimRight(config.Get(s.db, "mediaPath", ""), "/")
	if walked && !hasVideo && mediaPath != "" {
		slog.Warn("No supported video files — blacklisting and re-queuing", "id", entry.ID, "save_path", *entry.SavePath)
		s.blacklistTorrent(ctx, entry)
		qbtURL, username, password := s.qbtConfig()
		if err := s.login(ctx, qbtURL, username, password); err == nil && entry.TorrentHash != nil {
			_ = s.deleteTorrent(ctx, qbtURL, *entry.TorrentHash, true)
		}
		s.db.NewUpdate().Model((*models.DownloadQueue)(nil)).
			Set("deleted_at = now(), updated_at = now()").
			Where("id = ?", entry.ID).Exec(ctx)
		if entry.MonitorID != nil {
			s.db.NewUpdate().Model((*models.Monitor)(nil)).
				Set("status = 'pending', updated_at = now()").
				Where("id = ?", *entry.MonitorID).Exec(ctx)
			s.releaseEpisodeMonitors(ctx, *entry.MonitorID)
		}
		return
	}

	// Mark as completed (keep row visible in UI)
	s.db.NewUpdate().
		Model((*models.DownloadQueue)(nil)).
		Set("status = 'completed', progress = 1, updated_at = now()").
		Where("id = ?", entry.ID).
		Exec(ctx)

	if mon == nil {
		return
	}

	s.db.NewUpdate().
		Model((*models.Monitor)(nil)).
		Set("status = 'downloaded', updated_at = now()").
		Where("id = ?", *entry.MonitorID).
		Exec(ctx)

	// Season pack completion: also mark all episode monitors as downloaded
	if mon.IsSeason != nil && *mon.IsSeason && mon.LibraryID != nil {
		s.db.NewUpdate().
			Model((*models.Monitor)(nil)).
			Set("status = 'downloaded', updated_at = now()").
			Where("library_id = ? AND is_episode = true AND deleted_at IS NULL", *mon.LibraryID).
			Exec(ctx)
		slog.Info("Season pack complete — all episodes marked downloaded, awaiting availability check", "monitor_id", *entry.MonitorID, "library_id", *mon.LibraryID)
	} else {
		slog.Info("Episode complete — marked downloaded, awaiting availability check", "monitor_id", *entry.MonitorID)
	}
}

func (s *DownloaderService) createSymlinks(_ context.Context, savePath string, entryTitle *string, mediaFolder string, episodeHint int) (walked, hasVideo bool) {
	title := ""
	if entryTitle != nil {
		title = *entryTitle
	}
	return s.CreateSymlinks(savePath, title, mediaFolder, episodeHint)
}

// CreateSymlinksForEntry is used by the manual /symlinks/create endpoint.
// It looks up the download queue entry by ID, queries qBittorrent for the
// current content_path (falling back to the DB save_path if the torrent is
// no longer tracked), and calls CreateSymlinks with the resolved path.
func (s *DownloaderService) CreateSymlinksForEntry(ctx context.Context, id int64) (walked, hasVideo bool) {
	var entry models.DownloadQueue
	if err := s.db.NewSelect().Model(&entry).Where("id = ?", id).Scan(ctx); err != nil {
		slog.Error("CreateSymlinksForEntry: entry not found", "id", id, "error", err)
		return
	}

	title := ""
	if entry.Title != nil {
		title = *entry.Title
	}

	savePath := ""
	if entry.SavePath != nil {
		savePath = *entry.SavePath
	}

	// Prefer the live content_path from qBittorrent over the potentially-stale DB value.
	if entry.TorrentHash != nil && *entry.TorrentHash != "" {
		qbtURL, username, password := s.qbtConfig()
		if err := s.login(ctx, qbtURL, username, password); err == nil {
			if items, err := s.fetchTorrents(ctx, qbtURL, *entry.TorrentHash, ""); err == nil && len(items) > 0 {
				if rel := items[0].contentRelPath(); rel != "" {
					downloadPath := strings.TrimRight(config.Get(s.db, "downloadPath", ""), "/")
					if downloadPath != "" {
						savePath = filepath.Join(downloadPath, rel)
						slog.Info("CreateSymlinksForEntry: resolved live path from qBittorrent", "id", id, "save_path", savePath)
					}
				}
			}
		}
	}

	var mediaFolder string
	episodeHint := 0
	if entry.MonitorID != nil {
		mon := s.resolveMonitor(ctx, *entry.MonitorID)
		mediaFolder = s.resolveMediaFolder(ctx, mon)
		if mon != nil && mon.IsEpisode != nil && *mon.IsEpisode && mon.EpisodeNumber != nil && *mon.EpisodeNumber > 0 {
			episodeHint = int(*mon.EpisodeNumber)
		}
	}
	return s.CreateSymlinks(savePath, title, mediaFolder, episodeHint)
}

// CreateSymlinks is the public entry point used by both onComplete and the manual /symlinks/create endpoint.
// It returns (walked, hasVideo): walked=true if the save path was found and traversed,
// hasVideo=true if at least one supported video file was encountered.
func (s *DownloaderService) CreateSymlinks(savePath, entryTitle, mediaFolder string, episodeHint int) (walked, hasVideo bool) {
	mediaPath := strings.TrimRight(config.Get(s.db, "mediaPath", ""), "/")
	rawExts := config.Get(s.db, "allowedVideoExtensions", ".mkv,.mp4,.avi,.mov,.wmv,.m4v")

	// Always rebase against the current downloadPath setting so that stale DB entries
	// (e.g. Windows paths from before a settings change) resolve correctly.
	downloadPath := strings.TrimRight(config.Get(s.db, "downloadPath", ""), "/")
	if downloadPath != "" && savePath != "" {
		dp := filepath.ToSlash(strings.TrimRight(downloadPath, "/")) + "/"
		sp := filepath.ToSlash(savePath)
		if strings.HasPrefix(sp, dp) {
			// Path is already under downloadPath — preserve the full sub-path so
			// torrents with a root folder (TorrentFolder/file.mkv) resolve correctly.
			savePath = filepath.Join(downloadPath, sp[len(dp):])
		} else {
			// Different root (e.g. stale Windows path or old entry) — remap using
			// just the base name as before.
			savePath = filepath.Join(downloadPath, pathBase(savePath))
		}
	}

	slog.Info("createHardlinks: start", "save_path", savePath, "media_path", mediaPath, "entry_title", entryTitle)

	if mediaPath == "" {
		slog.Warn("createHardlinks: mediaPath is not configured — skipping")
		return
	}
	if savePath == "" {
		slog.Warn("createHardlinks: savePath is empty — skipping")
		return
	}

	allowedExts := map[string]bool{}
	for _, e := range strings.Split(rawExts, ",") {
		if t := strings.TrimSpace(strings.ToLower(e)); t != "" {
			allowedExts[t] = true
		}
	}
	slog.Info("createHardlinks: allowed extensions", "exts", rawExts)

	walkRoot := savePath
	if _, err := os.Stat(savePath); err != nil {
		// savePath doesn't exist — single-file torrent where t.Name has no extension.
		// Look in the parent directory for a file whose base name (without ext) matches.
		parent := filepath.Dir(savePath)
		base := filepath.Base(savePath)
		entries, readErr := os.ReadDir(parent)
		if readErr != nil {
			slog.Warn("createHardlinks: save_path does not exist and parent is not readable", "path", savePath, "error", err)
			return
		}
		var match string
		for _, e := range entries {
			if !e.IsDir() && strings.TrimSuffix(e.Name(), filepath.Ext(e.Name())) == base {
				match = filepath.Join(parent, e.Name())
				break
			}
		}
		if match == "" {
			slog.Warn("createHardlinks: save_path does not exist or is not accessible", "path", savePath, "error", err)
			return
		}
		slog.Info("createHardlinks: resolved single-file torrent", "path", match)
		walkRoot = match
	}

	walked = true
	err := filepath.Walk(walkRoot, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			slog.Warn("createHardlinks: walk error on entry", "path", path, "error", err)
			return nil
		}
		if info.IsDir() {
			slog.Info("createHardlinks: entering directory", "path", path)
			return nil
		}

		ext := strings.ToLower(filepath.Ext(path))
		if !allowedExts[ext] {
			slog.Info("createHardlinks: skipping (extension not allowed)", "file", info.Name(), "ext", ext)
			return nil
		}
		hasVideo = true

		g := runGuessit(info.Name())
		slog.Info("createHardlinks: guessit result", "file", info.Name(), "title", g.Title, "season", g.Season, "episode", g.Episode, "type", g.Type)

		resolvedTitle := mediaFolder
		if resolvedTitle == "" {
			resolvedTitle = entryTitle
			if resolvedTitle == "" {
				resolvedTitle = g.Title
			}
			resolvedTitle = sanitizeFilename(resolvedTitle)
		}
		if resolvedTitle == "" {
			slog.Warn("createHardlinks: could not determine title, skipping", "file", path)
			return nil
		}

		episode := g.Episode
		if episode == 0 {
			episode = episodeHint
		}
		var linkName string
		if episode > 0 {
			season := g.Season
			if season == 0 {
				season = 1
			}
			if g.ReleaseGroup != "" {
				linkName = fmt.Sprintf("%s - S%02dE%02d [%s]%s", resolvedTitle, season, episode, g.ReleaseGroup, ext)
			} else {
				linkName = fmt.Sprintf("%s - S%02dE%02d%s", resolvedTitle, season, episode, ext)
			}
		} else {
			linkName = sanitizeFilename(info.Name())
		}
		linkPath := filepath.Join(mediaPath, resolvedTitle, linkName)
		slog.Info("createHardlinks: planned symlink", "src", path, "dst", linkPath)

		destDir := filepath.Dir(linkPath)
		if err := os.MkdirAll(destDir, 0755); err != nil {
			slog.Warn("createHardlinks: failed to create directory", "path", destDir, "error", err)
			return nil
		}

		if dstInfo, err := os.Stat(linkPath); err == nil {
			srcInfo, _ := os.Stat(path)
			if srcInfo != nil && os.SameFile(srcInfo, dstInfo) {
				slog.Info("createHardlinks: already linked, skipping", "dst", linkPath)
			} else {
				slog.Info("createHardlinks: destination exists with different content, skipping", "dst", linkPath)
			}
			return nil
		}

		if err := os.Link(path, linkPath); err != nil {
			slog.Warn("createHardlinks: failed", "src", path, "dst", linkPath, "error", err)
		} else {
			slog.Info("createHardlinks: created", "src", path, "dst", linkPath)
		}
		return nil
	})
	if err != nil {
		slog.Warn("createHardlinks: walk failed", "path", savePath, "error", err)
	}
	slog.Info("createHardlinks: done", "save_path", savePath)
	return
}

func (s *DownloaderService) resolveMonitor(ctx context.Context, monitorID int64) *models.Monitor {
	var mon models.Monitor
	if err := s.db.NewSelect().Model(&mon).Where("id = ?", monitorID).Scan(ctx); err != nil {
		return nil
	}
	return &mon
}

// resolveMediaFolder returns the pre-sanitized media_folder for the library entry
// associated with the given monitor. Falls back to empty string on any error.
func (s *DownloaderService) resolveMediaFolder(ctx context.Context, mon *models.Monitor) string {
	if mon == nil || mon.LibraryID == nil {
		return ""
	}
	var folder string
	if err := s.db.NewSelect().
		TableExpr("media").
		ColumnExpr("media_folder").
		Where("id = ?", *mon.LibraryID).
		Scan(ctx, &folder); err != nil {
		slog.Warn("resolveMediaFolder: failed", "library_id", *mon.LibraryID, "error", err)
	}
	return folder
}

// releaseEpisodeMonitors resets any episode monitors that were bulk-queued under
// a season pack back to pending, so the indexer can search for them individually.
// It is a no-op if monitorID does not belong to a season monitor.
func (s *DownloaderService) releaseEpisodeMonitors(ctx context.Context, monitorID int64) {
	res, err := s.db.NewUpdate().
		Model((*models.Monitor)(nil)).
		Set("status = 'pending', updated_at = now()").
		Where(`is_episode = true AND status = 'queued' AND deleted_at IS NULL
			AND library_id = (SELECT library_id FROM monitors WHERE id = ? AND is_season = true AND deleted_at IS NULL)`, monitorID).
		Exec(ctx)
	if err != nil {
		slog.Warn("releaseEpisodeMonitors: update failed", "monitor_id", monitorID, "error", err)
		return
	}
	if n, _ := res.RowsAffected(); n > 0 {
		slog.Info("releaseEpisodeMonitors: episode monitors released", "monitor_id", monitorID, "count", n)
	}
}

func (s *DownloaderService) addTorrent(ctx context.Context, qbtURL, torrentURL string) (string, error) {
	if qbtURL == "" {
		return "", fmt.Errorf("qBittorrent URL is not configured")
	}

	var (
		buf    strings.Builder
		writer = multipart.NewWriter(&buf)
	)

	effectiveURL := torrentURL
	if !strings.HasPrefix(torrentURL, "magnet:") {
		// Resolve the URL — it may be a .torrent file or redirect to a magnet link
		magnetURL, torrentBytes, err := s.resolveURL(ctx, torrentURL)
		if err != nil {
			return "", fmt.Errorf("failed to resolve torrent URL: %w", err)
		}
		if magnetURL != "" {
			// Prowlarr redirected to a magnet link
			effectiveURL = magnetURL
		} else {
			// Got actual .torrent bytes — upload as binary
			part, err := writer.CreateFormFile("torrents", "file.torrent")
			if err != nil {
				return "", err
			}
			if _, err := part.Write(torrentBytes); err != nil {
				return "", err
			}
			effectiveURL = ""
		}
	}

	if effectiveURL != "" {
		if err := writer.WriteField("urls", effectiveURL); err != nil {
			return "", err
		}
	}

	if err := writer.WriteField("tags", "kbarr"); err != nil {
		return "", err
	}
	if err := writer.Close(); err != nil {
		return "", err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, qbtURL+"/api/v2/torrents/add", strings.NewReader(buf.String()))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())

	resp, err := s.qbtClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("qBittorrent returned %s", resp.Status)
	}

	respBody, _ := io.ReadAll(resp.Body)
	bodyStr := strings.TrimSpace(string(respBody))
	slog.Info("qBittorrent add response", "status", resp.StatusCode, "body", bodyStr, "url_prefix", torrentURL[:min(len(torrentURL), 80)])
	if bodyStr == "Fails." {
		return "", fmt.Errorf("qBittorrent rejected the torrent (Fails.)")
	}

	// qBittorrent v5+ returns JSON with added_torrent_ids
	var jsonResp struct {
		AddedIDs []string `json:"added_torrent_ids"`
	}
	if err := json.Unmarshal([]byte(bodyStr), &jsonResp); err == nil && len(jsonResp.AddedIDs) > 0 {
		return jsonResp.AddedIDs[0], nil
	}

	// Fallback: extract hash from magnet link
	return extractMagnetHash(torrentURL), nil
}

// resolveURL fetches an HTTP URL. If the server redirects to a magnet link,
// it returns (magnetURL, nil, nil). Otherwise it returns ("", bytes, nil).
func (s *DownloaderService) resolveURL(ctx context.Context, torrentURL string) (magnetURL string, torrentBytes []byte, err error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, torrentURL, nil)
	if err != nil {
		return "", nil, err
	}
	resp, err := s.httpClient.Do(req)
	if err != nil {
		return "", nil, err
	}
	defer resp.Body.Close()

	// 3xx redirect to a magnet link — CheckRedirect stopped following it
	if resp.StatusCode/100 == 3 {
		loc := resp.Header.Get("Location")
		if strings.HasPrefix(loc, "magnet:") {
			return loc, nil, nil
		}
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", nil, fmt.Errorf("HTTP %s", resp.Status)
	}

	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", nil, err
	}

	// Prowlarr returns XML error responses even with 2xx status codes
	// e.g. <error code="429" description="Indexer is disabled..."/>
	if trimmed := bytes.TrimSpace(b); bytes.HasPrefix(trimmed, []byte("<?xml")) || bytes.HasPrefix(trimmed, []byte("<error")) {
		return "", nil, fmt.Errorf("indexer error response: %s", strings.TrimSpace(string(b)))
	}

	return "", b, nil
}

func (s *DownloaderService) fetchTorrentFile(ctx context.Context, torrentURL string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, torrentURL, nil)
	if err != nil {
		return nil, err
	}
	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("HTTP %s", resp.Status)
	}
	return io.ReadAll(resp.Body)
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func (s *DownloaderService) fetchTorrents(ctx context.Context, qbtURL, hash, category string) ([]torrentInfo, error) {
	if qbtURL == "" {
		return nil, fmt.Errorf("qBittorrent URL is not configured")
	}

	query := url.Values{}
	if category != "" {
		query.Set("category", category)
	}
	if hash != "" {
		query.Set("hashes", hash)
	}

	endpoint := qbtURL + "/api/v2/torrents/info"
	if encoded := query.Encode(); encoded != "" {
		endpoint += "?" + encoded
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, err
	}

	resp, err := s.qbtClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("qBittorrent returned %s", resp.Status)
	}

	var items []torrentInfo
	if err := json.NewDecoder(resp.Body).Decode(&items); err != nil {
		return nil, err
	}

	return items, nil
}

func isMagnet(u string) bool {
	return strings.HasPrefix(u, "magnet:") || strings.Contains(u, "urn:btih:")
}

var invalidFilenameChars = regexp.MustCompile(`[/\\:*?"<>|]`)

func sanitizeFilename(name string) string {
	return strings.TrimSpace(invalidFilenameChars.ReplaceAllString(name, "_"))
}

func extractMagnetHash(magnetURL string) string {
	parsed, err := url.Parse(magnetURL)
	if err != nil {
		return ""
	}
	xt := parsed.Query().Get("xt")
	if strings.HasPrefix(xt, "urn:btih:") {
		return strings.TrimPrefix(xt, "urn:btih:")
	}
	return ""
}
