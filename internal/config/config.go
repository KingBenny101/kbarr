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
		anidbClient = db.DefaultSettings["anidbClient"]
	}
	anidbIntervalStr, err := db.GetSetting("anidbSyncInterval")
	if err != nil {
		anidbIntervalStr = db.DefaultSettings["anidbSyncInterval"]
	}
	anidbInterval, err := time.ParseDuration(anidbIntervalStr + "s")
	if err != nil {
		anidbInterval = 60 * time.Second
	}
	anidbVersion, err := db.GetSetting("anidbVersion")
	if err != nil {
		anidbVersion = db.DefaultSettings["anidbVersion"]
	}

	serverPort := getEnv("KBARR_PORT", "8282")
	serverAddr := ":" + serverPort

	cfg := &Config{
		ServerPort:    serverPort,
		ServerAddr:    serverAddr,
		AniDBClient:   anidbClient,
		AniDBVersion:  anidbVersion,
		AniDBInterval: anidbInterval,
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