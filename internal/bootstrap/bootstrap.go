package bootstrap

import (
	"os"

	"github.com/kingbenny101/kbarr/internal/db"
	"github.com/kingbenny101/kbarr/internal/logger"
	"github.com/kingbenny101/kbarr/internal/version"
)

const dbPath = "data/kbarr.db"

func Init() {
	// Initialize the logger
	logger.Init()
	logger.Log.Infof("[Main] kbarr v%s starting... (commit: %s, built: %s)", version.Version, version.GitCommit, version.BuildTime)

	// Check for data/ directory and create if it doesn't exist
	logger.Log.Infof("[Main] Checking for data directory...")
	if _, err := os.Stat("data"); os.IsNotExist(err) {
		err := os.Mkdir("data", 0755)
		if err != nil {
			logger.Log.Fatalf("[Main] Failed to create data directory: %v", err)
			return
		}
	}

	// Initialize the database
	logger.Log.Infof("[Main] Initializing database...")
	err := db.Init(dbPath)
	if err != nil {
		logger.Log.Fatalf("[Main] Database initialization error: %v", err)
		return
	}

	logger.Log.Info("[Main] Initialization complete.")

}