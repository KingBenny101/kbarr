package config

import (
	"os"
	"sync"
	"time"

	"github.com/kingbenny101/kbarr/internal/db"
)

type Config struct {
	ServerPort       string
	ServerAddr       string
	AniDBClient      string
	AniDBVersion     string
	AniDBInterval    time.Duration
	TmDBApiKey       string
	ProwlarrUrl      string
	ProwlarrApiKey   string
	ProwlarrInterval time.Duration
	AutoMonitorOnAdd bool
}

var (
	global *Config
	mu     sync.RWMutex
)

func Load() *Config {
	// Server settings
	serverPort := getEnv("KBARR_PORT", "8282")
	serverAddr := ":" + serverPort

	// AniDB settings
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

	// TmDB settings
	tmdbApiKey, err := db.GetSetting("tmdbApiKey")
	if err != nil {
		tmdbApiKey = db.DefaultSettings["tmdbApiKey"]
	}

	// Prowlarr settings
	prowlarrUrl, err := db.GetSetting("prowlarrUrl")
	if err != nil {
		prowlarrUrl = db.DefaultSettings["prowlarrUrl"]
	}
	prowlarrApiKey, err := db.GetSetting("prowlarrApiKey")
	if err != nil {
		prowlarrApiKey = db.DefaultSettings["prowlarrApiKey"]
	}
	prowlarrIntervalStr, err := db.GetSetting("prowlarrInterval")
	if err != nil {
		prowlarrIntervalStr = db.DefaultSettings["prowlarrInterval"]
	}
	prowlarrIntervalInt, err := time.ParseDuration(prowlarrIntervalStr + "m")
	if err != nil {
		prowlarrIntervalInt = 60 * time.Minute
	}

	// Auto-monitor on add setting
	autoMonitorOnAddStr, err := db.GetSetting("autoMonitorOnAdd")
	if err != nil {
		autoMonitorOnAddStr = db.DefaultSettings["autoMonitorOnAdd"]
	}
	autoMonitorOnAdd := autoMonitorOnAddStr == "true"

	cfg := &Config{
		ServerPort:       serverPort,
		ServerAddr:       serverAddr,
		AniDBClient:      anidbClient,
		AniDBVersion:     anidbVersion,
		AniDBInterval:    anidbInterval,
		TmDBApiKey:       tmdbApiKey,
		ProwlarrUrl:      prowlarrUrl,
		ProwlarrApiKey:   prowlarrApiKey,
		ProwlarrInterval: prowlarrIntervalInt,
		AutoMonitorOnAdd: autoMonitorOnAdd,
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
