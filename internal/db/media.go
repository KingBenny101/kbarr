package db

import (
	"context"
	"fmt"

	"github.com/kingbenny101/kbarr/internal/models"
	"gorm.io/gorm"
)

func InsertMedia(m models.Media) (int64, error) {
	ctx := context.Background()
	if err := gorm.G[models.Media](DB).Create(ctx, &m); err != nil {
		return 0, fmt.Errorf("failed to insert media: %w", err)
	}
	return int64(m.ID), nil
}

func DeleteMedia(id string) error {
	ctx := context.Background()
	if _, err := gorm.G[models.Media](DB).Where("id = ?", id).Delete(ctx); err != nil {
		return fmt.Errorf("failed to delete media: %w", err)
	}
	return nil
}

func GetAllMedia() ([]models.Media, error) {
	ctx := context.Background()
	mediaList, err := gorm.G[models.Media](DB).Find(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to query media: %w", err)
	}
	return mediaList, nil
}

func UpdateMediaMonitorStatus(id string, monitored bool) error {
	ctx := context.Background()
	_, err := gorm.G[models.Media](DB).Where("id = ?", id).First(ctx)
	if err != nil {
		return fmt.Errorf("failed to fetch media: %w", err)
	}
	if _, err := gorm.G[models.Media](DB).Where("id = ?", id).Update(ctx, "Monitored", monitored); err != nil {
		return fmt.Errorf("failed to update monitor status: %w", err)
	}
	return nil
}

func GetMediaByID(id string) (models.Media, error) {
	ctx := context.Background()
	m, err := gorm.G[models.Media](DB).Where("id = ?", id).First(ctx)
	if err != nil {
		return models.Media{}, fmt.Errorf("failed to fetch media by id: %w", err)
	}
	return m, nil
}
