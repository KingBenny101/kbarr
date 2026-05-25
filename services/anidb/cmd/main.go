package main

import (
	"net"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/kingbenny101/kbarr/services/anidb/internal/db"
	"github.com/kingbenny101/kbarr/services/anidb/internal/service"
	"github.com/kingbenny101/kbarr/shared/config"
	"github.com/kingbenny101/kbarr/shared/logger"
	proto "github.com/kingbenny101/kbarr/shared/proto"
	"google.golang.org/grpc"
)

func main() {
	logger.Init()
	logger.Log.Infof("[AniDB Service] Starting service")

	if err := ensureDataDirs(); err != nil {
		logger.Log.Fatalf("[AniDB Service] Failed to prepare data directories: %v", err)
		return
	}

	if err := db.Init(); err != nil {
		logger.Log.Fatalf("[AniDB Service] Failed to initialize database: %v", err)
		return
	}

	if err := config.EnsureDefaults(db.SQLDB); err != nil {
		logger.Log.Fatalf("[AniDB Service] Failed to ensure default settings: %v", err)
		return
	}

	svc := service.New(db.SQLDB)
	titlesSyncStop := make(chan struct{})
	go svc.StartTitlesSync(titlesSyncStop)

	port := os.Getenv("ANIDB_SERVICE_PORT")
	if port == "" {
		port = "8081"
	}

	lis, err := net.Listen("tcp", ":"+port)
	if err != nil {
		logger.Log.Fatalf("[AniDB Service] Failed to listen on :%s: %v", port, err)
		return
	}

	grpcServer := grpc.NewServer()
	proto.RegisterAniDBServiceServer(grpcServer, service.NewGRPCServer(svc))

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		logger.Log.Infof("[AniDB Service] Listening on :%s", port)
		if err := grpcServer.Serve(lis); err != nil {
			logger.Log.Fatalf("[AniDB Service] Server failed: %v", err)
		}
	}()

	<-quit
	logger.Log.Info("[AniDB Service] Shutting down")
	close(titlesSyncStop)

	shutdownDone := make(chan struct{})
	go func() {
		grpcServer.GracefulStop()
		close(shutdownDone)
	}()

	select {
	case <-shutdownDone:
	case <-time.After(10 * time.Second):
		logger.Log.Warn("[AniDB Service] Graceful shutdown timed out, forcing stop")
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
