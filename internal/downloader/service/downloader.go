package service

import (
	"bytes"
	"context"
	"encoding/json"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"syscall"
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

// linkOrCopy hardlinks src to dst, falling back to a copy only when the two
// paths live on different filesystems (EXDEV). Hardlinks are impossible across
// devices — the common Docker case where downloads and media are separate
// mounts — so a copy is the only way to honour the user's layout. Any other
// error (permissions, dest exists, disk full) is returned unchanged rather than
// silently masked by a copy. Returns copied=true when the fallback ran.
func linkOrCopy(src, dst string) (copied bool, err error) {
	if err := os.Link(src, dst); err == nil {
		return false, nil
	} else if !errors.Is(err, syscall.EXDEV) {
		return false, err
	}
	return true, copyFile(src, dst)
}

// copyFile copies src to dst by writing a temp file alongside dst and renaming
// it into place, so a crash mid-copy cannot leave a half-written file that looks
// like a valid media file to Jellyfin.
func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	tmp := dst + ".kbarr.tmp"
	out, err := os.Create(tmp)
	if err != nil {
		return err
	}
	if _, err := io.Copy(out, in); err != nil {
		out.Close()
		os.Remove(tmp)
		return err
	}
	if err := out.Close(); err != nil {
		os.Remove(tmp)
		return err
	}
	if err := os.Rename(tmp, dst); err != nil {
		os.Remove(tmp)
		return err
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
			// The add may have failed because qBittorrent already has this torrent
			// (duplicate infohash). If so, adopt the existing torrent rather than
			// blacklisting a release that is already present.
			if existing := s.findExistingTorrent(ctx, entry); existing != nil {
				s.adoptExistingTorrent(ctx, entry, *existing)
				continue
			}
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

// findExistingTorrent returns the torrent already present in the download client
// that corresponds to this queue entry, or nil. It is used to detect duplicate
// adds (same infohash already loaded) so the release can be adopted rather than
// blacklisted. Matching is by torrent name, and by infohash when the entry is a
// magnet link.
func (s *DownloaderService) findExistingTorrent(ctx context.Context, entry models.DownloadQueue) *dlprovider.TorrentInfo {
	items, err := s.client.FetchTorrents(ctx, "", "")
	if err != nil {
		return nil
	}
	var wantHash string
	if entry.TorrentURL != nil && strings.HasPrefix(*entry.TorrentURL, "magnet:") {
		wantHash = strings.ToLower(magnetHash(*entry.TorrentURL))
	}
	wantName := ""
	if entry.TorrentName != nil {
		wantName = *entry.TorrentName
	}
	for i := range items {
		t := items[i]
		if wantHash != "" && strings.ToLower(t.Hash) == wantHash {
			return &t
		}
		if wantName != "" && t.Name == wantName {
			return &t
		}
	}
	return nil
}

// adoptExistingTorrent wires a torrent already present in the download client
// into this queue entry instead of blacklisting it. If the client has the
// torrent saved outside its default directory, it is moved back to the default
// (which the operator maps to downloadPath) so kbarr's path assumptions hold.
func (s *DownloaderService) adoptExistingTorrent(ctx context.Context, entry models.DownloadQueue, t dlprovider.TorrentInfo) {
	if def, err := s.client.DefaultSavePath(ctx); err == nil && def != "" && normPath(t.SavePath) != normPath(def) {
		if err := s.client.SetLocation(ctx, t.Hash, def); err != nil {
			slog.Warn("adopt: failed to move torrent to default save path", "id", entry.ID, "hash", t.Hash, "error", err)
		} else {
			slog.Info("adopt: moving torrent to default save path", "id", entry.ID, "hash", t.Hash, "location", def)
		}
	}

	downloadPath := strings.TrimRight(config.Get(s.db, "downloadPath", ""), "/")
	var savePath string
	if downloadPath != "" {
		savePath = filepath.Join(downloadPath, t.ContentName())
	}

	if _, err := s.db.NewUpdate().
		Model((*models.DownloadQueue)(nil)).
		Set("status = 'downloading', torrent_hash = ?, save_path = ?, progress_updated_at = now(), updated_at = now()", t.Hash, savePath).
		Where("id = ?", entry.ID).
		Exec(ctx); err != nil {
		slog.Error("adopt: failed to update download queue", "id", entry.ID, "error", err)
		return
	}

	if entry.MonitorID != nil {
		s.db.NewUpdate().
			Model((*models.Monitor)(nil)).
			Set("status = 'downloading', updated_at = now()").
			Where("id = ?", *entry.MonitorID).
			Exec(ctx)
	}

	slog.Info("adopted existing torrent", "id", entry.ID, "hash", t.Hash, "save_path", savePath)
}

// normPath normalises a path for comparison: forward slashes, no trailing slash.
func normPath(p string) string {
	return strings.TrimRight(strings.ReplaceAll(p, "\\", "/"), "/")
}

// magnetHash extracts the btih infohash from a magnet link, or "" if absent.
func magnetHash(magnetURL string) string {
	const marker = "xt=urn:btih:"
	i := strings.Index(magnetURL, marker)
	if i < 0 {
		return ""
	}
	rest := magnetURL[i+len(marker):]
	if amp := strings.IndexByte(rest, '&'); amp >= 0 {
		rest = rest[:amp]
	}
	return rest
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

		// Treat the torrent as making forward progress if either the progress value
		// moved or qBittorrent reports an active download speed. Very slow torrents
		// can download for longer than a poll interval without the progress float
		// changing between samples; keying the stall timer off dlspeed too prevents
		// them from being wrongly classified as stalled.
		if t.Progress != prevProgress || t.DLSpeed > 0 {
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
			slog.Debug("Stall check", "id", entry.ID, "stalled_for", time.Since(*entry.ProgressUpdatedAt).Round(time.Second), "timeout", stallTimeout, "progress", t.Progress, "dlspeed", t.DLSpeed)
		}
		if stallTimeout > 0 && t.DLSpeed == 0 && entry.ProgressUpdatedAt != nil && time.Since(*entry.ProgressUpdatedAt) > stallTimeout {
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

	var walked, hasVideo, placed bool
	if entry.SavePath != nil && *entry.SavePath != "" {
		walked, hasVideo, placed = s.createHardlinks(ctx, *entry.SavePath, entry.Title, mediaFolder, episodeHint)
	}

	// Video files were found but none could be placed into the media library
	// (e.g. media path unwritable, disk full). This is a local/filesystem problem,
	// not a bad release — do NOT blacklist or mark completed. Leave the entry as
	// 'downloading' so the next poll retries onComplete once the issue is resolved.
	if walked && hasVideo && !placed {
		slog.Error("onComplete: found video files but placed none into media library — check media path permissions and free space; will retry next poll", "id", entry.ID, "save_path", *entry.SavePath)
		return
	}

	if walked && hasVideo && placed && mon != nil {
		s.writeTVShowNFO(ctx, mon.LibraryID, mediaFolder)
		s.triggerJellyfinScan(ctx, mon.LibraryID)
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

func (s *DownloaderService) createHardlinks(_ context.Context, savePath string, entryTitle *string, mediaFolder string, episodeHint int) (walked, hasVideo, placed bool) {
	title := ""
	if entryTitle != nil {
		title = *entryTitle
	}
	return s.CreateHardlinks(savePath, title, mediaFolder, episodeHint)
}

// CreateHardlinksForEntry is used by the manual /hardlinks/create endpoint.
func (s *DownloaderService) CreateHardlinksForEntry(ctx context.Context, id int64) (walked, hasVideo, placed bool) {
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
	walked, hasVideo, placed = s.CreateHardlinks(savePath, title, mediaFolder, episodeHint)
	if walked && hasVideo && placed {
		s.writeTVShowNFO(ctx, libraryID, mediaFolder)
		s.triggerJellyfinScan(ctx, libraryID)
	}
	return
}

// CreateHardlinks is the public entry point used by both onComplete and the manual /hardlinks/create endpoint.
func (s *DownloaderService) CreateHardlinks(savePath, entryTitle, mediaFolder string, episodeHint int) (walked, hasVideo, placed bool) {
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

	// Parse the torrent/folder name once for series-level context. Scene packs
	// often carry the series title and season only on the folder (e.g.
	// "Grand Blue Dreaming S01 1080p BDRip ...") while the files inside are
	// title-less ("S01E01-Deep Blue [crc].mkv"). This fills gaps the per-file
	// parse leaves behind.
	folderCtx := runGuessit(pathBase(savePath))
	slog.Info("createHardlinks: folder context", "folder", pathBase(savePath), "title", folderCtx.Title, "season", folderCtx.Season)

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

		// Title precedence: explicit media folder > torrent entry title >
		// folder-name parse > per-file parse. Each later source is a weaker
		// guess used only when the stronger ones are empty.
		resolvedTitle := mediaFolder
		if resolvedTitle == "" {
			resolvedTitle = entryTitle
			if resolvedTitle == "" {
				resolvedTitle = folderCtx.Title
			}
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
		releaseGroup := g.ReleaseGroup
		if releaseGroup == "" {
			releaseGroup = folderCtx.ReleaseGroup
		}
		var linkName string
		if episode > 0 {
			// Season precedence: per-file > folder-name > default to 1.
			season := g.Season
			if season == 0 {
				season = folderCtx.Season
			}
			if season == 0 {
				season = 1
			}
			if releaseGroup != "" {
				linkName = fmt.Sprintf("%s - S%02dE%02d [%s]%s", resolvedTitle, season, episode, releaseGroup, ext)
			} else {
				linkName = fmt.Sprintf("%s - S%02dE%02d%s", resolvedTitle, season, episode, ext)
			}
		} else {
			// No episode number: a movie or single-file OVA. Name it after the
			// resolved series/title and preserve the real extension, rather than
			// scrubbing the raw release string (which strips the dot before the
			// extension and mangles the name).
			if releaseGroup != "" {
				linkName = fmt.Sprintf("%s [%s]%s", resolvedTitle, releaseGroup, ext)
			} else {
				linkName = resolvedTitle + ext
			}
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
				placed = true
				slog.Info("createHardlinks: already linked, skipping", "dst", linkPath)
			} else {
				slog.Info("createHardlinks: destination exists with different content, skipping", "dst", linkPath)
			}
			return nil
		}

		copied, err := linkOrCopy(path, linkPath)
		if err != nil {
			slog.Error("createHardlinks: failed to place file", "src", path, "dst", linkPath, "error", err)
		} else {
			placed = true
			if copied {
				slog.Info("createHardlinks: copied (cross-device, downloads and media are on different filesystems — this uses extra disk space)", "src", path, "dst", linkPath)
			} else {
				slog.Info("createHardlinks: created", "src", path, "dst", linkPath)
			}
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

	type tvshow struct {
		XMLName   xml.Name `xml:"tvshow"`
		Title     string   `xml:"title"`
		AnidbID   string   `xml:"anidbid"`
		TvdbID    string   `xml:"tvdbid,omitempty"`
		AnilistID string   `xml:"anilistid,omitempty"`
		ImdbID    string   `xml:"imdb_id,omitempty"`
		TmdbID    string   `xml:"tmdbid,omitempty"`
	}

	nfo := tvshow{
		Title:     d.Title,
		AnidbID:   d.SourceID,
		TvdbID:    d.TVDBID,
		AnilistID: d.AniListID,
		ImdbID:    d.IMDBID,
		TmdbID:    d.TMDBID,
	}

	var buf bytes.Buffer
	buf.WriteString(`<?xml version="1.0" encoding="utf-8" standalone="yes"?>` + "\n")
	enc := xml.NewEncoder(&buf)
	enc.Indent("", "  ")
	if err := enc.Encode(nfo); err != nil {
		slog.Warn("writeTVShowNFO: failed to encode", "path", nfoPath, "error", err)
		return
	}

	if err := os.WriteFile(nfoPath, buf.Bytes(), 0644); err != nil {
		slog.Warn("writeTVShowNFO: failed to write", "path", nfoPath, "error", err)
		return
	}
	slog.Info("writeTVShowNFO: written", "path", nfoPath)
}

// triggerJellyfinScan kicks off a Jellyfin library scan and then refreshes the
// individual show so the freshly-written tvshow.nfo (with the AniDB id) is used
// to fill in metadata. The clean flow is:
//   - POST /Library/Refresh to pick up the new files
//   - GET /Items?searchTerm=<title> to resolve the series item id
//   - POST /Items/{itemId}/Refresh (Default mode) to fill missing metadata
//
// The library scan is asynchronous, so the per-item refresh runs in a background
// goroutine after a short delay to give Jellyfin time to register the series.
func (s *DownloaderService) triggerJellyfinScan(ctx context.Context, libraryID uint) {
	jellyfinURL := strings.TrimRight(config.Get(s.db, "jellyfinUrl", ""), "/")
	if jellyfinURL == "" {
		return
	}
	apiKey := config.Get(s.db, "jellyfinApiKey", "")

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, jellyfinURL+"/Library/Refresh", nil)
	if err != nil {
		slog.Warn("triggerJellyfinScan: failed to build request", "error", err)
		return
	}
	req.Header.Set("X-Emby-Token", apiKey)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		slog.Warn("triggerJellyfinScan: request failed", "error", err)
		return
	}
	resp.Body.Close()
	slog.Info("triggerJellyfinScan: library refresh triggered", "status", resp.StatusCode)

	// Resolve the show title so the per-item refresh can locate the series.
	var title string
	if libraryID != 0 {
		_ = s.db.NewSelect().TableExpr("media").ColumnExpr("title").
			Where("id = ?", libraryID).Scan(ctx, &title)
	}
	if title == "" {
		return
	}

	go s.refreshJellyfinItem(jellyfinURL, apiKey, title)
}

// refreshJellyfinItem finds the series by title and triggers a Default-mode
// metadata refresh so the tvshow.nfo identifiers are honoured.
func (s *DownloaderService) refreshJellyfinItem(jellyfinURL, apiKey, title string) {
	// Give the asynchronous library scan time to register the new series.
	time.Sleep(15 * time.Second)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	itemID := s.findJellyfinSeries(ctx, jellyfinURL, apiKey, title)
	if itemID == "" {
		slog.Info("refreshJellyfinItem: series not found yet, relying on library scan", "title", title)
		return
	}

	u := fmt.Sprintf("%s/Items/%s/Refresh?metadataRefreshMode=Default&imageRefreshMode=Default&replaceAllMetadata=false", jellyfinURL, itemID)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, u, nil)
	if err != nil {
		slog.Warn("refreshJellyfinItem: failed to build request", "error", err)
		return
	}
	req.Header.Set("X-Emby-Token", apiKey)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		slog.Warn("refreshJellyfinItem: request failed", "title", title, "error", err)
		return
	}
	resp.Body.Close()
	slog.Info("refreshJellyfinItem: item refresh triggered", "title", title, "item_id", itemID, "status", resp.StatusCode)
}

// findJellyfinSeries returns the id of the Series item matching title, or "".
func (s *DownloaderService) findJellyfinSeries(ctx context.Context, jellyfinURL, apiKey, title string) string {
	u := fmt.Sprintf("%s/Items?searchTerm=%s&IncludeItemTypes=Series&Recursive=true&Limit=10",
		jellyfinURL, url.QueryEscape(title))
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return ""
	}
	req.Header.Set("X-Emby-Token", apiKey)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		slog.Warn("findJellyfinSeries: request failed", "title", title, "error", err)
		return ""
	}
	defer resp.Body.Close()

	var body struct {
		Items []struct {
			ID   string `json:"Id"`
			Name string `json:"Name"`
		} `json:"Items"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil || len(body.Items) == 0 {
		return ""
	}
	// Prefer an exact (case-insensitive) name match; otherwise take the first hit.
	for _, it := range body.Items {
		if strings.EqualFold(it.Name, title) {
			return it.ID
		}
	}
	return body.Items[0].ID
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

