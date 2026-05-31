package service

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"mime/multipart"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/kingbenny101/kbarr/services/downloader/internal/models"
	"github.com/uptrace/bun"
)

type DownloaderService struct {
	db        *bun.DB
	qbtURL    string
	qbtClient *http.Client
}

func NewDownloaderService(db *bun.DB, qbtURL string) *DownloaderService {
	return &DownloaderService{
		db:        db,
		qbtURL:    strings.TrimRight(strings.TrimSpace(qbtURL), "/"),
		qbtClient: &http.Client{Timeout: 30 * time.Second},
	}
}

type torrentInfo struct {
	Hash     string  `json:"hash"`
	Name     string  `json:"name"`
	State    string  `json:"state"`
	Size     int64   `json:"size"`
	Progress float32 `json:"progress"`
	ETA      int64   `json:"eta"`
	SavePath string  `json:"save_path"`
	Category string  `json:"category"`
}

func (s *DownloaderService) PollAndDownload(ctx context.Context) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			s.processPending(ctx)
			s.updateDownloading(ctx)
		case <-ctx.Done():
			return
		}
	}
}

func (s *DownloaderService) processPending(ctx context.Context) {
	status := "pending"
	var entries []models.DownloadQueue
	err := s.db.NewSelect().
		TableExpr("download_queue").
		ColumnExpr("*").
		Where("status = ? AND deleted_at IS NULL", status).
		Scan(ctx, &entries)
	if err != nil {
		slog.Error("Failed to fetch pending download queue entries", "error", err)
		return
	}

	for _, entry := range entries {
		if entry.TorrentURL == nil || *entry.TorrentURL == "" {
			continue
		}

		hash, err := s.addTorrent(ctx, *entry.TorrentURL, "")
		if err != nil {
			slog.Error("Failed to add torrent", "id", entry.ID, "error", err)
			continue
		}

		statusDownloading := "downloading"
		_, err = s.db.NewUpdate().
			TableExpr("download_queue").
			Set("status = ?, torrent_hash = ?, updated_at = now()", statusDownloading, hash).
			Where("id = ?", entry.ID).
			Exec(ctx)
		if err != nil {
			slog.Error("Failed to update download queue status", "id", entry.ID, "error", err)
			continue
		}

		slog.Info("Torrent added", "id", entry.ID, "hash", hash)
	}
}

func (s *DownloaderService) updateDownloading(ctx context.Context) {
	status := "downloading"
	var entries []models.DownloadQueue
	err := s.db.NewSelect().
		TableExpr("download_queue").
		ColumnExpr("*").
		Where("status = ? AND torrent_hash IS NOT NULL AND deleted_at IS NULL", status).
		Scan(ctx, &entries)
	if err != nil {
		slog.Error("Failed to fetch downloading entries", "error", err)
		return
	}

	for _, entry := range entries {
		if entry.TorrentHash == nil {
			continue
		}

		items, err := s.fetchTorrents(ctx, *entry.TorrentHash, "")
		if err != nil || len(items) == 0 {
			continue
		}

		t := items[0]
		newStatus := status
		if t.Progress >= 1.0 {
			newStatus = "completed"
		}

		_, err = s.db.NewUpdate().
			TableExpr("download_queue").
			Set("status = ?, updated_at = now()", newStatus).
			Where("id = ?", entry.ID).
			Exec(ctx)
		if err != nil {
			slog.Warn("Failed to update torrent status", "id", entry.ID, "error", err)
		}
	}
}

func (s *DownloaderService) addTorrent(ctx context.Context, magnetURL, savePath string) (string, error) {
	if s.qbtURL == "" {
		return "", fmt.Errorf("qBittorrent URL is not configured")
	}

	body := &strings.Builder{}
	writer := multipart.NewWriter(body)
	if err := writer.WriteField("urls", magnetURL); err != nil {
		return "", err
	}
	if savePath != "" {
		if err := writer.WriteField("savepath", savePath); err != nil {
			return "", err
		}
	}
	if err := writer.Close(); err != nil {
		return "", err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, s.qbtURL+"/api/v2/torrents/add", strings.NewReader(body.String()))
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

	return extractMagnetHash(magnetURL), nil
}

func (s *DownloaderService) fetchTorrents(ctx context.Context, hash string, category string) ([]torrentInfo, error) {
	if s.qbtURL == "" {
		return nil, fmt.Errorf("qBittorrent URL is not configured")
	}

	query := url.Values{}
	if category != "" {
		query.Set("category", category)
	}
	if hash != "" {
		query.Set("hashes", hash)
	}

	endpoint := s.qbtURL + "/api/v2/torrents/info"
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
