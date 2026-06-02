package handlers

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"time"

	"github.com/kingbenny101/kbarr/services/core/internal/clients"
	"github.com/kingbenny101/kbarr/services/core/internal/db"
)

func HandleMediaSearch(w http.ResponseWriter, r *http.Request, metadataClient *clients.MetadataClient) {
	query := r.URL.Query().Get("q")
	if query == "" {
		http.Error(w, "missing query parameter q", http.StatusBadRequest)
		return
	}

	slog.Info("Unified search request", "query", query)

	ctx, cancel := context.WithTimeout(r.Context(), 45*time.Second)
	defer cancel()

	results, err := metadataClient.SearchTitles(ctx, query)
	if err != nil {
		slog.Warn("Failed to search titles via metadata service", "error", err)
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode([]any{})
		return
	}

	if len(results) == 0 {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode([]clients.SearchResult{})
		return
	}

	mediaList, err := db.GetAllMedia()
	if err != nil {
		slog.Warn("Failed to load media list for search annotations", "error", err)
	} else {
		addedMedia := make(map[string]struct{}, len(mediaList))
		for _, media := range mediaList {
			addedMedia[media.Source+":"+media.SourceID] = struct{}{}
		}
		for i := range results {
			_, results[i].Added = addedMedia[results[i].Source+":"+results[i].SourceID]
		}
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(results)
}
