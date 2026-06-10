package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/danielgtaylor/huma/v2"
	"github.com/kingbenny101/kbarr/internal/core/db"
)

func GetDownloadList() func(context.Context, *struct{}) (*DownloadListOutput, error) {
	return func(ctx context.Context, _ *struct{}) (*DownloadListOutput, error) {
		entries, err := db.GetAllDownloadQueue()
		if err != nil {
			slog.Error("Failed to list download queue", "error", err)
			return nil, huma.Error500InternalServerError("failed to list downloads", err)
		}
		return &DownloadListOutput{Body: entries}, nil
	}
}

func AddTestDownload() func(context.Context, *AddTestDownloadInput) (*struct{}, error) {
	return func(ctx context.Context, input *AddTestDownloadInput) (*struct{}, error) {
		if input.Body.TorrentURL == "" {
			return nil, huma.Error400BadRequest("torrent_url is required")
		}
		if err := db.InsertTestDownloadQueueEntry(ctx, input.Body.TorrentURL, input.Body.Title); err != nil {
			slog.Error("Failed to insert test download queue entry", "error", err)
			return nil, huma.Error500InternalServerError("failed to insert entry", err)
		}
		return nil, nil
	}
}

func ClearBlacklist() func(context.Context, *struct{}) (*struct{}, error) {
	return func(ctx context.Context, _ *struct{}) (*struct{}, error) {
		if err := db.ClearBlacklist(ctx); err != nil {
			slog.Error("Failed to clear blacklist", "error", err)
			return nil, huma.Error500InternalServerError("failed to clear blacklist", err)
		}
		return nil, nil
	}
}

func CreateHardlinks(downloaderAddr string) func(context.Context, *HardlinkInput) (*struct{}, error) {
	return func(ctx context.Context, input *HardlinkInput) (*struct{}, error) {
		body, _ := json.Marshal(map[string]int64{"id": input.ID})
		req, err := http.NewRequestWithContext(ctx, http.MethodPost, downloaderAddr+"/hardlinks/create", bytes.NewReader(body))
		if err != nil {
			return nil, huma.Error500InternalServerError("failed to build request", err)
		}
		req.Header.Set("Content-Type", "application/json")
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			slog.Error("Failed to reach downloader for hardlink creation", "error", err)
			return nil, huma.Error502BadGateway("downloader unreachable", err)
		}
		resp.Body.Close()
		return nil, nil
	}
}

func DeleteDownload() func(context.Context, *DeleteDownloadInput) (*struct{}, error) {
	return func(ctx context.Context, input *DeleteDownloadInput) (*struct{}, error) {
		var opts db.DeleteOptions
		if input.Body != nil {
			opts = *input.Body
		}
		id := strconv.FormatInt(input.ID, 10)
		if err := db.DeleteDownloadQueueEntry(ctx, id, opts); err != nil {
			slog.Error("Failed to delete download queue entry", "id", id, "error", err)
			return nil, huma.Error500InternalServerError("failed to delete download", err)
		}
		return nil, nil
	}
}
