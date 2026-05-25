package service

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/kingbenny101/kbarr/shared/config"
	proto "github.com/kingbenny101/kbarr/shared/proto"
)

type cachedSearch struct {
	results   []*proto.ProwlarrSearchResult
	expiresAt time.Time
}

type ProwlarrService struct {
	db         *sql.DB
	httpClient *http.Client

	mu    sync.RWMutex
	cache map[string]cachedSearch
}

func New(db *sql.DB) *ProwlarrService {
	return &ProwlarrService{
		db:         db,
		httpClient: &http.Client{Timeout: 30 * time.Second},
		cache:      make(map[string]cachedSearch),
	}
}

func (s *ProwlarrService) Search(query string) ([]*proto.ProwlarrSearchResult, error) {
	cleanedQuery := strings.TrimSpace(query)
	if cleanedQuery == "" {
		return []*proto.ProwlarrSearchResult{}, nil
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

	results := make([]*proto.ProwlarrSearchResult, 0, len(rawResults))
	for _, item := range rawResults {
		results = append(results, &proto.ProwlarrSearchResult{
			Title:       item.Title,
			DownloadUrl: item.DownloadURL,
			Size:        item.Size,
			Indexer:     item.Indexer,
			Seeds:       int32(item.Seeds),
			Peers:       int32(item.Peers),
		})
	}

	ttl := cfg.ProwlarrInterval
	if ttl <= 0 {
		ttl = 5 * time.Minute
	}
	s.setCached(cacheKey, results, ttl)

	return results, nil
}

func (s *ProwlarrService) getCached(key string) ([]*proto.ProwlarrSearchResult, bool) {
	s.mu.RLock()
	cached, ok := s.cache[key]
	s.mu.RUnlock()

	if !ok || time.Now().After(cached.expiresAt) {
		return nil, false
	}

	out := make([]*proto.ProwlarrSearchResult, len(cached.results))
	copy(out, cached.results)
	return out, true
}

func (s *ProwlarrService) setCached(key string, results []*proto.ProwlarrSearchResult, ttl time.Duration) {
	copied := make([]*proto.ProwlarrSearchResult, len(results))
	copy(copied, results)

	s.mu.Lock()
	s.cache[key] = cachedSearch{results: copied, expiresAt: time.Now().Add(ttl)}
	s.mu.Unlock()
}
