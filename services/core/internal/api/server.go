package api

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/kingbenny101/kbarr/services/core/internal/api/handlers"
	downloaderpb "github.com/kingbenny101/kbarr/shared/proto/downloader"
	indexerpb "github.com/kingbenny101/kbarr/shared/proto/indexer"
	metadatapb "github.com/kingbenny101/kbarr/shared/proto/metadata"
)

type Server struct {
	anidb          metadatapb.MetadataServiceClient
	prowlarr       indexerpb.IndexerServiceClient
	downloader     downloaderpb.DownloaderClient
	version        string
	metadataAddr   string
	indexerAddr    string
	downloaderAddr string
}

func NewRouter(metadataClient metadatapb.MetadataServiceClient, indexerClient indexerpb.IndexerServiceClient, downloaderClient downloaderpb.DownloaderClient, version string, metadataAddr, indexerAddr, downloaderAddr string) http.Handler {
	router := &Server{anidb: metadataClient, prowlarr: indexerClient, downloader: downloaderClient, version: version, metadataAddr: metadataAddr, indexerAddr: indexerAddr, downloaderAddr: downloaderAddr}
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

	// Downloads
	r.Post("/api/downloads", router.handleAddTorrent)
	r.Get("/api/downloads", router.handleListTorrents)
	r.Get("/api/downloads/{hash}", router.handleGetTorrent)
	r.Delete("/api/downloads/{hash}", router.handleRemoveTorrent)

	// Library
	r.Get("/api/library", handlers.HandleGetMediaList)
	r.Post("/api/library", router.handleAddMedia)

	r.Get("/api/library/{id}", handlers.HandleGetDetailedByMediaID)
	r.Get("/api/library/{id}/monitored", handlers.HandleGetMonitorsByLibraryID)
	r.Delete("/api/library/{id}", handlers.HandleDeleteMedia)

	r.Put("/api/library/{id}/monitor", handlers.HandleUpdateMonitorStatus)
	r.Post("/api/library/{id}/search", router.handleTriggerSearch)

	// Monitor
	r.Get("/api/monitor", handlers.HandleGetMonitoredList)
	r.Post("/api/monitor", handlers.HandleAddMonitor)
	r.Post("/api/monitor/bulk", handlers.HandleBulkAddMonitor)
	r.Delete("/api/monitor/{id}", handlers.HandleDeleteMonitor)
	r.Post("/api/unmonitor", handlers.HandleUnmonitor)
	r.Post("/api/unmonitor/season", handlers.HandleUnmonitorSeason)

	// Compatibility endpoint for legacy UI; core no longer runs workers.
	r.Get("/api/workers", handlers.HandleGetWorkers(router.metadataAddr, router.indexerAddr, router.downloaderAddr))

	// Poster/image cache served from the shared data volume.
	r.Get("/api/images/{imageName}", handlers.HandleGetImage)

	// Serve embedded frontend on all other routes (catch-all for SPA)
	r.NotFound(staticHandler().ServeHTTP)

	return r
}

func (r *Server) handleMediaSearch(w http.ResponseWriter, req *http.Request) {
	handlers.HandleMediaSearch(w, req, r.anidb)
}

func (r *Server) handleAddMedia(w http.ResponseWriter, req *http.Request) {
	handlers.HandleAddMedia(w, req, r.anidb)
}

func (r *Server) handleTriggerSearch(w http.ResponseWriter, req *http.Request) {
	handlers.HandleTriggerSearch(w, req, r.prowlarr)
}

func (r *Server) handleAddTorrent(w http.ResponseWriter, req *http.Request) {
	handlers.HandleAddTorrent(w, req, r.downloader)
}

func (r *Server) handleListTorrents(w http.ResponseWriter, req *http.Request) {
	handlers.HandleListTorrents(w, req, r.downloader)
}

func (r *Server) handleGetTorrent(w http.ResponseWriter, req *http.Request) {
	handlers.HandleGetTorrent(w, req, r.downloader)
}

func (r *Server) handleRemoveTorrent(w http.ResponseWriter, req *http.Request) {
	handlers.HandleRemoveTorrent(w, req, r.downloader)
}
