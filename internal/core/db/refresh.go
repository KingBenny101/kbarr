package db

import (
	"context"
	"fmt"
	"strconv"

	"github.com/uptrace/bun"

	"github.com/kingbenny101/kbarr/internal/models"
)

// RefreshTarget is one library row the metadata refresh worker may re-fetch.
type RefreshTarget struct {
	LibraryID uint   `bun:"library_id"`
	Source    string `bun:"source"`
	SourceID  string `bun:"source_id"`
	Title     string `bun:"title"`
	EndDate   string `bun:"end_date"`
}

// OngoingRefreshTargets returns the detailed rows whose show has not finished
// airing — end_date is empty (AniDB leaves it blank while a show airs) — so the
// worker only ever re-fetches shows that can still gain episodes. Finished shows
// are never touched, which is what keeps refresh off AniDB's rate limit.
func OngoingRefreshTargets() ([]RefreshTarget, error) {
	if err := ensureDB(); err != nil {
		return nil, err
	}
	var targets []RefreshTarget
	err := DB.NewSelect().
		TableExpr("detaileds d").
		ColumnExpr("d.library_id, d.source, d.source_id, d.title, d.end_date").
		Join("JOIN media m ON m.id = d.library_id").
		Where("m.deleted_at IS NULL AND d.deleted_at IS NULL").
		Where("COALESCE(d.end_date, '') = ''").
		Scan(context.Background(), &targets)
	if err != nil {
		return nil, fmt.Errorf("failed to load refresh targets: %w", err)
	}
	return targets, nil
}

// ApplyRefresh upserts freshly-fetched metadata for one library: it updates the
// detailed row's mutable fields and inserts any episodes (and, for regular
// episodes, monitors) that did not exist before. Existing episode and monitor
// rows are never modified, so download state and user choices are preserved —
// the operation is purely additive. New episode monitors inherit the season's
// monitored flag, so a newly-aired episode auto-downloads exactly when the user
// is already monitoring that season. Returns the number of new episode monitors.
func ApplyRefresh(libraryID uint, prepared *models.AnimeMetadata) (newEpisodes int, err error) {
	if err := ensureDB(); err != nil {
		return 0, err
	}
	if prepared == nil {
		return 0, fmt.Errorf("no metadata to apply")
	}
	ctx := context.Background()

	err = DB.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		var detailed models.Detailed
		if err := tx.NewSelect().Model(&detailed).
			Where("library_id = ? AND deleted_at IS NULL", libraryID).
			Limit(1).Scan(ctx); err != nil {
			return fmt.Errorf("detailed row not found: %w", err)
		}

		// Refresh the fields that change over a show's life, plus genres and the
		// cross-database IDs so they backfill on existing entries (the IDs are stable,
		// but rows added before they were tracked still need them filled in). Title is
		// left alone — the folder/library identity is built from it.
		if _, err := tx.NewUpdate().Model((*models.Detailed)(nil)).
			Set("description = ?, release_date = ?, end_date = ?, total_episodes = ?, poster_url = ?, alternate_titles = ?, genres = ?, tvdb_id = ?, anilist_id = ?, imdb_id = ?, tmdb_id = ?, mal_id = ?, kitsu_id = ?, animeplanet_id = ?, anisearch_id = ?, updated_at = now()",
				prepared.Description, prepared.ReleaseDate, prepared.EndDate, prepared.TotalEpisodes, prepared.PosterURL, prepared.AlternateTitles, prepared.Genres,
				prepared.TVDBID, prepared.AniListID, prepared.IMDBID, prepared.TMDBID, prepared.MALID, prepared.KitsuID, prepared.AnimePlanetID, prepared.AniSearchID).
			Where("id = ?", detailed.ID).Exec(ctx); err != nil {
			return fmt.Errorf("failed to update detailed: %w", err)
		}

		// Existing episodes, keyed by external id, so we only insert genuinely new ones.
		var existing []models.Episode
		if err := tx.NewSelect().Model(&existing).
			Where("detailed_id = ?", detailed.ID).Scan(ctx); err != nil {
			return fmt.Errorf("failed to load existing episodes: %w", err)
		}
		haveEpisode := map[string]bool{}
		for _, e := range existing {
			haveEpisode[e.ExternalID] = true
		}

		// Existing episode monitors, keyed by episode number.
		var mons []models.Monitor
		if err := tx.NewSelect().Model(&mons).
			Where("library_id = ? AND is_episode = true AND deleted_at IS NULL", libraryID).Scan(ctx); err != nil {
			return fmt.Errorf("failed to load existing monitors: %w", err)
		}
		haveMonitor := map[int]bool{}
		for _, m := range mons {
			haveMonitor[m.EpisodeNumber] = true
		}

		// Does the user monitor this show at the season level? New episodes inherit it.
		seasonMonitored, err := tx.NewSelect().Model((*models.Monitor)(nil)).
			Where("library_id = ? AND is_season = true AND monitored = true AND deleted_at IS NULL", libraryID).
			Exists(ctx)
		if err != nil {
			return fmt.Errorf("failed to check season monitor: %w", err)
		}

		for _, ep := range prepared.Episodes {
			if !haveEpisode[ep.ExternalID] {
				newEp := models.Episode{
					DetailedID: detailed.ID,
					Source:     ep.Source,
					ExternalID: ep.ExternalID,
					Type:       ep.Type,
					EpNo:       ep.Number,
					Title:      ep.Title,
					AirDate:    ep.AirDate,
				}
				if _, err := tx.NewInsert().Model(&newEp).Exec(ctx); err != nil {
					return fmt.Errorf("failed to insert episode: %w", err)
				}
			}

			// Only regular episodes (AniDB EpNo type 1) get monitors, matching AddMedia.
			if ep.Type != 1 {
				continue
			}
			epNum, convErr := strconv.Atoi(ep.Number)
			if convErr != nil || haveMonitor[epNum] {
				continue
			}
			mon := models.Monitor{
				LibraryID:     libraryID,
				Title:         prepared.Title,
				EpisodeTitle:  ep.Title,
				Season:        1,
				EpisodeNumber: epNum,
				IsEpisode:     true,
				Source:        ep.Source,
				ExternalID:    ep.ExternalID,
				Status:        "pending",
				Monitored:     seasonMonitored,
			}
			if _, err := tx.NewInsert().Model(&mon).Exec(ctx); err != nil {
				return fmt.Errorf("failed to insert monitor: %w", err)
			}
			haveMonitor[epNum] = true
			newEpisodes++
		}
		return nil
	})
	if err != nil {
		return 0, err
	}
	return newEpisodes, nil
}
