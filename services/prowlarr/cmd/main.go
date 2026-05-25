package main

import (
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/kingbenny101/kbarr/services/prowlarr/internal/db"
	"github.com/kingbenny101/kbarr/services/prowlarr/internal/service"
	"github.com/kingbenny101/kbarr/shared/config"
	"github.com/kingbenny101/kbarr/shared/logger"
	proto "github.com/kingbenny101/kbarr/shared/proto"
	"google.golang.org/grpc"
)

func main() {
	logger.Init()
	logger.Log.Infof("[Prowlarr Service] Starting service")

	if err := db.Init(); err != nil {
		logger.Log.Fatalf("[Prowlarr Service] Failed to initialize database: %v", err)
		return
	}

	if err := config.EnsureDefaults(db.SQLDB); err != nil {
		logger.Log.Fatalf("[Prowlarr Service] Failed to ensure default settings: %v", err)
		return
	}

	svc := service.New(db.SQLDB)

	port := os.Getenv("PROWLARR_SERVICE_PORT")
	if port == "" {
		port = "8082"
	}

	lis, err := net.Listen("tcp", ":"+port)
	if err != nil {
		logger.Log.Fatalf("[Prowlarr Service] Failed to listen on :%s: %v", port, err)
		return
	}

	grpcServer := grpc.NewServer()
	proto.RegisterProwlarrServiceServer(grpcServer, service.NewGRPCServer(svc))

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		logger.Log.Infof("[Prowlarr Service] Listening on :%s", port)
		if err := grpcServer.Serve(lis); err != nil {
			logger.Log.Fatalf("[Prowlarr Service] Server failed: %v", err)
		}
	}()

	<-quit
	logger.Log.Info("[Prowlarr Service] Shutting down")

	shutdownDone := make(chan struct{})
	go func() {
		grpcServer.GracefulStop()
		close(shutdownDone)
	}()

	select {
	case <-shutdownDone:
	case <-time.After(10 * time.Second):
		logger.Log.Warn("[Prowlarr Service] Graceful shutdown timed out, forcing stop")
		grpcServer.Stop()
	}
}
