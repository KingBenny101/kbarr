package service

import (
	"context"
	"fmt"
	"log/slog"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/kingbenny101/kbarr/internal/config"
	iprovider "github.com/kingbenny101/kbarr/internal/indexer/provider"
	"github.com/kingbenny101/kbarr/internal/models"
	"github.com/kingbenny101/kbarr/internal/parser"
	"github.com/kingbenny101/kbarr/internal/textmatch"
	"github.com/uptrace/bun"
)

type IndexerService struct {
	db *bun.DB
}

func New(db *bun.DB) *IndexerService {
	return &IndexerService{db: db}
}

// ── parser ────────────────────────────────────────────────────────────────────

func parseFilename(filename string) parser.ParseResult {
	return parser.Parse(filename)
}

// ── title helpers ─────────────────────────────────────────────────────────────

// bestTitleSimilarity returns the highest similarity between guessedTitle and any candidate.
func bestTitleSimilarity(guessedTitle string, candidates []string) float64 {
	best := 0.0
	for _, c := range candidates {
		if s := textmatch.Similarity(guessedTitle, c); s > best {
			best = s
		}
	}
	return best
}

// ── search title lookup ───────────────────────────────────────────────────────

// getSearchTitles returns the main title followed by deduplicated alternate titles.
func (s *IndexerService) getSearchTitles(ctx context.Context, libraryID int64, mainTitle string) []string {
	titles := []string{mainTitle}
	seen := map[string]bool{textmatch.Normalize(mainTitle): true}

	var det models.Detailed
	if err := s.db.NewSelect().Model(&det).
		Where("library_id = ? AND deleted_at IS NULL", libraryID).
		Limit(1).Scan(ctx); err != nil || det.AlternateTitles == "" {
		return titles
	}

	for _, t := range strings.Split(det.AlternateTitles, "|") {
		t = strings.TrimSpace(t)
		if t == "" {
			continue
		}
		key := textmatch.Normalize(t)
		if !seen[key] {
			seen[key] = true
			titles = append(titles, t)
		}
	}
	return titles
}

// ── poll loop ─────────────────────────────────────────────────────────────────

func (s *IndexerService) PollAndQueue(ctx context.Context) {
	for {
		didWork := s.processMonitors(ctx)
		if didWork {
			select {
			case <-ctx.Done():
				return
			default:
			}
			continue
		}

		interval := s.currentMonitorInterval()
		select {
		case <-time.After(interval):
		case <-ctx.Done():
			return
		}
	}
}

func (s *IndexerService) currentMonitorInterval() time.Duration {
	return config.GetSeconds(s.db, "prowlarrInterval", 1*time.Second, 1*time.Second)
}

var (
	seasonInTitleRe  = regexp.MustCompile(`(?i)\bseason\s*(\d+)\b`)
	ordinalSeasonRe  = regexp.MustCompile(`(?i)\b(\d+)(?:st|nd|rd|th)\s+season\b`)
	westernEpPattern = regexp.MustCompile(`(?i)S\d{1,2}E\d{1,2}`)
)

// isIndividualEpisode returns true if the result is a single episode rather than a season pack.
func isIndividualEpisode(parsedEp int, filename, torrentTitle string) bool {
	return parsedEp > 0 || westernEpPattern.MatchString(filename) || westernEpPattern.MatchString(torrentTitle)
}

// extractSeasonFromTitle returns the season number embedded in a show title.
func extractSeasonFromTitle(title string) int {
	if m := ordinalSeasonRe.FindStringSubmatch(title); m != nil {
		n, _ := strconv.Atoi(m[1])
		return n
	}
	if m := seasonInTitleRe.FindStringSubmatch(title); m != nil {
		n, _ := strconv.Atoi(m[1])
		return n
	}
	return 0
}

// stripSeasonFromTitle removes season indicators from a title.
func stripSeasonFromTitle(title string) string {
	s := ordinalSeasonRe.ReplaceAllString(title, "")
	s = seasonInTitleRe.ReplaceAllString(s, "")
	return strings.Join(strings.Fields(s), " ")
}

// expandWithStrippedSeasons appends stripped-season variants to the titles list.
func expandWithStrippedSeasons(titles []string) []string {
	seen := map[string]bool{}
	out := make([]string, 0, len(titles)*2)
	for _, t := range titles {
		if !seen[t] {
			out = append(out, t)
			seen[t] = true
		}
		if stripped := stripSeasonFromTitle(t); stripped != t && stripped != "" && !seen[stripped] {
			out = append(out, stripped)
			seen[stripped] = true
		}
	}
	return out
}

func (s *IndexerService) isBlacklisted(ctx context.Context, name string) bool {
	q := s.db.NewSelect().Model((*models.TorrentBlacklist)(nil)).WhereOr("torrent_name = ?", name)
	exists, _ := q.Exists(ctx)
	return exists
}

