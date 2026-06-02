package config

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/uptrace/bun"
)

var DefaultSettings = map[string]string{
	"anidbClient":          "error",
	"anidbVersion":         "error",
	"anidbSyncInterval":    "1440",
	"monitorSyncInterval":  "1",
	"prowlarrUrl":          "http://host.docker.internal:9696",
	"prowlarrApiKey":       "error",
	"prowlarrInterval":     "1",
	"downloaderInterval":   "1",
	"matchThreshold":       "80",
	"cacheFileLimit":       "10",
	"prowlarrCacheAge":    "3600",
	"autoMonitorOnAdd":     "false",
	"preferredQuality":     "1080p",
	"minSeeders":           "1",
	"qbittorrentUrl":       "http://host.docker.internal:8080",
	"qbittorrentUsername":  "",
	"qbittorrentPassword":  "",
	"downloadPath":         "/data/torrents",
	"stallTimeout":         "300",
	"devMode":              "false",
	"authUsername":         "",
	"authPasswordHash":     "",
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
		v := value
		_, err := db.NewInsert().Model(&Setting{Key: key, Value: &v}).On("CONFLICT (key) DO NOTHING").Exec(ctx)
		if err != nil {
			return fmt.Errorf("failed to initialize default setting %s: %w", key, err)
		}
	}
	return nil
}

// Get returns the value for key from the settings table, or fallback if not found.
func Get(db *bun.DB, key, fallback string) string {
	if db == nil {
		return fallback
	}
	var s Setting
	err := db.NewSelect().Model(&s).Where("key = ? AND deleted_at IS NULL", key).Scan(context.Background())
	if err != nil || s.Value == nil {
		return fallback
	}
	return *s.Value
}

// GetSeconds parses a setting stored as a second count into a time.Duration.
func GetSeconds(db *bun.DB, key string, fallback, min time.Duration) time.Duration {
	raw := Get(db, key, "")
	if raw == "" {
		return fallback
	}
	n, err := strconv.Atoi(raw)
	if err != nil || time.Duration(n)*time.Second < min {
		return fallback
	}
	return time.Duration(n) * time.Second
}

// GetMinutes parses a setting stored as a minute count into a time.Duration.
// If the value is missing or below min, fallback is returned.
func GetMinutes(db *bun.DB, key string, fallback, min time.Duration) time.Duration {
	raw := Get(db, key, "")
	if raw == "" {
		return fallback
	}
	n, err := strconv.Atoi(raw)
	if err != nil || time.Duration(n)*time.Minute < min {
		return fallback
	}
	return time.Duration(n) * time.Minute
}

func SetSetting(db *bun.DB, key, value string) error {
	if db == nil {
		return fmt.Errorf("database is not initialized")
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
	values := make(map[string]string, len(settings))
	for _, s := range settings {
		if s.Value != nil {
			values[s.Key] = *s.Value
		}
	}
	return values, nil
}
