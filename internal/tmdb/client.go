package tmdb

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"

	"github.com/kingbenny101/kbarr/internal/config"
	"github.com/kingbenny101/kbarr/internal/logger"
)

const (
	tmdbBaseURL      = "https://api.themoviedb.org/3"
	tmdbImageBaseURL = "https://image.tmdb.org/t/p/w500"
)

type MultiSearchResult struct {
	ID            int     `json:"id"`
	MediaType     string  `json:"media_type"`               // "movie" or "tv"
	Title         string  `json:"title,omitempty"`          // for movies
	Name          string  `json:"name,omitempty"`           // for tv shows
	OriginalTitle string  `json:"original_title,omitempty"` // for movies
	OriginalName  string  `json:"original_name,omitempty"`  // for tv shows
	Overview      string  `json:"overview"`
	PosterPath    string  `json:"poster_path"`
	BackdropPath  string  `json:"backdrop_path"`
	ReleaseDate   string  `json:"release_date,omitempty"`   // for movies
	FirstAirDate  string  `json:"first_air_date,omitempty"` // for tv shows
	VoteAverage   float64 `json:"vote_average"`
}

type MultiSearchResponse struct {
	Page         int                 `json:"page"`
	Results      []MultiSearchResult `json:"results"`
	TotalPages   int                 `json:"total_pages"`
	TotalResults int                 `json:"total_results"`
}

// SearchMulti calls the TMDB multi-search API to find movies and TV shows
func SearchMulti(query string) (*MultiSearchResponse, error) {
	cfg := config.Get()
	if err := checkTMDBSettings(cfg); err != nil {
		logger.Log.Warnf("[TMDB] Skipping search due to invalid settings: %v", err)
		return nil, err
	}

	searchURL := fmt.Sprintf("%s/search/multi?query=%s&include_adult=true&language=en-US&page=1",
		tmdbBaseURL, url.QueryEscape(query),
	)

	req, err := http.NewRequest("GET", searchURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create search request: %w", err)
	}

	req.Header.Add("accept", "application/json")
	req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", cfg.TmDBApiKey))

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to call tmdb search: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("tmdb api returned status: %s", resp.Status)
	}

	var result MultiSearchResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode tmdb search response: %w", err)
	}

	return &result, nil
}

func checkTMDBSettings(cfg *config.Config) error {
	if cfg.TmDBApiKey == "" || cfg.TmDBApiKey == "error" {
		return fmt.Errorf("invalid or missing TMDB API key")
	}
	return nil
}

// GetPosterURL returns the full URL for a TMDB poster path
func GetPosterURL(path string) string {
	if path == "" {
		return ""
	}
	return tmdbImageBaseURL + path
}

// MediaDetails represents the unified details from TMDB (simplified)
type MediaDetails struct {
	ID            int    `json:"id"`
	Title         string `json:"title,omitempty"`          // for movies
	Name          string `json:"name,omitempty"`           // for tv shows
	OriginalTitle string `json:"original_title,omitempty"` // for movies
	OriginalName  string `json:"original_name,omitempty"`  // for tv shows
	Overview      string `json:"overview"`
	PosterPath    string `json:"poster_path"`
	ReleaseDate   string `json:"release_date,omitempty"`   // for movies
	FirstAirDate  string `json:"first_air_date,omitempty"` // for tv shows
}

func (m *MediaDetails) GetTitle() string {
	if m.Title != "" {
		return m.Title
	}
	return m.Name
}

func (m *MediaDetails) GetOriginalTitle() string {
	if m.OriginalTitle != "" {
		return m.OriginalTitle
	}
	return m.OriginalName
}

// GetMediaDetails fetches details for either a movie or a tv show from TMDB
func GetMediaDetails(tmdbID string, mediaType string) (*MediaDetails, error) {
	cfg := config.Get()
	if err := checkTMDBSettings(cfg); err != nil {
		return nil, err
	}

	// endpoint is /movie/{id} or /tv/{id}
	detailURL := fmt.Sprintf("%s/%s/%s?language=en-US", tmdbBaseURL, mediaType, tmdbID)

	req, err := http.NewRequest("GET", detailURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create detail request: %w", err)
	}

	req.Header.Add("accept", "application/json")
	req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", cfg.TmDBApiKey))

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to call tmdb details: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("tmdb api returned status: %s", resp.Status)
	}

	var result MediaDetails
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode tmdb detail response: %w", err)
	}

	return &result, nil
}
