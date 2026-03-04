package workers

import (
	"time"

	"github.com/kingbenny101/kbarr/internal/config"
	"github.com/kingbenny101/kbarr/internal/db"
	"github.com/kingbenny101/kbarr/internal/logger"
	"github.com/kingbenny101/kbarr/internal/prowlarr"
)

func StartMonitorWorker(stopCh <-chan struct{}) {
	cfg := config.Get()
	interval := cfg.ProwlarrInterval
	if interval <= 0 {
		interval = 60 * time.Minute
	}

	logger.Log.Infof("[Monitor] Background worker started (interval: %v)", interval)
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	// Initial run
	go runMonitorTask()

	for {
		select {
		case <-stopCh:
			logger.Log.Info("[Monitor] Background worker stopping...")
			return
		case <-ticker.C:
			runMonitorTask()
		}
	}
}

func runMonitorTask() {
	cfg := config.Get()
	if cfg.ProwlarrApiKey == "" || cfg.ProwlarrApiKey == "error" {
		logger.Log.Warn("[Monitor] Skipping task: Prowlarr API key not set")
		return
	}

	mediaList, err := db.GetAllMedia()
	if err != nil {
		logger.Log.Errorf("[Monitor] Failed to fetch media list: %v", err)
		return
	}

	for _, m := range mediaList {
		if !m.Monitored {
			continue
		}

		logger.Log.Infof("[Monitor] Checking Prowlarr for: %s", m.Title)

		results, err := prowlarr.Search(m.Title)
		if err != nil {
			logger.Log.Errorf("[Monitor] Prowlarr search failed for %s: %v", m.Title, err)
			continue
		}

		if len(results) > 0 {
			logger.Log.Infof("[Monitor] Found %d results for %s (Top result: %s via %s)",
				len(results), m.Title, results[0].Title, results[0].Indexer)
		} else {
			logger.Log.Infof("[Monitor] No results found for %s", m.Title)
		}
	}
}
