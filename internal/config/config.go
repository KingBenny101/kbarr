package config

import (
	"os"
	"sync"
	"time"

	"github.com/kingbenny101/kbarr/internal/db"
)

type Config struct {
	ServerPort    string
	ServerAddr    string
	AniDBClient   string
	AniDBVersion  string
	AniDBInterval time.Duration
}

var (
	global *Config
	mu     sync.RWMutex
)

func Load() *Config {
	anidbClient, err := db.GetSetting("anidbClient")
	if err != nil {
		anidbClient = "error"
	}

	anidbInterval, err := db.GetSetting("anidbInterval")
	if err != nil {
		anidbInterval = "24h"
	}

	anidbIntervalDuration, err := time.ParseDuration(anidbInterval)
	if err != nil {
		anidbIntervalDuration = 24 * time.Hour
	}

	anidbVersion, err := db.GetSetting("anidbVersion")
	if err != nil {
		anidbVersion = "error"
	}

	serverPort := getEnv("KBARR_PORT", "8282")
	serverAddr := ":" + serverPort

	cfg := &Config{
		ServerPort:    serverPort,
		ServerAddr:    serverAddr,
		AniDBClient:   anidbClient,
		AniDBVersion:  anidbVersion,
		AniDBInterval: anidbIntervalDuration,
	}

	mu.Lock()
	global = cfg
	mu.Unlock()

	return cfg
}

func Get() *Config {
	mu.RLock()
	defer mu.RUnlock()
	return global
}

func Reload() {
	Load()
}

func getEnv(key, fallback string) string {
	val := os.Getenv(key)
	if val == "" {
		return fallback
	}
	return val
}