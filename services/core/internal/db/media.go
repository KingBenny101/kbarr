package db

import (
	"context"
	"fmt"
	"strconv"

	"github.com/kingbenny101/kbarr/shared/models"
)

func InsertMedia(m models.Media) (int64, error) {
	if err := ensureDB(); err != nil {
		return 0, err
	}

	ctx := context.Background()
	created, err := createMedia(ctx, stringPtr(m.Title), int64Ptr(int64(m.AID)), stringPtr(m.PosterURL))
	if err != nil {
		return 0, fmt.Errorf("failed to insert media: %w", err)
	}
	return created.ID, nil
}

func DeleteMedia(id string) error {
	if err := ensureDB(); err != nil {
		return err
	}

	ctx := context.Background()
	numID, err := strconv.ParseInt(id, 10, 64)
	if err != nil {
		return fmt.Errorf("invalid media id %q: %w", id, err)
	}

	if err := softDeleteMediaByID(ctx, numID); err != nil {
		return fmt.Errorf("failed to delete media: %w", err)
	}

	// Clean up monitor entries
	if err := DeleteMonitorsByLibraryID(uint(numID)); err != nil {
		return fmt.Errorf("failed to delete monitor entries for media: %w", err)
	}
	if err := DeleteDetailedByLibraryID(uint(numID)); err != nil {
		return fmt.Errorf("failed to delete detailed media: %w", err)
	}

	return nil
}

func CheckMediaExists(aid uint) (bool, error) {
	if err := ensureDB(); err != nil {
		return false, err
	}

	ctx := context.Background()
	return countMediaByAID(ctx, int64Ptr(int64(aid))) > 0, nil
}

func GetAllMedia() ([]models.Media, error) {
	if err := ensureDB(); err != nil {
		return nil, err
	}

	ctx := context.Background()
	rows, err := listMedia(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to query media: %w", err)
	}

	mediaList := make([]models.Media, 0, len(rows))
	for _, row := range rows {
		mediaList = append(mediaList, mediumToModel(row))
	}

	return mediaList, nil
}

func UpdateMediaMonitorStatus(id string, monitored bool) error {
	if err := ensureDB(); err != nil {
		return err
	}

	ctx := context.Background()
	numID, err := strconv.ParseInt(id, 10, 64)
	if err != nil {
		return fmt.Errorf("invalid media id %q: %w", id, err)
	}

	if _, err := getMediaByID(ctx, numID); err != nil {
		return fmt.Errorf("failed to fetch media: %w", err)
	}

	// The media table does not have a dedicated monitored column.
	_ = monitored

	if err := touchMediaMonitorStatus(ctx, numID); err != nil {
		return fmt.Errorf("failed to update monitor status: %w", err)
	}
	return nil
}

func GetMediaByID(id string) (models.Media, error) {
	if err := ensureDB(); err != nil {
		return models.Media{}, err
	}

	ctx := context.Background()
	numID, err := strconv.ParseInt(id, 10, 64)
	if err != nil {
		return models.Media{}, fmt.Errorf("invalid media id %q: %w", id, err)
	}

	row, err := getMediaByID(ctx, numID)
	if err != nil {
		return models.Media{}, fmt.Errorf("failed to fetch media by id: %w", err)
	}
	return mediumToModel(*row), nil
}

func listMedia(ctx context.Context) ([]Medium, error) {
	var items []Medium
	if err := DB.NewSelect().Model(&items).WhereAllWithDeleted().Scan(ctx); err != nil {
		return nil, err
	}
	return items, nil
}

func getMediaByID(ctx context.Context, id int64) (*Medium, error) {
	item := &Medium{}
	if err := DB.NewSelect().Model(item).Where("id = ?", id).Scan(ctx); err != nil {
		return nil, err
	}
	return item, nil
}

func createMedia(ctx context.Context, title *string, aid *int64, posterUrl *string) (*Medium, error) {
	item := &Medium{
		Title:     title,
		Aid:       aid,
		PosterUrl: posterUrl,
	}
	if _, err := DB.NewInsert().Model(item).Returning("*").Exec(ctx); err != nil {
		return nil, err
	}
	return item, nil
}

func softDeleteMediaByID(ctx context.Context, id int64) error {
	_, err := DB.NewDelete().Model((*Medium)(nil)).Where("id = ?", id).Exec(ctx)
	return err
}

func countMediaByAID(ctx context.Context, aid *int64) int64 {
	count, err := DB.NewSelect().Model((*Medium)(nil)).Where("aid IS NOT DISTINCT FROM ?", aid).Count(ctx)
	if err != nil {
		return 0
	}
	return int64(count)
}

func touchMediaMonitorStatus(ctx context.Context, id int64) error {
	_, err := DB.NewUpdate().Model((*Medium)(nil)).Set("updated_at = NOW()").Where("id = ?", id).Exec(ctx)
	return err
}
