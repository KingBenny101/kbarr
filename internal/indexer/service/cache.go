package service

import (
	"encoding/json"
	"os"
	"path/filepath"

	iprovider "github.com/kingbenny101/kbarr/internal/indexer/provider"
	"github.com/kingbenny101/kbarr/internal/models"
	"github.com/kingbenny101/kbarr/internal/parser"
)

func indexerDir() string {
	return filepath.Join(DataRootDir(), "indexer")
}

// saveParserDebugForQuery parses every result filename and writes a debug file
// keyed by query — same filename as the prowlarr-cache entry.
func saveParserDebugForQuery(query string, results []models.TorrentResult, limit int) {
	if len(results) == 0 {
		return
	}
	dir := filepath.Join(indexerDir(), "parser-debug")
	if err := os.MkdirAll(dir, 0755); err != nil {
		return
	}
	path := filepath.Join(dir, iprovider.CacheKey(query))
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
	iprovider.EnforceFileLimit(dir, limit)
	_ = os.WriteFile(path, data, 0644)
}

type MatchEntry struct {
	TorrentTitle  string  `json:"torrent_title"`
	ParsedTitle   string  `json:"parsed_title"`
	ParsedSeason  int     `json:"parsed_season"`
	ParsedEp      int     `json:"parsed_episode,omitempty"`
	Similarity    float64 `json:"similarity"`
	Seeds         int     `json:"seeds"`
	Passed        bool    `json:"passed"`
	Reason        string  `json:"reason,omitempty"`
}

// saveMatchingDebug writes a matching-debug file for the given query showing
// which torrents passed/failed the title+season+episode filter.
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
	iprovider.EnforceFileLimit(dir, limit)
	_ = os.WriteFile(filepath.Join(dir, iprovider.CacheKey(query)), data, 0644)
}
