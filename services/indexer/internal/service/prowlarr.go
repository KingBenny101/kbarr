package service

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"os/exec"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
	"unicode"

	"github.com/kingbenny101/kbarr/services/indexer/internal/models"
	"github.com/kingbenny101/kbarr/shared/config"
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

// ── guessit ──────────────────────────────────────────────────────────────────

type rawGuessit struct {
	Title        string          `json:"title"`
	Season       json.RawMessage `json:"season"`
	Episode      json.RawMessage `json:"episode"`
	ScreenSize   string          `json:"screen_size"`
	VideoCodec   string          `json:"video_codec"`
	ColorDepth   string          `json:"color_depth"`
	Source       string          `json:"source"`
	ReleaseGroup string          `json:"release_group"`
	Type         string          `json:"type"`
}

type GuessitResult struct {
	Title        string
	Season       int
	Episode      int
	ScreenSize   string
	VideoCodec   string
	ColorDepth   string
	Source       string
	ReleaseGroup string
	Type         string
}

var guessitRawMemo sync.Map // filename → []byte

// runGuessitRaw runs the guessit binary and returns raw JSON bytes.
// Results are memoized so each unique filename is only parsed once per process.
func runGuessitRaw(filename string) []byte {
	if filename == "" {
		return nil
	}
	if v, ok := guessitRawMemo.Load(filename); ok {
		b, _ := v.([]byte)
		return b
	}
	exe := guessitExe()
	out, err := exec.Command(exe, "-j", "-n", filename).Output()
	if err != nil {
		slog.Warn("guessit failed", "exe", exe, "filename", filename, "error", err)
		guessitRawMemo.Store(filename, []byte(nil))
		return nil
	}
	slog.Debug("guessit output", "filename", filename, "output", string(out))
	guessitRawMemo.Store(filename, out)
	return out
}

func runGuessit(filename string) GuessitResult {
	out := runGuessitRaw(filename)
	if len(out) == 0 {
		return GuessitResult{}
	}
	var raw rawGuessit
	if err := json.Unmarshal(out, &raw); err != nil {
		return GuessitResult{}
	}
	g := GuessitResult{
		Title:        raw.Title,
		ScreenSize:   raw.ScreenSize,
		VideoCodec:   raw.VideoCodec,
		ColorDepth:   raw.ColorDepth,
		Source:       raw.Source,
		ReleaseGroup: raw.ReleaseGroup,
		Type:         raw.Type,
	}
	g.Season = intFromRaw(raw.Season)
	g.Episode = intFromRaw(raw.Episode)
	return g
}

