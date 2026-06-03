package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/kingbenny101/kbarr/services/core/internal/db"
)

func HandleListDownloads(w http.ResponseWriter, r *http.Request) {
	entries, err := db.GetAllDownloadQueue()
	if err != nil {
		slog.Error("Failed to list download queue", "error", err)
		http.Error(w, "failed to list downloads", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(entries)
}

func HandleAddTestDownload(w http.ResponseWriter, r *http.Request) {
	var body struct {
		TorrentURL string `json:"torrent_url"`
		Title      string `json:"title"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.TorrentURL == "" {
		http.Error(w, "torrent_url is required", http.StatusBadRequest)
		return
	}
	if err := db.InsertTestDownloadQueueEntry(context.Background(), body.TorrentURL, body.Title); err != nil {
		slog.Error("Failed to insert test download queue entry", "error", err)
		http.Error(w, "failed to insert entry", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusCreated)
}

func HandleClearBlacklist(w http.ResponseWriter, r *http.Request) {
	if err := db.ClearBlacklist(context.Background()); err != nil {
		slog.Error("Failed to clear blacklist", "error", err)
		http.Error(w, "failed to clear blacklist", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func HandleCreateSymlinks(downloaderAddr string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := chi.URLParam(r, "id")
		body, _ := json.Marshal(map[string]string{"id": id})
		req, err := http.NewRequestWithContext(r.Context(), http.MethodPost, downloaderAddr+"/symlinks/create", bytes.NewReader(body))
		if err != nil {
			http.Error(w, "failed to build request", http.StatusInternalServerError)
			return
		}
		req.Header.Set("Content-Type", "application/json")
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			slog.Error("Failed to reach downloader for symlink creation", "error", err)
			http.Error(w, "downloader unreachable", http.StatusBadGateway)
			return
		}
		resp.Body.Close()
		w.WriteHeader(http.StatusNoContent)
	}
}

func HandleDeleteDownload(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	var opts db.DeleteOptions
	// Body is optional — plain DELETE with no body still works
	if r.ContentLength > 0 {
		_ = json.NewDecoder(r.Body).Decode(&opts)
	}

	if err := db.DeleteDownloadQueueEntry(context.Background(), id, opts); err != nil {
		slog.Error("Failed to delete download queue entry", "id", id, "error", err)
		http.Error(w, "failed to delete download", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