// qualityRank maps a parsed screen_size string to a numeric rank (higher = better).
func qualityRank(screenSize string) int {
	switch strings.ToLower(screenSize) {
	case "4k", "2160p", "uhd":
		return 5
	case "1080p":
		return 4
	case "720p":
		return 3
	case "576p":
		return 2
	case "480p":
		return 1
	}
	return 0
}

// preferredQualityRank returns the rank cap for the configured preferred quality.
func preferredQualityRank(preferred string) int {
	switch strings.ToLower(preferred) {
	case "any", "":
		return int(^uint(0) >> 1)
	}
	return qualityRank(preferred)
}

// pickBest selects the best candidate from already title/season/episode-filtered results.
func (s *IndexerService) pickBest(ctx context.Context, results []models.TorrentResult) *models.TorrentResult {
	preferred := config.Get(s.db, "preferredQuality", "any")
	cap := preferredQualityRank(preferred)

	var candidates []models.TorrentResult
	for i := range results {
		if s.isBlacklisted(ctx, results[i].Title) {
			continue
		}
		filename := results[i].FileName
		if filename == "" {
			filename = results[i].Title
		}
		g := parseFilename(filename)
		rank := qualityRank(g.ScreenSize)
		if rank > cap {
			continue
		}
		candidates = append(candidates, results[i])
	}

	if len(candidates) == 0 {
		return nil
	}

	sort.Slice(candidates, func(i, j int) bool {
		fi := candidates[i].FileName
		if fi == "" {
			fi = candidates[i].Title
		}
		fj := candidates[j].FileName
		if fj == "" {
			fj = candidates[j].Title
		}
		ri := qualityRank(parseFilename(fi).ScreenSize)
		rj := qualityRank(parseFilename(fj).ScreenSize)
		if ri != rj {
			return ri > rj
		}
		return candidates[i].Seeds > candidates[j].Seeds
	})

	return &candidates[0]
}

func (s *IndexerService) matchThreshold() float64 {
	raw := config.Get(s.db, "matchThreshold", "80")
	n, err := strconv.Atoi(raw)
	if err != nil || n < 0 || n > 100 {
		return 80
	}
	return float64(n)
}

func (s *IndexerService) cacheFileLimit() int {
	raw := config.Get(s.db, "cacheFileLimit", "10")
	n, err := strconv.Atoi(raw)
	if err != nil || n < 0 {
		return 10
	}
	return n
}

// ── queue download ────────────────────────────────────────────────────────────

