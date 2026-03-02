package config

import (
	"os"
)

type Config struct {
	ServerPort   string
	DBPath       string
	LogLevel     string
	CachePath    string
	AniDBClient  string
	AniDBVersion string
}

func Load() *Config {
	return &Config{
		ServerPort:   getEnv("KBARR_PORT", "8282"),
		DBPath:       getEnv("KBARR_DB_PATH", "kbarr.db"),
		LogLevel:     getEnv("KBARR_LOG_LEVEL", "info"),
		CachePath:    getEnv("KBARR_CACHE_PATH", "."),
		AniDBClient:  getEnv("KBARR_ANIDB_CLIENT", "kbarr"),
		AniDBVersion: getEnv("KBARR_ANIDB_VERSION", "1"),
	}
}

func getEnv(key, fallback string) string {
	val := os.Getenv(key)
	if val == "" {
		return fallback
	}
	return val
}