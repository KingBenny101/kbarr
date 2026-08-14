package db

import (
	"context"
	"fmt"

	"github.com/uptrace/bun"

	"github.com/kingbenny101/kbarr/internal/models"
)

// upsertMonitor inserts m, or re-activates an existing matching monitor. It runs
// against any bun.IDB so callers can share a transaction.
//
// Monitors are keyed by their stable external id when one is present, so
// specials and credits that share an episode number (e.g. C1/C2, both falling
// back to episode_number 0) stay distinct rows. Without an external id, the
// (library, season, episodeNumber) fallback keeps old behavior.
func upsertMonitor(ctx context.Context, idb bun.IDB, m models.Monitor) error {
	where := func(q *bun.SelectQuery) *bun.SelectQuery {
		q = q.Where("library_id = ?", m.LibraryID).
			Where("is_episode = ?", m.IsEpisode).
			Where("deleted_at IS NULL")
		if m.ExternalID != "" {
			return q.Where("external_id = ?", m.ExternalID)
		}
		return q.Where("season = ?", m.Season).Where("episode_number = ?", m.EpisodeNumber)
	}

	count, err := where(idb.NewSelect().Model((*models.Monitor)(nil))).Count(ctx)
	if err != nil {
		return fmt.Errorf("failed to check monitor existence: %w", err)
	}

	if count > 0 {
		upd := idb.NewUpdate().Model((*models.Monitor)(nil)).
			Set("monitored = ?, status = 'pending', updated_at = CURRENT_TIMESTAMP", m.Monitored).
			Where("library_id = ?", m.LibraryID).
			Where("is_episode = ?", m.IsEpisode).
			Where("deleted_at IS NULL")
		if m.ExternalID != "" {
			upd = upd.Where("external_id = ?", m.ExternalID)
		} else {
			upd = upd.Where("season = ?", m.Season).Where("episode_number = ?", m.EpisodeNumber)
		}
		_, err := upd.Exec(ctx)
		return err
	}

	m.Status = "pending"
	if _, err = idb.NewInsert().Model(&m).Exec(ctx); err != nil {
		return fmt.Errorf("failed to insert monitor entry: %w", err)
	}
	return nil
}

func InsertMonitor(m models.Monitor) error {
	if err := ensureDB(); err != nil {
		return err
	}
	return upsertMonitor(context.Background(), DB, m)
}

// InsertMonitorsBulk upserts all monitors inside a single transaction, so a
// failure partway through rolls the whole batch back rather than leaving a
// partial monitor set.
func InsertMonitorsBulk(ms []models.Monitor) error {
	if err := ensureDB(); err != nil {
		return err
	}
	ctx := context.Background()
	return DB.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		for _, m := range ms {
			if err := upsertMonitor(ctx, tx, m); err != nil {
				return err
			}
		}
		return nil
	})
}

func DeleteMonitor(id uint) error {
	if err := ensureDB(); err != nil {
		return err
	}
	_, err := DB.NewUpdate().Model((*models.Monitor)(nil)).
		Set("monitored = false, updated_at = CURRENT_TIMESTAMP").
		Where("id = ? AND deleted_at IS NULL", id).
		Exec(context.Background())
	return err
}

func DeleteMonitorsByLibraryID(libraryID uint) error {
	if err := ensureDB(); err != nil {
		return err
	}
	ctx := context.Background()
	// Hard-delete any queued downloads for these monitors first so no dangling
	// monitor_id references remain in download_queue. Torrents are intentionally
	// left in qBittorrent for the user to manage.
	if _, err := DB.NewDelete().Model((*models.DownloadQueue)(nil)).
		Where("monitor_id IN (SELECT id FROM monitors WHERE library_id = ?)", libraryID).
		Exec(ctx); err != nil {
		return fmt.Errorf("failed to delete download queue entries for library ID %d: %w", libraryID, err)
	}
	if _, err := DB.NewDelete().Model((*models.Monitor)(nil)).
		Where("library_id = ?", libraryID).
		Exec(ctx); err != nil {
		return fmt.Errorf("failed to delete monitor entries for library ID %d: %w", libraryID, err)
	}
	return nil
}

func DeleteMonitorsBySeason(libraryID uint, season int) error {
	if err := ensureDB(); err != nil {
		return err
	}
	_, err := DB.NewUpdate().Model((*models.Monitor)(nil)).
		Set("monitored = false, updated_at = CURRENT_TIMESTAMP").
		Where("library_id = ? AND season = ? AND (is_season = true OR episode_number > 0) AND deleted_at IS NULL", libraryID, season).
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
		Set("monitored = false, updated_at = CURRENT_TIMESTAMP").
		Where("library_id = ? AND external_id = ? AND deleted_at IS NULL", libraryID, externalID).
		Exec(context.Background())
	if err != nil {
		return fmt.Errorf("failed to unmonitor: %w", err)
	}
	return nil
}

func UpdateMonitorStatusByID(id string, status string) error {
	if err := ensureDB(); err != nil {
		return err
	}
	_, err := DB.NewUpdate().Model((*models.Monitor)(nil)).
		Set("status = ?, updated_at = CURRENT_TIMESTAMP", status).
		Where("id = ? AND deleted_at IS NULL", id).
		Exec(context.Background())
	return err
}
