package service

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/kingbenny101/kbarr/services/indexer/internal/models"
	"github.com/kingbenny101/kbarr/shared/config"
	"github.com/uptrace/bun"
)

type cachedSearch struct {
	results   []models.SearchResult
	expiresAt time.Time
}

type IndexerService struct {
	db         *bun.DB
	httpClient *http.Client

	mu    sync.RWMutex
	cache map[string]cachedSearch
}

func New(db *bun.DB) *IndexerService {
	return &IndexerService{
		db:         db,
		httpClient: &http.Client{Timeout: 30 * time.Second},
		cache:      make(map[string]cachedSearch),
	}
}

func (s *IndexerService) Search(query string) ([]models.SearchResult, error) {
	cleanedQuery := strings.TrimSpace(query)
	if cleanedQuery == "" {
		return []models.SearchResult{}, nil
	}

	cacheKey := strings.ToLower(cleanedQuery)
	if results, ok := s.getCached(cacheKey); ok {
		return results, nil
	}

	cfg := config.Load(s.db)
	if cfg.ProwlarrApiKey == "" || cfg.ProwlarrApiKey == "error" {
		return nil, fmt.Errorf("prowlarr api key is not configured")
	}
	if strings.TrimSpace(cfg.ProwlarrUrl) == "" {
		return nil, fmt.Errorf("prowlarr url is not configured")
	}

	searchURL := fmt.Sprintf("%s/api/v1/search?query=%s&type=search", cfg.ProwlarrUrl, url.QueryEscape(cleanedQuery))
	req, err := http.NewRequest(http.MethodGet, searchURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Add("X-Api-Key", cfg.ProwlarrApiKey)

	resp, err := s.httpClient.Do(req)
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
		DownloadURL string `json:"downloadUrl"`
		Size        int64  `json:"size"`
		Indexer     string `json:"indexer"`
		Seeds       int    `json:"seeders"`
		Peers       int    `json:"leechers"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&rawResults); err != nil {
		return nil, fmt.Errorf("failed to decode prowlarr response: %w", err)
	}

	results := make([]models.SearchResult, 0, len(rawResults))
	for _, item := range rawResults {
		results = append(results, models.SearchResult{
			Title:       item.Title,
			DownloadURL: item.DownloadURL,
			Size:        item.Size,
			Indexer:     item.Indexer,
			Seeds:       item.Seeds,
			Peers:       item.Peers,
		})
	}

	ttl := cfg.ProwlarrInterval
	if ttl <= 0 {
		ttl = 5 * time.Minute
	}
	s.setCached(cacheKey, results, ttl)

	return results, nil
}

func (s *IndexerService) PollAndQueue(ctx context.Context) {
	cfg := config.Load(s.db)
	interval := cfg.MonitorInterval
	if interval < 30*time.Second {
		interval = 30 * time.Second
	}

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			s.processMonitors(ctx)

			nextInterval := config.Load(s.db).MonitorInterval
			if nextInterval < 30*time.Second {
				nextInterval = 30 * time.Second
			}
			if nextInterval != interval {
				interval = nextInterval
				ticker.Reset(interval)
			}
		case <-ctx.Done():
			return
		}
	}
}

func (s *IndexerService) processMonitors(ctx context.Context) {
	status := "monitored"
	var monitors []models.Monitor
	err := s.db.NewSelect().
		TableExpr("monitors").
		ColumnExpr("*").
		Where("status = ? AND deleted_at IS NULL", status).
		Scan(ctx, &monitors)
	if err != nil {
		slog.Error("Failed to fetch monitors", "error", err)
		return
	}

	for _, monitor := range monitors {
		title := ""
		if monitor.Title != nil {
			title = *monitor.Title
		}
		if title == "" {
			continue
		}

		results, err := s.Search(title)
		if err != nil {
			slog.Warn("Prowlarr search failed for monitor", "id", monitor.ID, "title", title, "error", err)
			continue
		}

		if len(results) == 0 {
			slog.Info("No results for monitor", "id", monitor.ID, "title", title)
			continue
		}

		best := results[0]
		statusSearching := "searching"
		_, err = s.db.NewUpdate().
			TableExpr("monitors").
			Set("status = ?", statusSearching).
			Where("id = ?", monitor.ID).
			Exec(ctx)
		if err != nil {
			slog.Error("Failed to update monitor status", "id", monitor.ID, "error", err)
			continue
		}

		queueStatus := "pending"
		entry := models.DownloadQueue{
			MonitorID:  &monitor.ID,
			Title:      monitor.Title,
			TorrentURL: &best.DownloadURL,
			Status:     &queueStatus,
		}
		_, err = s.db.NewInsert().TableExpr("download_queue").Model(&entry).Exec(ctx)
		if err != nil {
			slog.Error("Failed to insert download queue entry", "id", monitor.ID, "error", err)
			continue
		}

		slog.Info("Queued download for monitor", "monitor_id", monitor.ID, "title", title, "torrent", best.Title)
	}
}

func (s *IndexerService) getCached(key string) ([]models.SearchResult, bool) {
	s.mu.RLock()
	cached, ok := s.cache[key]
	s.mu.RUnlock()

	if !ok || time.Now().After(cached.expiresAt) {
		return nil, false
	}

	out := make([]models.SearchResult, len(cached.results))
	copy(out, cached.results)
	return out, true
}

func (s *IndexerService) setCached(key string, results []models.SearchResult, ttl time.Duration) {
	copied := make([]models.SearchResult, len(results))
	copy(copied, results)

	s.mu.Lock()
	s.cache[key] = cachedSearch{results: copied, expiresAt: time.Now().Add(ttl)}
	s.mu.Unlock()
}
