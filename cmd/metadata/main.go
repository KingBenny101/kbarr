package main

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	metaapi "github.com/kingbenny101/kbarr/internal/metadata/api"
	"github.com/kingbenny101/kbarr/internal/metadata/db"
	"github.com/kingbenny101/kbarr/internal/metadata/service"
	"github.com/kingbenny101/kbarr/internal/logger"
)

func main() {
	logger.Init()
	slog.Info("Starting metadata service")

	if err := ensureDataDirs(); err != nil {
		slog.Error("Failed to prepare data directories", "error", err)
		os.Exit(1)
	}

	if err := db.Init(); err != nil {
		slog.Error("Failed to initialize database", "error", err)
		os.Exit(1)
	}

	svc := service.New(db.DB)
	titlesSyncStop := make(chan struct{})
	go svc.StartTitlesSync(titlesSyncStop)

	port := os.Getenv("ANIDB_SERVICE_PORT")
	if port == "" {
		port = "8081"
	}

	handler := metaapi.NewHandler(svc)
	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusOK) })
	mux.HandleFunc("GET /logs", logger.HandleLogs)
	mux.HandleFunc("GET /search", handler.Search)
	mux.HandleFunc("GET /anime/{aid}", handler.GetAnimeDetails)
	mux.HandleFunc("GET /resolve/anilist/{anilistID}", handler.ResolveAniList)
	mux.HandleFunc("POST /prepare", handler.Prepare)
	mux.HandleFunc("POST /trigger", func(w http.ResponseWriter, _ *http.Request) {
		slog.Info("Trigger received")
		svc.Trigger()
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"ok": "true"})
	})

	server := &http.Server{
		Addr:    ":" + port,
		Handler: mux,
	}

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		slog.Info("Listening on", "port", port)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("Server failed", "error", err)
			os.Exit(1)
		}
	}()

	<-quit
	slog.Info("Shutting down metadata service")
	close(titlesSyncStop)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	_ = server.Shutdown(ctx)
}

func ensureDataDirs() error {
	dataDir := service.DataRootDir()
	dirs := []string{
		dataDir,
		filepath.Join(dataDir, "metadata"),
		filepath.Join(dataDir, "metadata", "images"),
		filepath.Join(dataDir, "metadata", "details"),
	}

	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return err
		}
	}

	return nil
}
