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
	"github.com/kingbenny101/kbarr/services/core/internal/db"
	coreversion "github.com/kingbenny101/kbarr/services/core/internal/version"
	"github.com/kingbenny101/kbarr/shared/config"
	"github.com/kingbenny101/kbarr/shared/logger"
	downloaderpb "github.com/kingbenny101/kbarr/shared/proto/downloader"
	indexerpb "github.com/kingbenny101/kbarr/shared/proto/indexer"
	metadatapb "github.com/kingbenny101/kbarr/shared/proto/metadata"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func main() {
	logger.Init()
	slog.Info("kbarr starting...")
	slog.Info("Initializing database...")
	if err := db.Init(); err != nil {
		slog.Error("Database initialization error", "error", err)
		os.Exit(1)
	}
	slog.Info("Initialization complete.")

	// Config
	cfg := config.Load(db.DB)

	appVersion, err := coreversion.Load()
	if err != nil {
		slog.Error("Failed to load version", "error", err)
		os.Exit(1)
	}
	slog.Info("kbarr version", "version", appVersion)

	metadataAddr := resolveGRPCAddr("METADATA_GRPC_ADDR", "localhost:8081")
	indexerAddr := resolveGRPCAddr("INDEXER_GRPC_ADDR", "localhost:8082")
	downloaderAddr := resolveGRPCAddr("DOWNLOADER_GRPC_ADDR", "downloader:8083")

	metadataConn, err := grpc.NewClient(metadataAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		slog.Error("Failed to connect to AniDB gRPC service", "error", err)
		os.Exit(1)
	}
	defer metadataConn.Close()

	indexerConn, err := grpc.NewClient(indexerAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		slog.Error("Failed to connect to Prowlarr gRPC service", "error", err)
		os.Exit(1)
	}
	defer indexerConn.Close()

	downloaderConn, err := grpc.NewClient(downloaderAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		slog.Error("Failed to connect to Downloader gRPC service", "error", err)
		os.Exit(1)
	}
	defer downloaderConn.Close()

	// API Router
	router := api.NewRouter(metadatapb.NewMetadataServiceClient(metadataConn), indexerpb.NewIndexerServiceClient(indexerConn), downloaderpb.NewDownloaderClient(downloaderConn), appVersion, metadataAddr, indexerAddr, downloaderAddr)

	server := &http.Server{
		Addr:    cfg.ServerAddr,
		Handler: router,
	}

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		// Start server
		slog.Info("Server running on", "url", "http://localhost:"+cfg.ServerPort)
		_ = server.ListenAndServe()
	}()

	<-quit

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	server.Shutdown(ctx)

	slog.Info("Server stopped")
}

func resolveGRPCAddr(envKey, fallback string) string {
	addr := strings.TrimSpace(os.Getenv(envKey))
	if addr == "" {
		return fallback
	}
	return addr
}
