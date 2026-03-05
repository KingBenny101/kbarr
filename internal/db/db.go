package db

import (
	"fmt"

	"github.com/kingbenny101/kbarr/internal/logger"
	"github.com/kingbenny101/kbarr/internal/models"

	_ "github.com/mattn/go-sqlite3"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

var DB *gorm.DB

func Init(dbPath string) error {
	var err error

	DB, err = gorm.Open(sqlite.Open(dbPath), &gorm.Config{})
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}

	logger.Log.Infof("[DB] Connected to database at %s", dbPath)

	if err := DB.AutoMigrate(&models.Media{}, &models.Setting{}); err != nil {
		return fmt.Errorf("failed to migrate database schema: %w", err)
	}

	logger.Log.Infof("[DB] Tables ready")

	checkSettings()

	return nil
}
