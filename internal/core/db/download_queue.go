package db

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/kingbenny101/kbarr/internal/models"
)

func GetAllDownloadQueue() ([]models.DownloadQueue, error) {
	if err := ensureDB(); err != nil {
		return nil, err
	}
	var entries []models.DownloadQueue
	if err := DB.NewSelect().Model(&entries).Where("deleted_at IS NULL").OrderExpr("created_at DESC").Scan(context.Background()); err != nil {
		return nil, fmt.Errorf("failed to query download queue: %w", err)
	}
	return entries, nil
}

func GetDownloadQueueEntry(ctx context.Context, id string, entry *models.DownloadQueue) error {
	if err := ensureDB(); err != nil {
		return err
	}
	numID, err := strconv.ParseInt(id, 10, 64)
	if err != nil {
		return fmt.Errorf("invalid id %q: %w", id, err)
	}
	return DB.NewSelect().Model(entry).Where("id = ? AND deleted_at IS NULL", numID).Scan(ctx)
}

func InsertTestDownloadQueueEntry(ctx context.Context, torrentURL, title string) error {
	if err := ensureDB(); err != nil {
		return err
	}
	status := "pending"
	if title == "" {
		title = torrentURL
	}
	entry := models.DownloadQueue{
		TorrentURL:  &torrentURL,
		TorrentName: &torrentURL,
		Title:       &title,
		Status:      &status,
	}
	_, err := DB.NewInsert().Model(&entry).Exec(ctx)
	return err
}

type DeleteOptions struct {
	Blacklist   bool `json:"blacklist"`
	Unmonitor   bool `json:"unmonitor"`
	DeleteFiles bool `json:"deleteFiles"`
}

func DeleteDownloadQueueEntry(ctx context.Context, id string, opts DeleteOptions) error {
	if err := ensureDB(); err != nil {
		return err
	}
	numID, err := strconv.ParseInt(id, 10, 64)
	if err != nil {
		return fmt.Errorf("invalid id %q: %w", id, err)
	}

	var entry models.DownloadQueue
	if err := DB.NewSelect().Model(&entry).Where("id = ? AND deleted_at IS NULL", numID).Scan(ctx); err == nil {
		status := ""
		if entry.Status != nil {
			status = *entry.Status
		}
		if entry.TorrentHash != nil && *entry.TorrentHash != "" {
			if status == "completed" {
				removeFromQBittorrent(ctx, *entry.TorrentHash, opts.DeleteFiles)
			} else {
				removeFromQBittorrent(ctx, *entry.TorrentHash, false)
			}
		}
		if opts.Blacklist {
			hash := ""
			if entry.TorrentHash != nil {
				hash = *entry.TorrentHash
			}
			name := ""
			if entry.TorrentName != nil {
				name = *entry.TorrentName
			}
			title := ""
			if entry.Title != nil {
				title = *entry.Title
			}
			_ = AddToBlacklist(ctx, hash, name, title)
		}
		if entry.MonitorID != nil {
			resetMonitorOnQueueDelete(ctx, *entry.MonitorID, opts.Unmonitor)
		}
		if opts.DeleteFiles && entry.SavePath != nil && *entry.SavePath != "" {
			title := ""
			if entry.Title != nil {
				title = *entry.Title
			}
			deleteDownloadedFiles(ctx, *entry.SavePath, title)
		}
	}

	_, err = DB.NewUpdate().Model((*models.DownloadQueue)(nil)).
		Set("deleted_at = now(), updated_at = now()").
		Where("id = ?", numID).
		Exec(ctx)
	return err
}

// ClearCompletedDownloads soft-deletes every completed queue entry so the
// downloads table is decluttered. It deliberately does not touch qBittorrent or
// any files on disk — the torrents keep seeding and the hardlinks remain.
func ClearCompletedDownloads(ctx context.Context) (int64, error) {
	if err := ensureDB(); err != nil {
		return 0, err
	}
	res, err := DB.NewUpdate().Model((*models.DownloadQueue)(nil)).
		Set("deleted_at = now(), updated_at = now()").
		Where("status = 'completed' AND deleted_at IS NULL").
		Exec(ctx)
	if err != nil {
		return 0, fmt.Errorf("failed to clear completed downloads: %w", err)
	}
	n, _ := res.RowsAffected()
	return n, nil
}

func resetMonitorOnQueueDelete(ctx context.Context, monitorID int64, unmonitor bool) {
	var mon models.Monitor
	if err := DB.NewSelect().Model(&mon).Where("id = ? AND deleted_at IS NULL", monitorID).Scan(ctx); err != nil {
		return
	}

	if unmonitor {
		DB.NewUpdate().Model((*models.Monitor)(nil)).
			Set("monitored = false, updated_at = now()").
			Where("id = ?", monitorID).
			Exec(ctx)
	} else {
		DB.NewUpdate().Model((*models.Monitor)(nil)).
			Set("monitored = true, status = 'pending', updated_at = now()").
			Where("id = ?", monitorID).
			Exec(ctx)
	}

	if mon.IsSeason && mon.LibraryID != 0 {
		if unmonitor {
			DB.NewUpdate().Model((*models.Monitor)(nil)).
				Set("monitored = false, updated_at = now()").
				Where("library_id = ? AND is_episode = true AND deleted_at IS NULL", mon.LibraryID).
				Exec(ctx)
		} else {
			DB.NewUpdate().Model((*models.Monitor)(nil)).
				Set("monitored = true, status = 'pending', updated_at = now()").
				Where("library_id = ? AND is_episode = true AND deleted_at IS NULL", mon.LibraryID).
				Exec(ctx)
		}
	}
}

func deleteDownloadedFiles(ctx context.Context, savePath, title string) {
	downloaderAddr := strings.TrimRight(os.Getenv("DOWNLOADER_HEALTH_ADDR"), "/")
	if downloaderAddr == "" {
		downloaderAddr = "http://localhost:8083"
	}
	body, _ := json.Marshal(map[string]string{"save_path": savePath, "hardlink_dir": title})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, downloaderAddr+"/files/delete", bytes.NewReader(body))
	if err != nil {
		slog.Warn("Failed to build files/delete request", "save_path", savePath, "error", err)
		return
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		slog.Warn("Failed to delete downloaded files", "save_path", savePath, "error", err)
		return
	}
	resp.Body.Close()
}

func removeFromQBittorrent(ctx context.Context, hash string, deleteFiles bool) {
	downloaderAddr := strings.TrimRight(os.Getenv("DOWNLOADER_HEALTH_ADDR"), "/")
	if downloaderAddr == "" {
		downloaderAddr = "http://localhost:8083"
	}
	body, _ := json.Marshal(map[string]interface{}{"hash": hash, "deleteFiles": deleteFiles})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, downloaderAddr+"/torrent/delete", bytes.NewReader(body))
	if err != nil {
		slog.Warn("Failed to build qBittorrent remove request", "hash", hash, "error", err)
		return
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		slog.Warn("Failed to remove torrent from qBittorrent", "hash", hash, "error", err)
		return
	}
	resp.Body.Close()
}
