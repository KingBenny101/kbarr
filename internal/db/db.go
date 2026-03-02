package db

import (
	"database/sql"
	"fmt"

	"github.com/kingbenny101/kbarr/internal/logger"
	_ "github.com/mattn/go-sqlite3"
)

var DB *sql.DB

func Init(dbPath string) error {
	var err error

	DB, err = sql.Open("sqlite3", dbPath)
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}

	err = DB.Ping()
	if err != nil {
		return fmt.Errorf("failed to connect to database: %w", err)
	}

	logger.Log.Infof("[DB] Connected to database at %s", dbPath)
	return nil
}

func CreateTables() error {
	query := `
	CREATE TABLE IF NOT EXISTS anime (
		id               INTEGER PRIMARY KEY AUTOINCREMENT,
		title            TEXT NOT NULL,
		title_jp         TEXT,
		alternate_titles TEXT DEFAULT '[]',
		description      TEXT,
		status           TEXT DEFAULT 'watching',
		episodes         INTEGER DEFAULT 0,
		anidb_id         TEXT,
		cover_image      TEXT,
		added_at         DATETIME DEFAULT CURRENT_TIMESTAMP
	);`

	_, err := DB.Exec(query)
	if err != nil {
		return fmt.Errorf("failed to create tables: %w", err)
	}

	logger.Log.Info("[DB] Tables ready")
	return nil
}
