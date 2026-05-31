package handlers

import (
	"encoding/json"
	"net/http"
	"os"
	"time"
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

func HandleGetWorkers() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		services := []struct {
			name        string
			displayName string
			envKey      string
			fallback    string
		}{
			{"metadata", "Metadata", "METADATA_ADDR", "http://localhost:8081"},
			{"indexer", "Indexer", "INDEXER_HEALTH_ADDR", "http://localhost:8082"},
			{"downloader", "Downloader", "DOWNLOADER_HEALTH_ADDR", "http://localhost:8083"},
		}

		out := make([]ServiceHealth, 0, len(services))
		for _, svc := range services {
			addr := os.Getenv(svc.envKey)
			if addr == "" {
				addr = svc.fallback
			}
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
