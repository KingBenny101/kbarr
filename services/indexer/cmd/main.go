package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/kingbenny101/kbarr/services/indexer/internal/db"
	"github.com/kingbenny101/kbarr/services/indexer/internal/service"
	"github.com/kingbenny101/kbarr/shared/logger"
)

func main() {
	logger.Init()
	slog.Info("Starting indexer service")

	if err := db.Init(); err != nil {
		slog.Error("Failed to initialize database", "error", err)
		os.Exit(1)
	}

	svc := service.New(db.DB)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go svc.PollAndQueue(ctx)
	slog.Info("Indexer polling started")

	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusOK) })
	mux.HandleFunc("GET /logs", logger.HandleLogs)
	go func() {
		slog.Info("Health endpoint listening", "port", "8082")
		if err := http.ListenAndServe(":8082", mux); err != nil {
			slog.Error("Health server failed", "error", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	slog.Info("Shutting down indexer")
	cancel()
}
