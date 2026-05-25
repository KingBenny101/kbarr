package db

import (
	"context"
	"fmt"
	"strconv"

	dbgen "github.com/kingbenny101/kbarr/services/core/internal/db/generated"
	"github.com/kingbenny101/kbarr/shared/models"
)

func InsertMedia(m models.Media) (int64, error) {
	if err := ensureQueries(); err != nil {
		return 0, err
	}

	ctx := context.Background()
	created, err := Queries.CreateMedia(ctx, dbgen.CreateMediaParams{
		Title:     toNullString(m.Title),
		Aid:       toNullInt64FromUint(m.AID),
		PosterUrl: toNullString(m.PosterURL),
	})
	if err != nil {
		return 0, fmt.Errorf("failed to insert media: %w", err)
	}
	return created.ID, nil
}

func DeleteMedia(id string) error {
	if err := ensureQueries(); err != nil {
		return err
	}

	ctx := context.Background()
	numID, err := strconv.ParseInt(id, 10, 64)
	if err != nil {
		return fmt.Errorf("invalid media id %q: %w", id, err)
	}

	if err := Queries.SoftDeleteMediaByID(ctx, numID); err != nil {
		return fmt.Errorf("failed to delete media: %w", err)
	}

	// Clean up monitor entries
	DeleteMonitorsByLibraryID(uint(numID))
	if err := DeleteDetailedByLibraryID(uint(numID)); err != nil {
		return fmt.Errorf("failed to delete detailed media: %w", err)
	}

	return nil
}

func CheckMediaExists(aid uint) (bool, error) {
	if err := ensureQueries(); err != nil {
		return false, err
	}

	ctx := context.Background()
	count, err := Queries.CountMediaByAID(ctx, toNullInt64FromUint(aid))
	if err != nil {
		return false, fmt.Errorf("failed to check media existence: %w", err)
	}
	return count > 0, nil
}

func GetAllMedia() ([]models.Media, error) {
	if err := ensureQueries(); err != nil {
		return nil, err
	}

	ctx := context.Background()
	rows, err := Queries.ListMedia(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to query media: %w", err)
	}

	mediaList := make([]models.Media, 0, len(rows))
	for _, row := range rows {
		mediaList = append(mediaList, toMediaModel(row))
	}

	return mediaList, nil
}

func UpdateMediaMonitorStatus(id string, monitored bool) error {
	if err := ensureQueries(); err != nil {
		return err
	}

	ctx := context.Background()
	numID, err := strconv.ParseInt(id, 10, 64)
	if err != nil {
		return fmt.Errorf("invalid media id %q: %w", id, err)
	}

	_, err = Queries.GetMediaByID(ctx, numID)
	if err != nil {
		return fmt.Errorf("failed to fetch media: %w", err)
	}

	// The media table does not have a dedicated monitored column.
	_ = monitored

	if err := Queries.TouchMediaMonitorStatus(ctx, numID); err != nil {
		return fmt.Errorf("failed to update monitor status: %w", err)
	}
	return nil
}

func GetMediaByID(id string) (models.Media, error) {
	if err := ensureQueries(); err != nil {
		return models.Media{}, err
	}

	ctx := context.Background()
	numID, err := strconv.ParseInt(id, 10, 64)
	if err != nil {
		return models.Media{}, fmt.Errorf("invalid media id %q: %w", id, err)
	}

	row, err := Queries.GetMediaByID(ctx, numID)
	if err != nil {
		return models.Media{}, fmt.Errorf("failed to fetch media by id: %w", err)
	}
	return toMediaModel(row), nil
}
