package handlers

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/kingbenny101/kbarr/services/core/internal/db"
	"github.com/kingbenny101/kbarr/shared/models"
)

func HandleGetMonitoredList(w http.ResponseWriter, r *http.Request) {
	monitors, err := db.GetAllMonitored()
	if err != nil {
		slog.Error("Failed to fetch monitored list", "error", err)
		http.Error(w, "failed to fetch monitored list", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(monitors)
}

func HandleAddMonitor(w http.ResponseWriter, r *http.Request) {
	var monitor models.Monitor
	err := json.NewDecoder(r.Body).Decode(&monitor)
	if err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	slog.Info("HandleAddMonitor called", "title", monitor.Title)

	db.InsertMonitor(monitor)

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"message": "Entry monitored successfully!",
	})
}

func HandleDeleteMonitor(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		http.Error(w, "invalid ID", http.StatusBadRequest)
		return
	}

	slog.Info("Delete monitor request", "id", id)

	err = db.DeleteMonitor(uint(id))
	if err != nil {
		slog.Error("Failed to delete monitor entry", "error", err)
		http.Error(w, "failed to delete monitor entry", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func HandleGetMonitorsByLibraryID(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		http.Error(w, "invalid ID", http.StatusBadRequest)
		return
	}

	monitors, err := db.GetMonitorsByLibraryID(uint(id))
	if err != nil {
		slog.Error("Failed to fetch monitored items for library ID", "id", id, "error", err)
		http.Error(w, "failed to fetch monitored items", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(monitors)
}

func HandleUnmonitor(w http.ResponseWriter, r *http.Request) {
	var body struct {
		LibraryID  uint   `json:"library_id"`
		ExternalID string `json:"external_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}

	if err := db.UnmonitorByDetails(body.LibraryID, body.ExternalID); err != nil {
		http.Error(w, "failed to unmonitor", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func HandleBulkAddMonitor(w http.ResponseWriter, r *http.Request) {
	var monitors []models.Monitor
	if err := json.NewDecoder(r.Body).Decode(&monitors); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	slog.Info("HandleBulkAddMonitor called", "items", len(monitors))

	db.InsertMonitorsBulk(monitors)

	w.WriteHeader(http.StatusCreated)
}

func HandleUnmonitorSeason(w http.ResponseWriter, r *http.Request) {
	var body struct {
		LibraryID uint `json:"library_id"`
		Season    int  `json:"season"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}

	if err := db.DeleteMonitorsBySeason(body.LibraryID, body.Season); err != nil {
		http.Error(w, "failed to unmonitor season", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
