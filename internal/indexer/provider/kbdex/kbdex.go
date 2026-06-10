package kbdex

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/kingbenny101/kbarr/internal/config"
	iprovider "github.com/kingbenny101/kbarr/internal/indexer/provider"
	"github.com/kingbenny101/kbarr/internal/models"
	"github.com/uptrace/bun"
)

func init() {
	iprovider.Register(func() iprovider.Provider {
		return &Kbdex{httpClient: &http.Client{Timeout: 5 * time.Minute}}
	})
}

type Kbdex struct {
	httpClient *http.Client
}

func (k *Kbdex) Name() string { return "kbdex" }

func (k *Kbdex) IsEnabled(db *bun.DB) bool {
	return config.Get(db, "indexerProvider", "prowlarr") == "kbdex"
}

type kbdexTorrentResult struct {
	Title         string  `json:"title"`
	MagnetLink    *string `json:"magnet_link"`
	TorrentURL    *string `json:"torrent_url"`
	SizeBytes     int64   `json:"size_bytes"`
	Seeders       int     `json:"seeders"`
	Leechers      int     `json:"leechers"`
	SourceIndexer string  `json:"source_indexer"`
}

type kbdexSearchResponse struct {
	Results []kbdexTorrentResult `json:"results"`
}

func (k *Kbdex) Search(ctx context.Context, db *bun.DB, req iprovider.SearchRequest) ([]models.TorrentResult, error) {
	anidbID, err := lookupAnidbID(ctx, db, req.LibraryID)
	if err != nil {
		return nil, fmt.Errorf("kbdex: could not resolve anidb ID: %w", err)
	}

	cacheQuery := fmt.Sprintf("kbdex:%d", anidbID)
	dir := cacheDir()
	limit := cacheFileLimit(db)

	if cached, ok := iprovider.CacheLoad(db, dir, cacheQuery); ok {
		slog.Info("kbdex search (cache hit)", "anidb_id", anidbID)
		return cached, nil
	}

	kbdexURL := strings.TrimRight(config.Get(db, "kbdexUrl", "http://localhost:8000"), "/")
	if kbdexURL == "" {
		return nil, fmt.Errorf("kbdexUrl is not configured")
	}

	reqURL := fmt.Sprintf("%s/search?anidb_id=%d", kbdexURL, anidbID)
	slog.Info("kbdex search", "url", reqURL)

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create kbdex request: %w", err)
	}

	resp, err := k.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to call kbdex: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusPartialContent {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("kbdex returned status %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	var sr kbdexSearchResponse
	if err := json.NewDecoder(resp.Body).Decode(&sr); err != nil {
		return nil, fmt.Errorf("failed to decode kbdex response: %w", err)
	}

	results := make([]models.TorrentResult, 0, len(sr.Results))
	for _, item := range sr.Results {
		dlURL := ""
		if item.MagnetLink != nil && *item.MagnetLink != "" {
			dlURL = *item.MagnetLink
		} else if item.TorrentURL != nil {
			dlURL = *item.TorrentURL
		}
		results = append(results, models.TorrentResult{
			Title:       item.Title,
			FileName:    item.Title,
			DownloadURL: dlURL,
			Size:        item.SizeBytes,
			Indexer:     item.SourceIndexer,
			Seeds:       item.Seeders,
			Peers:       item.Leechers,
		})
	}

	slog.Info("kbdex search complete", "anidb_id", anidbID, "results", len(results))
	iprovider.CacheSave(dir, cacheQuery, results, limit)
	return results, nil
}

func lookupAnidbID(ctx context.Context, db *bun.DB, libraryID int64) (int64, error) {
	var row struct {
		SourceID *string `bun:"source_id"`
	}
	err := db.NewSelect().
		TableExpr("media").
		Column("source_id").
		Where("id = ? AND source = 'anidb' AND deleted_at IS NULL", libraryID).
		Scan(ctx, &row)
	if err != nil || row.SourceID == nil {
		return 0, fmt.Errorf("anidb source_id not found for library_id %d: %w", libraryID, err)
	}
	id, err := strconv.ParseInt(*row.SourceID, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid anidb source_id %q: %w", *row.SourceID, err)
	}
	return id, nil
}

func cacheDir() string {
	return filepath.Join(iprovider.DataRootDir(), "indexer", "kbdex-cache")
}

func cacheFileLimit(db *bun.DB) int {
	raw := config.Get(db, "cacheFileLimit", "10")
	n, err := strconv.Atoi(raw)
	if err != nil || n < 0 {
		return 10
	}
	return n
}
