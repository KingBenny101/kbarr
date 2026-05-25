package main

import (
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/kingbenny101/kbarr/services/downloader/internal/db"
	"github.com/kingbenny101/kbarr/services/downloader/internal/service"
	"github.com/kingbenny101/kbarr/shared/config"
	"github.com/kingbenny101/kbarr/shared/logger"
	proto "github.com/kingbenny101/kbarr/shared/proto"
	"google.golang.org/grpc"
)

func main() {
	logger.Init()
	logger.Log.Infof("[Downloader Service] Starting service")

	if err := db.Init(); err != nil {
		logger.Log.Fatalf("[Downloader Service] Failed to initialize database: %v", err)
		return
	}

	if err := config.EnsureDefaults(db.SQLDB); err != nil {
		logger.Log.Fatalf("[Downloader Service] Failed to ensure default settings: %v", err)
		return
	}

	qbtURL := os.Getenv("QBITTORRENT_URL")
	svc := service.NewDownloaderService(qbtURL)

	port := os.Getenv("DOWNLOADER_SERVICE_PORT")
	if port == "" {
		port = "8083"
	}

	lis, err := net.Listen("tcp", ":"+port)
	if err != nil {
		logger.Log.Fatalf("[Downloader Service] Failed to listen on :%s: %v", port, err)
		return
	}

	grpcServer := grpc.NewServer()
	proto.RegisterDownloaderServer(grpcServer, service.NewGrpcServer(svc))

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		logger.Log.Infof("[Downloader Service] Listening on :%s", port)
		if err := grpcServer.Serve(lis); err != nil {
			logger.Log.Fatalf("[Downloader Service] Server failed: %v", err)
		}
	}()

	<-quit
	logger.Log.Info("[Downloader Service] Shutting down")

	shutdownDone := make(chan struct{})
	go func() {
		grpcServer.GracefulStop()
		close(shutdownDone)
	}()

	select {
	case <-shutdownDone:
	case <-time.After(10 * time.Second):
		logger.Log.Warn("[Downloader Service] Graceful shutdown timed out, forcing stop")
		grpcServer.Stop()
	}
}
