package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/kingbenny101/kbarr/services/core/internal/db"
	"github.com/kingbenny101/kbarr/shared/logger"
	"github.com/kingbenny101/kbarr/shared/models"
)

func HandleGetMonitoredList(w http.ResponseWriter, r *http.Request) {
	monitors, err := db.GetAllMonitored()
	if err != nil {
		logger.Log.Errorf("[API] Failed to fetch monitored list %v", err)
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

	logger.Log.Infof("[API] HandleAddMonitor called for %s", monitor.Title)

	db.InsertMonitor(monitor)

	// Add to search queue
	db.AddToSearchQueue(models.SearchQueue{
		LibraryID:     monitor.LibraryID,
		Title:         monitor.Title,
		EpisodeTitle:  monitor.EpisodeTitle,
		Season:        monitor.Season,
		EpisodeNumber: monitor.EpisodeNumber,
		IsEpisode:     monitor.IsEpisode,
		IsSeason:      monitor.IsSeason,
	})

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

	logger.Log.Infof("[API] Delete monitor request for ID %d", id)

	err = db.DeleteMonitor(uint(id))
	if err != nil {
		logger.Log.Errorf("[API] Failed to delete monitor entry %v", err)
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
		logger.Log.Errorf("[API] Failed to fetch monitored items for library ID %d: %v", id, err)
		http.Error(w, "failed to fetch monitored items", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(monitors)
}

func HandleUnmonitor(w http.ResponseWriter, r *http.Request) {
	var body struct {
		LibraryID uint   `json:"library_id"`
		AniDBID   string `json:"anidb_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}

	if err := db.UnmonitorByDetails(body.LibraryID, body.AniDBID); err != nil {
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

	logger.Log.Infof("[API] HandleBulkAddMonitor called for %d items", len(monitors))

	db.InsertMonitorsBulk(monitors)

	// Add to search queue with season-priority rule
	// 1. Identify seasons being monitored
	monitoredSeasons := make(map[int]bool)
	for _, m := range monitors {
		if m.IsSeason {
			monitoredSeasons[m.Season] = true
			db.AddToSearchQueue(models.SearchQueue{
				LibraryID:     m.LibraryID,
				Title:         m.Title,
				EpisodeTitle:  m.EpisodeTitle,
				Season:        m.Season,
				EpisodeNumber: m.EpisodeNumber,
				IsEpisode:     m.IsEpisode,
				IsSeason:      m.IsSeason,
			})
		}
	}

	// 2. Add episodes only if their season is not being monitored as a whole
	for _, m := range monitors {
		if m.IsEpisode && !monitoredSeasons[m.Season] {
			db.AddToSearchQueue(models.SearchQueue{
				LibraryID:     m.LibraryID,
				Title:         m.Title,
				EpisodeTitle:  m.EpisodeTitle,
				Season:        m.Season,
				EpisodeNumber: m.EpisodeNumber,
				IsEpisode:     m.IsEpisode,
				IsSeason:      m.IsSeason,
			})
		}
	}

	w.WriteHeader(http.StatusCreated)
}

// HandleUnmonitorSeason handles unmonitoring an entire season
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
