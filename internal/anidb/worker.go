package anidb

import (
	"time"

	"github.com/kingbenny101/kbarr/internal/config"
	"github.com/kingbenny101/kbarr/internal/logger"
)

// Check Whether the config is valid and if not, log a warning and skip the sync
// Download the titles dump, parse it, and store it in memory for fast access
func Start(cfg *config.Config, stop <-chan struct{}) {
    ticker := time.NewTicker(cfg.AniDBInterval)
    defer ticker.Stop()

    // run once immediately on start
	LoadTitlesDump(cfg)

    for {
        select {
        case <-ticker.C:
            LoadTitlesDump(cfg)
        case <-stop:
            logger.Log.Infof("[AniDB] Stopping sync worker...")
            return
        }
    }
}