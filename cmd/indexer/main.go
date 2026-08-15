package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/kingbenny101/kbarr/internal/config"
	"github.com/kingbenny101/kbarr/internal/indexer/db"
	_ "github.com/kingbenny101/kbarr/internal/indexer/provider/kbdex"
	_ "github.com/kingbenny101/kbarr/internal/indexer/provider/prowlarr"
	"github.com/kingbenny101/kbarr/internal/indexer/service"
	"github.com/kingbenny101/kbarr/internal/logger"
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
	testProwlarr := func(w http.ResponseWriter, r *http.Request) {
		prowlarrURL := config.Get(db.DB, "prowlarrUrl", "http://localhost:9696")
		apiKey := config.Get(db.DB, "prowlarrApiKey", "")
		w.Header().Set("Content-Type", "application/json")
		req, err := http.NewRequestWithContext(r.Context(), http.MethodGet,
			strings.TrimRight(prowlarrURL, "/")+"/api/v1/system/status", nil)
		if err != nil {
			json.NewEncoder(w).Encode(map[string]string{"ok": "false", "message": err.Error()})
			return
		}
		req.Header.Set("X-Api-Key", apiKey)
		client := &http.Client{Timeout: 10 * time.Second}
		resp, err := client.Do(req)
		if err != nil {
			json.NewEncoder(w).Encode(map[string]string{"ok": "false", "message": err.Error()})
			return
		}
		resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			msg := fmt.Sprintf("HTTP %d", resp.StatusCode)
			switch resp.StatusCode {
			case http.StatusUnauthorized:
				msg = "Invalid API key"
			case http.StatusForbidden:
				msg = "Access denied"
			case http.StatusNotFound:
				msg = "Prowlarr not found at this URL"
			case http.StatusServiceUnavailable:
				msg = "Prowlarr is unavailable"
			}
			json.NewEncoder(w).Encode(map[string]string{"ok": "false", "message": msg})
			return
		}
		json.NewEncoder(w).Encode(map[string]string{"ok": "true", "message": "Connected to Prowlarr"})
	}

	testKbdex := func(w http.ResponseWriter, r *http.Request) {
		kbdexURL := config.Get(db.DB, "kbdexUrl", "http://localhost:8000")
		w.Header().Set("Content-Type", "application/json")
		req, err := http.NewRequestWithContext(r.Context(), http.MethodGet,
			strings.TrimRight(kbdexURL, "/")+"/health", nil)
		if err != nil {
			json.NewEncoder(w).Encode(map[string]string{"ok": "false", "message": err.Error()})
			return
		}
		client := &http.Client{Timeout: 10 * time.Second}
		resp, err := client.Do(req)
		if err != nil {
			json.NewEncoder(w).Encode(map[string]string{"ok": "false", "message": err.Error()})
			return
		}
		resp.Body.Close()
		if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusServiceUnavailable {
			json.NewEncoder(w).Encode(map[string]string{"ok": "false", "message": fmt.Sprintf("HTTP %d", resp.StatusCode)})
			return
		}
		json.NewEncoder(w).Encode(map[string]string{"ok": "true", "message": "Connected to kbdex"})
	}

	mux.HandleFunc("POST /test/prowlarr", testProwlarr)
	mux.HandleFunc("POST /test/kbdex", testKbdex)
	mux.HandleFunc("POST /trigger", func(w http.ResponseWriter, _ *http.Request) {
		slog.Info("Trigger received")
		svc.Trigger()
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"ok": "true"})
	})
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
