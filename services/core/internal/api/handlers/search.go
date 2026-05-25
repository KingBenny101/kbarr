package handlers

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"time"

	proto "github.com/kingbenny101/kbarr/shared/proto"
)

func HandleMediaSearch(w http.ResponseWriter, r *http.Request, anidbClient proto.AniDBServiceClient) {
	query := r.URL.Query().Get("q")
	if query == "" {
		http.Error(w, "missing query parameter q", http.StatusBadRequest)
		return
	}

	slog.Info("Unified search request", "query", query)

	ctx, cancel := context.WithTimeout(r.Context(), 45*time.Second)
	defer cancel()

	response, err := anidbClient.SearchTitles(ctx, &proto.AniDBSearchTitlesRequest{Query: query})
	if err != nil {
		slog.Warn("Failed to search titles via service", "error", err)
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode([]any{})
		return
	}

	results := response.GetResults()
	if len(results) == 0 {
		results = []*proto.AniDBSearchResult{}
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(results)
}
