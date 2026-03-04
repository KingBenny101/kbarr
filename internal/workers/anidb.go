package workers

import (
	"time"

	"github.com/kingbenny101/kbarr/internal/anidb"
	"github.com/kingbenny101/kbarr/internal/config"
	"github.com/kingbenny101/kbarr/internal/logger"
)

// StartAniDBWorker handles AniDB titles dump sync on an interval
func StartAniDBWorker(stop <-chan struct{}) {
	currentCfg := config.Get()
	ticker := time.NewTicker(currentCfg.AniDBInterval)
	defer ticker.Stop()

	// run once immediately on start
	anidb.LoadTitlesDump()

	for {
		select {
		case <-ticker.C:
			anidb.LoadTitlesDump()
		case <-stop:
			logger.Log.Infof("[Worker] Stopping AniDB sync worker...")
			return
		}
	}
}
