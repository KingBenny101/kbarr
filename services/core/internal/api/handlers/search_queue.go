package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/kingbenny101/kbarr/services/core/internal/db"
	"github.com/kingbenny101/kbarr/shared/logger"
)

func HandleGetSearchQueue(w http.ResponseWriter, r *http.Request) {
	queue, err := db.GetSearchQueue()
	if err != nil {
		logger.Log.Errorf("[API] Failed to fetch search queue %v", err)
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
		logger.Log.Errorf("[API] Failed to delete search queue entry %v", err)
		http.Error(w, "failed to delete entry", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
