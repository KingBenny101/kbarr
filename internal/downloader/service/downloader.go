package service

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/kingbenny101/kbarr/internal/config"
	dlprovider "github.com/kingbenny101/kbarr/internal/downloader/provider"
	"github.com/kingbenny101/kbarr/internal/models"
	"github.com/kingbenny101/kbarr/internal/naming"
	"github.com/uptrace/bun"
)

type DownloaderService struct {
	db     *bun.DB
	client dlprovider.TorrentClient
}

func NewDownloaderService(db *bun.DB) *DownloaderService {
	return &DownloaderService{
		db:     db,
		client: dlprovider.Get(db),
	}
}

// pathBase extracts the final path component handling both '/' (POSIX) and '\'
// (Windows) separators. filepath.Base on Linux does not split on '\'.
func pathBase(p string) string {
	if i := strings.LastIndexAny(p, "/\\"); i >= 0 {
		return p[i+1:]
	}
	return p
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
	if s.client == nil {
		slog.Warn("No downloader client configured")
		return
	}

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

	if err := s.client.Login(ctx); err != nil {
		slog.Error("qBittorrent login failed", "error", err)
		return
	}

	for _, entry := range entries {
		if entry.TorrentURL == nil || *entry.TorrentURL == "" {
			continue
		}

		hash, err := s.client.AddTorrent(ctx, *entry.TorrentURL)
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
		if items, err := s.client.FetchTorrents(ctx, hash, ""); err == nil && len(items) > 0 {
			t := items[0]
			if hash == "" {
				hash = t.Hash
			}
			if downloadPath != "" {
				savePath = filepath.Join(downloadPath, t.ContentName())
			}
		} else if hash == "" && entry.TorrentName != nil && *entry.TorrentName != "" {
			// Fallback: search by name if hash still unknown
			if items, err := s.client.FetchTorrents(ctx, "", ""); err == nil {
				for _, t := range items {
					if t.Name == *entry.TorrentName {
						hash = t.Hash
						if downloadPath != "" {
							savePath = filepath.Join(downloadPath, t.ContentName())
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
	if s.client == nil {
		slog.Warn("No downloader client configured")
		return
	}

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

	if err := s.client.Login(ctx); err != nil {
		slog.Error("qBittorrent login failed", "error", err)
		return
	}

	for _, entry := range entries {
		if entry.TorrentHash == nil {
			continue
		}

		items, err := s.client.FetchTorrents(ctx, *entry.TorrentHash, "")
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
			// ContentRelPath preserves subdirectories (e.g. TorrentFolder/file.mkv) so
			// we land at the right location when the torrent has a root folder.
			if rel := t.ContentRelPath(); rel != "" {
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
			_ = s.client.DeleteTorrent(ctx, *entry.TorrentHash, true)
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

func (s *DownloaderService) LoginAndDelete(ctx context.Context, hash string, deleteFiles bool) error {
	if s.client == nil {
		return fmt.Errorf("no downloader client configured")
	}
	if err := s.client.Login(ctx); err != nil {
		return err
	}
	return s.client.DeleteTorrent(ctx, hash, deleteFiles)
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

func (s *DownloaderService) onComplete(ctx context.Context, entry models.DownloadQueue) {
	var mon *models.Monitor
	if entry.MonitorID != nil {
		mon = s.resolveMonitor(ctx, *entry.MonitorID)
	}
	mediaFolder := s.resolveMediaFolder(ctx, mon)
	episodeHint := 0
	if mon != nil && mon.IsEpisode && mon.EpisodeNumber > 0 {
		episodeHint = mon.EpisodeNumber
	}

	var walked, hasVideo bool
	if entry.SavePath != nil && *entry.SavePath != "" {
		walked, hasVideo = s.createHardlinks(ctx, *entry.SavePath, entry.Title, mediaFolder, episodeHint)
	}
	if walked && hasVideo && mon != nil {
		s.writeTVShowNFO(ctx, mon.LibraryID, mediaFolder)
	}

	// If we successfully walked the directory but found no supported video files,
	// blacklist this torrent and re-queue the monitor so a different release is tried.
	mediaPath := strings.TrimRight(config.Get(s.db, "mediaPath", ""), "/")
	if walked && !hasVideo && mediaPath != "" {
		slog.Warn("No supported video files — blacklisting and re-queuing", "id", entry.ID, "save_path", *entry.SavePath)
		s.blacklistTorrent(ctx, entry)
		if s.client != nil {
			if err := s.client.Login(ctx); err == nil && entry.TorrentHash != nil {
				_ = s.client.DeleteTorrent(ctx, *entry.TorrentHash, true)
			}
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

	if mon.IsSeason && mon.LibraryID != 0 {
		s.db.NewUpdate().
			Model((*models.Monitor)(nil)).
			Set("status = 'downloaded', updated_at = now()").
			Where("library_id = ? AND is_episode = true AND deleted_at IS NULL", mon.LibraryID).
			Exec(ctx)
		slog.Info("Season pack complete — all episodes marked downloaded, awaiting availability check", "monitor_id", *entry.MonitorID, "library_id", mon.LibraryID)
	} else {
		slog.Info("Episode complete — marked downloaded, awaiting availability check", "monitor_id", *entry.MonitorID)
	}
}

func (s *DownloaderService) createHardlinks(_ context.Context, savePath string, entryTitle *string, mediaFolder string, episodeHint int) (walked, hasVideo bool) {
	title := ""
	if entryTitle != nil {
		title = *entryTitle
	}
	return s.CreateHardlinks(savePath, title, mediaFolder, episodeHint)
}

// CreateHardlinksForEntry is used by the manual /hardlinks/create endpoint.
func (s *DownloaderService) CreateHardlinksForEntry(ctx context.Context, id int64) (walked, hasVideo bool) {
	var entry models.DownloadQueue
	if err := s.db.NewSelect().Model(&entry).Where("id = ?", id).Scan(ctx); err != nil {
		slog.Error("CreateHardlinksForEntry: entry not found", "id", id, "error", err)
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
	if s.client != nil && entry.TorrentHash != nil && *entry.TorrentHash != "" {
		if err := s.client.Login(ctx); err == nil {
			if items, err := s.client.FetchTorrents(ctx, *entry.TorrentHash, ""); err == nil && len(items) > 0 {
				if rel := items[0].ContentRelPath(); rel != "" {
					downloadPath := strings.TrimRight(config.Get(s.db, "downloadPath", ""), "/")
					if downloadPath != "" {
						savePath = filepath.Join(downloadPath, rel)
						slog.Info("CreateHardlinksForEntry: resolved live path from qBittorrent", "id", id, "save_path", savePath)
					}
				}
			}
		}
	}

	var mediaFolder string
	var libraryID uint
	episodeHint := 0
	if entry.MonitorID != nil {
		mon := s.resolveMonitor(ctx, *entry.MonitorID)
		mediaFolder = s.resolveMediaFolder(ctx, mon)
		if mon != nil {
			libraryID = mon.LibraryID
			if mon.IsEpisode && mon.EpisodeNumber > 0 {
				episodeHint = mon.EpisodeNumber
			}
		}
	}
	walked, hasVideo = s.CreateHardlinks(savePath, title, mediaFolder, episodeHint)
	if walked && hasVideo {
		s.writeTVShowNFO(ctx, libraryID, mediaFolder)
	}
	return
}

// CreateHardlinks is the public entry point used by both onComplete and the manual /hardlinks/create endpoint.
func (s *DownloaderService) CreateHardlinks(savePath, entryTitle, mediaFolder string, episodeHint int) (walked, hasVideo bool) {
	mediaPath := strings.TrimRight(config.Get(s.db, "mediaPath", ""), "/")
	rawExts := config.Get(s.db, "allowedVideoExtensions", ".mkv,.mp4,.avi,.mov,.wmv,.m4v")

	// Always rebase against the current downloadPath setting so that stale DB entries
	// (e.g. Windows paths from before a settings change) resolve correctly.
	downloadPath := strings.TrimRight(config.Get(s.db, "downloadPath", ""), "/")
	if downloadPath != "" && savePath != "" {
		dp := filepath.ToSlash(strings.TrimRight(downloadPath, "/")) + "/"
		sp := filepath.ToSlash(savePath)
		if strings.HasPrefix(sp, dp) {
			savePath = filepath.Join(downloadPath, sp[len(dp):])
		} else {
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
			resolvedTitle = naming.SanitizeFilename(resolvedTitle)
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
			linkName = naming.SanitizeFilename(info.Name())
		}
		linkPath := filepath.Join(mediaPath, resolvedTitle, linkName)
		slog.Info("createHardlinks: planned hardlink", "src", path, "dst", linkPath)

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

// writeTVShowNFO writes a Jellyfin-compatible tvshow.nfo into the show's media
// folder so Jellyfin can match the series to the correct AniDB entry without
// relying on title fuzzy matching.
func (s *DownloaderService) writeTVShowNFO(ctx context.Context, libraryID uint, mediaFolder string) {
	if libraryID == 0 || mediaFolder == "" {
		return
	}
	mediaPath := strings.TrimRight(config.Get(s.db, "mediaPath", ""), "/")
	if mediaPath == "" {
		return
	}

	var d models.Detailed
	if err := s.db.NewSelect().Model(&d).
		Where("library_id = ?", libraryID).
		Scan(ctx); err != nil {
		slog.Warn("writeTVShowNFO: failed to load detailed", "library_id", libraryID, "error", err)
		return
	}

	nfoPath := filepath.Join(mediaPath, mediaFolder, "tvshow.nfo")
	if _, err := os.Stat(nfoPath); err == nil {
		return // already exists
	}

	content := fmt.Sprintf(`<?xml version="1.0" encoding="utf-8"?>
<tvshow>
  <title>%s</title>
  <uniqueid type="anidb" default="true">%s</uniqueid>
</tvshow>
`, d.Title, d.SourceID)

	if err := os.WriteFile(nfoPath, []byte(content), 0644); err != nil {
		slog.Warn("writeTVShowNFO: failed to write", "path", nfoPath, "error", err)
		return
	}
	slog.Info("writeTVShowNFO: written", "path", nfoPath)
}

func (s *DownloaderService) resolveMediaFolder(ctx context.Context, mon *models.Monitor) string {
	if mon == nil || mon.LibraryID == 0 {
		return ""
	}
	var folder string
	if err := s.db.NewSelect().
		TableExpr("media").
		ColumnExpr("media_folder").
		Where("id = ?", mon.LibraryID).
		Scan(ctx, &folder); err != nil {
		slog.Warn("resolveMediaFolder: failed", "library_id", mon.LibraryID, "error", err)
	}
	return folder
}

// releaseEpisodeMonitors resets any episode monitors that were bulk-queued under
// a season pack back to pending, so the indexer can search for them individually.
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

