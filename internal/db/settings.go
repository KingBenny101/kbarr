package db

import (
	"database/sql"
	"fmt"

	"github.com/kingbenny101/kbarr/internal/models"
)

var DefaultSettings = map[string]string{
	"anidbClient":  "error",
	"anidbVersion": "error",
	"anidbSyncInterval": "86400",
}

func InitDefaultSettings() error {
	for key := range DefaultSettings {
		err := SetDefaultSetting(key)
		if err != nil {
			return fmt.Errorf("failed to initialize default settings: %w", err)
		}
	}
	return nil
}

func SetDefaultSetting(key string) error {
	_, err := DB.Exec(`
		INSERT INTO settings (key, value)
		VALUES (?, ?)
		ON CONFLICT(key) DO NOTHING
	`, key, DefaultSettings[key])
	if err != nil {
		return fmt.Errorf("failed to set default setting: %w", err)
	}
	return nil
}

func SetSetting(key, value string) error {

	if _, exists := DefaultSettings[key]; !exists {
		return fmt.Errorf("invalid setting key: %s", key)
	}

	_, err := DB.Exec(`
		INSERT INTO settings (key, value)
		VALUES (?, ?)
		ON CONFLICT(key) DO UPDATE SET value = excluded.value
	`, key, value)
	if err != nil {
		return fmt.Errorf("failed to set setting: %w", err)
	}
	return nil
}

func GetSetting(key string) (string, error) {
	var value string
	err := DB.QueryRow("SELECT value FROM settings WHERE key = ?", key).Scan(&value)
	if err != nil {
		if err == sql.ErrNoRows {
			return "", nil
		}
		return "", fmt.Errorf("failed to get setting: %w", err)
	}
	return value, nil
}

func GetAllSettings() ([]models.Setting, error) {
	rows, err := DB.Query("SELECT id, key, value FROM settings")
	if err != nil {
		return nil, fmt.Errorf("failed to query settings: %w", err)
	}
	defer rows.Close()

	var settings []models.Setting

	for rows.Next() {
		var s models.Setting
		err := rows.Scan(&s.ID, &s.Key, &s.Value)
		if err != nil {
			return nil, fmt.Errorf("failed to scan row: %w", err)
		}
		settings = append(settings, s)
	}

	return settings, nil
}
