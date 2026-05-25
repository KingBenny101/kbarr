package service

import (
	"os"
	"strings"
	"time"

	"github.com/kingbenny101/kbarr/shared/config"
	"github.com/kingbenny101/kbarr/shared/logger"
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
		logger.Log.Warnf("[AniDB Service] Initial titles sync failed: %v", err)
	}

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			if err := s.LoadTitlesDump(); err != nil {
				logger.Log.Warnf("[AniDB Service] Scheduled titles sync failed: %v", err)
			}

			nextInterval := s.currentTitlesInterval()
			if nextInterval != interval {
				interval = nextInterval
				ticker.Reset(interval)
			}
		case <-stop:
			logger.Log.Info("[AniDB Service] Titles sync loop stopped")
			return
		}
	}
}

func (s *AniDBService) currentTitlesInterval() time.Duration {
	interval := config.Load(s.db).AniDBInterval
	if interval < time.Minute {
		return time.Minute
	}

	return interval
}
