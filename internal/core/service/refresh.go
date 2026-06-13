package service

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/kingbenny101/kbarr/internal/config"
	"github.com/kingbenny101/kbarr/internal/core/clients"
	"github.com/kingbenny101/kbarr/internal/core/db"
)

// RefreshLibrary re-fetches a single show's metadata from the source (bypassing
// the per-aid cache) and additively upserts any episodes that have aired since
// it was added. It is the shared body of both the manual per-show refresh button
// and the background worker. Returns the number of newly-monitored episodes.
func RefreshLibrary(ctx context.Context, mc *clients.MetadataClient, libraryID uint) (int, error) {
	media, err := db.GetMediaByID(fmt.Sprintf("%d", libraryID))
	if err != nil {
		return 0, fmt.Errorf("media not found: %w", err)
	}
	prepared, err := mc.Prepare(ctx, media.Source, media.SourceID, media.Title, libraryID, true)
	if err != nil {
		return 0, fmt.Errorf("failed to fetch metadata: %w", err)
	}
	n, err := db.ApplyRefresh(libraryID, prepared)
	if err != nil {
		return 0, fmt.Errorf("failed to apply refresh: %w", err)
	}
	if n > 0 {
		slog.Info("refresh: new episodes added", "library_id", libraryID, "title", media.Title, "count", n)
	}
	return n, nil
}

// PollMetadataRefresh runs the background refresh loop: on each tick it re-fetches
// every still-airing library show and upserts any new episodes. Finished shows
// (those with an end date) are never re-fetched, so the per-tick AniDB load is
// bounded by the handful of shows currently airing — and each request still goes
// through the metadata service's 2s throttle and ban cooldown.
func PollMetadataRefresh(ctx context.Context, mc *clients.MetadataClient) {
	// First pass is delayed a little so it doesn't pile onto startup work.
	timer := time.NewTimer(time.Minute)
	defer timer.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-timer.C:
		}
		runRefreshPass(ctx, mc)
		timer.Reset(config.GetMinutes(db.DB, "metadataRefreshInterval", 1440*time.Minute, 60*time.Minute))
	}
}

func runRefreshPass(ctx context.Context, mc *clients.MetadataClient) {
	targets, err := db.OngoingRefreshTargets()
	if err != nil {
		slog.Error("refresh: failed to load targets", "error", err)
		return
	}
	if len(targets) == 0 {
		slog.Debug("refresh: no airing shows to refresh")
		return
	}
	slog.Info("refresh: starting pass", "shows", len(targets))
	for _, t := range targets {
		if ctx.Err() != nil {
			return
		}
		if _, err := RefreshLibrary(ctx, mc, t.LibraryID); err != nil {
			slog.Warn("refresh: failed", "library_id", t.LibraryID, "title", t.Title, "error", err)
		}
	}
	slog.Debug("refresh: pass done")
}
