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
	"sort"
	"strconv"
	"strings"
	"time"
	"unicode"

	"github.com/adrg/strutil/metrics"
	"github.com/kingbenny101/kbarr/services/indexer/internal/models"
	"github.com/kingbenny101/kbarr/shared/config"
	"github.com/kingbenny101/kbarr/shared/parser"
	"github.com/uptrace/bun"
)

type IndexerService struct {
	db         *bun.DB
	httpClient *http.Client
}

func New(db *bun.DB) *IndexerService {
	return &IndexerService{
		db:         db,
		httpClient: &http.Client{Timeout: 5 * time.Minute},
	}
}

// ── parser ────────────────────────────────────────────────────────────────────

func runGuessit(filename string) parser.ParseResult {
	return parser.Parse(filename)
}

// ── title helpers ─────────────────────────────────────────────────────────────

func normalizeTitle(s string) string {
	s = strings.ToLower(s)
	var b strings.Builder
	prevSpace := true
	for _, r := range s {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			b.WriteRune(r)
			prevSpace = false
		} else if !prevSpace {
			b.WriteRune(' ')
			prevSpace = true
		}
	}
	return strings.TrimSpace(b.String())
}

var bigramDice = func() *metrics.SorensenDice {
	m := metrics.NewSorensenDice()
	m.NgramSize = 2
	return m
}()

// titleSimilarity returns a 0–100 similarity between two titles after normalisation.
// It takes the max of:
//   - word-level Dice (symmetric baseline)
//   - word-level overlap (rewards containment, but only when titles are similarly
//     sized — guarded by a 0.6 length ratio to avoid short generic prefixes like
//     "Sword Art Online" scoring 100% against "Sword Art Online II")
//   - character bigram Dice on space-stripped strings (handles romanized Japanese
//     compounding, e.g. "futari inai" vs "futariinai")
func titleSimilarity(a, b string) float64 {
	na, nb := normalizeTitle(a), normalizeTitle(b)
	if na == "" || nb == "" {
		return 0
	}
	if na == nb {
		return 100
	}

	tA, tB := strings.Fields(na), strings.Fields(nb)
	freq := map[string]int{}
	for _, t := range tA {
		freq[t]++
	}
	inter := 0
	for _, t := range tB {
		if freq[t] > 0 {
			inter++
			freq[t]--
		}
	}
	minLen, maxLen := len(tA), len(tB)
	if minLen > maxLen {
		minLen, maxLen = maxLen, minLen
	}

	diceScore := float64(2*inter) / float64(len(tA)+len(tB)) * 100

	// Only promote the overlap score when the shorter title is at least 60% as
	// long as the longer. Below that the parsed title is too short to reliably
	// distinguish related shows sharing a common prefix.
	overlapScore := diceScore
	if float64(minLen)/float64(maxLen) >= 0.6 {
		overlapScore = float64(inter) / float64(minLen) * 100
	}

	// Character bigram Dice on space-stripped strings
	sa := strings.ReplaceAll(na, " ", "")
	sb := strings.ReplaceAll(nb, " ", "")
	charScore := bigramDice.Compare(sa, sb) * 100

	best := diceScore
	if overlapScore > best {
		best = overlapScore
	}
	if charScore > best {
		best = charScore
	}
	return best
}

// bestTitleSimilarity returns the highest similarity between guessedTitle and any candidate.
func bestTitleSimilarity(guessedTitle string, candidates []string) float64 {
	best := 0.0
	for _, c := range candidates {
		if s := titleSimilarity(guessedTitle, c); s > best {
			best = s
		}
	}
	return best
}

// ── Prowlarr search with cache ────────────────────────────────────────────────

