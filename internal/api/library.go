package api

import (
	"encoding/json"
	"net/http"
	
	"github.com/go-chi/chi/v5"
	"github.com/kingbenny101/kbarr/internal/anidb"
	"github.com/kingbenny101/kbarr/internal/db"
	"github.com/kingbenny101/kbarr/internal/logger"
	"github.com/kingbenny101/kbarr/internal/models"
)

func handleMediaSearch(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query().Get("q")
	if query == "" {
		http.Error(w, "missing query parameter q", http.StatusBadRequest)
		return
	}

	logger.Log.Infof("[API] Search request: %s", query)

	results := anidb.SearchTitles(query)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(results)
}

func handleMediaDetails(w http.ResponseWriter, r *http.Request) {
	aid := chi.URLParam(r, "aid")

	logger.Log.Infof("[API] Details request for AID: %s", aid)

	details, err := anidb.GetAnimeDetails(aid)
	if err != nil {
		logger.Log.Errorf("[API] Failed to get details for AID %s: %v", aid, err)
		http.Error(w, "failed to fetch anime details", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(details)
}

func handleAddMedia(w http.ResponseWriter, r *http.Request) {
	var body struct {
		AID string `json:"aid"`
	}

	err := json.NewDecoder(r.Body).Decode(&body)
	if err != nil || body.AID == "" {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	logger.Log.Infof("[API] Add media request for AID: %s", body.AID)

	details, err := anidb.GetAnimeDetails(body.AID)
	if err != nil {
		logger.Log.Errorf("[API] Failed to fetch details for AID %s: %v", body.AID, err)
		http.Error(w, "failed to fetch media details", http.StatusInternalServerError)
		return
	}

	anime := models.Anime{
		AnidbID:         details.AID,
		Title:           details.MainTitle(),
		Description:     details.Description,
		Episodes:        details.EpisodeCount,
		Status:          "watching",
		AlternateTitles: details.OfficialTitles(),
	}

	id, err := db.InsertAnime(anime)
	if err != nil {
		logger.Log.Errorf("[API] Failed to insert media: %v", err)
		http.Error(w, "failed to save media", http.StatusInternalServerError)
		return
	}

	logger.Log.Infof("[API] Media added with ID %d: %s", id, anime.Title)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"id":    id,
		"title": anime.Title,
	})
}

func handleGetMediaList(w http.ResponseWriter, r *http.Request) {
	animeList, err := db.GetAllAnime()
	if err != nil {
		logger.Log.Errorf("[API] Failed to fetch media list: %v", err)
		http.Error(w, "failed to fetch media list", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(animeList)
}
