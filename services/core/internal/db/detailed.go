package db

import (
	"context"
	"fmt"

	"github.com/kingbenny101/kbarr/shared/models"
	"github.com/uptrace/bun"
)

func InsertDetailed(d models.Detailed) (int64, error) {
	if err := ensureDB(); err != nil {
		return 0, err
	}

	ctx := context.Background()
	created, err := createDetailed(
		ctx,
		stringPtr(d.Title),
		stringPtr(d.Source),
		stringPtr(d.SourceID),
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
		_, err := createEpisode(ctx, int64Ptr(created.ID), stringPtr(episode.Source), stringPtr(episode.ExternalID), int64Ptr(int64(episode.Type)), stringPtr(episode.EpNo), stringPtr(episode.Title), stringPtr(episode.AirDate))
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

	return detailedToModel(*row, nil), nil
}

type EpisodeQueryParams struct {
	Types     []int
	SortField string // "ep_no" | "title"
	SortOrder string // "asc" | "desc"
	Page      int
	Limit     int
}

type EpisodeQueryResult struct {
	Episodes     []models.Episode `json:"episodes"`
	Total        int              `json:"total"`
	Page         int              `json:"page"`
	Limit        int              `json:"limit"`
	PresentTypes []int            `json:"present_types"`
}

func QueryEpisodesByLibraryID(libraryID uint, p EpisodeQueryParams) (EpisodeQueryResult, error) {
	if err := ensureDB(); err != nil {
		return EpisodeQueryResult{}, err
	}

	ctx := context.Background()

	// Resolve detailed_id for this library entry.
	det, err := getDetailedByLibraryID(ctx, int64Ptr(int64(libraryID)))
	if err != nil {
		return EpisodeQueryResult{}, fmt.Errorf("failed to get detailed record: %w", err)
	}
	detailedID := int64Ptr(det.ID)

	// Collect all distinct types present (unfiltered) for UI filter buttons.
	var rawTypes []struct{ Type int64 `bun:"type"` }
	_ = DB.NewSelect().
		TableExpr("episodes").
		ColumnExpr("DISTINCT type").
		Where("detailed_id IS NOT DISTINCT FROM ?", detailedID).
		Where("deleted_at IS NULL").
		Scan(ctx, &rawTypes)
	presentTypes := make([]int, 0, len(rawTypes))
	for _, rt := range rawTypes {
		presentTypes = append(presentTypes, int(rt.Type))
	}

	// Build filtered + sorted + paginated query.
	q := DB.NewSelect().Model((*Episode)(nil)).
		Where("detailed_id IS NOT DISTINCT FROM ?", detailedID).
		Where("deleted_at IS NULL")

	if len(p.Types) > 0 {
		q = q.Where("type IN (?)", bun.In(p.Types))
	}

	total, err := q.Count(ctx)
	if err != nil {
		return EpisodeQueryResult{}, fmt.Errorf("failed to count episodes: %w", err)
	}

	sortCol := "ep_no"
	if p.SortField == "title" {
		sortCol = "title"
	}
	sortDir := "ASC"
	if p.SortOrder == "desc" {
		sortDir = "DESC"
	}

	if sortCol == "ep_no" {
		// Numeric-aware sort: pure-numeric ep_no values sort before non-numeric ones.
		q = q.OrderExpr(
			"CASE WHEN ep_no ~ '^[0-9]+(\\.[0-9]+)?$' THEN CAST(ep_no AS NUMERIC) END "+sortDir+" NULLS LAST, ep_no "+sortDir,
		)
	} else {
		q = q.OrderExpr("? "+sortDir, bun.Ident(sortCol))
	}

	if p.Limit > 0 {
		q = q.Limit(p.Limit).Offset((p.Page - 1) * p.Limit)
	}

	var rows []Episode
	if err := q.Scan(ctx, &rows); err != nil {
		return EpisodeQueryResult{}, fmt.Errorf("failed to query episodes: %w", err)
	}

	episodes := make([]models.Episode, 0, len(rows))
	for _, row := range rows {
		episodes = append(episodes, episodeToModel(row))
	}

	return EpisodeQueryResult{
		Episodes:     episodes,
		Total:        total,
		Page:         p.Page,
		Limit:        p.Limit,
		PresentTypes: presentTypes,
	}, nil
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

func createDetailed(ctx context.Context, title *string, source *string, sourceID *string, libraryID *int64, alternateTitles *string, description *string, releaseDate *string, genres *string, posterUrl *string, totalEpisodes *int64, totalSeasons *int64) (*Detailed, error) {
	item := &Detailed{
		Title:           title,
		Source:          source,
		SourceID:        sourceID,
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

func createEpisode(ctx context.Context, detailedID *int64, source *string, externalID *string, typeVal *int64, epNo *string, title *string, airDate *string) (*Episode, error) {
	item := &Episode{
		DetailedID: detailedID,
		Source:     source,
		ExternalID: externalID,
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
