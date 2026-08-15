package service

import (
	"context"
	"log/slog"
	"os"
	"strings"
	"time"

	"github.com/kingbenny101/kbarr/internal/config"
	"github.com/kingbenny101/kbarr/internal/cycle"
)

func DataRootDir() string {
	dataDir := strings.TrimSpace(os.Getenv("KBARR_DATA_DIR"))
	if dataDir == "" {
		return "data"
	}
	return dataDir
}

func (s *AniDBService) StartTitlesSync(stop <-chan struct{}) {
	interval := s.currentTitlesInterval()

	if err := s.LoadTitlesDump(); err != nil {
		slog.Warn("Initial titles sync failed", "error", err)
	}
	if err := s.LoadAnimeListsMapping(); err != nil {
		slog.Warn("Initial anime-lists sync failed", "error", err)
	}

	rec := cycle.NewRecorder(s.db)
	syncCycle := cycle.Cycle{Service: "metadata", Cycle: "anidb_sync", DisplayName: "AniDB title sync"}

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			nextInterval := s.syncOnce(rec, syncCycle)
			if nextInterval != interval {
				interval = nextInterval
				ticker.Reset(interval)
			}
		case <-s.trigger:
			slog.Info("AniDB sync woken by trigger")
			nextInterval := s.syncOnce(rec, syncCycle)
			if nextInterval != interval {
				interval = nextInterval
				ticker.Reset(interval)
			}
		case <-stop:
			slog.Info("Titles sync loop stopped")
			return
		}
	}
}

// syncOnce runs one titles dump + anime-lists mapping sync, recording the
// cycle, and returns the next interval so the caller can reset its ticker.
func (s *AniDBService) syncOnce(rec *cycle.Recorder, syncCycle cycle.Cycle) time.Duration {
	_ = rec.Start(context.Background(), syncCycle)
	if err := s.LoadTitlesDump(); err != nil {
		slog.Warn("Scheduled titles sync failed", "error", err)
	}
	if err := s.LoadAnimeListsMapping(); err != nil {
		slog.Warn("Scheduled anime-lists sync failed", "error", err)
	}
	nextInterval := s.currentTitlesInterval()
	_ = rec.End(context.Background(), syncCycle, time.Now().Add(nextInterval))
	return nextInterval
}

func (s *AniDBService) currentTitlesInterval() time.Duration {
	interval := config.GetMinutes(s.db, "anidbSyncInterval", 1440*time.Minute, time.Minute)
	if interval < time.Minute {
		return time.Minute
	}
	return interval
}
