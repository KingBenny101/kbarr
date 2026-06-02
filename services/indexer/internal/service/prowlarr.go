package service

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
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
		httpClient: &http.Client{Timeout: 5 * time.Minute},
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

	prowlarrURL := config.Get(s.db, "prowlarrUrl", "http://localhost:9696")
	prowlarrKey := config.Get(s.db, "prowlarrApiKey", "error")

	if prowlarrKey == "" || prowlarrKey == "error" {
		return nil, fmt.Errorf("prowlarr api key is not configured")
	}
	if strings.TrimSpace(prowlarrURL) == "" {
		return nil, fmt.Errorf("prowlarr url is not configured")
	}

	searchURL := fmt.Sprintf("%s/api/v1/search?query=%s&type=search", prowlarrURL, url.QueryEscape(cleanedQuery))
	req, err := http.NewRequest(http.MethodGet, searchURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Add("X-Api-Key", prowlarrKey)

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

	ttl := config.GetMinutes(s.db, "prowlarrInterval", 60*time.Minute, time.Minute)
	s.setCached(cacheKey, results, ttl)

	return results, nil
}

func (s *IndexerService) PollAndQueue(ctx context.Context) {
	for {
		didWork := s.processMonitors(ctx)
		if didWork {
			// More monitors may be waiting — loop immediately
			select {
			case <-ctx.Done():
				return
			default:
			}
			continue
		}

		// Nothing to do — wait the configured interval before checking again
		interval := s.currentMonitorInterval()
		select {
		case <-time.After(interval):
		case <-ctx.Done():
			return
		}
	}
}

func (s *IndexerService) currentMonitorInterval() time.Duration {
	interval := config.GetMinutes(s.db, "monitorSyncInterval", time.Minute, 5*time.Second)
	if interval < 5*time.Second {
		return 5 * time.Second
	}
	return interval
}

var episodeInTitlePattern = regexp.MustCompile(`(?i)S\d{2}E\d{2}`)

func seasonQuery(title string, season int64) string {
	return fmt.Sprintf("%s S%02d", title, season)
}

func episodeQuery(title string, season, episode int64) string {
	return fmt.Sprintf("%s S%02dE%02d", title, season, episode)
}

func isSeasonPack(r models.SearchResult) bool {
	return !episodeInTitlePattern.MatchString(r.Title)
}

func (s *IndexerService) score(r models.SearchResult) int {
	quality := config.Get(s.db, "preferredQuality", "1080p")
	minSeeders, _ := strconv.Atoi(config.Get(s.db, "minSeeders", "1"))
	if minSeeders < 1 {
		minSeeders = 1
	}
	if r.Seeds < minSeeders {
		return -1
	}
	sc := r.Seeds
	if quality != "any" && strings.Contains(strings.ToLower(r.Title), strings.ToLower(quality)) {
		sc += 1000
	}
	return sc
}

func (s *IndexerService) pickBest(results []models.SearchResult) *models.SearchResult {
	var best *models.SearchResult
	bestScore := -2
	for i := range results {
		if sc := s.score(results[i]); sc > bestScore {
			best = &results[i]
			bestScore = sc
		}
	}
	return best
}

func (s *IndexerService) queueDownload(ctx context.Context, mon models.Monitor, best *models.SearchResult) bool {
	queueStatus := "pending"
	entry := models.DownloadQueue{
		MonitorID:   &mon.ID,
		Title:       mon.Title,
		TorrentName: &best.Title,
		TorrentURL:  &best.DownloadURL,
		Indexer:     &best.Indexer,
		Size:        &best.Size,
		Seeders:     &best.Seeds,
		Status:      &queueStatus,
	}
	_, err := s.db.NewInsert().Model(&entry).Exec(ctx)
	if err != nil {
		slog.Error("Failed to insert download queue entry", "id", mon.ID, "error", err)
		return false
	}

	_, err = s.db.NewUpdate().
		Model((*models.Monitor)(nil)).
		Set("status = 'queued', updated_at = now()").
		Where("id = ?", mon.ID).
		Exec(ctx)
	if err != nil {
		slog.Error("Failed to update monitor status to queued", "id", mon.ID, "error", err)
	}

	slog.Info("Queued download", "monitor_id", mon.ID, "torrent", best.Title)
	return true
}

