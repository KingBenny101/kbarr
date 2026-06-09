package main

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/kingbenny101/kbarr/internal/downloader/db"
	"github.com/kingbenny101/kbarr/internal/downloader/service"
	"github.com/kingbenny101/kbarr/internal/config"
	"github.com/kingbenny101/kbarr/internal/logger"
)

func main() {
	logger.Init()
	slog.Info("Starting downloader service")

	if err := db.Init(); err != nil {
		slog.Error("Failed to initialize database", "error", err)
		os.Exit(1)
	}

	svc := service.NewDownloaderService(db.DB)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go svc.PollAndDownload(ctx)
	slog.Info("Downloader polling started")

	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusOK) })
	mux.HandleFunc("GET /logs", logger.HandleLogs)
	mux.HandleFunc("POST /torrent/delete", func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			Hash        string `json:"hash"`
			DeleteFiles bool   `json:"deleteFiles"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.Hash == "" {
			http.Error(w, "hash is required", http.StatusBadRequest)
			return
		}
		qbtURL := strings.TrimRight(config.Get(db.DB, "qbittorrentUrl", "http://localhost:8080"), "/")
		username := config.Get(db.DB, "qbittorrentUsername", "")
		password := config.Get(db.DB, "qbittorrentPassword", "")
		if err := svc.LoginAndDelete(r.Context(), qbtURL, username, password, body.Hash, body.DeleteFiles); err != nil {
			slog.Error("Failed to delete torrent from qBittorrent", "hash", body.Hash, "error", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"ok": "true"})
	})
	mux.HandleFunc("POST /symlinks/create", func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			ID int64 `json:"id"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.ID == 0 {
			http.Error(w, "id is required", http.StatusBadRequest)
			return
		}
		svc.CreateSymlinksForEntry(r.Context(), body.ID)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"ok": "true"})
	})
	mux.HandleFunc("POST /files/delete", func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			SavePath   string `json:"save_path"`
			SymlinkDir string `json:"symlink_dir"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			http.Error(w, "invalid request body", http.StatusBadRequest)
			return
		}
		mediaPath := strings.TrimRight(config.Get(db.DB, "mediaPath", ""), "/")
		if body.SymlinkDir != "" {
			cleaned := filepath.Join(mediaPath, filepath.Base(body.SymlinkDir))
			if err := os.RemoveAll(cleaned); err != nil {
				slog.Warn("files/delete: failed to remove symlink dir", "path", cleaned, "error", err)
			} else {
				slog.Info("files/delete: removed symlink dir", "path", cleaned)
			}
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"ok": "true"})
	})
	mux.HandleFunc("POST /trigger", func(w http.ResponseWriter, r *http.Request) {
		go func() {
			svc.ProcessPending(context.Background())
			svc.UpdateDownloading(context.Background())
		}()
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"ok": "true", "message": "Triggered"})
	})
	mux.HandleFunc("POST /test", func(w http.ResponseWriter, r *http.Request) {
		qbtURL := strings.TrimRight(config.Get(db.DB, "qbittorrentUrl", "http://localhost:8080"), "/")
		username := config.Get(db.DB, "qbittorrentUsername", "")
		password := config.Get(db.DB, "qbittorrentPassword", "")
		w.Header().Set("Content-Type", "application/json")
		client := &http.Client{Timeout: 10 * time.Second}
		if username == "" {
			resp, err := client.Get(qbtURL + "/api/v2/app/version")
			if err != nil {
				json.NewEncoder(w).Encode(map[string]string{"ok": "false", "message": err.Error()})
				return
			}
			resp.Body.Close()
			if resp.StatusCode == http.StatusNotFound {
				json.NewEncoder(w).Encode(map[string]string{"ok": "false", "message": "qBittorrent not found at this URL"})
				return
			}
			if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
				json.NewEncoder(w).Encode(map[string]string{"ok": "false", "message": "qBittorrent requires credentials"})
				return
			}
			json.NewEncoder(w).Encode(map[string]string{"ok": "true", "message": "Connected to qBittorrent"})
			return
		}
		resp, err := client.PostForm(qbtURL+"/api/v2/auth/login", url.Values{
			"username": {username}, "password": {password},
		})
		if err != nil {
			json.NewEncoder(w).Encode(map[string]string{"ok": "false", "message": err.Error()})
			return
		}
		defer resp.Body.Close()
		body, _ := io.ReadAll(resp.Body)
		trimmed := strings.TrimSpace(string(body))
		if resp.StatusCode < 200 || resp.StatusCode >= 300 || trimmed == "Fails." {
			msg := "Authentication failed"
			switch {
			case trimmed == "Fails.":
				msg = "Invalid username or password"
			case resp.StatusCode == http.StatusUnauthorized:
				msg = "Invalid credentials"
			case resp.StatusCode == http.StatusForbidden:
				msg = "Access denied"
			case resp.StatusCode == http.StatusNotFound:
				msg = "qBittorrent not found at this URL"
			case resp.StatusCode == http.StatusServiceUnavailable:
				msg = "qBittorrent is unavailable"
			}
			json.NewEncoder(w).Encode(map[string]string{"ok": "false", "message": msg})
			return
		}
		json.NewEncoder(w).Encode(map[string]string{"ok": "true", "message": "Connected to qBittorrent"})
	})
	go func() {
		slog.Info("Health endpoint listening", "port", "8083")
		if err := http.ListenAndServe(":8083", mux); err != nil {
			slog.Error("Health server failed", "error", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	slog.Info("Shutting down downloader")
	cancel()
}
