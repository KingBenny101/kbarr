package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/kingbenny101/kbarr/services/core/internal/api"
	"github.com/kingbenny101/kbarr/services/core/internal/auth"
	"github.com/kingbenny101/kbarr/services/core/internal/clients"
	"github.com/kingbenny101/kbarr/services/core/internal/db"
	coreversion "github.com/kingbenny101/kbarr/services/core/internal/version"
	"github.com/kingbenny101/kbarr/shared/logger"
)

func main() {
	logger.Init()
	slog.Info("kbarr starting...")
	if err := db.Init(); err != nil {
		slog.Error("Database initialization error", "error", err)
		os.Exit(1)
	}
	slog.Info("Initialization complete.")

	authStore := auth.NewStore()
	if err := auth.EnsureDefaults(db.DB); err != nil {
		slog.Error("Failed to seed auth defaults", "error", err)
		os.Exit(1)
	}

	port := resolveAddr("PORT", "8080")

	appVersion, err := coreversion.Load()
	if err != nil {
		slog.Error("Failed to load version", "error", err)
		os.Exit(1)
	}
	slog.Info("kbarr version", "version", appVersion)

	metadataAddr := resolveAddr("METADATA_ADDR", "http://localhost:8081")
	metadataClient := clients.NewMetadataClient(metadataAddr)

	router := api.NewRouter(metadataClient, appVersion, authStore)

	server := &http.Server{
		Addr:    ":" + port,
		Handler: router,
	}

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		slog.Info("Server running on", "url", "http://localhost:"+port)
		_ = server.ListenAndServe()
	}()

	<-quit

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	server.Shutdown(ctx)

	slog.Info("Server stopped")
}

func resolveAddr(envKey, fallback string) string {
	addr := strings.TrimSpace(os.Getenv(envKey))
	if addr == "" {
		return fallback
	}
	return addr
}
