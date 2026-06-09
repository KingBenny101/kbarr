package handlers

import (
	"encoding/json"
	"net/http"
)

func HandleGetVersion(version string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]string{
			"version": version,
		})
	}
}
