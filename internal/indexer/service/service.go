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
	"github.com/kingbenny101/kbarr/internal/cycle"
	iprovider "github.com/kingbenny101/kbarr/internal/indexer/provider"
	"github.com/kingbenny101/kbarr/internal/models"
	"github.com/kingbenny101/kbarr/internal/naming"
	"github.com/kingbenny101/kbarr/internal/parser"
	"github.com/kingbenny101/kbarr/internal/textmatch"
	"github.com/uptrace/bun"
)

type IndexerService struct {
	db      *bun.DB
	trigger chan struct{}
}

func New(db *bun.DB) *IndexerService {
	return &IndexerService{db: db, trigger: make(chan struct{}, 1)}
}

// Trigger wakes the poll loop immediately. Non-blocking: if the loop is already
// running a pass or a wake is already pending, the signal is dropped.
func (s *IndexerService) Trigger() {
	select {
	case s.trigger <- struct{}{}:
	default:
	}
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
	rec := cycle.NewRecorder(s.db)
	pollCycle := cycle.Cycle{Service: "indexer", Cycle: "monitor_poll", DisplayName: "Monitor poll"}
	forceRetry := false
	for {
		_ = rec.Start(ctx, pollCycle)
		didWork := s.processMonitors(ctx, forceRetry)
		forceRetry = false
		next := time.Now().Add(s.currentMonitorInterval())
		if didWork {
			// Work is available immediately; the next pass starts right away.
			next = time.Now()
		}
		_ = rec.End(ctx, pollCycle, next)

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
		case <-s.trigger:
			slog.Info("Monitor poll woken by trigger")
			forceRetry = true
		case <-ctx.Done():
			return
		}
	}
}

func (s *IndexerService) currentMonitorInterval() time.Duration {
	return config.GetSeconds(s.db, "prowlarrInterval", 1*time.Second, 1*time.Second)
}

var westernEpPattern = regexp.MustCompile(`(?i)S\d{1,2}E\d{1,2}`)

// isIndividualEpisode returns true if the result is a single episode rather than a season pack.
func isIndividualEpisode(parsedEp int, filename, torrentTitle string) bool {
	return parsedEp > 0 || westernEpPattern.MatchString(filename) || westernEpPattern.MatchString(torrentTitle)
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
		if stripped := naming.StripSeasonFromTitle(t); stripped != t && stripped != "" && !seen[stripped] {
			out = append(out, stripped)
			seen[stripped] = true
		}
	}
	return out
}

