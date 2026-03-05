package db

import (
	"context"
	"errors"
	"fmt"

	"github.com/kingbenny101/kbarr/internal/logger"
	"github.com/kingbenny101/kbarr/internal/models"
	"gorm.io/gorm"
)

var DefaultSettings = map[string]string{
	"anidbClient":       "error",
	"anidbVersion":      "error",
	"anidbSyncInterval": "86400",
	"tmdbApiKey":        "error",
	"prowlarrUrl":       "http://localhost:9696",
	"prowlarrApiKey":    "error",
	"prowlarrInterval":  "60", // in minutes
	"autoMonitorOnAdd":  "false",
}

func InitDefaultSettings() error {
	for key := range DefaultSettings {
		if err := SetSetting(key,DefaultSettings[key]); err != nil {
			return fmt.Errorf("failed to initialize default settings: %w", err)
		}
	}
	return nil
}

func SetSetting(key, value string) error {
    if _, exists := DefaultSettings[key]; !exists {
        return fmt.Errorf("invalid setting key: %s", key)
    }

    ctx := context.Background()
    _, err := gorm.G[models.Setting](DB).Where("key = ?", key).First(ctx)
    if err != nil {
        if !errors.Is(err, gorm.ErrRecordNotFound) {
            return fmt.Errorf("failed to get setting: %w", err)
        }
        // doesn't exist — create it
        s := models.Setting{Key: key, Value: value}
        return gorm.G[models.Setting](DB).Create(ctx, &s)
    }
    // exists — update it
	_,err = gorm.G[models.Setting](DB).Where("key = ?", key).Update(ctx, "Value", value)
    if err != nil {
        return fmt.Errorf("failed to update setting: %w", err)
    }
    return nil
}

func GetSetting(key string) (string, error) {
	ctx := context.Background()
	setting, err := gorm.G[models.Setting](DB).Where("key = ?", key).First(ctx)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return "", nil
		}
		return "", fmt.Errorf("failed to get setting: %w", err)
	}
	return setting.Value, nil
}

func GetAllSettings() ([]models.Setting, error) {
	ctx := context.Background()
	settings, err := gorm.G[models.Setting](DB).Find(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to query settings: %w", err)
	}
	return settings, nil
}

func checkSettings() error {
	ctx := context.Background()
	count, err := gorm.G[models.Setting](DB).Count(ctx,"key")
	if err != nil {
		return fmt.Errorf("failed to check settings count: %w", err)
	}

	if count == 0 {
		logger.Log.Info("[DB] Initializing default settings...")
		InitDefaultSettings()
	}
	return nil
}
