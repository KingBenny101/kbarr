package handlers

import (
	"encoding/json"
	"io"
	"net/http"
	"os"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/kingbenny101/kbarr/shared/logger"
)

type ServiceHealth struct {
	Name        string `json:"name"`
	DisplayName string `json:"display_name"`
	Running     bool   `json:"running"`
	Error       string `json:"error,omitempty"`
}

var httpProbe = &http.Client{Timeout: 2 * time.Second}

func probe(url string) (bool, string) {
	resp, err := httpProbe.Get(url)
	if err != nil {
		return false, err.Error()
	}
	resp.Body.Close()
	return resp.StatusCode == http.StatusOK, ""
}

var sidecarServices = []struct {
	name        string
	displayName string
	envKey      string
	fallback    string
}{
	{"metadata", "Metadata", "METADATA_ADDR", "http://localhost:8081"},
	{"indexer", "Indexer", "INDEXER_HEALTH_ADDR", "http://localhost:8082"},
	{"downloader", "Downloader", "DOWNLOADER_HEALTH_ADDR", "http://localhost:8083"},
}

func svcAddr(envKey, fallback string) string {
	if v := os.Getenv(envKey); v != "" {
		return v
	}
	return fallback
}

func HandleGetWorkers() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		out := []ServiceHealth{
			{Name: "core", DisplayName: "Core", Running: true},
		}
		for _, svc := range sidecarServices {
			addr := svcAddr(svc.envKey, svc.fallback)
			running, errMsg := probe(addr + "/health")
			out = append(out, ServiceHealth{
				Name:        svc.name,
				DisplayName: svc.displayName,
				Running:     running,
				Error:       errMsg,
			})
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(out)
	}
}

func HandleGetServiceLogs() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		name := chi.URLParam(r, "name")

		if name == "core" {
			w.Header().Set("Content-Type", "application/json")
			logger.HandleLogs(w, r)
			return
		}

		for _, svc := range sidecarServices {
			if svc.name == name {
				addr := svcAddr(svc.envKey, svc.fallback)
				resp, err := httpProbe.Get(addr + "/logs")
				if err != nil {
					http.Error(w, err.Error(), http.StatusBadGateway)
					return
				}
				defer resp.Body.Close()
				w.Header().Set("Content-Type", "application/json")
				io.Copy(w, resp.Body)
				return
			}
		}

		http.NotFound(w, r)
	}
}
