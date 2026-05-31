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

	_, err = DB.NewDelete().Model((*DownloadQueue)(nil)).Where("id = ?", numID).Exec(ctx)
	return err
}
