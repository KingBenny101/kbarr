package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/kingbenny101/kbarr/services/core/internal/db"
	"github.com/kingbenny101/kbarr/shared/models"
	proto "github.com/kingbenny101/kbarr/shared/proto"
)

func HandleAddMedia(w http.ResponseWriter, r *http.Request, anidbClient proto.AniDBServiceClient) {
	var media models.Media
	err := json.NewDecoder(r.Body).Decode(&media)
	if err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	slog.Info("HandleAddMedia called", "title", media.Title, "aid", media.AID)

	exists, err := db.CheckMediaExists(media.AID)
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

	slog.Info("Preparing detailed info", "title", media.Title)
	ctx, cancel := context.WithTimeout(r.Context(), 45*time.Second)
	defer cancel()

	prepared, err := anidbClient.PrepareDetailedFromMedia(ctx, toAniDBMediaProto(media))
	if err != nil {
		slog.Error("Failed to get anime details from AniDB", "error", err)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadGateway)
		_ = json.NewEncoder(w).Encode(map[string]string{
			"error": "Failed to fetch anime details from AniDB. Check your AniDB client settings.",
		})
		return
	} else if prepared != nil && prepared.GetPosterUrl() != "" {
		media.PosterURL = prepared.GetPosterUrl()
		slog.Info("Set poster URL", "posterURL", media.PosterURL)

	}

	id, err := db.InsertMedia(media)
	if err != nil {
		slog.Error("Failed to insert media", "error", err)
		http.Error(w, "failed to save media", http.StatusInternalServerError)
		return
	}

	slog.Info("Media added", "id", id, "title", media.Title)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"message": "Media added successfully!",
	})

	if prepared != nil {
		prepared.LibraryId = uint64(id)

		detailed := toDetailedModel(prepared)
		detailed.LibraryID = uint(id)

		if _, insertErr := db.InsertDetailed(detailed); insertErr != nil {
			slog.Error("Failed to insert detailed info for media ID", "id", id, "error", insertErr)
			if deleteErr := db.DeleteMedia(strconv.FormatInt(id, 10)); deleteErr != nil {
				slog.Error("Failed to rollback media insert after detailed insert failure", "id", id, "error", deleteErr)
			}
			http.Error(w, "failed to save media", http.StatusInternalServerError)
			return
		}

		slog.Info("Detailed info added", "id", id)
	}

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

func HandleTriggerSearch(w http.ResponseWriter, r *http.Request, prowlarrClient proto.ProwlarrServiceClient) {
	id := chi.URLParam(r, "id")

	slog.Info("Trigger search for media ID", "id", id)

	media, err := db.GetMediaByID(id)
	if err != nil {
		slog.Error("Failed to fetch media with ID", "id", id, "error", err)
		http.Error(w, "media not found", http.StatusNotFound)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()

	resultsResponse, err := prowlarrClient.Search(ctx, &proto.ProwlarrSearchRequest{Query: media.Title})
	if err != nil {
		slog.Error("Prowlarr search failed", "error", err)
		http.Error(w, fmt.Sprintf("search failed: %v", err), http.StatusInternalServerError)
		return
	}

	results := toProwlarrSearchResults(resultsResponse.GetResults())

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"media_title":   media.Title,
		"results_count": len(results),
		"results":       results,
	})
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
