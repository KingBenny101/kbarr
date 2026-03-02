package api

import (
	"encoding/json"
	"io/fs"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/kingbenny101/kbarr/internal/anidb"
	"github.com/kingbenny101/kbarr/internal/config"
	"github.com/kingbenny101/kbarr/internal/db"
	"github.com/kingbenny101/kbarr/internal/logger"
	"github.com/kingbenny101/kbarr/internal/models"
	"github.com/kingbenny101/kbarr/internal/version"
)

var cfg *config.Config

func NewRouter(c *config.Config) http.Handler {
	cfg = c
	r := chi.NewRouter()

	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins: []string{"http://localhost:5173"},
		AllowedMethods: []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders: []string{"Content-Type"},
	}))

	r.Get("/api/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("KBArr is running"))
	})

	r.Get("/api/version", handleGetVersion)
	r.Get("/api/anime/search", handleAnimeSearch)
	r.Get("/api/anime", handleGetAnimeList)
	r.Post("/api/anime", handleAddAnime)

	distFS, err := fs.Sub(frontendFS, "frontend")
	if err != nil {
		logger.Log.Warnf("[API] Frontend not embedded: %v", err)
		return r
	}

	fileServer := http.FileServer(http.FS(distFS))

	r.Get("/*", func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path[1:]

		if path == "" {
			path = "index.html"
		}

		_, err := distFS.Open(path)
		if err != nil {
			indexBytes, _ := fs.ReadFile(distFS, "index.html")
			w.Header().Set("Content-Type", "text/html")
			w.Write(indexBytes)
			return
		}
		fileServer.ServeHTTP(w, r)
	})

	logger.Log.Info("[API] Frontend embedded and ready")
	return r
}

func handleGetVersion(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"version": version.Version,
		"commit":  version.GitCommit,
		"built":   version.BuildTime,
	})
}

func handleAnimeSearch(w http.ResponseWriter, r *http.Request) {
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

func handleAnimeDetails(w http.ResponseWriter, r *http.Request) {
	aid := chi.URLParam(r, "aid")

	logger.Log.Infof("[API] Details request for AID: %s", aid)

	details, err := anidb.GetAnimeDetails(aid, cfg)
	if err != nil {
		logger.Log.Errorf("[API] Failed to get details for AID %s: %v", aid, err)
		http.Error(w, "failed to fetch anime details", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(details)
}

func handleAddAnime(w http.ResponseWriter, r *http.Request) {
	var body struct {
		AID string `json:"aid"`
	}

	err := json.NewDecoder(r.Body).Decode(&body)
	if err != nil || body.AID == "" {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	logger.Log.Infof("[API] Add anime request for AID: %s", body.AID)

	details, err := anidb.GetAnimeDetails(body.AID, cfg)
	if err != nil {
		logger.Log.Errorf("[API] Failed to fetch details for AID %s: %v", body.AID, err)
		http.Error(w, "failed to fetch anime details", http.StatusInternalServerError)
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
		logger.Log.Errorf("[API] Failed to insert anime: %v", err)
		http.Error(w, "failed to save anime", http.StatusInternalServerError)
		return
	}

	logger.Log.Infof("[API] Anime added with ID %d: %s", id, anime.Title)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"id":    id,
		"title": anime.Title,
	})
}

func handleGetAnimeList(w http.ResponseWriter, r *http.Request) {
	animeList, err := db.GetAllAnime()
	if err != nil {
		logger.Log.Errorf("[API] Failed to fetch anime list: %v", err)
		http.Error(w, "failed to fetch anime list", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(animeList)
}
