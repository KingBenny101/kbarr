package api

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/kingbenny101/kbarr/services/core/internal/api/handlers"
	"github.com/kingbenny101/kbarr/services/core/internal/auth"
	"github.com/kingbenny101/kbarr/services/core/internal/clients"
)

type Server struct {
	metadata *clients.MetadataClient
	version  string
}

func NewRouter(metadataClient *clients.MetadataClient, version string, authStore *auth.Store) http.Handler {
	router := &Server{metadata: metadataClient, version: version}
	r := chi.NewRouter()

	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins: []string{"http://localhost:5173", "http://localhost:3000"},
		AllowedMethods: []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders: []string{"Content-Type", "Authorization"},
	}))

	// Public routes — no auth required
	r.Get("/api/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("KBArr is running"))
	})
	r.Post("/api/auth/login", handlers.HandleLogin(authStore))
	r.Get("/api/images/{imageName}", handlers.HandleGetImage)

	// Protected routes
	r.Group(func(r chi.Router) {
		r.Use(authStore.Middleware)

		r.Get("/api/version", handlers.HandleGetVersion(router.version))

		// Auth
		r.Get("/api/auth/me", handlers.HandleMe(authStore))
		r.Post("/api/auth/logout", handlers.HandleLogout(authStore))
		r.Post("/api/auth/credentials", handlers.HandleChangeCredentials())

		// Settings
		r.Get("/api/settings", handlers.HandleGetSettings)
		r.Post("/api/settings", handlers.HandleUpdateSettings)
		r.Post("/api/settings/test/indexer", handlers.HandleTestIndexer(handlers.SvcAddr("INDEXER_HEALTH_ADDR", "http://localhost:8082")))
		r.Post("/api/settings/test/downloader", handlers.HandleTestDownloader(handlers.SvcAddr("DOWNLOADER_HEALTH_ADDR", "http://localhost:8083")))

		// Search
		r.Get("/api/search", router.handleMediaSearch)
		r.Get("/api/search-queue", handlers.HandleGetSearchQueue)
		r.Delete("/api/search-queue/{id}", handlers.HandleDeleteSearchQueueEntry)

		// Downloads (download_queue table)
		r.Get("/api/downloads", handlers.HandleListDownloads)
		r.Delete("/api/downloads/{id}", handlers.HandleDeleteDownload)

		// Library
		r.Get("/api/library", handlers.HandleGetMediaList)
		r.Post("/api/library", router.handleAddMedia)

		r.Get("/api/library/{id}", handlers.HandleGetDetailedByMediaID)
		r.Get("/api/library/{id}/episodes", handlers.HandleGetEpisodes)
		r.Get("/api/library/{id}/monitored", handlers.HandleGetMonitorsByLibraryID)
		r.Delete("/api/library/{id}", handlers.HandleDeleteMedia)

		r.Put("/api/library/{id}/monitor", handlers.HandleUpdateMonitorStatus)

		// Monitor
		r.Get("/api/monitor", handlers.HandleGetMonitoredList)
		r.Post("/api/monitor", handlers.HandleAddMonitor)
		r.Post("/api/monitor/bulk", handlers.HandleBulkAddMonitor)
		r.Delete("/api/monitor/{id}", handlers.HandleDeleteMonitor)
		r.Post("/api/unmonitor", handlers.HandleUnmonitor)
		r.Post("/api/unmonitor/season", handlers.HandleUnmonitorSeason)

		// Workers / service health
		r.Get("/api/workers", handlers.HandleGetWorkers())
		r.Get("/api/workers/{name}/logs", handlers.HandleGetServiceLogs())

	})

	// Serve embedded frontend on all other routes (catch-all for SPA)
	r.NotFound(staticHandler().ServeHTTP)

	return r
}

func (r *Server) handleMediaSearch(w http.ResponseWriter, req *http.Request) {
	handlers.HandleMediaSearch(w, req, r.metadata)
}

func (r *Server) handleAddMedia(w http.ResponseWriter, req *http.Request) {
	handlers.HandleAddMedia(w, req, r.metadata)
}
