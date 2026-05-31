package handlers

import (
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

func HandleDeleteDownload(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if err := db.DeleteDownloadQueueEntry(context.Background(), id); err != nil {
		slog.Error("Failed to delete download queue entry", "id", id, "error", err)
		http.Error(w, "failed to delete download", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
