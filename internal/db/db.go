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

	createTables()
	checkSettings()
	return nil
}

func createTables() error {
	query := `
	CREATE TABLE IF NOT EXISTS media (
		id               INTEGER PRIMARY KEY AUTOINCREMENT,
		title            TEXT NOT NULL,
		title_original   TEXT,
		alternate_titles TEXT DEFAULT '[]',
		description      TEXT,
		status           TEXT DEFAULT 'watching',
		type             TEXT,
		episodes         INTEGER DEFAULT 0,
		seasons          INTEGER DEFAULT 0,
		year             INTEGER,
		cover_image      TEXT,
		banner_image     TEXT,
		external_id      TEXT,
		source           TEXT,
		added_at         DATETIME DEFAULT CURRENT_TIMESTAMP,
		updated_at       DATETIME DEFAULT CURRENT_TIMESTAMP
	);
	
	CREATE TABLE IF NOT EXISTS settings (
		id    INTEGER PRIMARY KEY AUTOINCREMENT,
		key   TEXT NOT NULL UNIQUE,
		value TEXT
	);
	`

	_, err := DB.Exec(query)
	if err != nil {
		return fmt.Errorf("failed to create tables: %w", err)
	}

	logger.Log.Infof("[DB] Tables ready")
	return nil
}

func checkSettings() error {
	query := `SELECT COUNT(*) FROM settings`
	var count int
	err := DB.QueryRow(query).Scan(&count)
	if err != nil {
		return fmt.Errorf("failed to check settings count: %w", err)
	}

	if count == 0 {
		logger.Log.Info("[DB] Initializing default settings...")
		InitDefaultSettings()
	}
	return nil
}
