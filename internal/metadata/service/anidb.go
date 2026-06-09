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

	"github.com/kingbenny101/kbarr/internal/metadata/models"
	"github.com/kingbenny101/kbarr/internal/config"
	"github.com/uptrace/bun"
)

const (
	anidbHTTPAPI  = "http://api.anidb.net:9001/httpapi"
	anidbCDN      = "https://cdn.anidb.net/images/main/"
	titlesDumpURL = "https://anidb.net/api/anime-titles.xml.gz"
)

type AniDBService struct {
	db         *bun.DB
	httpClient *http.Client

	mu         sync.RWMutex
	titlesDump *models.AnimeTitlesDump
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

func (s *AniDBService) SearchTitles(query string) ([]models.SearchResult, error) {
	s.mu.RLock()
	dumpReady := s.titlesDump != nil
	s.mu.RUnlock()

	if !dumpReady {
		if err := s.LoadTitlesDump(); err != nil {
			return nil, err
		}
	}

	s.mu.RLock()
	dump := s.titlesDump
	s.mu.RUnlock()

	query = strings.ToLower(strings.TrimSpace(query))
	if query == "" {
		return []models.SearchResult{}, nil
	}

	results := make([]models.SearchResult, 0)
	for _, anime := range dump.Anime {
		for _, t := range anime.Titles {
			if strings.Contains(strings.ToLower(t.Title), query) {
				results = append(results, models.SearchResult{
					Source:   "anidb",
					SourceID: strconv.FormatUint(uint64(anime.AID), 10),
					Title:    t.Title,
				})
				break
			}
		}
	}

	return results, nil
}

func (s *AniDBService) GetAnimeDetails(aid uint) (*models.AnimeDetails, error) {
	client := config.Get(s.db, "anidbClient", "error")
	version := config.Get(s.db, "anidbVersion", "error")
	if err := validateAniDBSettings(client, version); err != nil {
		return nil, err
	}

	ttl := config.GetMinutes(s.db, "anidbSyncInterval", 1440*time.Minute, time.Minute)
	cacheFile := filepath.Join(DataRootDir(), "metadata", "details", fmt.Sprintf("%d.xml", aid))
	if details, ok := s.loadCachedAnimeDetails(cacheFile, ttl); ok {
		return details, nil
	}

	details, raw, err := s.fetchAnimeDetails(aid, client, version)
	if err != nil {
		return nil, err
	}

	if err := os.WriteFile(cacheFile, raw, 0644); err != nil {
		slog.Warn("Failed to write anime details cache", "cacheFile", cacheFile, "error", err)
	}

	return details, nil
}

func (s *AniDBService) PrepareDetailed(aid uint, title string, libraryID uint) (models.AnimeMetadata, error) {
	metadata := models.AnimeMetadata{
		Source:    "anidb",
		SourceID:  strconv.FormatUint(uint64(aid), 10),
		Title:     title,
		LibraryID: libraryID,
	}

	details, err := s.GetAnimeDetails(aid)
	if err != nil {
		return models.AnimeMetadata{}, fmt.Errorf("failed to get anime details: %w", err)
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
		metadata.Episodes = append(metadata.Episodes, models.EpisodeMetadata{
			ExternalID: ep.ID,
			Source:     "anidb",
			Type:       epType,
			Number:     ep.EpNo.Value,
			Title:      title,
			AirDate:    ep.AirDate,
		})
	}

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

	if err := os.WriteFile(titlesFile, data, 0644); err != nil {
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

	var dump models.AnimeTitlesDump
	if err := xml.Unmarshal(data, &dump); err != nil {
		return fmt.Errorf("failed to parse titles dump: %w", err)
	}

	s.mu.Lock()
	s.titlesDump = &dump
	s.mu.Unlock()

	slog.Info("Titles dump loaded", "entries", len(dump.Anime))
	return nil
}

func (s *AniDBService) loadCachedAnimeDetails(cacheFile string, ttl time.Duration) (*models.AnimeDetails, bool) {
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

	var details models.AnimeDetails
	if err := xml.Unmarshal(data, &details); err != nil {
		return nil, false
	}

	slog.Info("Using cached anime details", "aid", details.AID)
	return &details, true
}

func (s *AniDBService) fetchAnimeDetails(aid uint, client, version string) (*models.AnimeDetails, []byte, error) {
	apiURL := fmt.Sprintf("%s?request=anime&client=%s&clientver=%s&protover=1&aid=%d", anidbHTTPAPI, client, version, aid)

	req, err := http.NewRequest(http.MethodGet, apiURL, nil)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("User-Agent", fmt.Sprintf("%s/%s", client, version))

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

	var details models.AnimeDetails
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
