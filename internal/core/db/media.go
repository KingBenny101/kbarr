package db

import (
	"context"
	"fmt"
	"strconv"

	"github.com/uptrace/bun"

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

// AddMediaWithMonitors inserts a media row together with its detailed metadata
// and episode monitors in a single transaction. If any step fails the whole add
// rolls back, so the library never contains a half-added show. The new media id
// is stamped onto detailed and every monitor. Monitors are inserted in one bulk
// statement (a brand-new library id cannot collide, so no per-row upsert is
// needed); each is forced to status 'pending' to match InsertMonitor.
func AddMediaWithMonitors(media models.Media, detailed models.Detailed, monitors []models.Monitor) (int64, error) {
	if err := ensureDB(); err != nil {
		return 0, err
	}
	ctx := context.Background()

	var id int64
	err := DB.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		if _, err := tx.NewInsert().Model(&media).Returning("id").Exec(ctx); err != nil {
			return fmt.Errorf("failed to insert media: %w", err)
		}
		id = int64(media.ID)

		detailed.LibraryID = uint(id)
		if _, err := tx.NewInsert().Model(&detailed).Exec(ctx); err != nil {
			return fmt.Errorf("failed to insert detailed: %w", err)
		}

		if len(monitors) > 0 {
			for i := range monitors {
				monitors[i].LibraryID = uint(id)
				monitors[i].Status = "pending"
			}
			if _, err := tx.NewInsert().Model(&monitors).Exec(ctx); err != nil {
				return fmt.Errorf("failed to insert monitors: %w", err)
			}
		}
		return nil
	})
	if err != nil {
		return 0, err
	}
	return id, nil
}

// MediaFolderTaken reports whether any existing media row already uses folder.
func MediaFolderTaken(folder string) (bool, error) {
	if err := ensureDB(); err != nil {
		return false, err
	}
	count, err := DB.NewSelect().Model((*models.Media)(nil)).
		Where("media_folder = ?", folder).
		Where("deleted_at IS NULL").
		Count(context.Background())
	return count > 0, err
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
