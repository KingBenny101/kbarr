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
	Hash     string  `json:"hash"`
	Name     string  `json:"name"`
	State    string  `json:"state"`
	Size     int64   `json:"size"`
	Progress float64 `json:"progress"`
	ETA      int64   `json:"eta"`
	SavePath string  `json:"save_path"`
	Category string  `json:"category"`
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
					Set("status = 'monitored', updated_at = now()").
					Where("id = ?", *entry.MonitorID).Exec(ctx)
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
				savePath = filepath.Join(downloadPath, t.Name)
			}
		} else if hash == "" && entry.TorrentName != nil && *entry.TorrentName != "" {
			// Fallback: search by name if hash still unknown
			if items, err := s.fetchTorrents(ctx, qbtURL, "", ""); err == nil {
				for _, t := range items {
					if t.Name == *entry.TorrentName {
						hash = t.Hash
						if downloadPath != "" {
							savePath = filepath.Join(downloadPath, t.Name)
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
			_ = s.deleteTorrent(ctx, qbtURL, *entry.TorrentHash)
			s.db.NewUpdate().
				Model((*models.DownloadQueue)(nil)).
				Set("deleted_at = now(), updated_at = now()").
				Where("id = ?", entry.ID).
				Exec(ctx)
			if entry.MonitorID != nil {
				s.db.NewUpdate().
					Model((*models.Monitor)(nil)).
					Set("status = 'monitored', updated_at = now()").
					Where("id = ?", *entry.MonitorID).
					Exec(ctx)
			}
		}
	}
}

func (s *DownloaderService) LoginAndDelete(ctx context.Context, qbtURL, username, password, hash string) error {
	if err := s.login(ctx, qbtURL, username, password); err != nil {
		return err
	}
	return s.deleteTorrent(ctx, qbtURL, hash)
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

func (s *DownloaderService) deleteTorrent(ctx context.Context, qbtURL, hash string) error {
	form := url.Values{"hashes": {hash}, "deleteFiles": {"false"}}
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
	// Mark as completed (keep row visible in UI)
	s.db.NewUpdate().
		Model((*models.DownloadQueue)(nil)).
		Set("status = 'completed', progress = 1, updated_at = now()").
		Where("id = ?", entry.ID).
		Exec(ctx)

	// Create symlinks for all video files in the download directory
	if entry.SavePath != nil && *entry.SavePath != "" {
		s.createSymlinks(ctx, *entry.SavePath, entry.Title)
	}

	if entry.MonitorID == nil {
		return
	}

	// Look up monitor to check if it's a season monitor
	mon := s.resolveMonitor(ctx, *entry.MonitorID)
	if mon == nil {
		return
	}

	s.db.NewUpdate().
		Model((*models.Monitor)(nil)).
		Set("status = 'available', updated_at = now()").
		Where("id = ?", *entry.MonitorID).
		Exec(ctx)

	// Season pack completion: also mark all episode monitors as available
	if mon.IsSeason != nil && *mon.IsSeason && mon.LibraryID != nil {
		s.db.NewUpdate().
			Model((*models.Monitor)(nil)).
			Set("status = 'available', updated_at = now()").
			Where("library_id = ? AND is_episode = true AND deleted_at IS NULL", *mon.LibraryID).
			Exec(ctx)
		slog.Info("Season pack complete — all episodes marked available", "monitor_id", *entry.MonitorID, "library_id", *mon.LibraryID)
	} else {
		slog.Info("Episode complete — marked available", "monitor_id", *entry.MonitorID)
	}
}

func (s *DownloaderService) createSymlinks(_ context.Context, savePath string, entryTitle *string) {
	title := ""
	if entryTitle != nil {
		title = *entryTitle
	}
	s.CreateSymlinks(savePath, title)
}

// CreateSymlinks is the public entry point used by both onComplete and the manual /symlinks/create endpoint.
func (s *DownloaderService) CreateSymlinks(savePath, entryTitle string) {
	mediaPath := strings.TrimRight(config.Get(s.db, "mediaPath", ""), "/")
	rawExts := config.Get(s.db, "allowedVideoExtensions", ".mkv,.mp4,.avi,.mov,.wmv,.m4v")

	// Always rebase using the current downloadPath setting so that old DB entries
	// with a stale base path (e.g. /data/torrents/...) still resolve correctly.
	downloadPath := strings.TrimRight(config.Get(s.db, "downloadPath", ""), "/")
	if downloadPath != "" && savePath != "" {
		savePath = filepath.Join(downloadPath, filepath.Base(savePath))
	}

	slog.Info("createSymlinks: start", "save_path", savePath, "media_path", mediaPath, "entry_title", entryTitle)

	if mediaPath == "" {
		slog.Warn("createSymlinks: mediaPath is not configured — skipping")
		return
	}
	if savePath == "" {
		slog.Warn("createSymlinks: savePath is empty — skipping")
		return
	}

	allowedExts := map[string]bool{}
	for _, e := range strings.Split(rawExts, ",") {
		if t := strings.TrimSpace(strings.ToLower(e)); t != "" {
			allowedExts[t] = true
		}
	}
	slog.Info("createSymlinks: allowed extensions", "exts", rawExts)

	if _, err := os.Stat(savePath); err != nil {
		slog.Warn("createSymlinks: save_path does not exist or is not accessible", "path", savePath, "error", err)
		return
	}

	err := filepath.Walk(savePath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			slog.Warn("createSymlinks: walk error on entry", "path", path, "error", err)
			return nil
		}
		if info.IsDir() {
			slog.Info("createSymlinks: entering directory", "path", path)
			return nil
		}

		ext := strings.ToLower(filepath.Ext(path))
		if !allowedExts[ext] {
			slog.Info("createSymlinks: skipping (extension not allowed)", "file", info.Name(), "ext", ext)
			return nil
		}

		g := runGuessit(info.Name())
		slog.Info("createSymlinks: guessit result", "file", info.Name(), "title", g.Title, "season", g.Season, "episode", g.Episode, "type", g.Type)

		resolvedTitle := g.Title
		if resolvedTitle == "" {
			resolvedTitle = entryTitle
		}
		resolvedTitle = sanitizeFilename(resolvedTitle)
		if resolvedTitle == "" {
			slog.Warn("createSymlinks: could not determine title, skipping", "file", path)
			return nil
		}

		var linkName string
		if g.Episode > 0 {
			season := g.Season
			if season == 0 {
				season = 1
			}
			linkName = fmt.Sprintf("%s - S%02dE%02d%s", resolvedTitle, season, g.Episode, ext)
		} else {
			linkName = sanitizeFilename(info.Name())
		}
		linkPath := filepath.Join(mediaPath, resolvedTitle, linkName)
		slog.Info("createSymlinks: planned symlink", "src", path, "dst", linkPath)

		destDir := filepath.Dir(linkPath)
		if err := os.MkdirAll(destDir, 0755); err != nil {
			slog.Warn("createSymlinks: failed to create directory", "path", destDir, "error", err)
			return nil
		}

		// Remove any existing symlink in the target directory that points to the same source,
		// so stale/raw-named symlinks get replaced with the clean name on retry.
		if entries, err := os.ReadDir(destDir); err == nil {
			for _, de := range entries {
				candidate := filepath.Join(destDir, de.Name())
				if target, err := os.Readlink(candidate); err == nil && target == path {
					if candidate == linkPath {
						slog.Info("createSymlinks: symlink already correct, skipping", "dst", linkPath)
						return nil
					}
					slog.Info("createSymlinks: removing stale symlink", "old", candidate, "new", linkPath)
					os.Remove(candidate)
				}
			}
		}

		if _, err := os.Lstat(linkPath); err == nil {
			slog.Info("createSymlinks: symlink already exists, skipping", "dst", linkPath)
			return nil
		}

		if err := os.Symlink(path, linkPath); err != nil {
			slog.Warn("createSymlinks: failed to create symlink", "src", path, "dst", linkPath, "error", err)
		} else {
			slog.Info("createSymlinks: created symlink", "src", path, "dst", linkPath)
		}
		return nil
	})
	if err != nil {
		slog.Warn("createSymlinks: walk failed", "path", savePath, "error", err)
	}
	slog.Info("createSymlinks: done", "save_path", savePath)
}

func (s *DownloaderService) resolveMonitor(ctx context.Context, monitorID int64) *models.Monitor {
	var mon models.Monitor
	if err := s.db.NewSelect().Model(&mon).Where("id = ?", monitorID).Scan(ctx); err != nil {
		return nil
	}
	return &mon
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
