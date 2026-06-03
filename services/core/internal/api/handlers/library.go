package handlers

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/kingbenny101/kbarr/services/core/internal/clients"
	"github.com/kingbenny101/kbarr/services/core/internal/db"
	"github.com/kingbenny101/kbarr/shared/models"
)

func HandleAddMedia(w http.ResponseWriter, r *http.Request, metadataClient *clients.MetadataClient) {
	var media models.Media
	err := json.NewDecoder(r.Body).Decode(&media)
	if err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	slog.Info("HandleAddMedia called", "title", media.Title, "source", media.Source, "source_id", media.SourceID)

	exists, err := db.CheckMediaExists(media.Source, media.SourceID)
	if err != nil {
		slog.Error("Failed to check media existence", "error", err)
		http.Error(w, "failed to check media existence", http.StatusInternalServerError)
		return
	}

	if exists {
		slog.Info("Media already exists", "title", media.Title)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"message": "Media already exists in library!",
		})
		return
	}

	slog.Info("Preparing detailed info", "title", media.Title, "source", media.Source, "source_id", media.SourceID)
	ctx, cancel := context.WithTimeout(r.Context(), 45*time.Second)
	defer cancel()

	prepared, err := metadataClient.Prepare(ctx, media.Source, media.SourceID, media.Title, 0)
	if err != nil {
		slog.Error("Failed to get anime details from metadata service", "error", err)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadGateway)
		_ = json.NewEncoder(w).Encode(map[string]string{
			"error": "Failed to fetch anime details. Check your metadata source settings.",
		})
		return
	}

	if prepared.PosterURL != "" {
		media.PosterURL = prepared.PosterURL
	}

	id, err := db.InsertMedia(media)
	if err != nil {
		slog.Error("Failed to insert media", "error", err)
		http.Error(w, "failed to save media", http.StatusInternalServerError)
		return
	}

	slog.Info("Media added", "id", id, "title", media.Title)

	detailed := toDetailedModel(prepared, uint(id))
	if _, err := db.InsertDetailed(detailed); err != nil {
		slog.Error("Failed to insert detailed info", "id", id, "error", err)
		_ = db.DeleteMedia(strconv.FormatInt(id, 10))
		http.Error(w, "failed to save media details", http.StatusInternalServerError)
		return
	}

	slog.Info("Detailed info added", "id", id)

	var monitors []models.Monitor
	for _, ep := range prepared.Episodes {
		epNum, err := strconv.Atoi(ep.Number)
		if err != nil {
			// Skip non-integer episode numbers (specials, credits, etc.)
			continue
		}
		monitors = append(monitors, models.Monitor{
			LibraryID:     uint(id),
			Title:         prepared.Title,
			EpisodeTitle:  ep.Title,
			Season:        1,
			EpisodeNumber: epNum,
			IsEpisode:     true,
			IsSeason:      false,
			Source:        ep.Source,
			ExternalID:    ep.ExternalID,
			Status:        "unmonitored",
		})
	}
	if len(monitors) > 0 {
		if err := db.InsertMonitorsBulk(monitors); err != nil {
			slog.Warn("Failed to create episode monitors on add", "id", id, "error", err)
		} else {
			slog.Info("Created episode monitors", "id", id, "count", len(monitors), "status", "unmonitored")
		}
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"message": "Media added successfully!",
	})
}

