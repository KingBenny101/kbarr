package clients

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"github.com/kingbenny101/kbarr/internal/models"
)

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
		return nil, fmt.Errorf("metadata service returned status %d", resp.StatusCode)
	}

	var results []models.SearchResult
	if err := json.NewDecoder(resp.Body).Decode(&results); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return results, nil
}

func (c *MetadataClient) Prepare(ctx context.Context, source, sourceID, title string, libraryID uint) (*models.AnimeMetadata, error) {
	body := map[string]any{
		"source":     source,
		"source_id":  sourceID,
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

	var metadata models.AnimeMetadata
	if err := json.NewDecoder(resp.Body).Decode(&metadata); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &metadata, nil
}
