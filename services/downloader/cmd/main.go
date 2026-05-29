package main

import (
	"log/slog"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/kingbenny101/kbarr/services/downloader/internal/db"
	"github.com/kingbenny101/kbarr/services/downloader/internal/service"
	"github.com/kingbenny101/kbarr/shared/logger"
	proto "github.com/kingbenny101/kbarr/shared/proto"
	"google.golang.org/grpc"
)

func main() {
	logger.Init()
	slog.Info("Starting service")

	if err := db.Init(); err != nil {
		slog.Error("Failed to initialize database", "error", err)
		os.Exit(1)
	}

	qbtURL := os.Getenv("QBITTORRENT_URL")
	svc := service.NewDownloaderService(qbtURL)

	port := os.Getenv("DOWNLOADER_SERVICE_PORT")
	if port == "" {
		port = "8083"
	}

	lis, err := net.Listen("tcp", ":"+port)
	if err != nil {
		slog.Error("Failed to listen on", "port", port, "error", err)
		os.Exit(1)
	}

	grpcServer := grpc.NewServer()
	proto.RegisterDownloaderServer(grpcServer, service.NewGrpcServer(svc))

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		slog.Info("Listening on", "port", port)
		if err := grpcServer.Serve(lis); err != nil {
			slog.Error("Server failed", "error", err)
			os.Exit(1)
		}
	}()

	<-quit
	slog.Info("Shutting down")

	shutdownDone := make(chan struct{})
	go func() {
		grpcServer.GracefulStop()
		close(shutdownDone)
	}()

	select {
	case <-shutdownDone:
	case <-time.After(10 * time.Second):
		slog.Warn("Graceful shutdown timed out, forcing stop")
		grpcServer.Stop()
	}
}
