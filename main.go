package main

import (
	"net/http"

	"github.com/kingbenny101/kbarr/internal/anidb"
	"github.com/kingbenny101/kbarr/internal/api"
	"github.com/kingbenny101/kbarr/internal/config"
	"github.com/kingbenny101/kbarr/internal/db"
	"github.com/kingbenny101/kbarr/internal/logger"
	"github.com/kingbenny101/kbarr/internal/version"
)

func main() {
	cfg := config.Load()

	logger.Init(cfg.LogLevel == "debug")
	logger.Log.Infof("[Main] KBArr v%s starting... (commit: %s, built: %s)", version.Version, version.GitCommit, version.BuildTime)

	err := db.Init(cfg.DBPath)
	if err != nil {
		logger.Log.Fatalf("[Main] Database error: %v", err)
		return
	}

	err = db.CreateTables()
	if err != nil {
		logger.Log.Fatalf("[Main] Table creation error: %v", err)
		return
	}

	err = anidb.LoadTitlesDump(cfg)
	if err != nil {
		logger.Log.Fatalf("[Main] Failed to load AniDB titles: %v", err)
		return
	}

	router := api.NewRouter(cfg)

	addr := ":" + cfg.ServerPort
	logger.Log.Infof("[Main] Server running on http://localhost:%s", cfg.ServerPort)
	http.ListenAndServe(addr, router)
}
