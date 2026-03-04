package api

import (
	"io/fs"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/kingbenny101/kbarr/internal/config"
	"github.com/kingbenny101/kbarr/internal/logger"
	"github.com/kingbenny101/kbarr/internal/workers"
)

var cfg *config.Config
var WorkerMgr *workers.Manager

func NewRouter(wm *workers.Manager) http.Handler {
	cfg = config.Get()
	WorkerMgr = wm
	r := chi.NewRouter()

	// r.Use(middleware.Logger)
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
	r.Get("/api/library/search", handleMediaSearch)
	r.Delete("/api/library/{id}", handleDeleteMedia)

	r.Get("/api/library", handleGetMediaList)
	r.Post("/api/library", handleAddMedia)

	r.Get("/api/settings", handleGetSettings)
	r.Post("/api/settings", handleUpdateSettings)

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
