package main

import (
	"log/slog"
	"net"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/kingbenny101/kbarr/services/metadata/internal/db"
	"github.com/kingbenny101/kbarr/services/metadata/internal/service"
	"github.com/kingbenny101/kbarr/shared/logger"
	metadatapb "github.com/kingbenny101/kbarr/shared/proto/metadata"
	"google.golang.org/grpc"
)

func main() {
	logger.Init()
	slog.Info("Starting service")

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

	lis, err := net.Listen("tcp", ":"+port)
	if err != nil {
		slog.Error("Failed to listen on", "port", port, "error", err)
		os.Exit(1)
	}

	grpcServer := grpc.NewServer()
	metadatapb.RegisterMetadataServiceServer(grpcServer, service.NewGRPCServer(svc))

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
	close(titlesSyncStop)

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

func ensureDataDirs() error {
	dataDir := service.DataRootDir()
	dirs := []string{
		dataDir,
		filepath.Join(dataDir, "images"),
		filepath.Join(dataDir, "details"),
	}

	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return err
		}
	}

	return nil
}
