package handlers

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/kingbenny101/kbarr/services/core/internal/db"
)

func HandleGetSearchQueue(w http.ResponseWriter, r *http.Request) {
	queue, err := db.GetSearchQueue()
	if err != nil {
		slog.Error("Failed to fetch search queue", "error", err)
		http.Error(w, "failed to fetch search queue", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(queue)
}

func HandleDeleteSearchQueueEntry(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		http.Error(w, "invalid ID", http.StatusBadRequest)
		return
	}

	if err := db.DeleteSearchQueueEntry(uint(id)); err != nil {
		slog.Error("Failed to delete search queue entry", "error", err)
		http.Error(w, "failed to delete entry", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
