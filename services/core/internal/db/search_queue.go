package db

import (
	"context"
	"fmt"

	"github.com/kingbenny101/kbarr/shared/models"
)

func AddToSearchQueue(m models.SearchQueue) error {
	if err := ensureDB(); err != nil {
		return err
	}

	ctx := context.Background()
	// Check if already in queue
	count := countSearchQueueExactMatch(ctx, int64Ptr(int64(m.LibraryID)), int64Ptr(int64(m.Season)), int64Ptr(int64(m.EpisodeNumber)), boolPtr(m.IsEpisode))
	if count < 0 {
		return fmt.Errorf("failed to check search queue existence")
	}

	if count > 0 {
		return nil // Already in queue
	}

	_, err := createSearchQueueEntry(ctx, int64Ptr(int64(m.LibraryID)), stringPtr(m.Title), stringPtr(m.EpisodeTitle), int64Ptr(int64(m.Season)), int64Ptr(int64(m.EpisodeNumber)), boolPtr(m.IsEpisode), boolPtr(m.IsSeason), stringPtr(m.Status))
	if err != nil {
		return fmt.Errorf("failed to add to search queue: %w", err)
	}
	return nil
}

func GetSearchQueue() ([]models.SearchQueue, error) {
	if err := ensureDB(); err != nil {
		return nil, err
	}

	ctx := context.Background()
	rows := listSearchQueue(ctx)

	queue := make([]models.SearchQueue, 0, len(rows))
	for _, row := range rows {
		queue = append(queue, searchQueueToModel(row))
	}

	return queue, nil
}

func DeleteSearchQueueEntry(id uint) error {
	if err := ensureDB(); err != nil {
		return err
	}

	ctx := context.Background()
	if err := softDeleteSearchQueueEntryByID(ctx, int64(id)); err != nil {
		return fmt.Errorf("failed to delete search queue entry: %w", err)
	}
	return nil
}

func listSearchQueue(ctx context.Context) []SearchQueue {
	var items []SearchQueue
	if err := DB.NewSelect().Model(&items).Scan(ctx); err != nil {
		return nil
	}
	return items
}

func createSearchQueueEntry(ctx context.Context, libraryID *int64, title *string, episodeTitle *string, season *int64, episodeNumber *int64, isEpisode *bool, isSeason *bool, status *string) (*SearchQueue, error) {
	item := &SearchQueue{
		LibraryID:     libraryID,
		Title:         title,
		EpisodeTitle:  episodeTitle,
		Season:        season,
		EpisodeNumber: episodeNumber,
		IsEpisode:     isEpisode,
		IsSeason:      isSeason,
		Status:        status,
	}
	if _, err := DB.NewInsert().Model(item).Returning("*").Exec(ctx); err != nil {
		return nil, err
	}
	return item, nil
}

func softDeleteSearchQueueEntryByID(ctx context.Context, id int64) error {
	_, err := DB.NewDelete().Model((*SearchQueue)(nil)).Where("id = ?", id).Exec(ctx)
	return err
}

func countSearchQueueExactMatch(ctx context.Context, libraryID *int64, season *int64, episodeNumber *int64, isEpisode *bool) int64 {
	count, err := DB.NewSelect().Model((*SearchQueue)(nil)).Where("library_id IS NOT DISTINCT FROM ?", libraryID).Where("season IS NOT DISTINCT FROM ?", season).Where("episode_number IS NOT DISTINCT FROM ?", episodeNumber).Where("is_episode IS NOT DISTINCT FROM ?", isEpisode).Count(ctx)
	if err != nil {
		return -1
	}
	return int64(count)
}
