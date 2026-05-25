package db

import (
	"context"
	"fmt"

	dbgen "github.com/kingbenny101/kbarr/services/core/internal/db/generated"
	"github.com/kingbenny101/kbarr/shared/models"
)

func InsertMonitor(m models.Monitor) error {
	if err := ensureQueries(); err != nil {
		return err
	}

	ctx := context.Background()
	count, err := Queries.CountMonitorExactMatch(ctx, dbgen.CountMonitorExactMatchParams{
		LibraryID:     toNullInt64FromUint(m.LibraryID),
		Season:        toNullInt64(int64(m.Season)),
		EpisodeNumber: toNullInt64(int64(m.EpisodeNumber)),
		IsEpisode:     toNullBool(m.IsEpisode),
		IsSeason:      toNullBool(m.IsSeason),
	})

	if err != nil {
		return fmt.Errorf("failed to check monitor existence: %w", err)
	}

	if count > 0 {
		return nil // Already monitored
	}

	_, err = Queries.CreateMonitor(ctx, dbgen.CreateMonitorParams{
		LibraryID:     toNullInt64FromUint(m.LibraryID),
		Title:         toNullString(m.Title),
		EpisodeTitle:  toNullString(m.EpisodeTitle),
		Season:        toNullInt64(int64(m.Season)),
		EpisodeNumber: toNullInt64(int64(m.EpisodeNumber)),
		IsEpisode:     toNullBool(m.IsEpisode),
		IsSeason:      toNullBool(m.IsSeason),
		AnidbID:       toNullString(m.AniDBID),
		Column9:       m.Status,
	})
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
	if err := ensureQueries(); err != nil {
		return err
	}

	ctx := context.Background()
	if err := Queries.SoftDeleteMonitorByID(ctx, int64(id)); err != nil {
		return fmt.Errorf("failed to delete monitor entry: %w", err)
	}
	return nil
}

func DeleteMonitorsByLibraryID(libraryID uint) error {
	if err := ensureQueries(); err != nil {
		return err
	}

	ctx := context.Background()
	if err := Queries.SoftDeleteMonitorsByLibraryID(ctx, toNullInt64FromUint(libraryID)); err != nil {
		return fmt.Errorf("failed to delete monitor entries for library ID %d: %w", libraryID, err)
	}
	return nil
}

func DeleteMonitorsBySeason(libraryID uint, season int) error {
	if err := ensureQueries(); err != nil {
		return err
	}

	ctx := context.Background()
	if err := Queries.SoftDeleteMonitorsByLibraryIDAndSeason(ctx, dbgen.SoftDeleteMonitorsByLibraryIDAndSeasonParams{
		LibraryID: toNullInt64FromUint(libraryID),
		Season:    toNullInt64(int64(season)),
	}); err != nil {
		return fmt.Errorf("failed to delete monitor entries for library ID %d season %d: %w", libraryID, season, err)
	}
	return nil
}

func GetAllMonitored() ([]models.Monitor, error) {
	if err := ensureQueries(); err != nil {
		return nil, err
	}

	ctx := context.Background()
	rows, err := Queries.ListMonitors(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to query monitor entries: %w", err)
	}

	monitors := make([]models.Monitor, 0, len(rows))
	for _, row := range rows {
		monitors = append(monitors, toMonitorModel(row))
	}

	return monitors, nil
}

func GetMonitorsByLibraryID(libraryID uint) ([]models.Monitor, error) {
	if err := ensureQueries(); err != nil {
		return nil, err
	}

	ctx := context.Background()
	rows, err := Queries.ListMonitorsByLibraryID(ctx, toNullInt64FromUint(libraryID))
	if err != nil {
		return nil, fmt.Errorf("failed to query monitor entries by library ID %d: %w", libraryID, err)
	}

	monitors := make([]models.Monitor, 0, len(rows))
	for _, row := range rows {
		monitors = append(monitors, toMonitorModel(row))
	}

	return monitors, nil
}

func UnmonitorByDetails(libraryID uint, anidbID string) error {
	if err := ensureQueries(); err != nil {
		return err
	}

	ctx := context.Background()
	if err := Queries.SoftDeleteMonitorsByLibraryIDAndAniDBID(ctx, dbgen.SoftDeleteMonitorsByLibraryIDAndAniDBIDParams{
		LibraryID: toNullInt64FromUint(libraryID),
		AnidbID:   toNullString(anidbID),
	}); err != nil {
		return fmt.Errorf("failed to unmonitor: %w", err)
	}
	return nil
}
