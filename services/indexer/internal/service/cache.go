package service

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"time"

	"github.com/kingbenny101/kbarr/services/indexer/internal/models"
	"github.com/kingbenny101/kbarr/shared/config"
	"github.com/uptrace/bun"
)

func indexerDir() string {
	return filepath.Join(DataRootDir(), "indexer")
}

func cacheDir() string {
	return filepath.Join(indexerDir(), "prowlarr-cache")
}

func cacheKey(query string) string {
	h := sha256.Sum256([]byte(query))
	return fmt.Sprintf("%x.json", h)
}

func cacheTTL(db *bun.DB) time.Duration {
	raw, err := strconv.Atoi(config.Get(db, "prowlarrCacheAge", "3600"))
	if err != nil || raw <= 0 {
		return time.Hour
	}
	return time.Duration(raw) * time.Second
}

// enforceFileLimit deletes the oldest files in dir until fewer than limit files remain.
func enforceFileLimit(dir string, limit int) {
	if limit <= 0 {
		return
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return
	}
	type fi struct {
		name    string
		modTime time.Time
	}
	var files []fi
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		info, err := e.Info()
		if err != nil {
			continue
		}
		files = append(files, fi{name: e.Name(), modTime: info.ModTime()})
	}
	sort.Slice(files, func(i, j int) bool {
		return files[i].modTime.Before(files[j].modTime)
	})
	for len(files) >= limit {
		os.Remove(filepath.Join(dir, files[0].name))
		files = files[1:]
	}
}

func cacheLoad(db *bun.DB, query string) ([]models.SearchResult, bool) {
	path := filepath.Join(cacheDir(), cacheKey(query))
	info, err := os.Stat(path)
	if err != nil {
		return nil, false
	}
	if time.Since(info.ModTime()) > cacheTTL(db) {
		return nil, false
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, false
	}
	var results []models.SearchResult
	if err := json.Unmarshal(data, &results); err != nil {
		return nil, false
	}
	slog.Debug("Prowlarr cache hit", "query", query)
	return results, true
}

// saveGuessitDebugForQuery runs guessit on every result filename and writes
// a single debug file keyed by query — same filename as the prowlarr-cache entry.
func saveGuessitDebugForQuery(query string, results []models.SearchResult, limit int) {
	if len(results) == 0 {
		return
	}
	dir := filepath.Join(indexerDir(), "guessit-debug")
	if err := os.MkdirAll(dir, 0755); err != nil {
		return
	}
	path := filepath.Join(dir, cacheKey(query))
	if _, err := os.Stat(path); err == nil {
		return // already written for this query
	}
	type entry struct {
		Filename string          `json:"filename"`
		Result   json.RawMessage `json:"result"`
	}
	var entries []entry
	for _, r := range results {
		filename := r.FileName
		if filename == "" {
			filename = r.Title
		}
		raw := runGuessitRaw(filename)
		if len(raw) == 0 {
			continue
		}
		entries = append(entries, entry{Filename: filename, Result: json.RawMessage(raw)})
	}
	if len(entries) == 0 {
		return
	}
	data, err := json.MarshalIndent(entries, "", "  ")
	if err != nil {
		return
	}
	enforceFileLimit(dir, limit)
	_ = os.WriteFile(path, data, 0644)
}

type MatchEntry struct {
	TorrentTitle  string  `json:"torrent_title"`
	GuessitTitle  string  `json:"guessit_title"`
	GuessitSeason int     `json:"guessit_season"`
	GuessitEp     int     `json:"guessit_episode,omitempty"`
	Similarity    float64 `json:"similarity"`
	Seeds         int     `json:"seeds"`
	Passed        bool    `json:"passed"`
	Reason        string  `json:"reason,omitempty"`
}

// saveMatchingDebug writes a matching-debug file for the given query showing
// which torrents passed/failed the title+season+episode filter.
// The file is always overwritten so it reflects the latest search.
func saveMatchingDebug(query string, entries []MatchEntry, limit int) {
	if len(entries) == 0 {
		return
	}
	dir := filepath.Join(indexerDir(), "matching-debug")
	if err := os.MkdirAll(dir, 0755); err != nil {
		return
	}
	data, err := json.MarshalIndent(entries, "", "  ")
	if err != nil {
		return
	}
	enforceFileLimit(dir, limit)
	_ = os.WriteFile(filepath.Join(dir, cacheKey(query)), data, 0644)
}

func cacheSave(query string, results []models.SearchResult, limit int) {
	dir := cacheDir()
	if err := os.MkdirAll(dir, 0755); err != nil {
		slog.Warn("Failed to create prowlarr cache dir", "error", err)
		return
	}
	data, err := json.Marshal(results)
	if err != nil {
		return
	}
	enforceFileLimit(dir, limit)
	path := filepath.Join(dir, cacheKey(query))
	if err := os.WriteFile(path, data, 0644); err != nil {
		slog.Warn("Failed to write prowlarr cache", "query", query, "error", err)
	}
}