func HandleGetMediaList(w http.ResponseWriter, r *http.Request) {
	mediaList, err := db.GetAllMedia()
	if err != nil {
		slog.Error("Failed to fetch media list", "error", err)
		http.Error(w, "failed to fetch media list", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(mediaList)
}

func HandleDeleteMedia(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	slog.Info("Delete media request", "id", id)

	err := db.DeleteMedia(id)
	if err != nil {
		slog.Error("Failed to delete media with ID", "id", id, "error", err)
		http.Error(w, "failed to delete media", http.StatusInternalServerError)
		return
	}

	slog.Info("Media deleted successfully", "id", id)

	w.WriteHeader(http.StatusNoContent)
}

func HandleUpdateMonitorStatus(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var body struct {
		Monitored bool `json:"monitored"`
	}

	err := json.NewDecoder(r.Body).Decode(&body)
	if err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	slog.Info("Update monitor status", "id", id, "monitored", body.Monitored)

	err = db.UpdateMediaMonitorStatus(id, body.Monitored)
	if err != nil {
		slog.Error("Failed to update monitor status", "error", err)
		http.Error(w, "failed to update monitor status", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func HandleGetDetailedByMediaID(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	slog.Info("Fetch detailed info for media ID", "id", id)

	library_id, err := strconv.ParseUint(id, 10, 64)
	if err != nil {
		slog.Error("Invalid media ID format", "id", id)
		http.Error(w, "invalid media ID format", http.StatusBadRequest)
		return
	}
	detailed, err := db.GetDetailedByLibraryID(uint(library_id))
	if err != nil {
		slog.Error("Failed to fetch detailed info for media ID", "id", id, "error", err)
		http.Error(w, "failed to fetch detailed info", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(detailed)
}

func HandleGetEpisodes(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	libraryID, err := strconv.ParseUint(id, 10, 64)
	if err != nil {
		http.Error(w, "invalid media ID", http.StatusBadRequest)
		return
	}

	// Parse type filter: ?types=1,2,3
	var types []int
	if raw := r.URL.Query().Get("types"); raw != "" {
		for _, part := range strings.Split(raw, ",") {
			if t, err := strconv.Atoi(strings.TrimSpace(part)); err == nil {
				types = append(types, t)
			}
		}
	}

	// Parse sort: ?sort=ep_no|title  ?order=asc|desc
	sortField := r.URL.Query().Get("sort")
	if sortField != "title" {
		sortField = "ep_no"
	}
	sortOrder := r.URL.Query().Get("order")
	if sortOrder != "desc" {
		sortOrder = "asc"
	}

	// Parse pagination: ?page=1&limit=10
	page := 1
	if p, err := strconv.Atoi(r.URL.Query().Get("page")); err == nil && p > 0 {
		page = p
	}
	limit := 10
	if l, err := strconv.Atoi(r.URL.Query().Get("limit")); err == nil && l > 0 && l <= 10000 {
		limit = l
	}

	result, err := db.QueryEpisodesByLibraryID(uint(libraryID), db.EpisodeQueryParams{
		Types:     types,
		SortField: sortField,
		SortOrder: sortOrder,
		Page:      page,
		Limit:     limit,
	})
	if err != nil {
		slog.Error("Failed to query episodes", "id", id, "error", err)
		http.Error(w, "failed to fetch episodes", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

func toDetailedModel(d *clients.AnimeMetadata, libraryID uint) models.Detailed {
	if d == nil {
		return models.Detailed{}
	}

	result := models.Detailed{
		Source:          d.Source,
		SourceID:        d.SourceID,
		LibraryID:       libraryID,
		Title:           d.Title,
		AlternateTitles: d.AlternateTitles,
		Description:     d.Description,
		ReleaseDate:     d.ReleaseDate,
		Genres:          d.Genres,
		PosterURL:       d.PosterURL,
		TotalEpisodes:   d.TotalEpisodes,
		TotalSeasons:    d.TotalSeasons,
	}

	if len(d.Episodes) > 0 {
		result.Episodes = make([]models.Episode, 0, len(d.Episodes))
		for _, ep := range d.Episodes {
			result.Episodes = append(result.Episodes, models.Episode{
				Source:     ep.Source,
				ExternalID: ep.ExternalID,
				Type:       ep.Type,
				EpNo:       ep.Number,
				Title:      ep.Title,
				AirDate:    ep.AirDate,
			})
		}
	}

	return result
}
