package db

import (
	"context"
	"fmt"
	"strconv"

	"github.com/kingbenny101/kbarr/internal/models"
)

func InsertMedia(m models.Media) (int64, error) {
	if err := ensureDB(); err != nil {
		return 0, err
	}
	ctx := context.Background()
	if _, err := DB.NewInsert().Model(&m).Returning("id").Exec(ctx); err != nil {
		return 0, fmt.Errorf("failed to insert media: %w", err)
	}
	return int64(m.ID), nil
}

func DeleteMedia(id string) error {
	if err := ensureDB(); err != nil {
		return err
	}
	ctx := context.Background()
	numID, err := strconv.ParseUint(id, 10, 64)
	if err != nil {
		return fmt.Errorf("invalid media id %q: %w", id, err)
	}
	if _, err := DB.NewDelete().Model((*models.Media)(nil)).Where("id = ?", numID).Exec(ctx); err != nil {
		return fmt.Errorf("failed to delete media: %w", err)
	}
	if err := DeleteMonitorsByLibraryID(uint(numID)); err != nil {
		return fmt.Errorf("failed to delete monitor entries for media: %w", err)
	}
	if err := DeleteDetailedByLibraryID(uint(numID)); err != nil {
		return fmt.Errorf("failed to delete detailed media: %w", err)
	}
	return nil
}

func CheckMediaExists(source, sourceID string) (bool, error) {
	if err := ensureDB(); err != nil {
		return false, err
	}
	ctx := context.Background()
	count, err := DB.NewSelect().Model((*models.Media)(nil)).
		Where("source = ?", source).
		Where("source_id = ?", sourceID).
		Count(ctx)
	return count > 0, err
}

func GetAllMedia() ([]models.Media, error) {
	if err := ensureDB(); err != nil {
		return nil, err
	}
	var items []models.Media
	if err := DB.NewSelect().Model(&items).Where("deleted_at IS NULL").Scan(context.Background()); err != nil {
		return nil, fmt.Errorf("failed to query media: %w", err)
	}
	return items, nil
}

func GetMediaByID(id string) (models.Media, error) {
	if err := ensureDB(); err != nil {
		return models.Media{}, err
	}
	numID, err := strconv.ParseUint(id, 10, 64)
	if err != nil {
		return models.Media{}, fmt.Errorf("invalid media id %q: %w", id, err)
	}
	var m models.Media
	if err := DB.NewSelect().Model(&m).Where("id = ?", numID).Scan(context.Background()); err != nil {
		return models.Media{}, fmt.Errorf("failed to fetch media by id: %w", err)
	}
	return m, nil
}

func UpdateMediaMonitorStatus(id string, monitored bool) error {
	if err := ensureDB(); err != nil {
		return err
	}
	numID, err := strconv.ParseUint(id, 10, 64)
	if err != nil {
		return fmt.Errorf("invalid media id %q: %w", id, err)
	}
	_ = monitored
	_, err = DB.NewUpdate().Model((*models.Media)(nil)).
		Set("updated_at = NOW()").
		Where("id = ?", numID).
		Exec(context.Background())
	return err
}

func UpdateMediaNSFW(id string, nsfw bool) error {
	if err := ensureDB(); err != nil {
		return err
	}
	numID, err := strconv.ParseUint(id, 10, 64)
	if err != nil {
		return fmt.Errorf("invalid media id %q: %w", id, err)
	}
	_, err = DB.NewUpdate().Model((*models.Media)(nil)).
		Set("is_nsfw = ?", nsfw).
		Where("id = ?", numID).
		Exec(context.Background())
	return err
}

func UpdateMediaPosterURL(id int64, posterURL string) error {
	if err := ensureDB(); err != nil {
		return err
	}
	_, err := DB.NewUpdate().Model((*models.Media)(nil)).
		Set("poster_url = ?", posterURL).
		Where("id = ?", id).
		Exec(context.Background())
	return err
}
