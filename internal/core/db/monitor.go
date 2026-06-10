package db

import (
	"context"
	"fmt"

	"github.com/kingbenny101/kbarr/internal/models"
)

func InsertMonitor(m models.Monitor) error {
	if err := ensureDB(); err != nil {
		return err
	}
	ctx := context.Background()

	count, err := DB.NewSelect().Model((*models.Monitor)(nil)).
		Where("library_id = ?", m.LibraryID).
		Where("season = ?", m.Season).
		Where("episode_number = ?", m.EpisodeNumber).
		Where("is_episode = ?", m.IsEpisode).
		Where("deleted_at IS NULL").
		Count(ctx)
	if err != nil {
		return fmt.Errorf("failed to check monitor existence: %w", err)
	}

	if count > 0 {
		_, err := DB.NewUpdate().Model((*models.Monitor)(nil)).
			Set("monitored = ?, status = 'pending', updated_at = now()", m.Monitored).
			Where("library_id = ?", m.LibraryID).
			Where("season = ?", m.Season).
			Where("episode_number = ?", m.EpisodeNumber).
			Where("is_episode = ?", m.IsEpisode).
			Where("deleted_at IS NULL").
			Exec(ctx)
		return err
	}

	m.Status = "pending"
	_, err = DB.NewInsert().Model(&m).Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to insert monitor entry: %w", err)
	}
	return nil
}

func InsertMonitorsBulk(ms []models.Monitor) error {
	for _, m := range ms {
		if err := InsertMonitor(m); err != nil {
			return err
		}
	}
	return nil
}

func DeleteMonitor(id uint) error {
	if err := ensureDB(); err != nil {
		return err
	}
	_, err := DB.NewUpdate().Model((*models.Monitor)(nil)).
		Set("monitored = false, updated_at = now()").
		Where("id = ? AND deleted_at IS NULL", id).
		Exec(context.Background())
	return err
}

func DeleteMonitorsByLibraryID(libraryID uint) error {
	if err := ensureDB(); err != nil {
		return err
	}
	if _, err := DB.NewDelete().Model((*models.Monitor)(nil)).
		Where("library_id = ?", libraryID).
		Exec(context.Background()); err != nil {
		return fmt.Errorf("failed to delete monitor entries for library ID %d: %w", libraryID, err)
	}
	return nil
}

func DeleteMonitorsBySeason(libraryID uint, season int) error {
	if err := ensureDB(); err != nil {
		return err
	}
	_, err := DB.NewUpdate().Model((*models.Monitor)(nil)).
		Set("monitored = false, updated_at = now()").
		Where("library_id = ? AND season = ? AND deleted_at IS NULL", libraryID, season).
		Exec(context.Background())
	return err
}

func GetAllMonitored() ([]models.Monitor, error) {
	if err := ensureDB(); err != nil {
		return nil, err
	}
	var items []models.Monitor
	if err := DB.NewSelect().Model(&items).
		Where("monitored = true AND deleted_at IS NULL").
		Scan(context.Background()); err != nil {
		return nil, err
	}
	return items, nil
}

func GetMonitorsByLibraryID(libraryID uint) ([]models.Monitor, error) {
	if err := ensureDB(); err != nil {
		return nil, err
	}
	var items []models.Monitor
	if err := DB.NewSelect().Model(&items).
		Where("library_id = ? AND deleted_at IS NULL", libraryID).
		Scan(context.Background()); err != nil {
		return nil, err
	}
	return items, nil
}

func UnmonitorByDetails(libraryID uint, externalID string) error {
	if err := ensureDB(); err != nil {
		return err
	}
	_, err := DB.NewUpdate().Model((*models.Monitor)(nil)).
		Set("monitored = false, updated_at = now()").
		Where("library_id = ? AND external_id = ? AND deleted_at IS NULL", libraryID, externalID).
		Exec(context.Background())
	if err != nil {
		return fmt.Errorf("failed to unmonitor: %w", err)
	}
	return nil
}
