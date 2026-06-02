package db

import (
	"context"
	"fmt"
	"strconv"
)

func GetAllDownloadQueue() ([]DownloadQueue, error) {
	if err := ensureDB(); err != nil {
		return nil, err
	}

	var entries []DownloadQueue
	if err := DB.NewSelect().Model(&entries).Where("deleted_at IS NULL").OrderExpr("created_at DESC").Scan(context.Background()); err != nil {
		return nil, fmt.Errorf("failed to query download queue: %w", err)
	}
	return entries, nil
}

func DeleteDownloadQueueEntry(ctx context.Context, id string) error {
	if err := ensureDB(); err != nil {
		return err
	}

	numID, err := strconv.ParseInt(id, 10, 64)
	if err != nil {
		return fmt.Errorf("invalid id %q: %w", id, err)
	}

	// Fetch row before deleting so we can reset the linked monitor
	var entry DownloadQueue
	if err := DB.NewSelect().Model(&entry).Where("id = ? AND deleted_at IS NULL", numID).Scan(ctx); err == nil && entry.MonitorID != nil {
		resetMonitorOnQueueDelete(ctx, *entry.MonitorID)
	}

	_, err = DB.NewDelete().Model((*DownloadQueue)(nil)).Where("id = ?", numID).Exec(ctx)
	return err
}

func resetMonitorOnQueueDelete(ctx context.Context, monitorID int64) {
	var mon Monitor
	if err := DB.NewSelect().Model(&mon).Where("id = ? AND deleted_at IS NULL", monitorID).Scan(ctx); err != nil {
		return
	}

	DB.NewUpdate().Model((*Monitor)(nil)).
		Set("status = 'monitored', updated_at = now()").
		Where("id = ?", monitorID).
		Exec(ctx)

	// For season monitors, also reset all episode monitors in the same library
	if mon.IsSeason != nil && *mon.IsSeason && mon.LibraryID != nil {
		DB.NewUpdate().Model((*Monitor)(nil)).
			Set("status = 'monitored', updated_at = now()").
			Where("library_id = ? AND is_episode = true AND deleted_at IS NULL", *mon.LibraryID).
			Exec(ctx)
	}
}
