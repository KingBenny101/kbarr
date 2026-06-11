package service

import (
	"compress/gzip"
	"encoding/xml"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/kingbenny101/kbarr/internal/config"
	imodels "github.com/kingbenny101/kbarr/internal/models"
	mdmodels "github.com/kingbenny101/kbarr/internal/metadata/models"
	"github.com/uptrace/bun"
)

const (
	anidbHTTPAPI  = "http://api.anidb.net:9001/httpapi"
	anidbCDN      = "https://cdn.anidb.net/images/main/"
	titlesDumpURL = "https://anidb.net/api/anime-titles.xml.gz"

	// minAniDBInterval is the minimum spacing between requests to AniDB. AniDB's
	// flood protection bans clients that exceed roughly one request per 2s; this
	// is a fixed floor (not a setting) so it can't be lowered into a ban.
	minAniDBInterval = 2 * time.Second
	// aniDBBanCooldown is how long we stop calling AniDB after it reports a ban,
	// so a ban halts the hammering instead of amplifying it.
	aniDBBanCooldown = 30 * time.Minute
	// detailsCacheTTL is the lifetime of the per-aid anime details cache. Details
	// are effectively static, so this is long (and independent of anidbSyncInterval,
	// which only governs the titles-dump refresh) to avoid needless AniDB calls —
	// and ban risk — when a show is reopened.
	detailsCacheTTL = 7 * 24 * time.Hour
)

type AniDBService struct {
	db         *bun.DB
	httpClient *http.Client

	mu          sync.RWMutex
	titlesDump  *mdmodels.AnimeTitlesDump
	searchIndex []searchEntry

	// loadMu serializes the lazy titles-dump load so concurrent first-time
	// searches don't each download the multi-MB dump (thundering herd).
	loadMu sync.Mutex

	alMu       sync.RWMutex
	animeLists map[uint]ExternalIDs

	// apiMu serializes AniDB requests so the min-interval pacing holds across
	// concurrent callers; it guards lastCall and bannedUntil.
	apiMu       sync.Mutex
	lastCall    time.Time
	bannedUntil time.Time
}

// throttle blocks until at least minAniDBInterval has elapsed since the previous
// AniDB request, serializing all callers. Stamps the call time at request start.
func (s *AniDBService) throttle() {
	s.apiMu.Lock()
	defer s.apiMu.Unlock()
	if wait := minAniDBInterval - time.Since(s.lastCall); wait > 0 {
		time.Sleep(wait)
	}
	s.lastCall = time.Now()
}

// banCooldownRemaining returns how long the AniDB ban cooldown still has to run,
// or 0 if not currently backing off.
func (s *AniDBService) banCooldownRemaining() time.Duration {
	s.apiMu.Lock()
	defer s.apiMu.Unlock()
	if d := time.Until(s.bannedUntil); d > 0 {
		return d
	}
	return 0
}

// markBanned starts the ban cooldown.
func (s *AniDBService) markBanned() {
	s.apiMu.Lock()
	s.bannedUntil = time.Now().Add(aniDBBanCooldown)
	s.apiMu.Unlock()
}

// ensureIndexLoaded loads the titles dump exactly once across concurrent callers.
// The first caller downloads/parses; the rest wait on loadMu and then observe the
// ready index instead of triggering their own download.
func (s *AniDBService) ensureIndexLoaded() error {
	s.mu.RLock()
	ready := s.searchIndex != nil
	s.mu.RUnlock()
	if ready {
		return nil
	}

	s.loadMu.Lock()
	defer s.loadMu.Unlock()
	// Re-check after acquiring: another goroutine may have loaded it while we waited.
	s.mu.RLock()
	ready = s.searchIndex != nil
	s.mu.RUnlock()
	if ready {
		return nil
	}
	return s.LoadTitlesDump()
}

// atomicWriteFile writes data to path via a temp file in the same directory then
// renames it into place, so a crash mid-write can't leave a truncated file that
// breaks the next load. The temp file shares the destination's directory to keep
// the rename on one filesystem (where rename is atomic).
func atomicWriteFile(path string, data []byte, perm os.FileMode) error {
	tmp, err := os.CreateTemp(filepath.Dir(path), "."+filepath.Base(path)+".tmp-*")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		os.Remove(tmpName)
		return err
	}
	if err := tmp.Close(); err != nil {
		os.Remove(tmpName)
		return err
	}
	if err := os.Chmod(tmpName, perm); err != nil {
		os.Remove(tmpName)
		return err
	}
	if err := os.Rename(tmpName, path); err != nil {
		os.Remove(tmpName)
		return err
	}
	return nil
}

// anidbErrorBody reports whether an AniDB HTTP API response is an <error>…</error>
// document (returned with HTTP 200) and, if so, its message. AniDB signals bans
// and other failures this way rather than via the status code.
func anidbErrorBody(raw []byte) (string, bool) {
	var probe struct {
		XMLName xml.Name `xml:"error"`
		Msg     string   `xml:",chardata"`
	}
	if err := xml.Unmarshal(raw, &probe); err == nil && probe.XMLName.Local == "error" {
		return strings.TrimSpace(probe.Msg), true
	}
	return "", false
}

