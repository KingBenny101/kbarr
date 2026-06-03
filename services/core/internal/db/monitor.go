package db

import (
	"context"
	"fmt"

	"github.com/kingbenny101/kbarr/shared/models"
)

func InsertMonitor(m models.Monitor) error {
	if err := ensureDB(); err != nil {
		return err
	}

	ctx := context.Background()
	count := countMonitorExactMatch(ctx, int64Ptr(int64(m.LibraryID)), int64Ptr(int64(m.Season)), int64Ptr(int64(m.EpisodeNumber)), boolPtr(m.IsEpisode))
	if count < 0 {
		return fmt.Errorf("failed to check monitor existence")
	}

	if count > 0 {
		// Row exists — update monitored flag and reset status to pending when explicitly (re)monitoring.
		_, err := DB.NewUpdate().Model((*Monitor)(nil)).
			Set("monitored = ?, status = 'pending', updated_at = now()", m.Monitored).
			Where("library_id IS NOT DISTINCT FROM ?", int64Ptr(int64(m.LibraryID))).
			Where("season IS NOT DISTINCT FROM ?", int64Ptr(int64(m.Season))).
			Where("episode_number IS NOT DISTINCT FROM ?", int64Ptr(int64(m.EpisodeNumber))).
			Where("is_episode IS NOT DISTINCT FROM ?", boolPtr(m.IsEpisode)).
			Where("deleted_at IS NULL").
			Exec(ctx)
		return err
	}

	_, err := createMonitor(ctx, int64Ptr(int64(m.LibraryID)), stringPtr(m.Title), stringPtr(m.EpisodeTitle), int64Ptr(int64(m.Season)), int64Ptr(int64(m.EpisodeNumber)), boolPtr(m.IsEpisode), boolPtr(m.IsSeason), stringPtr(m.Source), stringPtr(m.ExternalID), m.Monitored)
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
	ctx := context.Background()
	_, err := DB.NewUpdate().Model((*Monitor)(nil)).
		Set("monitored = false, updated_at = now()").
		Where("id = ? AND deleted_at IS NULL", int64(id)).
		Exec(ctx)
	return err
}

// DeleteMonitorsByLibraryID hard-deletes all monitors for a library (used when removing the whole series).
func DeleteMonitorsByLibraryID(libraryID uint) error {
	if err := ensureDB(); err != nil {
		return err
	}
	ctx := context.Background()
	if err := softDeleteMonitorsByLibraryID(ctx, int64Ptr(int64(libraryID))); err != nil {
		return fmt.Errorf("failed to delete monitor entries for library ID %d: %w", libraryID, err)
	}
	return nil
}

func DeleteMonitorsBySeason(libraryID uint, season int) error {
	if err := ensureDB(); err != nil {
		return err
	}
	ctx := context.Background()
	_, err := DB.NewUpdate().Model((*Monitor)(nil)).
		Set("monitored = false, updated_at = now()").
		Where("library_id = ? AND season = ? AND deleted_at IS NULL", int64(libraryID), int64(season)).
		Exec(ctx)
	return err
}

func GetAllMonitored() ([]models.Monitor, error) {
	if err := ensureDB(); err != nil {
		return nil, err
	}

	ctx := context.Background()
	rows := listMonitors(ctx)

	monitors := make([]models.Monitor, 0, len(rows))
	for _, row := range rows {
		monitors = append(monitors, monitorToModel(row))
	}

	return monitors, nil
}

func GetMonitorsByLibraryID(libraryID uint) ([]models.Monitor, error) {
	if err := ensureDB(); err != nil {
		return nil, err
	}

	ctx := context.Background()
	rows := listMonitorsByLibraryID(ctx, int64Ptr(int64(libraryID)))

	monitors := make([]models.Monitor, 0, len(rows))
	for _, row := range rows {
		monitors = append(monitors, monitorToModel(row))
	}

	return monitors, nil
}

func UnmonitorByDetails(libraryID uint, externalID string) error {
	if err := ensureDB(); err != nil {
		return err
	}
	ctx := context.Background()
	_, err := DB.NewUpdate().Model((*Monitor)(nil)).
		Set("monitored = false, updated_at = now()").
		Where("library_id = ? AND external_id = ? AND deleted_at IS NULL", int64(libraryID), externalID).
		Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to unmonitor: %w", err)
	}
	return nil
}

func listMonitors(ctx context.Context) []Monitor {
	var items []Monitor
	if err := DB.NewSelect().Model(&items).Where("monitored = true AND deleted_at IS NULL").Scan(ctx); err != nil {
		return nil
	}
	return items
}

func listMonitorsByLibraryID(ctx context.Context, libraryID *int64) []Monitor {
	var items []Monitor
	if err := DB.NewSelect().Model(&items).Where("library_id IS NOT DISTINCT FROM ? AND deleted_at IS NULL", libraryID).Scan(ctx); err != nil {
		return nil
	}
	return items
}

func createMonitor(ctx context.Context, libraryID *int64, title *string, episodeTitle *string, season *int64, episodeNumber *int64, isEpisode *bool, isSeason *bool, source *string, externalID *string, monitored bool) (*Monitor, error) {
	status := stringPtr("pending")
	item := &Monitor{
		LibraryID:     libraryID,
		Title:         title,
		EpisodeTitle:  episodeTitle,
		Season:        season,
		EpisodeNumber: episodeNumber,
		IsEpisode:     isEpisode,
		IsSeason:      isSeason,
		Source:        source,
		ExternalID:    externalID,
		Status:        status,
		Monitored:     monitored,
	}
	if _, err := DB.NewInsert().Model(item).Returning("*").Exec(ctx); err != nil {
		return nil, err
	}
	return item, nil
}

func softDeleteMonitorByID(ctx context.Context, id int64) error {
	_, err := DB.NewDelete().Model((*Monitor)(nil)).Where("id = ?", id).Exec(ctx)
	return err
}

func softDeleteMonitorsByLibraryID(ctx context.Context, libraryID *int64) error {
	_, err := DB.NewDelete().Model((*Monitor)(nil)).Where("library_id IS NOT DISTINCT FROM ?", libraryID).Exec(ctx)
	return err
}

func softDeleteMonitorsByLibraryIDAndSeason(ctx context.Context, libraryID *int64, season *int64) error {
	_, err := DB.NewDelete().Model((*Monitor)(nil)).Where("library_id IS NOT DISTINCT FROM ?", libraryID).Where("season IS NOT DISTINCT FROM ?", season).Exec(ctx)
	return err
}

func softDeleteMonitorsByLibraryIDAndExternalID(ctx context.Context, libraryID *int64, externalID *string) error {
	_, err := DB.NewDelete().Model((*Monitor)(nil)).Where("library_id IS NOT DISTINCT FROM ?", libraryID).Where("external_id IS NOT DISTINCT FROM ?", externalID).Exec(ctx)
	return err
}

func countMonitorExactMatch(ctx context.Context, libraryID *int64, season *int64, episodeNumber *int64, isEpisode *bool) int64 {
	count, err := DB.NewSelect().Model((*Monitor)(nil)).Where("library_id IS NOT DISTINCT FROM ?", libraryID).Where("season IS NOT DISTINCT FROM ?", season).Where("episode_number IS NOT DISTINCT FROM ?", episodeNumber).Where("is_episode IS NOT DISTINCT FROM ?", isEpisode).Count(ctx)
	if err != nil {
		return -1
	}
	return int64(count)
}
