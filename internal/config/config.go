package config

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	imodels "github.com/kingbenny101/kbarr/internal/models"
	"github.com/uptrace/bun"
)

// settingsCacheTTL bounds how long a setting value is served from the in-process
// cache before re-reading the DB. Kept short because each service runs as its own
// process: a write in one process can only invalidate its own cache, so other
// processes converge on a changed value within this window.
const settingsCacheTTL = 3 * time.Second

type cachedSetting struct {
	value   string
	present bool
	expiry  time.Time
}

var (
	cacheMu      sync.RWMutex
	settingCache = map[string]cachedSetting{}
)

func cacheLoad(key string) (cachedSetting, bool) {
	cacheMu.RLock()
	e, ok := settingCache[key]
	cacheMu.RUnlock()
	if !ok || time.Now().After(e.expiry) {
		return cachedSetting{}, false
	}
	return e, true
}

func cacheStore(key, value string, present bool) {
	cacheMu.Lock()
	settingCache[key] = cachedSetting{value: value, present: present, expiry: time.Now().Add(settingsCacheTTL)}
	cacheMu.Unlock()
}

// DefaultSettings is derived from Schema so there is one source of truth.
var DefaultSettings = func() map[string]string {
	m := make(map[string]string, len(Schema))
	for _, def := range Schema {
		m[def.Key] = def.Default
	}
	return m
}()

func EnsureDefaults(db *bun.DB) error {
	if db == nil {
		return fmt.Errorf("database is not initialized")
	}
	ctx := context.Background()
	for key, value := range DefaultSettings {
		v := value
		_, err := db.NewInsert().Model(&imodels.Setting{Key: key, Value: &v}).On("CONFLICT (key) DO NOTHING").Exec(ctx)
		if err != nil {
			return fmt.Errorf("failed to initialize default setting %s: %w", key, err)
		}
	}
	return nil
}

// Get returns the value for key from the settings table, or fallback if not found.
// Results are cached in-process for settingsCacheTTL to spare the DB a round-trip on
// every read (the poll loops read several settings per cycle).
func Get(db *bun.DB, key, fallback string) string {
	if db == nil {
		return fallback
	}
	if e, ok := cacheLoad(key); ok {
		if e.present {
			return e.value
		}
		return fallback
	}
	var s imodels.Setting
	err := db.NewSelect().Model(&s).Where("key = ?", key).Scan(context.Background())
	present := err == nil && s.Value != nil
	value := ""
	if present {
		value = *s.Value
	}
	cacheStore(key, value, present)
	if present {
		return value
	}
	return fallback
}

// GetSeconds parses a setting stored as a second count into a time.Duration.
func GetSeconds(db *bun.DB, key string, fallback, min time.Duration) time.Duration {
	raw := Get(db, key, "")
	if raw == "" {
		return fallback
	}
	n, err := strconv.Atoi(raw)
	if err != nil {
		return fallback
	}
	if d := time.Duration(n) * time.Second; d < min {
		return min
	} else {
		return d
	}
}

// GetMinutes parses a setting stored as a minute count into a time.Duration.
func GetMinutes(db *bun.DB, key string, fallback, min time.Duration) time.Duration {
	raw := Get(db, key, "")
	if raw == "" {
		return fallback
	}
	n, err := strconv.Atoi(raw)
	if err != nil {
		return fallback
	}
	if d := time.Duration(n) * time.Minute; d < min {
		return min
	} else {
		return d
	}
}

func SetSetting(db *bun.DB, key, value string) error {
	if db == nil {
		return fmt.Errorf("database is not initialized")
	}
	s := imodels.Setting{Key: key, Value: &value}
	_, err := db.NewInsert().Model(&s).On("CONFLICT (key) DO UPDATE").Set("value = EXCLUDED.value").Set("deleted_at = NULL").Exec(context.Background())
	if err != nil {
		return fmt.Errorf("failed to upsert setting %s: %w", key, err)
	}
	// Refresh this process's cache immediately so the writer sees its own change
	// without waiting out the TTL.
	cacheStore(key, value, true)
	return nil
}

// ValidateSetting checks value against the schema definition for key. It returns
// known=false for keys not in the schema so callers can ignore unknown keys, and a
// non-nil error when a known setting's value violates its type/options.
func ValidateSetting(key, value string) (known bool, err error) {
	def, ok := schemaByKey[key]
	if !ok {
		return false, nil
	}
	switch def.Type {
	case TypeInt:
		n, e := strconv.Atoi(strings.TrimSpace(value))
		if e != nil {
			return true, fmt.Errorf("%q must be a whole number", def.Key)
		}
		if n < 0 {
			return true, fmt.Errorf("%q must not be negative", def.Key)
		}
	case TypeBool:
		switch strings.ToLower(strings.TrimSpace(value)) {
		case "true", "false":
		default:
			return true, fmt.Errorf("%q must be true or false", def.Key)
		}
	case TypeSelect:
		for _, opt := range def.Options {
			if strings.EqualFold(opt, strings.TrimSpace(value)) {
				return true, nil
			}
		}
		return true, fmt.Errorf("%q must be one of %s", def.Key, strings.Join(def.Options, ", "))
	}
	return true, nil
}

// schemaByKey indexes Schema for O(1) lookup during validation.
var schemaByKey = func() map[string]SettingDef {
	m := make(map[string]SettingDef, len(Schema))
	for _, d := range Schema {
		m[d.Key] = d
	}
	return m
}()

func GetSettingsMap(db *bun.DB) (map[string]string, error) {
	if db == nil {
		return nil, fmt.Errorf("database is not initialized")
	}
	var settings []imodels.Setting
	if err := db.NewSelect().Model(&settings).Scan(context.Background()); err != nil {
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