func New(db *bun.DB) *AniDBService {
	return &AniDBService{
		db:         db,
		httpClient: &http.Client{Timeout: 45 * time.Second},
	}
}

func (s *AniDBService) LoadTitlesDump() error {
	client := config.Get(s.db, "anidbClient", "error")
	version := config.Get(s.db, "anidbVersion", "error")
	ttl := config.GetMinutes(s.db, "anidbSyncInterval", 1440*time.Minute, time.Minute)
	titlesFile := filepath.Join(DataRootDir(), "metadata", "anidb-titles.xml")

	if s.shouldDownloadTitles(titlesFile, ttl) {
		if err := s.downloadTitlesDump(titlesFile, client, version); err != nil {
			slog.Error("Failed to download titles dump", "error", err)
			return err
		}
	} else {
		slog.Info("Titles dump is fresh, loading from cache")
	}

	return s.parseTitlesDump(titlesFile)
}

func (s *AniDBService) GetAnimeDetails(aid uint) (*mdmodels.AnimeDetails, error) {
	client := config.Get(s.db, "anidbClient", "error")
	version := config.Get(s.db, "anidbVersion", "error")
	if err := validateAniDBSettings(client, version); err != nil {
		return nil, err
	}

	cacheFile := filepath.Join(DataRootDir(), "metadata", "details", fmt.Sprintf("%d.xml", aid))
	if details, ok := s.loadCachedAnimeDetails(cacheFile, detailsCacheTTL); ok {
		return details, nil
	}

	// Don't touch AniDB while a ban cooldown is active — serving from cache above
	// is still fine, but a live fetch would only deepen the ban.
	if d := s.banCooldownRemaining(); d > 0 {
		return nil, fmt.Errorf("AniDB temporarily unavailable: rate-limit cooldown, ~%s remaining", d.Round(time.Second))
	}

	details, raw, err := s.fetchAnimeDetails(aid, client, version)
	if err != nil {
		return nil, err
	}

	if err := atomicWriteFile(cacheFile, raw, 0644); err != nil {
		slog.Warn("Failed to write anime details cache", "cacheFile", cacheFile, "error", err)
	}

	return details, nil
}

func (s *AniDBService) PrepareDetailed(aid uint, title string, libraryID uint) (imodels.AnimeMetadata, error) {
	metadata := imodels.AnimeMetadata{
		Source:    "anidb",
		SourceID:  strconv.FormatUint(uint64(aid), 10),
		Title:     title,
		LibraryID: libraryID,
	}

	details, err := s.GetAnimeDetails(aid)
	if err != nil {
		return imodels.AnimeMetadata{}, fmt.Errorf("failed to get anime details: %w", err)
	}

	metadata.Description = details.Description
	metadata.ReleaseDate = details.StartDate
	metadata.TotalEpisodes = details.EpisodeCount

	var altTitles []string
	for _, t := range details.Titles {
		if t.Value != "" {
			altTitles = append(altTitles, t.Value)
		}
	}
	metadata.AlternateTitles = strings.Join(altTitles, "|")

	for _, ep := range details.Episodes {
		title := ""
		for _, t := range ep.Titles {
			if t.Lang == "en" {
				title = t.Value
				break
			}
		}
		if title == "" && len(ep.Titles) > 0 {
			title = ep.Titles[0].Value
		}
		epType, _ := strconv.Atoi(ep.EpNo.Type)
		metadata.Episodes = append(metadata.Episodes, imodels.EpisodeMetadata{
			ExternalID: ep.ID,
			Source:     "anidb",
			Type:       epType,
			Number:     ep.EpNo.Value,
			Title:      title,
			AirDate:    ep.AirDate,
		})
	}

	ids := s.LookupExternalIDs(aid)
	metadata.TVDBID = ids.TVDB
	metadata.AniListID = ids.AniList
	metadata.IMDBID = ids.IMDB
	metadata.TMDBID = ids.TMDB
	metadata.IsNSFW = details.Restricted == "true"

	if details.Picture != "" {
		metadata.PosterURL = "/api/images/" + details.Picture
		go func(filename string, libID uint) {
			imageURL := anidbCDN + filename
			if err := downloadAndSaveImage(s.httpClient, imageURL, filename); err != nil {
				slog.Warn("Failed to download image", "imageURL", imageURL, "error", err)
			} else {
				slog.Info("Cached image for media ID", "mediaID", libID, "filename", filename)
			}
		}(details.Picture, libraryID)
	}

	return metadata, nil
}

func (s *AniDBService) shouldDownloadTitles(titlesFile string, ttl time.Duration) bool {
	info, err := os.Stat(titlesFile)
	if err != nil {
		return true
	}
	return time.Since(info.ModTime()) > ttl
}

