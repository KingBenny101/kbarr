package service

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strconv"
	"strings"

	"github.com/kingbenny101/kbarr/services/indexer/internal/models"
	"github.com/kingbenny101/kbarr/shared/config"
)

// kbdex API response types

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

// lookupAnidbID fetches the AniDB series ID for a library entry.
func (s *IndexerService) lookupAnidbID(ctx context.Context, libraryID int64) (int64, error) {
	var row struct {
		SourceID *string `bun:"source_id"`
	}
	err := s.db.NewSelect().
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

// SearchKbdex queries kbdex for all torrents matching the given AniDB series ID.
// Results are cached per-ID in the kbdex-specific cache directory.
func (s *IndexerService) SearchKbdex(ctx context.Context, anidbID int64) ([]models.SearchResult, error) {
	cacheQuery := fmt.Sprintf("kbdex:%d", anidbID)

	if cached, ok := cacheLoad(s.db, kbdexCacheDir(), cacheQuery); ok {
		slog.Info("kbdex search (cache hit)", "anidb_id", anidbID)
		return cached, nil
	}

	kbdexURL := strings.TrimRight(config.Get(s.db, "kbdexUrl", "http://localhost:8000"), "/")
	if kbdexURL == "" {
		return nil, fmt.Errorf("kbdexUrl is not configured")
	}

	reqURL := fmt.Sprintf("%s/search?anidb_id=%d", kbdexURL, anidbID)
	slog.Info("kbdex search", "url", reqURL)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create kbdex request: %w", err)
	}

	resp, err := s.httpClient.Do(req)
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

	results := make([]models.SearchResult, 0, len(sr.Results))
	for _, item := range sr.Results {
		dlURL := ""
		if item.MagnetLink != nil && *item.MagnetLink != "" {
			dlURL = *item.MagnetLink
		} else if item.TorrentURL != nil {
			dlURL = *item.TorrentURL
		}
		results = append(results, models.SearchResult{
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
	cacheSave(kbdexCacheDir(), cacheQuery, results, s.cacheFileLimit())
	return results, nil
}

// runKbdexSearch fetches all results for the series, writes parser-debug, then
// delegates to the season or episode sub-function for matching and queueing.
func (s *IndexerService) runKbdexSearch(ctx context.Context, mon models.Monitor, title string, libraryID int64) {
	anidbID, err := s.lookupAnidbID(ctx, libraryID)
	if err != nil {
		slog.Warn("kbdex: could not resolve anidb ID, marking missing", "monitor_id", mon.ID, "error", err)
		s.db.NewUpdate().Model((*models.Monitor)(nil)).
			Set("status = 'missing', updated_at = now()").
			Where("id = ?", mon.ID).Exec(ctx)
		return
	}

	results, err := s.SearchKbdex(ctx, anidbID)
	if err != nil {
		slog.Warn("kbdex search failed", "monitor_id", mon.ID, "error", err)
		s.db.NewUpdate().Model((*models.Monitor)(nil)).
			Set("status = 'missing', updated_at = now()").
			Where("id = ?", mon.ID).Exec(ctx)
		return
	}

	debugKey := fmt.Sprintf("kbdex_%d", anidbID)
	go saveParserDebugForQuery(debugKey, results, s.cacheFileLimit())

	titles := s.getSearchTitles(ctx, libraryID, title)
	threshold := s.matchThreshold()
	season := effectiveSeason(mon, title)

	if mon.IsSeason != nil && *mon.IsSeason {
		s.runKbdexSeasonSearch(ctx, mon, title, anidbID, libraryID, season, results, titles, threshold)
	} else {
		s.runKbdexEpisodeSearch(ctx, mon, anidbID, libraryID, season, results, titles, threshold)
	}
}

func (s *IndexerService) runKbdexSeasonSearch(ctx context.Context, mon models.Monitor, title string, anidbID, libraryID, season int64, results []models.SearchResult, titles []string, threshold float64) {
	episodeCount, _ := s.db.NewSelect().
		Model((*models.Monitor)(nil)).
		Where("library_id = ? AND season = ? AND is_episode = true AND deleted_at IS NULL", libraryID, season).
		Count(ctx)

	matchLog := buildMatchLog(results, titles, threshold, int(season), -1, int(episodeCount))
	saveMatchingDebug(fmt.Sprintf("kbdex_%d_s%d", anidbID, season), matchLog, s.cacheFileLimit())

	var candidates []models.SearchResult
	for _, e := range matchLog {
		if !e.Passed {
			continue
		}
		for i := range results {
			if results[i].Title == e.TorrentTitle {
				candidates = append(candidates, results[i])
				break
			}
		}
	}

	best := s.pickBest(ctx, candidates)
	if best != nil {
		slog.Info("kbdex season pack selected", "monitor_id", mon.ID, "torrent", best.Title, "seeds", best.Seeds, "size_mb", best.Size/1024/1024)
		if s.queueDownload(ctx, mon, best) {
			s.db.NewUpdate().
				Model((*models.Monitor)(nil)).
				Set("status = 'queued', updated_at = now()").
				Where("library_id = ? AND is_episode = true AND monitored = true AND status = 'pending' AND deleted_at IS NULL", libraryID).
				Exec(ctx)
		}
	} else {
		slog.Info("kbdex: no qualifying season pack — marking missing", "monitor_id", mon.ID, "title", title)
		s.db.NewUpdate().Model((*models.Monitor)(nil)).
			Set("status = 'missing', updated_at = now()").
			Where("id = ?", mon.ID).Exec(ctx)
	}
}

func (s *IndexerService) runKbdexEpisodeSearch(ctx context.Context, mon models.Monitor, anidbID, libraryID, season int64, results []models.SearchResult, titles []string, threshold float64) {
	// If a season pack is already in-progress, hold off.
	seasonActive, _ := s.db.NewSelect().
		Model((*models.Monitor)(nil)).
		Where("library_id = ? AND is_season = true AND status IN ('queued','downloading','available') AND deleted_at IS NULL", libraryID).
		Exists(ctx)
	if seasonActive {
		slog.Info("kbdex: skipping episode — season pack in progress, resetting to pending", "monitor_id", mon.ID)
		s.db.NewUpdate().Model((*models.Monitor)(nil)).
			Set("status = 'pending', updated_at = now()").
			Where("id = ?", mon.ID).Exec(ctx)
		return
	}

	episode := int64(0)
	if mon.EpisodeNumber != nil {
		episode = *mon.EpisodeNumber
	}

	matchLog := buildMatchLog(results, titles, threshold, int(season), int(episode), 0)
	saveMatchingDebug(fmt.Sprintf("kbdex_%d_s%d_e%d", anidbID, season, episode), matchLog, s.cacheFileLimit())

	var candidates []models.SearchResult
	for _, e := range matchLog {
		if !e.Passed {
			continue
		}
		for i := range results {
			if results[i].Title == e.TorrentTitle {
				candidates = append(candidates, results[i])
				break
			}
		}
	}

	best := s.pickBest(ctx, candidates)
	if best != nil {
		slog.Info("kbdex episode torrent selected", "monitor_id", mon.ID, "torrent", best.Title, "seeds", best.Seeds)
		s.queueDownload(ctx, mon, best)
	} else {
		slog.Info("kbdex: no qualifying torrent for episode, marking missing", "monitor_id", mon.ID)
		s.db.NewUpdate().Model((*models.Monitor)(nil)).
			Set("status = 'missing', updated_at = now()").
			Where("id = ?", mon.ID).Exec(ctx)
	}
}
