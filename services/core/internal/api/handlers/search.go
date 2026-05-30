package handlers

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"time"

	"github.com/kingbenny101/kbarr/services/core/internal/db"
	metadatapb "github.com/kingbenny101/kbarr/shared/proto/metadata"
)

func HandleMediaSearch(w http.ResponseWriter, r *http.Request, metadataClient metadatapb.MetadataServiceClient) {
	query := r.URL.Query().Get("q")
	if query == "" {
		http.Error(w, "missing query parameter q", http.StatusBadRequest)
		return
	}

	slog.Info("Unified search request", "query", query)

	ctx, cancel := context.WithTimeout(r.Context(), 45*time.Second)
	defer cancel()

	response, err := metadataClient.SearchTitles(ctx, &metadatapb.AniDBSearchTitlesRequest{Query: query})
	if err != nil {
		slog.Warn("Failed to search titles via service", "error", err)
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode([]any{})
		return
	}

	results := response.GetResults()
	if len(results) == 0 {
		results = []*metadatapb.AniDBSearchResult{}
	}

	mediaList, err := db.GetAllMedia()
	if err != nil {
		slog.Warn("Failed to load media list for search annotations", "error", err)
	} else {
		addedMedia := make(map[uint64]struct{}, len(mediaList))
		for _, media := range mediaList {
			addedMedia[uint64(media.AID)] = struct{}{}
		}

		for _, result := range results {
			if result == nil {
				continue
			}
			_, ok := addedMedia[result.GetAid()]
			result.Added = ok
		}
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(results)
}
