package prowlarr

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/kingbenny101/kbarr/internal/config"
	iprovider "github.com/kingbenny101/kbarr/internal/indexer/provider"
	"github.com/kingbenny101/kbarr/internal/models"
	"github.com/kingbenny101/kbarr/internal/naming"
	"github.com/uptrace/bun"
)

func init() {
	iprovider.Register(func() iprovider.Provider {
		return &Prowlarr{httpClient: &http.Client{Timeout: 5 * time.Minute}}
	})
}

type Prowlarr struct {
	httpClient *http.Client
}

func (p *Prowlarr) Name() string { return "prowlarr" }

// Prematched is false: Prowlarr does keyword search, so results must still be
// title-matched against the monitor.
func (p *Prowlarr) Prematched() bool { return false }

func (p *Prowlarr) IsEnabled(db *bun.DB) bool {
	return iprovider.IndexerEnabled(db, "prowlarr")
}

// Search builds all applicable query variants, queries Prowlarr for each,
// and returns deduplicated results merged from all variants.
func (p *Prowlarr) Search(ctx context.Context, db *bun.DB, req iprovider.SearchRequest) ([]models.TorrentResult, error) {
	queryTitles := req.AllTitles()
	limit := cacheFileLimit(db)

	var buildQueries func(string) []string
	if req.IsSeason {
		buildQueries = seasonQueries
	} else {
		buildQueries = func(t string) []string { return episodeQueries(t, req.EpisodeNumber) }
	}

	seen := map[string]bool{}
	var merged []models.TorrentResult

	for _, t := range queryTitles {
		for _, q := range buildQueries(t) {
			results, err := p.searchRaw(ctx, db, q, limit)
			if err != nil {
				slog.Warn("Prowlarr search failed", "query", q, "error", err)
				continue
			}
			slog.Info("Prowlarr search", "query", q, "results", len(results))
			for _, r := range results {
				key := r.DownloadURL
				if key == "" {
					key = r.Title
				}
				if !seen[key] {
					seen[key] = true
					merged = append(merged, r)
				}
			}
		}
	}

	return merged, nil
}

func (p *Prowlarr) searchRaw(ctx context.Context, db *bun.DB, query string, limit int) ([]models.TorrentResult, error) {
	cleanedQuery := strings.TrimSpace(query)
	if cleanedQuery == "" {
		return nil, nil
	}

	dir := cacheDir()
	if cached, ok := iprovider.CacheLoad(db, dir, cleanedQuery); ok {
		slog.Info("Prowlarr search (cache hit)", "query", cleanedQuery, "results", len(cached))
		return cached, nil
	}

	prowlarrURL := config.Get(db, "prowlarrUrl", "http://localhost:9696")
	prowlarrKey := config.Get(db, "prowlarrApiKey", "error")

	if prowlarrKey == "" || prowlarrKey == "error" {
		return nil, fmt.Errorf("prowlarr api key is not configured")
	}
	if strings.TrimSpace(prowlarrURL) == "" {
		return nil, fmt.Errorf("prowlarr url is not configured")
	}

	searchURL := fmt.Sprintf("%s/api/v1/search?query=%s&type=search", prowlarrURL, url.QueryEscape(cleanedQuery))
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, searchURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Add("X-Api-Key", prowlarrKey)

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to call prowlarr: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("prowlarr returned status %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	var rawResults []struct {
		Title       string `json:"title"`
		FileName    string `json:"fileName"`
		DownloadURL string `json:"downloadUrl"`
		MagnetURL   string `json:"magnetUrl"`
		Size        int64  `json:"size"`
		Indexer     string `json:"indexer"`
		Seeds       int    `json:"seeders"`
		Peers       int    `json:"leechers"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&rawResults); err != nil {
		return nil, fmt.Errorf("failed to decode prowlarr response: %w", err)
	}

	results := make([]models.TorrentResult, 0, len(rawResults))
	for _, item := range rawResults {
		dlURL := item.DownloadURL
		if dlURL == "" {
			dlURL = item.MagnetURL
		}
		results = append(results, models.TorrentResult{
			Title:       item.Title,
			FileName:    item.FileName,
			DownloadURL: dlURL,
			Size:        item.Size,
			Indexer:     item.Indexer,
			Seeds:       item.Seeds,
			Peers:       item.Peers,
		})
	}

	slog.Info("Prowlarr search complete", "query", cleanedQuery, "results", len(results))
	iprovider.CacheSave(dir, cleanedQuery, results, limit)
	return results, nil
}

func cacheDir() string {
	return filepath.Join(iprovider.DataRootDir(), "indexer", "prowlarr-cache")
}

func cacheFileLimit(db *bun.DB) int {
	raw := config.Get(db, "cacheFileLimit", "10")
	n, err := strconv.Atoi(raw)
	if err != nil || n < 0 {
		return 10
	}
	return n
}

// ── query builders ─────────────────────────────────────────────────────────────

func seasonQueries(title string) []string {
	queries := []string{title, title + " Batch", title + " Complete"}
	if stripped := naming.StripSeasonFromTitle(title); stripped != title && stripped != "" {
		queries = append(queries, stripped, stripped+" Batch", stripped+" Complete")
	}
	return queries
}

func episodeQueries(title string, episode int64) []string {
	return []string{
		fmt.Sprintf("%s - %02d", title, episode),
		fmt.Sprintf("%s %02d", title, episode),
		fmt.Sprintf("%s E%02d", title, episode),
	}
}
