package config

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/uptrace/bun"
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

type Setting struct {
	bun.BaseModel `bun:"table:settings,alias:s"`
	Key           string  `bun:"key,notnull,unique"`
	Value         *string `bun:"value"`
}

func EnsureDefaults(db *bun.DB) error {
	if db == nil {
		return fmt.Errorf("database is not initialized")
	}

	ctx := context.Background()
	for key, value := range DefaultSettings {
		_, err := db.NewInsert().Model(&Setting{Key: key, Value: &value}).On("CONFLICT (key) DO NOTHING").Exec(ctx)
		if err != nil {
			return fmt.Errorf("failed to initialize default setting %s: %w", key, err)
		}
	}

	return nil
}

func SetSetting(db *bun.DB, key, value string) error {
	if db == nil {
		return fmt.Errorf("database is not initialized")
	}
	if err := validateSettingKey(key); err != nil {
		return err
	}

	s := Setting{Key: key, Value: &value}
	_, err := db.NewInsert().Model(&s).On("CONFLICT (key) DO UPDATE").Set("value = EXCLUDED.value").Set("deleted_at = NULL").Exec(context.Background())
	if err != nil {
		return fmt.Errorf("failed to upsert setting %s: %w", key, err)
	}

	return nil
}

func GetSettingsMap(db *bun.DB) (map[string]string, error) {
	if db == nil {
		return nil, fmt.Errorf("database is not initialized")
	}

	var settings []Setting
	if err := db.NewSelect().Model(&settings).Where("deleted_at IS NULL").Scan(context.Background()); err != nil {
		return nil, fmt.Errorf("failed to list settings: %w", err)
	}

	values := make(map[string]string, len(DefaultSettings))
	for _, setting := range settings {
		if setting.Value != nil {
			values[setting.Key] = *setting.Value
		} else {
			values[setting.Key] = ""
		}
	}

	return values, nil
}

func Load(db *bun.DB) *Config {
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