func (s *IndexerService) Search(query string) ([]models.SearchResult, error) {
	cleanedQuery := strings.TrimSpace(query)
	if cleanedQuery == "" {
		return []models.SearchResult{}, nil
	}

	if cached, ok := cacheLoad(s.db, cacheDir(), cleanedQuery); ok {
		slog.Info("Prowlarr search (cache hit)", "query", cleanedQuery, "results", len(cached), "file", cacheKey(cleanedQuery))
		go saveParserDebugForQuery(cleanedQuery, cached, s.cacheFileLimit())
		return cached, nil
	}

	slog.Info("Prowlarr search", "query", cleanedQuery)

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

	results := make([]models.SearchResult, 0, len(rawResults))
	for _, item := range rawResults {
		dlURL := item.DownloadURL
		if dlURL == "" {
			dlURL = item.MagnetURL
		}
		results = append(results, models.SearchResult{
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
	cacheSave(cacheDir(), cleanedQuery, results, s.cacheFileLimit())
	go saveParserDebugForQuery(cleanedQuery, results, s.cacheFileLimit())
	return results, nil
}

// ── search with title fallback ────────────────────────────────────────────────

// getSearchTitles returns the main title followed by deduplicated alternate titles.
func (s *IndexerService) getSearchTitles(ctx context.Context, libraryID int64, mainTitle string) []string {
	titles := []string{mainTitle}
	seen := map[string]bool{normalizeTitle(mainTitle): true}

	var det models.Detailed
	if err := s.db.NewSelect().Model(&det).
		Where("library_id = ? AND deleted_at IS NULL", libraryID).
		Limit(1).Scan(ctx); err != nil || det.AlternateTitles == nil {
		return titles
	}

	for _, t := range strings.Split(*det.AlternateTitles, "|") {
		t = strings.TrimSpace(t)
		if t == "" {
			continue
		}
		key := normalizeTitle(t)
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
	seasonInTitleRe = regexp.MustCompile(`(?i)\bseason\s*(\d+)\b`)
	ordinalSeasonRe = regexp.MustCompile(`(?i)\b(\d+)(?:st|nd|rd|th)\s+season\b`)
	westernEpPattern = regexp.MustCompile(`(?i)S\d{1,2}E\d{1,2}`)
)

// isIndividualEpisode returns true if the result is a single episode rather than
// a season pack. It checks anitogo's parsed episode number first, then falls back
// to a SxxExx regex on the raw filename and torrent title — anitogo can miss the
// pattern when non-standard delimiters (e.g. double dashes) surround it.
func isIndividualEpisode(parsedEp int, filename, torrentTitle string) bool {
	return parsedEp > 0 || westernEpPattern.MatchString(filename) || westernEpPattern.MatchString(torrentTitle)
}

// extractSeasonFromTitle returns the season number embedded in a show title,
// e.g. "Grand Blue Dreaming Season 2" → 2, "Attack on Titan 4th Season" → 4.
// Returns 0 if no season pattern is found.
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

// stripSeasonFromTitle removes season indicators ("Season 2", "2nd Season") from a title
// so that base-title similarity comparisons work correctly.
func stripSeasonFromTitle(title string) string {
	s := ordinalSeasonRe.ReplaceAllString(title, "")
	s = seasonInTitleRe.ReplaceAllString(s, "")
	return strings.Join(strings.Fields(s), " ")
}

// expandWithStrippedSeasons appends stripped-season variants to the titles list so
// guessit-parsed titles (which drop "Season N") still score well in similarity checks.
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

// seasonQueries returns query variants for a season pack search, tried in order
// until one yields a usable result. Bare title first; batch/complete keywords as
// fallbacks to find packs that indexers label explicitly.
func seasonQueries(title string) []string {
	return []string{
		title,
		title + " Batch",
		title + " Complete",
	}
}

// episodeQueries returns query variants for a single-episode search, tried in
// order. The dash-space convention is most common in anime releases; bare number
// and E-prefix are fallbacks for releases that use different schemes.
func episodeQueries(title string, episode int64) []string {
	return []string{
		fmt.Sprintf("%s - %02d", title, episode),
		fmt.Sprintf("%s %02d", title, episode),
		fmt.Sprintf("%s E%02d", title, episode),
	}
}


func (s *IndexerService) isBlacklisted(ctx context.Context, name string) bool {
	q := s.db.NewSelect().Model((*models.TorrentBlacklist)(nil)).WhereOr("torrent_name = ?", name)
	exists, _ := q.Exists(ctx)
	return exists
}

// qualityRank maps a guessit screen_size string to a numeric rank (higher = better).
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
	return 0 // unknown
}

// preferredQualityRank returns the rank cap for the configured preferred quality.
// "any" returns MaxInt so all qualities are eligible.
func preferredQualityRank(preferred string) int {
	switch strings.ToLower(preferred) {
	case "any", "":
		return int(^uint(0) >> 1) // MaxInt
	}
	return qualityRank(preferred)
}

// pickBest selects the best candidate from already title/season/episode-filtered results.
//
// Quality preference (from settings):
//   - "any"             → sort by quality rank desc, then seeds; pick first
//   - specific (e.g. "720p") → cap at that quality rank (skip anything higher),
//     then sort by quality rank desc, then seeds; pick first
func (s *IndexerService) pickBest(ctx context.Context, results []models.SearchResult) *models.SearchResult {
	preferred := config.Get(s.db, "preferredQuality", "any")
	cap := preferredQualityRank(preferred)

	var candidates []models.SearchResult
	for i := range results {
		if s.isBlacklisted(ctx, results[i].Title) {
			continue
		}
		filename := results[i].FileName
		if filename == "" {
			filename = results[i].Title
		}
		g := runGuessit(filename)
		rank := qualityRank(g.ScreenSize)
		if rank > cap {
			continue // exceeds quality cap
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
		ri := qualityRank(runGuessit(fi).ScreenSize) // memoized
		rj := qualityRank(runGuessit(fj).ScreenSize)
		if ri != rj {
			return ri > rj // higher quality first
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
	if mon.Title == nil || mon.LibraryID == nil {
		return false
	}
	title := *mon.Title
	libraryID := *mon.LibraryID
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

	provider := config.Get(s.db, "indexerProvider", "prowlarr")
	if provider == "kbdex" {
		s.runKbdexSearch(ctx, mon, title, libraryID)
	} else {
		titles := s.getSearchTitles(ctx, libraryID, title)
		slog.Info("Search titles", "monitor_id", mon.ID, "count", len(titles))
		threshold := s.matchThreshold()
		if mon.IsSeason != nil && *mon.IsSeason {
			s.runSeasonSearch(ctx, mon, title, libraryID, titles, threshold)
		} else {
			s.runEpisodeSearch(ctx, mon, title, libraryID, titles, threshold)
		}
	}
	return true
}

// effectiveSeason returns the season number for a monitor. If the DB season is 1
// but the title embeds a higher number (e.g. "Grand Blue Dreaming Season 2"),
// the embedded value is used — AniDB records such sequels as standalone entries.
func effectiveSeason(mon models.Monitor, title string) int64 {
	season := int64(1)
	if mon.Season != nil && *mon.Season > 0 {
		season = *mon.Season
	}
	if season == 1 {
		if embedded := extractSeasonFromTitle(title); embedded > 1 {
			season = int64(embedded)
		}
	}
	return season
}

// asciiQueryTitles returns the subset of titles that contain only ASCII characters,
// since Prowlarr searches torrent names which are almost always in romanized English.
// Non-ASCII titles (Japanese, Chinese, etc.) are kept for similarity matching inside
// buildMatchLog but skipped as search queries.
// Falls back to the full list if every title is non-ASCII (shouldn't happen in practice).
func asciiQueryTitles(titles []string) []string {
	var out []string
	for _, t := range titles {
		ascii := true
		for _, r := range t {
			if r > 127 {
				ascii = false
				break
			}
		}
		if ascii {
			out = append(out, t)
		}
	}
	if len(out) == 0 {
		return titles
	}
	return out
}

// searchForBest tries each queryTitle then each query variant in order, stopping as
// soon as pickBest returns a usable result. matchTitles (the full set including
// non-ASCII alternates) is passed to buildMatchLog for similarity scoring.
// Variants for a given title are only tried when the previous variant returned zero
// matching candidates — if results came back but failed pickBest that is a
// quality/blacklist problem, so we advance to the next title rather than burning more
// API calls.
func (s *IndexerService) searchForBest(ctx context.Context, queryTitles, matchTitles []string, threshold float64, buildQueries func(string) []string, season, episode, episodeCount int) *models.SearchResult {
	for _, t := range queryTitles {
		for _, q := range buildQueries(t) {
			results, err := s.Search(q)
			if err != nil {
				slog.Warn("Search failed", "query", q, "error", err)
				continue
			}
			slog.Info("Search", "query", q, "results", len(results))
			matchLog := buildMatchLog(results, matchTitles, threshold, season, episode, episodeCount)
			saveMatchingDebug(q, matchLog, s.cacheFileLimit())
			var candidates []models.SearchResult
			for _, e := range matchLog {
				if e.Passed {
					for i := range results {
						if results[i].Title == e.TorrentTitle {
							candidates = append(candidates, results[i])
							break
						}
					}
				}
			}
			if len(candidates) == 0 {
				// No matching results for this variant — try next variant.
				continue
			}
			if best := s.pickBest(ctx, candidates); best != nil {
				return best
			}
			// Candidates exist but all failed pickBest (blacklisted/quality cap).
			// No point trying more query variants for this title.
			break
		}
	}
	return nil
}

func (s *IndexerService) runSeasonSearch(ctx context.Context, mon models.Monitor, title string, libraryID int64, titles []string, threshold float64) {
	season := effectiveSeason(mon, title)

	episodeCount, _ := s.db.NewSelect().
		Model((*models.Monitor)(nil)).
		Where("library_id = ? AND season = ? AND is_episode = true AND deleted_at IS NULL", libraryID, season).
		Count(ctx)

	best := s.searchForBest(ctx, asciiQueryTitles(titles), titles, threshold, func(t string) []string {
		return seasonQueries(t)
	}, int(season), -1, int(episodeCount))

	if best != nil {
		slog.Info("Season pack selected", "monitor_id", mon.ID, "torrent", best.Title, "seeds", best.Seeds, "size_mb", best.Size/1024/1024)
		if s.queueDownload(ctx, mon, best) {
			s.db.NewUpdate().
				Model((*models.Monitor)(nil)).
				Set("status = 'queued', updated_at = now()").
				Where("library_id = ? AND is_episode = true AND monitored = true AND status = 'pending' AND deleted_at IS NULL", libraryID).
				Exec(ctx)
		}
	} else {
		slog.Info("No qualifying season pack — falling back to individual episodes", "monitor_id", mon.ID, "title", title)
		s.db.NewUpdate().
			Model((*models.Monitor)(nil)).
			Set("status = 'missing', updated_at = now()").
			Where("id = ?", mon.ID).
			Exec(ctx)
	}
}

func (s *IndexerService) runEpisodeSearch(ctx context.Context, mon models.Monitor, title string, libraryID int64, titles []string, threshold float64) {
	// If a season pack is already in-progress for this library, hold off and let
	// the downloader resolve episodes from it.
	seasonActive, _ := s.db.NewSelect().
		Model((*models.Monitor)(nil)).
		Where("library_id = ? AND is_season = true AND status IN ('queued','downloading','available') AND deleted_at IS NULL", libraryID).
		Exists(ctx)
	if seasonActive {
		slog.Info("Skipping episode — season pack already in progress, resetting to pending", "monitor_id", mon.ID)
		s.db.NewUpdate().Model((*models.Monitor)(nil)).
			Set("status = 'pending', updated_at = now()").
			Where("id = ?", mon.ID).Exec(ctx)
		return
	}

	season := effectiveSeason(mon, title)
	episode := int64(0)
	if mon.EpisodeNumber != nil {
		episode = *mon.EpisodeNumber
	}

	best := s.searchForBest(ctx, asciiQueryTitles(titles), titles, threshold, func(t string) []string {
		return episodeQueries(t, episode)
	}, int(season), int(episode), 0)

	if best != nil {
		slog.Info("Episode torrent selected", "monitor_id", mon.ID, "torrent", best.Title, "seeds", best.Seeds, "size_mb", best.Size/1024/1024)
		s.queueDownload(ctx, mon, best)
	} else {
		slog.Info("No qualifying torrent for episode, marking missing", "monitor_id", mon.ID)
		s.db.NewUpdate().Model((*models.Monitor)(nil)).
			Set("status = 'missing', updated_at = now()").
			Where("id = ?", mon.ID).Exec(ctx)
	}
}

// buildMatchLog evaluates every result with guessit+similarity, marks passing ones,
// and returns the list sorted by similarity descending.
// episode == -1 means season-pack mode (episode number is not checked).
// episodeCount is the total number of episodes in the season (0 = unknown).
// When episodeCount == 1, a torrent with guessit episode == 1 is accepted as a season pack.
func buildMatchLog(results []models.SearchResult, titles []string, threshold float64, season, episode, episodeCount int) []MatchEntry {
	// Include stripped-season variants so guessit-parsed titles (which drop "Season N")
	// still match our DB titles at full similarity.
	expandedTitles := expandWithStrippedSeasons(titles)
	entries := make([]MatchEntry, 0, len(results))
	for _, r := range results {
		filename := r.FileName
		if filename == "" {
			filename = r.Title
		}
		g := runGuessit(filename)
		sim := bestTitleSimilarity(g.Title, expandedTitles)
		e := MatchEntry{
			TorrentTitle:  r.Title,
			GuessitTitle:  g.Title,
			GuessitSeason: g.Season,
			GuessitEp:     g.Episode,
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