func (s *IndexerService) queueDownload(ctx context.Context, mon models.Monitor, best *models.TorrentResult) bool {
	queueStatus := "pending"
	monID := int64(mon.ID)
	monTitle := mon.Title
	entry := models.DownloadQueue{
		MonitorID:   &monID,
		Title:       &monTitle,
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

// ── process monitors ──────────────────────────────────────────────────────────

func (s *IndexerService) processMonitors(ctx context.Context) bool {
	var monitors []models.Monitor
	err := s.db.NewSelect().
		Model(&monitors).
		Where("monitored = true AND available = false AND (status = 'pending' OR (status = 'searching' AND updated_at < now() - interval '10 minutes')) AND deleted_at IS NULL").
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
	if mon.Title == "" || mon.LibraryID == 0 {
		return false
	}
	title := mon.Title
	libraryID := int64(mon.LibraryID)
	slog.Info("Processing monitor", "id", mon.ID, "title", title)

	res, err := s.db.NewUpdate().
		Model((*models.Monitor)(nil)).
		Set("status = 'searching', updated_at = now()").
		Where("id = ? AND (status = 'pending' OR status = 'searching')", mon.ID).
		Exec(ctx)
	if err != nil {
		slog.Error("Failed to claim monitor", "id", mon.ID, "error", err)
		return false
	}
	if n, _ := res.RowsAffected(); n == 0 {
		slog.Info("Monitor already claimed by another poll, skipping", "id", mon.ID)
		return false
	}

	season := effectiveSeason(mon, title)

	// For episode monitors: hold off if a season pack is already in progress.
	if !mon.IsSeason {
		seasonActive, _ := s.db.NewSelect().
			Model((*models.Monitor)(nil)).
			Where("library_id = ? AND is_season = true AND (status IN ('queued','downloading') OR available = true) AND deleted_at IS NULL", libraryID).
			Exists(ctx)
		if seasonActive {
			slog.Info("Skipping episode — season pack already in progress, resetting to pending", "monitor_id", mon.ID)
			s.db.NewUpdate().Model((*models.Monitor)(nil)).
				Set("status = 'pending', updated_at = now()").
				Where("id = ?", mon.ID).Exec(ctx)
			return true
		}
	}

	titles := s.getSearchTitles(ctx, libraryID, title)
	threshold := s.matchThreshold()

	var episodeCount int
	if mon.IsSeason {
		c, _ := s.db.NewSelect().
			Model((*models.Monitor)(nil)).
			Where("library_id = ? AND season = ? AND is_episode = true AND deleted_at IS NULL", libraryID, season).
			Count(ctx)
		episodeCount = int(c)
	}

	var alts []string
	if len(titles) > 1 {
		alts = titles[1:]
	}

	req := iprovider.SearchRequest{
		LibraryID:       libraryID,
		Title:           title,
		AlternateTitles: alts,
		IsSeason:        mon.IsSeason,
		Season:          season,
		EpisodeNumber:   int64(mon.EpisodeNumber),
		EpisodeCount:    episodeCount,
		Threshold:       threshold,
	}

	s.runSearch(ctx, mon, req)
	return true
}

func (s *IndexerService) runSearch(ctx context.Context, mon models.Monitor, req iprovider.SearchRequest) {
	providers := iprovider.GetEnabled(s.db)
	if len(providers) == 0 {
		slog.Warn("No indexer providers enabled", "monitor_id", mon.ID)
		s.markMissing(ctx, mon.ID)
		return
	}

	limit := s.cacheFileLimit()

	for _, p := range providers {
		results, err := p.Search(ctx, s.db, req)
		if err != nil {
			slog.Warn("Provider search failed", "provider", p.Name(), "monitor_id", mon.ID, "error", err)
			continue
		}

		slog.Info("Provider search complete", "provider", p.Name(), "monitor_id", mon.ID, "results", len(results))
		go saveParserDebugForQuery(req.Title, results, limit)

		ep := int(req.EpisodeNumber)
		if req.IsSeason {
			ep = -1
		}
		debugKey := fmt.Sprintf("%s_s%d_e%d", req.Title, req.Season, req.EpisodeNumber)
		matchLog := buildMatchLog(results, req.AllTitles(), req.Threshold, int(req.Season), ep, req.EpisodeCount)
		saveMatchingDebug(debugKey, matchLog, limit)

		var candidates []models.TorrentResult
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

		if best := s.pickBest(ctx, candidates); best != nil {
			slog.Info("Selected torrent", "provider", p.Name(), "monitor_id", mon.ID, "torrent", best.Title, "seeds", best.Seeds, "size_mb", best.Size/1024/1024)
			if s.queueDownload(ctx, mon, best) && req.IsSeason {
				s.db.NewUpdate().
					Model((*models.Monitor)(nil)).
					Set("status = 'queued', updated_at = now()").
					Where("library_id = ? AND is_episode = true AND monitored = true AND status = 'pending' AND deleted_at IS NULL", mon.LibraryID).
					Exec(ctx)
			}
			return
		}
	}

	slog.Info("No qualifying torrent found, marking missing", "monitor_id", mon.ID, "title", req.Title)
	s.markMissing(ctx, mon.ID)
}

func (s *IndexerService) markMissing(ctx context.Context, monitorID uint) {
	s.db.NewUpdate().Model((*models.Monitor)(nil)).
		Set("status = 'missing', updated_at = now()").
		Where("id = ?", monitorID).Exec(ctx)
}

// effectiveSeason returns the season number for a monitor.
func effectiveSeason(mon models.Monitor, title string) int64 {
	season := int64(1)
	if mon.Season > 0 {
		season = int64(mon.Season)
	}
	if season == 1 {
		if embedded := extractSeasonFromTitle(title); embedded > 1 {
			season = int64(embedded)
		}
	}
	return season
}

// buildMatchLog evaluates every result with anitogo+similarity, marks passing ones.
func buildMatchLog(results []models.TorrentResult, titles []string, threshold float64, season, episode, episodeCount int) []MatchEntry {
	expandedTitles := expandWithStrippedSeasons(titles)
	entries := make([]MatchEntry, 0, len(results))
	for _, r := range results {
		filename := r.FileName
		if filename == "" {
			filename = r.Title
		}
		g := parseFilename(filename)
		sim := bestTitleSimilarity(g.Title, expandedTitles)
		e := MatchEntry{
			TorrentTitle:  r.Title,
			ParsedTitle:  g.Title,
			ParsedSeason: g.Season,
			ParsedEp:     g.Episode,
			Similarity:    sim,
			Seeds:         r.Seeds,
		}
		effectiveSeason := g.Season
		if effectiveSeason == 0 {
			effectiveSeason = 1
		}
		if sim < threshold {
			e.Reason = fmt.Sprintf("similarity %.0f%% < threshold %.0f%%", sim, threshold)
		} else if effectiveSeason != season {
			e.Reason = fmt.Sprintf("season %d != %d", effectiveSeason, season)
		} else if episode < 0 && isIndividualEpisode(g.Episode, filename, r.Title) && !(episodeCount == 1 && g.Episode == 1) {
			e.Reason = fmt.Sprintf("individual episode (anitogo_ep=%d), not a season pack", g.Episode)
		} else if episode >= 0 && g.Episode != episode {
			e.Reason = fmt.Sprintf("episode %d != %d", g.Episode, episode)
		} else {
			e.Passed = true
		}
		entries = append(entries, e)
	}
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].Similarity > entries[j].Similarity
	})
	return entries
}
