package db

import (
	"context"
	"fmt"

	"github.com/kingbenny101/kbarr/shared/models"
)

func InsertDetailed(d models.Detailed) (int64, error) {
	if err := ensureDB(); err != nil {
		return 0, err
	}

	ctx := context.Background()
	created, err := createDetailed(
		ctx,
		stringPtr(d.Title),
		int64Ptr(int64(d.AID)),
		int64Ptr(int64(d.LibraryID)),
		stringPtr(d.AlternateTitles),
		stringPtr(d.Description),
		stringPtr(d.ReleaseDate),
		stringPtr(d.Genres),
		stringPtr(d.PosterURL),
		int64Ptr(int64(d.TotalEpisodes)),
		int64Ptr(int64(d.TotalSeasons)),
	)
	if err != nil {
		return 0, fmt.Errorf("failed to insert detailed media: %w", err)
	}

	for _, episode := range d.Episodes {
		_, err := createEpisode(ctx, int64Ptr(created.ID), stringPtr(episode.AniDBID), int64Ptr(int64(episode.Type)), stringPtr(episode.EpNo), stringPtr(episode.Title), stringPtr(episode.AirDate))
		if err != nil {
			return 0, fmt.Errorf("failed to insert episode for detailed media %d: %w", created.ID, err)
		}
	}

	return created.ID, nil
}

func GetDetailedByLibraryID(libraryID uint) (models.Detailed, error) {
	if err := ensureDB(); err != nil {
		return models.Detailed{}, err
	}

	ctx := context.Background()
	row, err := getDetailedByLibraryID(ctx, int64Ptr(int64(libraryID)))
	if err != nil {
		return models.Detailed{}, fmt.Errorf("failed to get detailed media: %w", err)
	}

	episodes := listEpisodesByDetailedID(ctx, int64Ptr(row.ID))

	return detailedToModel(*row, episodes), nil
}

func DeleteDetailedByLibraryID(libraryID uint) error {
	if err := ensureDB(); err != nil {
		return err
	}

	ctx := context.Background()
	if err := softDeleteDetailedByLibraryID(ctx, int64Ptr(int64(libraryID))); err != nil {
		return fmt.Errorf("failed to delete detailed media: %w", err)
	}
	return nil
}

func getDetailedByLibraryID(ctx context.Context, libraryID *int64) (*Detailed, error) {
	item := &Detailed{}
	if err := DB.NewSelect().Model(item).Where("library_id IS NOT DISTINCT FROM ?", libraryID).Scan(ctx); err != nil {
		return nil, err
	}
	return item, nil
}

func createDetailed(ctx context.Context, title *string, aid *int64, libraryID *int64, alternateTitles *string, description *string, releaseDate *string, genres *string, posterUrl *string, totalEpisodes *int64, totalSeasons *int64) (*Detailed, error) {
	item := &Detailed{
		Title:           title,
		Aid:             aid,
		LibraryID:       libraryID,
		AlternateTitles: alternateTitles,
		Description:     description,
		ReleaseDate:     releaseDate,
		Genres:          genres,
		PosterUrl:       posterUrl,
		TotalEpisodes:   totalEpisodes,
		TotalSeasons:    totalSeasons,
	}
	if _, err := DB.NewInsert().Model(item).Returning("*").Exec(ctx); err != nil {
		return nil, err
	}
	return item, nil
}

func createEpisode(ctx context.Context, detailedID *int64, anidbID *string, typeVal *int64, epNo *string, title *string, airDate *string) (*Episode, error) {
	item := &Episode{
		DetailedID: detailedID,
		AnidbID:    anidbID,
		Type:       typeVal,
		EpNo:       epNo,
		Title:      title,
		AirDate:    airDate,
	}
	if _, err := DB.NewInsert().Model(item).Returning("*").Exec(ctx); err != nil {
		return nil, err
	}
	return item, nil
}

func listEpisodesByDetailedID(ctx context.Context, detailedID *int64) []Episode {
	var items []Episode
	if err := DB.NewSelect().Model(&items).Where("detailed_id IS NOT DISTINCT FROM ?", detailedID).Scan(ctx); err != nil {
		return nil
	}
	return items
}

func softDeleteDetailedByLibraryID(ctx context.Context, libraryID *int64) error {
	_, err := DB.NewDelete().Model((*Detailed)(nil)).Where("library_id IS NOT DISTINCT FROM ?", libraryID).Exec(ctx)
	return err
}
