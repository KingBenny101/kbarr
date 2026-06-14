package clients

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/kingbenny101/kbarr/internal/models"
)

// errorFromResponse reads the response body and returns a descriptive error.
// It prefers a JSON {"error":"..."} field, falls back to the raw body, and
// falls back further to just the HTTP status.
func errorFromResponse(resp *http.Response) error {
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
	var errResp map[string]string
	if json.Unmarshal(body, &errResp) == nil {
		if msg, ok := errResp["error"]; ok && msg != "" {
			return fmt.Errorf("%s", msg)
		}
	}
	if trimmed := strings.TrimSpace(string(body)); trimmed != "" {
		return fmt.Errorf("metadata service: %s", trimmed)
	}
	return fmt.Errorf("metadata service returned status %d", resp.StatusCode)
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

func (c *MetadataClient) SearchTitles(ctx context.Context, query string) ([]models.SearchResult, error) {
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
		return nil, errorFromResponse(resp)
	}

	var results []models.SearchResult
	if err := json.NewDecoder(resp.Body).Decode(&results); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return results, nil
}

// ResolveAniListID resolves an AniList ID to its AniDB ID via the metadata
// service. It first tries the Fribb reverse index; the optional titles drive a
// fuzzy-match fallback when no direct mapping exists. matched is "mapping",
// "title", or "" (with a nil error) when nothing resolved.
func (c *MetadataClient) ResolveAniListID(ctx context.Context, anilistID int, titles []string) (uint, string, error) {
	endpoint := fmt.Sprintf("%s/resolve/anilist/%d", c.baseURL, anilistID)
	if len(titles) > 0 {
		q := url.Values{}
		for _, t := range titles {
			if t != "" {
				q.Add("title", t)
			}
		}
		if len(q) > 0 {
			endpoint += "?" + q.Encode()
		}
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return 0, "", fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return 0, "", fmt.Errorf("failed to call metadata service: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return 0, "", nil
	}
	if resp.StatusCode != http.StatusOK {
		return 0, "", errorFromResponse(resp)
	}

	var body struct {
		AID     uint   `json:"aid"`
		Matched string `json:"matched"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return 0, "", fmt.Errorf("failed to decode response: %w", err)
	}

	return body.AID, body.Matched, nil
}

func (c *MetadataClient) Prepare(ctx context.Context, source, sourceID, title string, libraryID uint, force bool) (*models.AnimeMetadata, error) {
	body := map[string]any{
		"source":     source,
		"source_id":  sourceID,
		"title":      title,
		"library_id": libraryID,
		"force":      force,
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
		return nil, errorFromResponse(resp)
	}

	var metadata models.AnimeMetadata
	if err := json.NewDecoder(resp.Body).Decode(&metadata); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &metadata, nil
}
