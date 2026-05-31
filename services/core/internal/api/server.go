package api

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/kingbenny101/kbarr/services/core/internal/api/handlers"
	"github.com/kingbenny101/kbarr/services/core/internal/clients"
)

type Server struct {
	metadata *clients.MetadataClient
	version  string
}

func NewRouter(metadataClient *clients.MetadataClient, version string) http.Handler {
	router := &Server{metadata: metadataClient, version: version}
	r := chi.NewRouter()

	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins: []string{"http://localhost:5173", "http://localhost:3000"},
		AllowedMethods: []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders: []string{"Content-Type"},
	}))

	r.Get("/api/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("KBArr is running"))
	})
	r.Get("/api/version", handlers.HandleGetVersion(router.version))

	// Settings
	r.Get("/api/settings", handlers.HandleGetSettings)
	r.Post("/api/settings", handlers.HandleUpdateSettings)

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

	// Poster/image cache served from the shared data volume.
	r.Get("/api/images/{imageName}", handlers.HandleGetImage)

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
