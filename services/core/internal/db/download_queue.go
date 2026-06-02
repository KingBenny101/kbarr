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
)

func GetAllDownloadQueue() ([]DownloadQueue, error) {
	if err := ensureDB(); err != nil {
		return nil, err
	}

	var entries []DownloadQueue
	if err := DB.NewSelect().Model(&entries).Where("deleted_at IS NULL").OrderExpr("created_at DESC").Scan(context.Background()); err != nil {
		return nil, fmt.Errorf("failed to query download queue: %w", err)
	}
	return entries, nil
}

func InsertTestDownloadQueueEntry(ctx context.Context, torrentURL, title string) error {
	if err := ensureDB(); err != nil {
		return err
	}
	status := "pending"
	if title == "" {
		title = torrentURL
	}
	entry := DownloadQueue{
		TorrentURL:  &torrentURL,
		TorrentName: &torrentURL,
		Title:       &title,
		Status:      &status,
	}
	_, err := DB.NewInsert().Model(&entry).Exec(ctx)
	return err
}

type DeleteOptions struct {
	Blacklist bool `json:"blacklist"`
	Unmonitor bool `json:"unmonitor"`
}

func DeleteDownloadQueueEntry(ctx context.Context, id string, opts DeleteOptions) error {
	if err := ensureDB(); err != nil {
		return err
	}

	numID, err := strconv.ParseInt(id, 10, 64)
	if err != nil {
		return fmt.Errorf("invalid id %q: %w", id, err)
	}

	// Fetch row before deleting so we can remove from qBittorrent, blacklist, and reset monitor
	var entry DownloadQueue
	if err := DB.NewSelect().Model(&entry).Where("id = ? AND deleted_at IS NULL", numID).Scan(ctx); err == nil {
		if entry.TorrentHash != nil && *entry.TorrentHash != "" {
			removeFromQBittorrent(ctx, *entry.TorrentHash)
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
	}

	_, err = DB.NewDelete().Model((*DownloadQueue)(nil)).Where("id = ?", numID).Exec(ctx)
	return err
}

func resetMonitorOnQueueDelete(ctx context.Context, monitorID int64, unmonitor bool) {
	var mon Monitor
	if err := DB.NewSelect().Model(&mon).Where("id = ? AND deleted_at IS NULL", monitorID).Scan(ctx); err != nil {
		return
	}

	newStatus := "monitored"
	if unmonitor {
		newStatus = "unmonitored"
	}

	DB.NewUpdate().Model((*Monitor)(nil)).
		Set("status = ?, updated_at = now()", newStatus).
		Where("id = ?", monitorID).
		Exec(ctx)

	// For season monitors, also reset all episode monitors in the same library
	if mon.IsSeason != nil && *mon.IsSeason && mon.LibraryID != nil {
		DB.NewUpdate().Model((*Monitor)(nil)).
			Set("status = ?, updated_at = now()", newStatus).
			Where("library_id = ? AND is_episode = true AND deleted_at IS NULL", *mon.LibraryID).
			Exec(ctx)
	}
}

func removeFromQBittorrent(ctx context.Context, hash string) {
	downloaderAddr := strings.TrimRight(os.Getenv("DOWNLOADER_HEALTH_ADDR"), "/")
	if downloaderAddr == "" {
		downloaderAddr = "http://localhost:8083"
	}
	body, _ := json.Marshal(map[string]string{"hash": hash})
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
