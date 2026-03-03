package api

import (
	"encoding/json"
	"net/http"

	"github.com/kingbenny101/kbarr/internal/version"
)


func handleGetVersion(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"version": version.Version,
		"commit":  version.GitCommit,
		"built":   version.BuildTime,
	})
}