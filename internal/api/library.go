package api

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/kingbenny101/kbarr/internal/anidb"
	"github.com/kingbenny101/kbarr/internal/config"
	"github.com/kingbenny101/kbarr/internal/db"
	"github.com/kingbenny101/kbarr/internal/logger"
	"github.com/kingbenny101/kbarr/internal/models"
	"github.com/kingbenny101/kbarr/internal/prowlarr"
	"github.com/kingbenny101/kbarr/internal/tmdb"
)

type UnifiedSearchResult struct {
	ID        string `json:"id"`
	Title     string `json:"title"`
	Source    string `json:"source"`     // "anidb", "tmdb"
	MediaType string `json:"media_type"` // "anime", "movie", "tv"
	Poster    string `json:"poster"`
	Year      string `json:"year"`
}

func handleMediaSearch(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query().Get("q")
	if query == "" {
		http.Error(w, "missing query parameter q", http.StatusBadRequest)
		return
	}

	logger.Log.Infof("[API] Unified search request: %s", query)

	var unifiedResults []UnifiedSearchResult

	// 1. Search AniDB
	anidbResults := anidb.SearchTitles(query)
	for _, res := range anidbResults {
		unifiedResults = append(unifiedResults, UnifiedSearchResult{
			ID:        res.AID,
			Title:     res.Title,
			Source:    "anidb",
			MediaType: "anime",
		})
	}

	// 2. Search TMDB
	tmdbResults, err := tmdb.SearchMulti(query)
	if err == nil && tmdbResults != nil {
		for _, res := range tmdbResults.Results {
			// TMDB Multi-search returns "movie", "tv", and "person". We skip "person".
			if res.MediaType == "person" {
				continue
			}

			title := res.Title
			if title == "" {
				title = res.Name
			}

			year := res.ReleaseDate
			if year == "" {
				year = res.FirstAirDate
			}
			if len(year) > 4 {
				year = year[:4]
			}

			unifiedResults = append(unifiedResults, UnifiedSearchResult{
				ID:        fmt.Sprintf("%d", res.ID),
				Title:     title,
				Source:    "tmdb",
				MediaType: res.MediaType,
				Poster:    tmdb.GetPosterURL(res.PosterPath),
				Year:      year,
			})
		}
	} else if err != nil {
		logger.Log.Warnf("[API] TMDB search failed or skipped: %v", err)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(unifiedResults)
}

func handleAddMedia(w http.ResponseWriter, r *http.Request) {
	var body struct {
		AID       string `json:"aid"`        // used for anidb
		TMDBID    string `json:"tmdb_id"`    // used for tmdb
		MediaType string `json:"media_type"` // for tmdb (movie, tv)
		Source    string `json:"source"`     // manually specifying source if possible
	}

	err := json.NewDecoder(r.Body).Decode(&body)
	if err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	var media models.Media

	if body.AID != "" || body.Source == "anidb" {
		id := body.AID
		logger.Log.Infof("[API] Add media request for AniDB AID: %s", id)

		details, err := anidb.GetAnimeDetails(id)
		if err != nil {
			logger.Log.Errorf("[API] Failed to fetch details for AID %s: %v", id, err)
			http.Error(w, "failed to fetch media details", http.StatusInternalServerError)
			return
		}

		media = models.Media{
			ExternalID:      details.AID,
			Source:          "anidb",
			Type:            "anime",
			Title:           details.MainTitle(),
			TitleOriginal:   details.MainTitle(),
			Description:     details.Description,
			Episodes:        details.EpisodeCount,
			Status:          "watching",
			AlternateTitles: details.OfficialTitles(),
			CoverImage:      details.PosterURL(),
		}
	} else if body.TMDBID != "" || body.Source == "tmdb" {
		id := body.TMDBID
		mType := body.MediaType
		logger.Log.Infof("[API] Add media request for TMDB ID: %s (%s)", id, mType)

		details, err := tmdb.GetMediaDetails(id, mType)
		if err != nil {
			logger.Log.Errorf("[API] Failed to fetch details for TMDB ID %s: %v", id, err)
			http.Error(w, "failed to fetch media details", http.StatusInternalServerError)
			return
		}

		media = models.Media{
			ExternalID:    fmt.Sprintf("%d", details.ID),
			Source:        "tmdb",
			Type:          mType,
			Title:         details.GetTitle(),
			TitleOriginal: details.GetOriginalTitle(),
			Description:   details.Overview,
			Status:        "watching",
			CoverImage:    tmdb.GetPosterURL(details.PosterPath),
		}
	} else {
		http.Error(w, "missing id/source in request", http.StatusBadRequest)
		return
	}

	id, err := db.InsertMedia(media)
	if err != nil {
		logger.Log.Errorf("[API] Failed to insert media: %v", err)
		http.Error(w, "failed to save media", http.StatusInternalServerError)
		return
	}

	logger.Log.Infof("[API] Media added with ID %d: %s", id, media.Title)

	// Check if auto-monitor on add is enabled
	cfg := config.Get()
	if cfg.AutoMonitorOnAdd {
		logger.Log.Infof("[API] Auto-monitoring enabled for media ID %d: %s", id, media.Title)

		// Update media to monitored status
		err = db.UpdateMediaMonitorStatus(fmt.Sprintf("%d", id), true)
		if err != nil {
			logger.Log.Errorf("[API] Failed to auto-monitor media: %v", err)
		}

		// Trigger immediate search if Prowlarr is configured
		if cfg.ProwlarrApiKey != "" && cfg.ProwlarrApiKey != "error" {
			logger.Log.Infof("[API] Triggering auto-search for media ID %d: %s", id, media.Title)
			results, err := prowlarr.Search(media.Title)
			if err != nil {
				logger.Log.Errorf("[API] Auto-search failed for media ID %d: %v", id, err)
			} else {
				logger.Log.Infof("[API] Auto-search found %d results for media ID %d", len(results), id)
			}
		}
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"id":    id,
		"title": media.Title,
	})
}