// intFromRaw handles guessit fields that can be either int or []int.
func intFromRaw(raw json.RawMessage) int {
	if len(raw) == 0 {
		return 0
	}
	var n int
	if err := json.Unmarshal(raw, &n); err == nil {
		return n
	}
	var arr []int
	if err := json.Unmarshal(raw, &arr); err == nil && len(arr) > 0 {
		return arr[0]
	}
	return 0
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

// titleSimilarity returns a 0–100 Dice-coefficient similarity between two titles
// after normalisation. It is equivalent in spirit to rapidfuzz token_set_ratio.
func titleSimilarity(a, b string) float64 {
	na, nb := normalizeTitle(a), normalizeTitle(b)
	if na == "" || nb == "" {
		return 0
	}
	if na == nb {
		return 100
	}
	tA := strings.Fields(na)
	tB := strings.Fields(nb)
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
	return float64(2*inter) / float64(len(tA)+len(tB)) * 100
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

	if cached, ok := cacheLoad(s.db, cleanedQuery); ok {
		slog.Info("Prowlarr search (cache hit)", "query", cleanedQuery, "results", len(cached), "file", cacheKey(cleanedQuery))
		go saveGuessitDebugForQuery(cleanedQuery, cached, s.cacheFileLimit())
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
	cacheSave(cleanedQuery, results, s.cacheFileLimit())
	go saveGuessitDebugForQuery(cleanedQuery, results, s.cacheFileLimit())
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

// isGoodSeasonResult returns true if guessit parses the filename, the title
// similarity meets the threshold, and the season number matches.
func (s *IndexerService) isGoodSeasonResult(r models.SearchResult, titles []string, season int, threshold float64) bool {
	filename := r.FileName
	if filename == "" {
		filename = r.Title
	}
	g := runGuessit(filename)
	effectiveSeason := g.Season
	if effectiveSeason == 0 {
		effectiveSeason = 1
	}
	sim := bestTitleSimilarity(g.Title, titles)
	if sim < threshold {
		slog.Debug("Season result rejected (title similarity)", "torrent", r.Title, "guessit_title", g.Title, "similarity", sim, "threshold", threshold)
		return false
	}
	if g.Episode > 0 {
		return false // individual episode, not a season pack
	}
	return effectiveSeason == season
}

// isGoodEpisodeResult returns true if guessit parses the filename, the title
// similarity meets the threshold, and season+episode numbers match.
func (s *IndexerService) isGoodEpisodeResult(r models.SearchResult, titles []string, season, episode int, threshold float64) bool {
	filename := r.FileName
	if filename == "" {
		filename = r.Title
	}
	g := runGuessit(filename)
	effectiveSeason := g.Season
	if effectiveSeason == 0 {
		effectiveSeason = 1
	}
	sim := bestTitleSimilarity(g.Title, titles)
	if sim < threshold {
		slog.Debug("Episode result rejected (title similarity)", "torrent", r.Title, "guessit_title", g.Title, "similarity", sim, "threshold", threshold)
		return false
	}
	return effectiveSeason == season && g.Episode == episode
}

// searchUntilGood queries Prowlarr with each title in order and returns the
// first batch that contains at least one result passing isGood. This avoids
// hitting every alternate title when the first query already yields a match.
func (s *IndexerService) searchUntilGood(ctx context.Context, titles []string, buildQuery func(string) string, isGood func(models.SearchResult) bool) []models.SearchResult {
	for _, title := range titles {
		q := buildQuery(title)
		results, err := s.Search(q)
		if err != nil {
			slog.Warn("Search failed", "query", q, "error", err)
			continue
		}
		for _, r := range results {
			if isGood(r) {
				slog.Info("Good result found", "query", q, "torrent", r.Title)
				return results
			}
		}
		slog.Info("No good result for query, trying next title", "query", q, "results", len(results))
	}
	return nil
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

var episodeInTitlePattern = regexp.MustCompile(`(?i)S\d{2}E\d{2}`)

// seasonQuery searches by title only — anime seasons are separate titles,
// not S01/S02 suffixes. Guessit + alternate-title filtering handles season matching.
func seasonQuery(title string, _ int64) string {
	return title
}

// episodeQuery appends just the episode number in the common anime format ("- 01").
// Prowlarr indexers and guessit both handle this reliably for anime.
func episodeQuery(title string, _ int64, episode int64) string {
	return fmt.Sprintf("%s %02d", title, episode)
}

func isSeasonPack(r models.SearchResult) bool {
	return !episodeInTitlePattern.MatchString(r.Title)
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

	// Claim the monitor atomically. Accept both 'monitored' and 'searching' so that
	// a monitor re-queued after a 10-minute stall can be re-processed.
	res, err := s.db.NewUpdate().
		Model((*models.Monitor)(nil)).
		Set("status = 'searching', updated_at = now()").
		Where("id = ? AND (status = 'monitored' OR status = 'searching')", mon.ID).
		Exec(ctx)
	if err != nil {
		slog.Error("Failed to claim monitor", "id", mon.ID, "error", err)
		return false
	}
	if n, _ := res.RowsAffected(); n == 0 {
		slog.Info("Monitor already claimed by another poll, skipping", "id", mon.ID)
		return false
	}

	// Load alternate titles for this library
	libraryID := int64(0)
	if mon.LibraryID != nil {
		libraryID = *mon.LibraryID
	}
	titles := s.getSearchTitles(ctx, libraryID, title)
	slog.Info("Search titles", "monitor_id", mon.ID, "count", len(titles))

	// Load ALL season monitors to know which libraries already have a season in-progress
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
		case "queued", "downloading", "available":
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

	threshold := s.matchThreshold()
	slog.Info("Match threshold", "threshold", threshold)

	// Season monitors: try to find a full season pack first
	for _, mon := range seasonMons {
		if mon.Title == nil || mon.LibraryID == nil {
			continue
		}
		season := int64(1)
		if mon.Season != nil {
			season = *mon.Season
		}

		// Count episode monitors for this library/season to detect single-episode seasons.
		episodeCount, _ := s.db.NewSelect().
			Model((*models.Monitor)(nil)).
			Where("library_id = ? AND season = ? AND is_episode = true AND deleted_at IS NULL", *mon.LibraryID, season).
			Count(ctx)

		var packs []models.SearchResult
		for _, t := range titles {
			q := seasonQuery(t, season)
			results, err := s.Search(q)
			if err != nil {
				slog.Warn("Search failed", "query", q, "error", err)
				continue
			}
			slog.Info("Season search", "query", q, "results", len(results), "episode_count", episodeCount)

			// Evaluate ALL results with guessit+similarity, sort, save debug file.
			matchLog := buildMatchLog(results, titles, threshold, int(season), -1, episodeCount)
			saveMatchingDebug(q, matchLog, s.cacheFileLimit())

			for _, e := range matchLog {
				if e.Passed && isSeasonPackTitle(e.TorrentTitle) {
					for i := range results {
						if results[i].Title == e.TorrentTitle {
							packs = append(packs, results[i])
							break
						}
					}
				}
			}
			if len(packs) > 0 {
				break
			}
		}
		slog.Info("Season packs after filtering", "monitor_id", mon.ID, "packs", len(packs))

		if best := s.pickBest(ctx, packs); best != nil {
			slog.Info("Season pack selected", "monitor_id", mon.ID, "torrent", best.Title, "seeds", best.Seeds, "size_mb", best.Size/1024/1024)
			if s.queueDownload(ctx, mon, best) {
				seasonFound[*mon.LibraryID] = true
				s.db.NewUpdate().
					Model((*models.Monitor)(nil)).
					Set("status = 'queued', updated_at = now()").
					Where("library_id = ? AND is_episode = true AND status = 'monitored' AND deleted_at IS NULL", *mon.LibraryID).
					Exec(ctx)
			}
		} else {
			slog.Info("No qualifying season pack, falling back to individual episodes", "monitor_id", mon.ID, "title", *mon.Title)
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
				slog.Info("Skipping episode — season pack already queued, resetting to monitored", "monitor_id", mon.ID)
				s.db.NewUpdate().Model((*models.Monitor)(nil)).
					Set("status = 'monitored', updated_at = now()").
					Where("id = ?", mon.ID).Exec(ctx)
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

		var goodEpisodes []models.SearchResult
		for _, t := range titles {
			q := episodeQuery(t, season, episode)
			results, err := s.Search(q)
			if err != nil {
				slog.Warn("Search failed", "query", q, "error", err)
				continue
			}
			slog.Info("Episode search", "query", q, "results", len(results))

			// Evaluate ALL results with guessit+similarity, sort, save debug file.
			matchLog := buildMatchLog(results, titles, threshold, int(season), int(episode), 0)
			saveMatchingDebug(q, matchLog, s.cacheFileLimit())

			for _, e := range matchLog {
				if e.Passed {
					for i := range results {
						if results[i].Title == e.TorrentTitle {
							goodEpisodes = append(goodEpisodes, results[i])
							break
						}
					}
				}
			}
			if len(goodEpisodes) > 0 {
				break
			}
		}
		slog.Info("Episode results", "monitor_id", mon.ID, "matching", len(goodEpisodes))

		if best := s.pickBest(ctx, goodEpisodes); best != nil {
			slog.Info("Episode torrent selected", "monitor_id", mon.ID, "torrent", best.Title, "seeds", best.Seeds, "size_mb", best.Size/1024/1024)
			s.queueDownload(ctx, mon, best)
		} else {
			slog.Info("No qualifying torrent for episode, marking missing", "monitor_id", mon.ID)
			s.db.NewUpdate().Model((*models.Monitor)(nil)).
				Set("status = 'missing', updated_at = now()").
				Where("id = ?", mon.ID).Exec(ctx)
		}
	}
	return true
}

// buildMatchLog evaluates every result with guessit+similarity, marks passing ones,
// and returns the list sorted by similarity descending.
// episode == -1 means season-pack mode (episode number is not checked).
// episodeCount is the total number of episodes in the season (0 = unknown).
// When episodeCount == 1, a torrent with guessit episode == 1 is accepted as a season pack.
func buildMatchLog(results []models.SearchResult, titles []string, threshold float64, season, episode, episodeCount int) []MatchEntry {
	entries := make([]MatchEntry, 0, len(results))
	for _, r := range results {
		filename := r.FileName
		if filename == "" {
			filename = r.Title
		}
		g := runGuessit(filename)
		sim := bestTitleSimilarity(g.Title, titles)
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
		} else if episode < 0 && g.Episode > 0 && !(episodeCount == 1 && g.Episode == 1) {
			e.Reason = fmt.Sprintf("individual episode (ep %d), not a season pack", g.Episode)
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

func isSeasonPackTitle(title string) bool {
	return !episodeInTitlePattern.MatchString(title)
}
