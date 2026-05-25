package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/kingbenny101/kbarr/services/core/internal/db"
	"github.com/kingbenny101/kbarr/shared/logger"
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

	logger.Log.Infof("[API] HandleAddMedia called with title %s, aid %d", media.Title, media.AID)

	exists, err := db.CheckMediaExists(media.AID)
	if err != nil {
		logger.Log.Errorf("[API] Failed to check media existence %v", err)
		http.Error(w, "failed to check media existence", http.StatusInternalServerError)
		return
	}

	if exists {
		logger.Log.Infof("[API] Media already exists %s", media.Title)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"message": "Media already exists in library!",
		})
		return
	}

	logger.Log.Infof("[API] Preparing detailed info for %s", media.Title)
	ctx, cancel := context.WithTimeout(r.Context(), 45*time.Second)
	defer cancel()

	prepared, err := anidbClient.PrepareDetailedFromMedia(ctx, toAniDBMediaProto(media))
	if err != nil {
		logger.Log.Warnf("[API] Failed to get anime details for poster URL %v", err)

	} else if prepared != nil && prepared.GetPosterUrl() != "" {
		media.PosterURL = prepared.GetPosterUrl()
		logger.Log.Infof("[API] Set poster URL %s", media.PosterURL)

	}

	id, err := db.InsertMedia(media)
	if err != nil {
		logger.Log.Errorf("[API] Failed to insert media %v", err)
		http.Error(w, "failed to save media", http.StatusInternalServerError)
		return
	}

	logger.Log.Infof("[API] Media added with ID %d %s", id, media.Title)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"message": "Media added successfully!",
	})

	// Process adding Detailed and download image asynchronously
	if prepared != nil {
		go func(prepared *proto.AniDBDetailed, media models.Media, id int64) {
			logger.Log.Infof("[API] Starting async detailed info processing for media ID %d %s", id, media.Title)

			prepared.LibraryId = uint64(id)

			detailed := toDetailedModel(prepared)
			detailed.LibraryID = uint(id) // Link Detailed to Media

			if _, insertErr := db.InsertDetailed(detailed); insertErr != nil {
				logger.Log.Errorf("[API] Failed to insert detailed info for media ID %d %v", id, insertErr)

				return
			}

			logger.Log.Infof("[API] Detailed info added for media ID %d", id)

		}(prepared, media, id)
	}

}

func HandleGetMediaList(w http.ResponseWriter, r *http.Request) {
	mediaList, err := db.GetAllMedia()
	if err != nil {
		logger.Log.Errorf("[API] Failed to fetch media list %v", err)
		http.Error(w, "failed to fetch media list", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(mediaList)
}

func HandleDeleteMedia(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	logger.Log.Infof("[API] Delete media request for ID %s", id)

	err := db.DeleteMedia(id)
	if err != nil {
		logger.Log.Errorf("[API] Failed to delete media with ID %s %v", id, err)
		http.Error(w, "failed to delete media", http.StatusInternalServerError)
		return
	}

	logger.Log.Infof("[API] Media with ID %s deleted successfully", id)

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

	logger.Log.Infof("[API] Update monitor status for ID %s to %v", id, body.Monitored)

	err = db.UpdateMediaMonitorStatus(id, body.Monitored)
	if err != nil {
		logger.Log.Errorf("[API] Failed to update monitor status %v", err)
		http.Error(w, "failed to update monitor status", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func HandleTriggerSearch(w http.ResponseWriter, r *http.Request, prowlarrClient proto.ProwlarrServiceClient) {
	id := chi.URLParam(r, "id")

	logger.Log.Infof("[API] Trigger search for media ID %s", id)

	media, err := db.GetMediaByID(id)
	if err != nil {
		logger.Log.Errorf("[API] Failed to fetch media with ID %s %v", id, err)
		http.Error(w, "media not found", http.StatusNotFound)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()

	resultsResponse, err := prowlarrClient.Search(ctx, &proto.ProwlarrSearchRequest{Query: media.Title})
	if err != nil {
		logger.Log.Errorf("[API] Prowlarr search failed %v", err)
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

	logger.Log.Infof("[API] Fetch detailed info for media ID %s", id)

	library_id, err := strconv.ParseUint(id, 10, 64)
	if err != nil {
		logger.Log.Errorf("[API] Invalid media ID format %s", id)
		http.Error(w, "invalid media ID format", http.StatusBadRequest)
		return
	}
	detailed, err := db.GetDetailedByLibraryID(uint(library_id))
	if err != nil {
		logger.Log.Errorf("[API] Failed to fetch detailed info for media ID %s %v", id, err)
		http.Error(w, "failed to fetch detailed info", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(detailed)
}
