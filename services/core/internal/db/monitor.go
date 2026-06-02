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
		return nil // Already monitored
	}

	_, err := createMonitor(ctx, int64Ptr(int64(m.LibraryID)), stringPtr(m.Title), stringPtr(m.EpisodeTitle), int64Ptr(int64(m.Season)), int64Ptr(int64(m.EpisodeNumber)), boolPtr(m.IsEpisode), boolPtr(m.IsSeason), stringPtr(m.Source), stringPtr(m.ExternalID), stringPtr(m.Status))
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
	if err := softDeleteMonitorByID(ctx, int64(id)); err != nil {
		return fmt.Errorf("failed to delete monitor entry: %w", err)
	}
	return nil
}

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
	if err := softDeleteMonitorsByLibraryIDAndSeason(ctx, int64Ptr(int64(libraryID)), int64Ptr(int64(season))); err != nil {
		return fmt.Errorf("failed to delete monitor entries for library ID %d season %d: %w", libraryID, season, err)
	}
	return nil
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
	if err := softDeleteMonitorsByLibraryIDAndExternalID(ctx, int64Ptr(int64(libraryID)), stringPtr(externalID)); err != nil {
		return fmt.Errorf("failed to unmonitor: %w", err)
	}
	return nil
}

func listMonitors(ctx context.Context) []Monitor {
	var items []Monitor
	if err := DB.NewSelect().Model(&items).Scan(ctx); err != nil {
		return nil
	}
	return items
}

func listMonitorsByLibraryID(ctx context.Context, libraryID *int64) []Monitor {
	var items []Monitor
	if err := DB.NewSelect().Model(&items).Where("library_id IS NOT DISTINCT FROM ?", libraryID).Scan(ctx); err != nil {
		return nil
	}
	return items
}

func createMonitor(ctx context.Context, libraryID *int64, title *string, episodeTitle *string, season *int64, episodeNumber *int64, isEpisode *bool, isSeason *bool, source *string, externalID *string, status *string) (*Monitor, error) {
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