func (s *IndexerService) processMonitors(ctx context.Context) bool {
	slog.Info("Running monitor poll")
	var monitors []models.Monitor
	err := s.db.NewSelect().
		Model(&monitors).
		Where("(status = 'monitored' OR (status = 'searching' AND updated_at < now() - interval '10 minutes')) AND deleted_at IS NULL").
		OrderExpr("is_season DESC").
		Limit(1).
		Scan(ctx)
	if err != nil {
		slog.Error("Failed to fetch monitors", "error", err)
		return false
	}
	if len(monitors) == 0 {
		return false
	}
	mon := monitors[0]
	title := ""
	if mon.Title != nil {
		title = *mon.Title
	}
	slog.Info("Processing monitor", "id", mon.ID, "title", title)

	// Claim the monitor immediately so concurrent polls don't pick it up
	res, err := s.db.NewUpdate().
		Model((*models.Monitor)(nil)).
		Set("status = 'searching', updated_at = now()").
		Where("id = ? AND status = 'monitored'", mon.ID).
		Exec(ctx)
	if err != nil {
		slog.Error("Failed to claim monitor", "id", mon.ID, "error", err)
		return false
	}
	if n, _ := res.RowsAffected(); n == 0 {
		slog.Info("Monitor already claimed by another poll, skipping", "id", mon.ID)
		return false
	}

	// Load ALL season monitors (any status) to know which libraries are already in-progress
	var allSeasonMons []models.Monitor
	_ = s.db.NewSelect().
		Model(&allSeasonMons).
		Where("is_season = true AND deleted_at IS NULL").
		Scan(ctx)

	seasonFound := map[int64]bool{}
	for _, m := range allSeasonMons {
		if m.LibraryID == nil || m.Status == nil {
			continue
		}
		switch *m.Status {
		case "queued", "downloading", "completed":
			seasonFound[*m.LibraryID] = true
		}
	}

	var seasonMons, episodeMons []models.Monitor
	for _, m := range monitors {
		if m.IsSeason != nil && *m.IsSeason {
			seasonMons = append(seasonMons, m)
		}
		if m.IsEpisode != nil && *m.IsEpisode {
			episodeMons = append(episodeMons, m)
		}
	}

	// Season monitors: try to find a full season pack first
	for _, mon := range seasonMons {
		if mon.Title == nil || mon.LibraryID == nil {
			continue
		}
		season := int64(1)
		if mon.Season != nil {
			season = *mon.Season
		}
		query := seasonQuery(*mon.Title, season)
		slog.Info("Searching for season pack", "monitor_id", mon.ID, "query", query)

		results, err := s.Search(query)
		if err != nil {
			slog.Warn("Season pack search failed", "monitor_id", mon.ID, "query", query, "error", err)
			continue
		}
		slog.Info("Season search returned results", "monitor_id", mon.ID, "total", len(results))

		var packs []models.SearchResult
		for _, r := range results {
			if isSeasonPack(r) {
				packs = append(packs, r)
			}
		}
		slog.Info("Season packs after filtering", "monitor_id", mon.ID, "packs", len(packs))

		if best := s.pickBest(packs); best != nil {
			slog.Info("Season pack selected", "monitor_id", mon.ID, "torrent", best.Title, "seeds", best.Seeds, "size_mb", best.Size/1024/1024)
			if s.queueDownload(ctx, mon, best) {
				seasonFound[*mon.LibraryID] = true
				// Mark all episode monitors for this library as queued
				s.db.NewUpdate().
					Model((*models.Monitor)(nil)).
					Set("status = 'queued', updated_at = now()").
					Where("library_id = ? AND is_episode = true AND status = 'monitored' AND deleted_at IS NULL", *mon.LibraryID).
					Exec(ctx)
			}
		} else {
			slog.Info("No qualifying season pack, falling back to individual episodes", "monitor_id", mon.ID, "title", *mon.Title)
			// Mark season monitor as "searching" so it's excluded from future season polls
			s.db.NewUpdate().
				Model((*models.Monitor)(nil)).
				Set("status = 'searching', updated_at = now()").
				Where("id = ?", mon.ID).
				Exec(ctx)
		}
	}

	// Episode monitors: skip those whose library already has a season pack queued
	for _, mon := range episodeMons {
		if mon.LibraryID == nil || seasonFound[*mon.LibraryID] {
			if mon.LibraryID != nil && seasonFound[*mon.LibraryID] {
				slog.Info("Skipping episode — season pack already queued", "monitor_id", mon.ID)
			}
			continue
		}
		if mon.Title == nil {
			continue
		}

		season := int64(1)
		if mon.Season != nil {
			season = *mon.Season
		}
		episode := int64(0)
		if mon.EpisodeNumber != nil {
			episode = *mon.EpisodeNumber
		}
		query := episodeQuery(*mon.Title, season, episode)
		slog.Info("Searching for episode", "monitor_id", mon.ID, "query", query)

		results, err := s.Search(query)
		if err != nil {
			slog.Warn("Episode search failed", "monitor_id", mon.ID, "query", query, "error", err)
			continue
		}
		slog.Info("Episode search returned results", "monitor_id", mon.ID, "total", len(results))

		if best := s.pickBest(results); best != nil {
			slog.Info("Episode torrent selected", "monitor_id", mon.ID, "torrent", best.Title, "seeds", best.Seeds, "size_mb", best.Size/1024/1024)
			s.queueDownload(ctx, mon, best)
		} else {
			slog.Info("No qualifying torrent for episode", "monitor_id", mon.ID, "query", query)
		}
	}
	return true
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
