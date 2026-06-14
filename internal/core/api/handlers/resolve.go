package handlers

import (
	"context"
	"log/slog"
	"strconv"
	"time"

	"github.com/kingbenny101/kbarr/internal/core/clients"
)

// ResolveAniList resolves an AniList ID to its canonical AniDB ID via the Fribb
// mapping held by the metadata service. AniList includes titles with no AniDB
// entry (e.g. Chinese/Korean content); those resolve to Found=false rather than
// an error so the frontend can route the user to manual search.
func ResolveAniList(mc *clients.MetadataClient) func(context.Context, *ResolveAniListInput) (*ResolveAniListOutput, error) {
	return func(ctx context.Context, input *ResolveAniListInput) (*ResolveAniListOutput, error) {
		ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
		defer cancel()

		out := &ResolveAniListOutput{}
		aid, found, err := mc.ResolveAniListID(ctx, input.ID)
		if err != nil {
			slog.Warn("Failed to resolve AniList ID", "anilist_id", input.ID, "error", err)
			return out, nil
		}

		out.Body.Found = found
		if found {
			out.Body.AID = strconv.FormatUint(uint64(aid), 10)
		}
		return out, nil
	}
}
