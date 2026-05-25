package db

import (
	"context"
	"fmt"

	dbgen "github.com/kingbenny101/kbarr/services/core/internal/db/generated"
	"github.com/kingbenny101/kbarr/shared/models"
)

func AddToSearchQueue(m models.SearchQueue) error {
	if err := ensureQueries(); err != nil {
		return err
	}

	ctx := context.Background()
	// Check if already in queue
	count, err := Queries.CountSearchQueueExactMatch(ctx, dbgen.CountSearchQueueExactMatchParams{
		LibraryID:     toNullInt64FromUint(m.LibraryID),
		Season:        toNullInt64(int64(m.Season)),
		EpisodeNumber: toNullInt64(int64(m.EpisodeNumber)),
		IsEpisode:     toNullBool(m.IsEpisode),
		IsSeason:      toNullBool(m.IsSeason),
	})

	if err != nil {
		return fmt.Errorf("failed to check search queue existence: %w", err)
	}

	if count > 0 {
		return nil // Already in queue
	}

	_, err = Queries.CreateSearchQueueEntry(ctx, dbgen.CreateSearchQueueEntryParams{
		LibraryID:     toNullInt64FromUint(m.LibraryID),
		Title:         toNullString(m.Title),
		EpisodeTitle:  toNullString(m.EpisodeTitle),
		Season:        toNullInt64(int64(m.Season)),
		EpisodeNumber: toNullInt64(int64(m.EpisodeNumber)),
		IsEpisode:     toNullBool(m.IsEpisode),
		IsSeason:      toNullBool(m.IsSeason),
		Column8:       m.Status,
	})
	if err != nil {
		return fmt.Errorf("failed to add to search queue: %w", err)
	}
	return nil
}

func GetSearchQueue() ([]models.SearchQueue, error) {
	if err := ensureQueries(); err != nil {
		return nil, err
	}

	ctx := context.Background()
	rows, err := Queries.ListSearchQueue(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch search queue: %w", err)
	}

	queue := make([]models.SearchQueue, 0, len(rows))
	for _, row := range rows {
		queue = append(queue, toSearchQueueModel(row))
	}

	return queue, nil
}

func DeleteSearchQueueEntry(id uint) error {
	if err := ensureQueries(); err != nil {
		return err
	}

	ctx := context.Background()
	if err := Queries.SoftDeleteSearchQueueEntryByID(ctx, int64(id)); err != nil {
		return fmt.Errorf("failed to delete search queue entry: %w", err)
	}
	return nil
}