func (s *IndexerService) isBlacklisted(ctx context.Context, name string) bool {
	exists, _ := s.db.NewSelect().Model((*models.TorrentBlacklist)(nil)).
		Where("torrent_name = ?", name).
		Exists(ctx)
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

// ── release group helpers ─────────────────────────────────────────────────────

// releaseGroupConfig holds the parsed release group ranking configuration.
type releaseGroupConfig struct {
	order []string // canonical order; empty means no preference
	mode  string   // "tie-break" or "strong"
}

// loadReleaseGroupConfig reads and parses release group settings from the DB.
func (s *IndexerService) loadReleaseGroupConfig() releaseGroupConfig {
	raw := config.Get(s.db, "releaseGroupPreferenceOrder", "Erai-raws,SubsPlease,EMBER,Judas,ASW,Yameii,Golumpa,Nekomoe kissaten,ToxicRUS,Baws,HorribleSubs")
	order, _ := config.ValidateReleaseGroupOrder(raw)
	if order == nil {
		order = []string{}
	}
	mode := config.Get(s.db, "releaseGroupPriorityMode", "tie-break")
	return releaseGroupConfig{order: order, mode: mode}
}

// releaseGroupRank returns the index of group within the preference order.
// Groups not in the list (or empty/unknown) return len(order), sorting after
// all preferred groups.  When order is empty every group returns 0 (no effect).
func releaseGroupRank(group string, order []string) int {
	if len(order) == 0 {
		return 0
	}
	if group == "" {
		return len(order)
	}
	normalized := config.NormalizeReleaseGroup(group)
	for i, g := range order {
		if config.NormalizeReleaseGroup(g) == normalized {
			return i
		}
	}
	return len(order)
}

// subtitleRank returns the best (lowest) index of any subtitle language in p
// that appears in order.  When no language matches or order is empty it returns
// len(order), sorting after all preferred languages (or equal when no preference).
func subtitleRank(p parser.ParseResult, order []string) int {
	if len(order) == 0 {
		return 0
	}
	best := len(order)
	for _, lang := range p.SubtitleLangs {
		for i, preferred := range order {
			if strings.EqualFold(lang, preferred) {
				if i < best {
					best = i
				}
				break
			}
		}
	}
	return best
}

// ── pickBest ───────────────────────────────────────────────────────────────────

// pickBest selects the best candidate from already title/season/episode-filtered results.
func (s *IndexerService) pickBest(ctx context.Context, results []models.TorrentResult) *models.TorrentResult {
	preferred := config.Get(s.db, "preferredQuality", "any")
	cap := preferredQualityRank(preferred)
	subRaw := config.Get(s.db, "preferredSubtitleLanguage", "multi,eng,jpn,spa,por,ara,fre,ger")
	subOrder, _ := config.ValidateSubtitleLanguageOrder(subRaw)
	if subOrder == nil {
		subOrder = []string{}
	}
	minSeeders := s.minSeeders()
	rgCfg := s.loadReleaseGroupConfig()

	type cand struct {
		result models.TorrentResult
		parsed parser.ParseResult
	}

	var candidates []cand
	for i := range results {
		if s.isBlacklisted(ctx, results[i].Title) {
			continue
		}
		if results[i].Seeds < minSeeders {
			slog.Debug("Skipping torrent below minimum seeders", "torrent", results[i].Title, "seeds", results[i].Seeds, "min", minSeeders)
			continue
		}
		fn := results[i].FileName
		if fn == "" {
			fn = results[i].Title
		}
		candidates = append(candidates, cand{result: results[i], parsed: parseFilename(fn)})
	}

	if len(candidates) == 0 {
		return nil
	}

	// preferredQuality is a preference, not a hard ceiling. Releases at or below the
	// cap are chosen first (best quality within budget); only when none exist do we
	// fall back to the closest release above the cap, so a show with only
	// higher-than-preferred releases still downloads instead of being stuck missing.
	sort.Slice(candidates, func(i, j int) bool {
		ri, rj := qualityRank(candidates[i].parsed.ScreenSize), qualityRank(candidates[j].parsed.ScreenSize)
		wi, wj := ri <= cap, rj <= cap
		if wi != wj {
			return wi // within-cap sorts ahead of above-cap
		}

		if rgCfg.mode == "strong" {
			rgi, rgj := releaseGroupRank(candidates[i].parsed.ReleaseGroup, rgCfg.order), releaseGroupRank(candidates[j].parsed.ReleaseGroup, rgCfg.order)
			if rgi != rgj {
				return rgi < rgj
			}
		}

		if ri != rj {
			if wi {
				return ri > rj // within cap: higher quality is better
			}
			return ri < rj // above cap: closer to the cap is better
		}

		// Ordered subtitle-language preference: at equal quality, a release whose
		// subtitle language ranks higher in the configured order sorts ahead.
		si, sj := subtitleRank(candidates[i].parsed, subOrder), subtitleRank(candidates[j].parsed, subOrder)
		if si != sj {
			return si < sj
		}

		if rgCfg.mode == "tie-break" {
			rgi, rgj := releaseGroupRank(candidates[i].parsed.ReleaseGroup, rgCfg.order), releaseGroupRank(candidates[j].parsed.ReleaseGroup, rgCfg.order)
			if rgi != rgj {
				return rgi < rgj
			}
		}

		return candidates[i].result.Seeds > candidates[j].result.Seeds
	})

	return &candidates[0].result
}

func (s *IndexerService) matchThreshold() float64 {
	raw := config.Get(s.db, "matchThreshold", "80")
	n, err := strconv.Atoi(raw)
	if err != nil || n < 0 || n > 100 {
		return 80
	}
	return float64(n)
}

func (s *IndexerService) minSeeders() int {
	raw := config.Get(s.db, "minSeeders", "1")
	n, err := strconv.Atoi(raw)
	if err != nil || n < 0 {
		return 1
	}
	return n
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
		Set("status = 'queued', updated_at = CURRENT_TIMESTAMP").
		Where("id = ?", mon.ID).
		Exec(ctx)
	if err != nil {
		slog.Error("Failed to update monitor status to queued", "id", mon.ID, "error", err)
	}

	slog.Info("Queued download", "monitor_id", mon.ID, "torrent", best.Title)
	return true
}

// ── process monitors ──────────────────────────────────────────────────────────

// processMonitors picks work for one pass of the poll loop. A triggered pass
// (forceRetry) re-searches every monitored missing monitor immediately,
// bypassing the retry-interval gate; a scheduled pass processes at most one
// monitor whose time has come.
func (s *IndexerService) processMonitors(ctx context.Context, forceRetry bool) bool {
	if forceRetry {
		return s.retryMissing(ctx)
	}
	return s.processOne(ctx)
}

func (s *IndexerService) processOne(ctx context.Context) bool {
	// Re-search items marked 'missing' (no qualifying torrent found earlier) once
	// the configured retry interval has elapsed, so temporarily-unavailable
	// releases (no seeders, not yet aired) are picked up later.
	missingRetryMin := config.GetMinutes(s.db, "missingRetryInterval", 1440*time.Minute, time.Minute).Minutes()

	var monitors []models.Monitor
	err := s.db.NewSelect().
		Model(&monitors).
		Where("monitored = true AND available = false AND ("+
			"status = 'pending' "+
			"OR (status = 'searching' AND updated_at < now() - interval '10 minutes') "+
			"OR (status = 'missing' AND updated_at < now() - (interval '1 minute' * ?))"+
			") AND deleted_at IS NULL", missingRetryMin).
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
	return s.claimAndSearch(ctx, monitors[0])
}

// retryMissing re-searches every monitored missing monitor right away. It
// runs when the poll loop is woken by a trigger, so "Run now" produces an
// immediate search instead of waiting for the configured retry interval.
func (s *IndexerService) retryMissing(ctx context.Context) bool {
	var monitors []models.Monitor
	err := s.db.NewSelect().
		Model(&monitors).
		Where("monitored = true AND available = false AND status = 'missing' AND deleted_at IS NULL").
		OrderExpr("is_season DESC, updated_at ASC").
		Scan(ctx)
	if err != nil {
		slog.Error("Failed to fetch missing monitors for retry", "error", err)
		return false
	}
	if len(monitors) == 0 {
		slog.Info("No missing monitors to retry")
		return false
	}

	didWork := false
	for _, mon := range monitors {
		if s.claimAndSearch(ctx, mon) {
			didWork = true
		}
	}
	return didWork
}

// claimAndSearch claims a monitor and searches for a release, recording the
// process_missing cycle when the monitor was missing. Returns true when the
// monitor was claimed.
func (s *IndexerService) claimAndSearch(ctx context.Context, mon models.Monitor) bool {
	if mon.Title == "" || mon.LibraryID == 0 {
		return false
	}
	title := mon.Title
	libraryID := int64(mon.LibraryID)
	slog.Info("Processing monitor", "id", mon.ID, "title", title)

	res, err := s.db.NewUpdate().
		Model((*models.Monitor)(nil)).
		Set("status = 'searching', updated_at = CURRENT_TIMESTAMP").
		Where("id = ? AND status IN ('pending', 'searching', 'missing')", mon.ID).
		Exec(ctx)
	if err != nil {
		slog.Error("Failed to claim monitor", "id", mon.ID, "error", err)
		return false
	}
	if n, _ := res.RowsAffected(); n == 0 {
		slog.Info("Monitor already claimed by another poll, skipping", "id", mon.ID)
		return false
	}

	// Record process_missing cycle only when a missing monitor is actually processed.
	if mon.Status == "missing" {
		rec := cycle.NewRecorder(s.db)
		missingCycle := cycle.Cycle{Service: "indexer", Cycle: "process_missing", DisplayName: "Missing search retry"}
		missingRetryMin := config.GetMinutes(s.db, "missingRetryInterval", 1440*time.Minute, time.Minute)
		_ = rec.Start(ctx, missingCycle)
		_ = rec.End(ctx, missingCycle, time.Now().Add(missingRetryMin))
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
				Set("status = 'pending', updated_at = CURRENT_TIMESTAMP").
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
		s.markMissing(ctx, mon)
		return
	}

	limit := s.cacheFileLimit()

	ep := int(req.EpisodeNumber)
	if req.IsSeason {
		ep = -1
	}

	// Query every enabled provider and merge their qualifying candidates so the
	// best release across all indexers is chosen, rather than the first provider
	// that returns anything.
	var allResults []models.TorrentResult
	var candidates []models.TorrentResult
	seen := map[string]bool{}
	for _, p := range providers {
		results, err := p.Search(ctx, s.db, req)
		if err != nil {
			slog.Warn("Provider search failed", "provider", p.Name(), "monitor_id", mon.ID, "error", err)
			continue
		}

		slog.Info("Provider search complete", "provider", p.Name(), "monitor_id", mon.ID, "results", len(results))
		allResults = append(allResults, results...)

		matchLog := buildMatchLog(results, req.AllTitles(), req.Threshold, int(req.Season), ep, req.EpisodeCount, p.Prematched())
		debugKey := fmt.Sprintf("%s_%s_s%d_e%d", p.Name(), req.Title, req.Season, req.EpisodeNumber)
		saveMatchingDebug(debugKey, matchLog, limit)

		for _, e := range matchLog {
			if !e.Passed {
				continue
			}
			for i := range results {
				if results[i].Title == e.TorrentTitle {
					if key := results[i].DownloadURL; key == "" || !seen[key] {
						if key != "" {
							seen[key] = true
						}
						candidates = append(candidates, results[i])
					}
					break
				}
			}
		}
	}

	go saveParserDebugForQuery(req.Title, allResults, limit)

	if best := s.pickBest(ctx, candidates); best != nil {
		slog.Info("Selected torrent", "provider", best.Indexer, "monitor_id", mon.ID, "torrent", best.Title, "seeds", best.Seeds, "size_mb", best.Size/1024/1024)
		if s.queueDownload(ctx, mon, best) && req.IsSeason {
			s.db.NewUpdate().
				Model((*models.Monitor)(nil)).
				Set("status = 'queued', updated_at = CURRENT_TIMESTAMP").
				Where("library_id = ? AND is_episode = true AND monitored = true AND status = 'pending' AND deleted_at IS NULL", mon.LibraryID).
				Exec(ctx)
		}
		return
	}

	slog.Info("No qualifying torrent found, marking missing", "monitor_id", mon.ID, "title", req.Title)
	s.markMissing(ctx, mon)
}

// canMarkMissing reports whether the indexer may still mark a monitor missing
// after a search concludes. It must be false when the monitor's file became
// available while the search was in flight (the availability scanner found it
// on disk) or when the monitor is no longer waiting on the indexer — otherwise
// the stale search result clobbers a newer state and the UI shows "missing"
// for episodes that are on disk.
func canMarkMissing(status string, available bool) bool {
	if available {
		return false
	}
	switch status {
	case "pending", "searching", "missing":
		return true
	}
	return false
}

func (s *IndexerService) markMissing(ctx context.Context, mon models.Monitor) {
	if !canMarkMissing(mon.Status, mon.Available) {
		slog.Info("Skipping mark missing — monitor no longer waiting or already available", "monitor_id", mon.ID)
		return
	}
	// The in-memory snapshot can be stale (a long search lets the scanner mark
	// the file available mid-flight), so the update re-checks available in SQL.
	s.db.NewUpdate().Model((*models.Monitor)(nil)).
		Set("status = 'missing', updated_at = CURRENT_TIMESTAMP").
		Where("id = ? AND available = false", mon.ID).
		Exec(ctx)
}

// effectiveSeason returns the season number for a monitor.
func effectiveSeason(mon models.Monitor, title string) int64 {
	season := int64(1)
	if mon.Season > 0 {
		season = int64(mon.Season)
	}
	if season == 1 {
		if embedded := naming.ExtractSeasonFromTitle(title); embedded > 1 {
			season = int64(embedded)
		}
	}
	return season
}

// buildMatchLog evaluates every result with anitogo+similarity, marks passing ones.
func buildMatchLog(results []models.TorrentResult, titles []string, threshold float64, season, episode, episodeCount int, prematched bool) []MatchEntry {
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
		if !prematched && sim < threshold {
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
