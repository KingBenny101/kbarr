package handlers

import (
	"context"
	"log/slog"
	"time"

	"github.com/kingbenny101/kbarr/internal/core/clients"
	"github.com/kingbenny101/kbarr/internal/core/db"
	"github.com/kingbenny101/kbarr/internal/models"
)

func Search(mc *clients.MetadataClient) func(context.Context, *SearchInput) (*SearchOutput, error) {
	return func(ctx context.Context, input *SearchInput) (*SearchOutput, error) {
		slog.Info("Search request", "query", input.Q)

		ctx, cancel := context.WithTimeout(ctx, 45*time.Second)
		defer cancel()

		results, err := mc.SearchTitles(ctx, input.Q)
		if err != nil {
			slog.Warn("Failed to search titles via metadata service", "error", err)
			return &SearchOutput{Body: []models.SearchResult{}}, nil
		}

		if len(results) == 0 {
			return &SearchOutput{Body: []models.SearchResult{}}, nil
		}

		mediaList, err := db.GetAllMedia()
		if err != nil {
			slog.Warn("Failed to load media list for search annotations", "error", err)
		} else {
			added := make(map[string]struct{}, len(mediaList))
			for _, m := range mediaList {
				added[m.Source+":"+m.SourceID] = struct{}{}
			}
			for i := range results {
				_, results[i].Added = added[results[i].Source+":"+results[i].SourceID]
			}
		}

		return &SearchOutput{Body: results}, nil
	}
}
