package clients

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"time"
)

type SearchResult struct {
	AID   uint   `json:"aid"`
	Title string `json:"title"`
	Added bool   `json:"added"`
}

type Episode struct {
	AniDBID string `json:"anidb_id"`
	Type    int    `json:"type"`
	EpNo    string `json:"ep_no"`
	Title   string `json:"title"`
	AirDate string `json:"air_date"`
}

type Detailed struct {
	AID             uint      `json:"aid"`
	LibraryID       uint      `json:"library_id"`
	Title           string    `json:"title"`
	AlternateTitles string    `json:"alternate_titles"`
	Description     string    `json:"description"`
	ReleaseDate     string    `json:"release_date"`
	Genres          string    `json:"genres"`
	PosterURL       string    `json:"poster_url"`
	TotalEpisodes   int       `json:"total_episodes"`
	TotalSeasons    int       `json:"total_seasons"`
	Episodes        []Episode `json:"episodes"`
}

type MetadataClient struct {
	baseURL    string
	httpClient *http.Client
}

func NewMetadataClient(baseURL string) *MetadataClient {
	return &MetadataClient{
		baseURL:    baseURL,
		httpClient: &http.Client{Timeout: 45 * time.Second},
	}
}

func (c *MetadataClient) SearchTitles(ctx context.Context, query string) ([]SearchResult, error) {
	endpoint := c.baseURL + "/search?q=" + url.QueryEscape(query)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to call metadata service: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("metadata service returned status %d", resp.StatusCode)
	}

	var results []SearchResult
	if err := json.NewDecoder(resp.Body).Decode(&results); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return results, nil
}

func (c *MetadataClient) Prepare(ctx context.Context, aid uint, title string, libraryID uint) (*Detailed, error) {
	body := map[string]any{
		"aid":        aid,
		"title":      title,
		"library_id": libraryID,
	}
	encoded, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("failed to encode request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/prepare", bytes.NewReader(encoded))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to call metadata service: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		var errResp map[string]string
		_ = json.NewDecoder(resp.Body).Decode(&errResp)
		if msg, ok := errResp["error"]; ok {
			return nil, fmt.Errorf("%s", msg)
		}
		return nil, fmt.Errorf("metadata service returned status %d", resp.StatusCode)
	}

	var detailed Detailed
	if err := json.NewDecoder(resp.Body).Decode(&detailed); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &detailed, nil
}
