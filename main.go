package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/kingbenny101/kbarr/internal/api"
	"github.com/kingbenny101/kbarr/internal/bootstrap"
	"github.com/kingbenny101/kbarr/internal/config"
	"github.com/kingbenny101/kbarr/internal/logger"
	"github.com/kingbenny101/kbarr/internal/anidb"
)

func main() {
	// Bootstrap
	bootstrap.Init()

	// Config
	cfg := config.Load()

	// API Router
	router := api.NewRouter(cfg)

	// Start AniDB Sync Worker
	logger.Log.Info("[AniDB] Starting AniDB sync worker...")
	stop := make(chan struct{})
	go anidb.Start(cfg, stop)

	server := &http.Server{
		Addr:    cfg.ServerAddr,
		Handler: router,
	}

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		// Start server
		logger.Log.Infof("[Main] Server running on http://localhost:%s", cfg.ServerPort)
		server.ListenAndServe()
	}()

	<-quit

	logger.Log.Info("[Main] Shutting down...")
	close(stop)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	server.Shutdown(ctx)

	logger.Log.Info("[Main] Server stopped")
}