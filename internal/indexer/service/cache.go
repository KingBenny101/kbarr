package service

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/kingbenny101/kbarr/internal/indexer/models"
	"github.com/kingbenny101/kbarr/internal/config"
	"github.com/kingbenny101/kbarr/internal/parser"
	"github.com/uptrace/bun"
)

func indexerDir() string {
	return filepath.Join(DataRootDir(), "indexer")
}

func cacheDir() string {
	return filepath.Join(indexerDir(), "prowlarr-cache")
}

func kbdexCacheDir() string {
	return filepath.Join(indexerDir(), "kbdex-cache")
}

var nonAlnum = regexp.MustCompile(`[^a-z0-9]+`)

func cacheKey(query string) string {
	slug := nonAlnum.ReplaceAllString(strings.ToLower(query), "_")
	slug = strings.Trim(slug, "_")
	if len(slug) > 60 {
		slug = slug[:60]
	}
	h := sha256.Sum256([]byte(query))
	return fmt.Sprintf("%s_%x.json", slug, h[:4])
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

func cacheLoad(db *bun.DB, dir, query string) ([]models.SearchResult, bool) {
	path := filepath.Join(dir, cacheKey(query))
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
	slog.Debug("Cache hit", "dir", filepath.Base(dir), "query", query)
	return results, true
}

// saveParserDebugForQuery parses every result filename and writes a debug file
// keyed by query — same filename as the prowlarr-cache entry.
func saveParserDebugForQuery(query string, results []models.SearchResult, limit int) {
	if len(results) == 0 {
		return
	}
	dir := filepath.Join(indexerDir(), "parser-debug")
	if err := os.MkdirAll(dir, 0755); err != nil {
		return
	}
	path := filepath.Join(dir, cacheKey(query))
	type entry struct {
		Filename string             `json:"filename"`
		Result   parser.ParseResult `json:"result"`
	}
	var entries []entry
	for _, r := range results {
		filename := r.FileName
		if filename == "" {
			filename = r.Title
		}
		entries = append(entries, entry{Filename: filename, Result: parser.Parse(filename)})
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

func cacheSave(dir, query string, results []models.SearchResult, limit int) {
	if err := os.MkdirAll(dir, 0755); err != nil {
		slog.Warn("Failed to create cache dir", "dir", dir, "error", err)
		return
	}
	data, err := json.Marshal(results)
	if err != nil {
		return
	}
	enforceFileLimit(dir, limit)
	path := filepath.Join(dir, cacheKey(query))
	if err := os.WriteFile(path, data, 0644); err != nil {
		slog.Warn("Failed to write cache", "dir", filepath.Base(dir), "query", query, "error", err)
	}
}
