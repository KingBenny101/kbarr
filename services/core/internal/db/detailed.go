package db

import (
	"context"
	"fmt"

	dbgen "github.com/kingbenny101/kbarr/services/core/internal/db/generated"
	"github.com/kingbenny101/kbarr/shared/models"
)

func InsertDetailed(d models.Detailed) (int64, error) {
	if err := ensureQueries(); err != nil {
		return 0, err
	}

	ctx := context.Background()
	created, err := Queries.CreateDetailed(ctx, dbgen.CreateDetailedParams{
		Title:           toNullString(d.Title),
		Aid:             toNullInt64FromUint(d.AID),
		LibraryID:       toNullInt64FromUint(d.LibraryID),
		AlternateTitles: toNullString(d.AlternateTitles),
		Description:     toNullString(d.Description),
		ReleaseDate:     toNullString(d.ReleaseDate),
		Genres:          toNullString(d.Genres),
		PosterUrl:       toNullString(d.PosterURL),
		TotalEpisodes:   toNullInt64(int64(d.TotalEpisodes)),
		TotalSeasons:    toNullInt64(int64(d.TotalSeasons)),
	})
	if err != nil {
		return 0, fmt.Errorf("failed to insert detailed media: %w", err)
	}

	for _, episode := range d.Episodes {
		_, err := Queries.CreateEpisode(ctx, dbgen.CreateEpisodeParams{
			DetailedID: toNullInt64(created.ID),
			AnidbID:    toNullString(episode.AniDBID),
			Type:       toNullInt64(int64(episode.Type)),
			EpNo:       toNullString(episode.EpNo),
			Title:      toNullString(episode.Title),
			AirDate:    toNullString(episode.AirDate),
		})
		if err != nil {
			return 0, fmt.Errorf("failed to insert episode for detailed media %d: %w", created.ID, err)
		}
	}

	return created.ID, nil
}

func GetDetailedByLibraryID(libraryID uint) (models.Detailed, error) {
	if err := ensureQueries(); err != nil {
		return models.Detailed{}, err
	}

	ctx := context.Background()
	row, err := Queries.GetDetailedByLibraryID(ctx, toNullInt64FromUint(libraryID))
	if err != nil {
		return models.Detailed{}, fmt.Errorf("failed to get detailed media: %w", err)
	}

	episodes, err := Queries.ListEpisodesByDetailedID(ctx, toNullInt64(row.ID))
	if err != nil {
		return models.Detailed{}, fmt.Errorf("failed to get detailed media episodes: %w", err)
	}

	return toDetailedModel(row, episodes), nil
}

func DeleteDetailedByLibraryID(libraryID uint) error {
	if err := ensureQueries(); err != nil {
		return err
	}

	ctx := context.Background()
	if err := Queries.SoftDeleteDetailedByLibraryID(ctx, toNullInt64FromUint(libraryID)); err != nil {
		return fmt.Errorf("failed to delete detailed media: %w", err)
	}
	return nil
}