func handleGetMediaList(w http.ResponseWriter, r *http.Request) {
	mediaList, err := db.GetAllMedia()
	if err != nil {
		logger.Log.Errorf("[API] Failed to fetch media list: %v", err)
		http.Error(w, "failed to fetch media list", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(mediaList)
}

func handleGetMediaByID(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	logger.Log.Infof("[API] Fetch media details for ID: %s", id)

	media, err := db.GetMediaByID(id)
	if err != nil {
		logger.Log.Errorf("[API] Failed to fetch media with ID %s: %v", id, err)
		http.Error(w, "failed to fetch media", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(media)
}

func handleDeleteMedia(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	logger.Log.Infof("[API] Delete media request for ID: %s", id)

	err := db.DeleteMedia(id)
	if err != nil {
		logger.Log.Errorf("[API] Failed to delete media with ID %s: %v", id, err)
		http.Error(w, "failed to delete media", http.StatusInternalServerError)
		return
	}

	logger.Log.Infof("[API] Media with ID %s deleted successfully", id)

	w.WriteHeader(http.StatusNoContent)
}

func handleUpdateMonitorStatus(w http.ResponseWriter, r *http.Request) {
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
		logger.Log.Errorf("[API] Failed to update monitor status: %v", err)
		http.Error(w, "failed to update monitor status", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func handleTriggerSearch(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	logger.Log.Infof("[API] Trigger search for media ID: %s", id)

	media, err := db.GetMediaByID(id)
	if err != nil {
		logger.Log.Errorf("[API] Failed to fetch media with ID %s: %v", id, err)
		http.Error(w, "media not found", http.StatusNotFound)
		return
	}

	cfg := config.Get()
	if cfg.ProwlarrApiKey == "" || cfg.ProwlarrApiKey == "error" {
		logger.Log.Warnf("[API] Prowlarr not configured for search")
		http.Error(w, "Prowlarr not configured", http.StatusBadRequest)
		return
	}

	results, err := prowlarr.Search(media.Title)
	if err != nil {
		logger.Log.Errorf("[API] Prowlarr search failed: %v", err)
		http.Error(w, fmt.Sprintf("search failed: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"media_title":   media.Title,
		"results_count": len(results),
		"results":       results,
	})
}
