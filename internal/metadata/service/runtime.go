package service

import (
	"log/slog"
	"os"
	"strings"
	"time"

	"github.com/kingbenny101/kbarr/internal/config"
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

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			if err := s.LoadTitlesDump(); err != nil {
				slog.Warn("Scheduled titles sync failed", "error", err)
			}

			nextInterval := s.currentTitlesInterval()
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

func (s *AniDBService) currentTitlesInterval() time.Duration {
	interval := config.GetMinutes(s.db, "anidbSyncInterval", 1440*time.Minute, time.Minute)
	if interval < time.Minute {
		return time.Minute
	}
	return interval
}
