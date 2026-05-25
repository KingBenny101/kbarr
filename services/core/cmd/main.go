package main

import (
	"context"
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
	proto "github.com/kingbenny101/kbarr/shared/proto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func main() {
	logger.Init()
	logger.Log.Infof("[Main] kbarr starting...")
	logger.Log.Infof("[Main] Initializing database...")
	if err := db.Init(); err != nil {
		logger.Log.Fatalf("[Main] Database initialization error %v", err)
		return
	}
	logger.Log.Info("[Main] Initialization complete.")

	// Config
	cfg := config.Load(db.DB)

	appVersion, err := coreversion.Load()
	if err != nil {
		logger.Log.Fatalf("[Main] Failed to load version: %v", err)
		return
	}
	logger.Log.Infof("[Main] kbarr version %s", appVersion)

	anidbConn, err := grpc.NewClient(resolveGRPCAddr("ANIDB_GRPC_ADDR", "localhost:8081"), grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		logger.Log.Fatalf("[Main] Failed to connect to AniDB gRPC service: %v", err)
		return
	}
	defer anidbConn.Close()

	prowlarrConn, err := grpc.NewClient(resolveGRPCAddr("PROWLARR_GRPC_ADDR", "localhost:8082"), grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		logger.Log.Fatalf("[Main] Failed to connect to Prowlarr gRPC service: %v", err)
		return
	}
	defer prowlarrConn.Close()

	downloaderConn, err := grpc.NewClient(resolveGRPCAddr("DOWNLOADER_GRPC_ADDR", "downloader:8083"), grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		logger.Log.Fatalf("[Main] Failed to connect to Downloader gRPC service: %v", err)
		return
	}
	defer downloaderConn.Close()

	// API Router
	router := api.NewRouter(proto.NewAniDBServiceClient(anidbConn), proto.NewProwlarrServiceClient(prowlarrConn), proto.NewDownloaderClient(downloaderConn), appVersion)

	server := &http.Server{
		Addr:    cfg.ServerAddr,
		Handler: router,
	}

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		// Start server
		logger.Log.Infof("[Main] Server running on http://localhost:%s", cfg.ServerPort)
		_ = server.ListenAndServe()
	}()

	<-quit

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	server.Shutdown(ctx)

	logger.Log.Info("[Main] Server stopped")
}

func resolveGRPCAddr(envKey, fallback string) string {
	addr := strings.TrimSpace(os.Getenv(envKey))
	if addr == "" {
		return fallback
	}
	return addr
}
