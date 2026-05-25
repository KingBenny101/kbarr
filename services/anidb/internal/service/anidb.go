package service

import (
	"compress/gzip"
	"database/sql"
	"encoding/xml"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/kingbenny101/kbarr/shared/config"
	"github.com/kingbenny101/kbarr/shared/models"
)

const (
	anidbHTTPAPI  = "http://api.anidb.net:9001/httpapi"
	anidbCDN      = "https://cdn.anidb.net/images/main/"
	titlesDumpURL = "https://anidb.net/api/anime-titles.xml.gz"
)

type AniDBService struct {
	db         *sql.DB
	httpClient *http.Client

	mu         sync.RWMutex
	titlesDump *models.AnimeTitlesDump
}

func New(db *sql.DB) *AniDBService {
	return &AniDBService{
		db:         db,
		httpClient: &http.Client{Timeout: 45 * time.Second},
	}
}

func (s *AniDBService) LoadTitlesDump() error {
	cfg := config.Load(s.db)
	titlesFile := filepath.Join(DataRootDir(), "anidb-titles.xml")

	if s.shouldDownloadTitles(titlesFile, cfg.AniDBInterval) {
		if err := s.downloadTitlesDump(titlesFile, cfg.AniDBClient, cfg.AniDBVersion); err != nil {
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
				results = append(results, models.SearchResult{AID: anime.AID, Title: t.Title})
				break
			}
		}
	}

	return results, nil
}

func (s *AniDBService) GetAnimeDetails(aid uint) (*models.AnimeDetails, error) {
	cfg := config.Load(s.db)
	if err := validateAniDBSettings(cfg.AniDBClient, cfg.AniDBVersion); err != nil {
		return nil, err
	}

	cacheFile := filepath.Join(DataRootDir(), "details", fmt.Sprintf("%d.xml", aid))
	if details, ok := s.loadCachedAnimeDetails(cacheFile, cfg.AniDBInterval); ok {
		return details, nil
	}

	details, raw, err := s.fetchAnimeDetails(aid, cfg)
	if err != nil {
		return nil, err
	}

	if err := os.WriteFile(cacheFile, raw, 0644); err != nil {
		slog.Warn("Failed to write anime details cache", "cacheFile", cacheFile, "error", err)
	}

	return details, nil
}

func (s *AniDBService) PrepareDetailedFromMedia(media *models.Media) (models.Detailed, error) {
	detailed := models.Detailed{
		Title:     media.Title,
		AID:       media.AID,
		LibraryID: media.ID,
	}

	details, err := s.GetAnimeDetails(media.AID)
	if err != nil {
		return models.Detailed{}, fmt.Errorf("failed to get anime details: %w", err)
	}

	detailed.Description = details.Description
	detailed.ReleaseDate = details.StartDate
	detailed.TotalEpisodes = details.EpisodeCount

	if details.Picture != "" {
		detailed.PosterURL = "/api/images/" + details.Picture
		go func(filename string, libraryID uint) {
			imageURL := anidbCDN + filename
			if err := downloadAndSaveImage(s.httpClient, imageURL, filename); err != nil {
				slog.Warn("Failed to download image", "imageURL", imageURL, "error", err)
			} else {
				slog.Info("Cached image for media ID", "mediaID", libraryID, "filename", filename)
			}
		}(details.Picture, media.ID)
	}

	return detailed, nil
}

func (s *AniDBService) shouldDownloadTitles(titlesFile string, titlesCacheMaxAge time.Duration) bool {
	info, err := os.Stat(titlesFile)
	if err != nil {
		return true
	}

	return time.Since(info.ModTime()) > titlesCacheMaxAge
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

func (s *AniDBService) fetchAnimeDetails(aid uint, cfg *config.Config) (*models.AnimeDetails, []byte, error) {
	apiURL := fmt.Sprintf("%s?request=anime&client=%s&clientver=%s&protover=1&aid=%d", anidbHTTPAPI, cfg.AniDBClient, cfg.AniDBVersion, aid)

	req, err := http.NewRequest(http.MethodGet, apiURL, nil)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("User-Agent", fmt.Sprintf("%s/%s", cfg.AniDBClient, cfg.AniDBVersion))

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to call anidb api: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, nil, fmt.Errorf("anidb api returned status %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	raw, err := io.ReadAll(resp.Body)
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
	dest := filepath.Join(DataRootDir(), "images", filename)
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