func (s *AniDBService) downloadTitlesDump(titlesFile, client, version string) error {
	if err := validateAniDBSettings(client, version); err != nil {
		return err
	}

	req, err := http.NewRequest(http.MethodGet, titlesDumpURL, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("User-Agent", fmt.Sprintf("%s/%s", client, version))

	s.throttle()
	resp, err := s.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to download titles dump: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("titles dump request failed with status %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	gz, err := gzip.NewReader(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to create gzip reader: %w", err)
	}
	defer gz.Close()

	data, err := io.ReadAll(gz)
	if err != nil {
		return fmt.Errorf("failed to read titles dump: %w", err)
	}

	if err := atomicWriteFile(titlesFile, data, 0644); err != nil {
		return fmt.Errorf("failed to cache titles dump: %w", err)
	}

	slog.Info("Titles dump cached to", "path", titlesFile)
	return nil
}

func (s *AniDBService) parseTitlesDump(titlesFile string) error {
	data, err := os.ReadFile(titlesFile)
	if err != nil {
		return fmt.Errorf("failed to read cached titles dump: %w", err)
	}

	var dump mdmodels.AnimeTitlesDump
	if err := xml.Unmarshal(data, &dump); err != nil {
		return fmt.Errorf("failed to parse titles dump: %w", err)
	}

	index := buildSearchIndex(&dump)

	s.mu.Lock()
	s.titlesDump = &dump
	s.searchIndex = index
	s.mu.Unlock()

	slog.Info("Titles dump loaded", "entries", len(dump.Anime), "search_index", len(index))
	return nil
}

func (s *AniDBService) loadCachedAnimeDetails(cacheFile string, ttl time.Duration) (*mdmodels.AnimeDetails, bool) {
	if ttl <= 0 {
		ttl = 24 * time.Hour
	}

	info, err := os.Stat(cacheFile)
	if err != nil {
		return nil, false
	}
	if time.Since(info.ModTime()) > ttl {
		return nil, false
	}

	data, err := os.ReadFile(cacheFile)
	if err != nil {
		return nil, false
	}

	var details mdmodels.AnimeDetails
	if err := xml.Unmarshal(data, &details); err != nil {
		return nil, false
	}

	slog.Info("Using cached anime details", "aid", details.AID)
	return &details, true
}

func (s *AniDBService) fetchAnimeDetails(aid uint, client, version string) (*mdmodels.AnimeDetails, []byte, error) {
	apiURL := fmt.Sprintf("%s?request=anime&client=%s&clientver=%s&protover=1&aid=%d", anidbHTTPAPI, client, version, aid)

	req, err := http.NewRequest(http.MethodGet, apiURL, nil)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("User-Agent", fmt.Sprintf("%s/%s", client, version))

	s.throttle()
	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to call anidb api: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, nil, fmt.Errorf("anidb api returned status %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	var reader io.Reader = resp.Body
	if resp.Header.Get("Content-Encoding") == "gzip" {
		gz, err := gzip.NewReader(resp.Body)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to create gzip reader: %w", err)
		}
		defer gz.Close()
		reader = gz
	}

	raw, err := io.ReadAll(reader)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to read anidb response: %w", err)
	}

	// AniDB returns errors (including bans) as an <error> document with HTTP 200.
	// Detect those before attempting to decode them as anime details.
	if msg, isErr := anidbErrorBody(raw); isErr {
		if strings.Contains(strings.ToLower(msg), "ban") {
			s.markBanned()
			return nil, nil, fmt.Errorf("AniDB has rate-limited or banned this client: %q — backing off for %s", msg, aniDBBanCooldown)
		}
		return nil, nil, fmt.Errorf("AniDB API error: %s", msg)
	}

	var details mdmodels.AnimeDetails
	if err := xml.Unmarshal(raw, &details); err != nil {
		return nil, nil, fmt.Errorf("failed to decode anidb response: %w", err)
	}

	return &details, raw, nil
}

func validateAniDBSettings(client, version string) error {
	if client == "" || client == "error" {
		return fmt.Errorf("invalid AniDB client setting")
	}
	if version == "" || version == "error" {
		return fmt.Errorf("invalid AniDB version setting")
	}
	return nil
}

func downloadAndSaveImage(client *http.Client, imageURL, filename string) error {
	dest := filepath.Join(DataRootDir(), "metadata", "images", filename)
	if _, err := os.Stat(dest); err == nil {
		return nil
	}

	req, err := http.NewRequest(http.MethodGet, imageURL, nil)
	if err != nil {
		return fmt.Errorf("failed to create image request: %w", err)
	}
	req.Header.Set("User-Agent", "Mozilla/5.0")

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to fetch image: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("image request failed with status %d", resp.StatusCode)
	}

	file, err := os.Create(dest)
	if err != nil {
		return fmt.Errorf("failed to create image file: %w", err)
	}
	defer file.Close()

	if _, err := io.Copy(file, resp.Body); err != nil {
		return fmt.Errorf("failed to write image: %w", err)
	}

	return nil
}
