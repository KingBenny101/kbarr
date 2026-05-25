package config

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"strconv"
	"time"
)

type Config struct {
	ServerPort       string
	ServerAddr       string
	AniDBClient      string
	AniDBVersion     string
	AniDBInterval    time.Duration
	MonitorInterval  time.Duration
	ProwlarrUrl      string
	ProwlarrApiKey   string
	ProwlarrInterval time.Duration
	AutoMonitorOnAdd bool
}

var DefaultSettings = map[string]string{
	"anidbClient":         "error",
	"anidbVersion":        "error",
	"anidbSyncInterval":   "1440",
	"monitorSyncInterval": "1",
	"prowlarrUrl":         "http://localhost:9696",
	"prowlarrApiKey":      "error",
	"prowlarrInterval":    "60",
	"autoMonitorOnAdd":    "false",
}

func EnsureDefaults(db *sql.DB) error {
	if db == nil {
		return fmt.Errorf("database is not initialized")
	}

	ctx := context.Background()
	for key, value := range DefaultSettings {
		_, err := db.ExecContext(ctx, `
INSERT INTO settings (key, value, deleted_at)
VALUES ($1, $2, NULL)
ON CONFLICT (key) DO NOTHING
`, key, value)
		if err != nil {
			return fmt.Errorf("failed to initialize default setting %s: %w", key, err)
		}
	}

	return nil
}

func SetSetting(db *sql.DB, key, value string) error {
	if db == nil {
		return fmt.Errorf("database is not initialized")
	}
	if err := validateSettingKey(key); err != nil {
		return err
	}

	_, err := db.ExecContext(context.Background(), `
INSERT INTO settings (key, value, created_at, updated_at)
VALUES ($1, $2, NOW(), NOW())
ON CONFLICT (key) DO UPDATE
SET value = EXCLUDED.value,
    updated_at = NOW(),
    deleted_at = NULL
`, key, value)
	if err != nil {
		return fmt.Errorf("failed to upsert setting %s: %w", key, err)
	}

	return nil
}

func GetSettingsMap(db *sql.DB) (map[string]string, error) {
	if db == nil {
		return nil, fmt.Errorf("database is not initialized")
	}

	rows, err := db.QueryContext(context.Background(), `
SELECT key, COALESCE(value, '')
FROM settings
WHERE deleted_at IS NULL
ORDER BY key ASC
`)
	if err != nil {
		return nil, fmt.Errorf("failed to list settings: %w", err)
	}
	defer rows.Close()

	values := make(map[string]string, len(DefaultSettings))
	for rows.Next() {
		var key string
		var value string
		if err := rows.Scan(&key, &value); err != nil {
			return nil, fmt.Errorf("failed to scan setting row: %w", err)
		}
		values[key] = value
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to iterate settings rows: %w", err)
	}

	return values, nil
}

func Load(db *sql.DB) *Config {
	serverPort := getEnv("PORT", getEnv("KBARR_PORT", "8080"))
	serverAddr := ":" + serverPort

	settings := make(map[string]string, len(DefaultSettings))
	for key, value := range DefaultSettings {
		settings[key] = value
	}

	if db != nil {
		dbSettings, err := GetSettingsMap(db)
		if err == nil {
			for key, value := range dbSettings {
				settings[key] = value
			}
		}
	}

	return &Config{
		ServerPort:       serverPort,
		ServerAddr:       serverAddr,
		AniDBClient:      settings["anidbClient"],
		AniDBVersion:     settings["anidbVersion"],
		AniDBInterval:    parseMinutesSetting(settings["anidbSyncInterval"], 1440*time.Minute, 1440*time.Minute),
		MonitorInterval:  parseMinutesSetting(settings["monitorSyncInterval"], time.Minute, time.Minute),
		ProwlarrUrl:      settings["prowlarrUrl"],
		ProwlarrApiKey:   settings["prowlarrApiKey"],
		ProwlarrInterval: parseMinutesSetting(settings["prowlarrInterval"], 60*time.Minute, time.Minute),
		AutoMonitorOnAdd: parseBoolSetting(settings["autoMonitorOnAdd"], false),
	}
}

func validateSettingKey(key string) error {
	if _, exists := DefaultSettings[key]; !exists {
		return fmt.Errorf("invalid setting key: %s", key)
	}
	return nil
}

func parseMinutesSetting(raw string, fallback time.Duration, min time.Duration) time.Duration {
	parsed, err := time.ParseDuration(raw + "m")
	if err != nil || parsed < min {
		return fallback
	}

	return parsed
}

func parseBoolSetting(raw string, fallback bool) bool {
	parsed, err := strconv.ParseBool(raw)
	if err != nil {
		return fallback
	}

	return parsed
}

func getEnv(key, fallback string) string {
	val := os.Getenv(key)
	if val == "" {
		return fallback
	}
	return val
}
