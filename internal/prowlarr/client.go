package prowlarr

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"

	"github.com/kingbenny101/kbarr/internal/config"
)

type SearchResult struct {
	Title       string `json:"title"`
	DownloadURL string `json:"downloadUrl"`
	Size        int64  `json:"size"`
	Indexer     string `json:"indexer"`
	Seeds       int    `json:"seeders"`
	Peers       int    `json:"leechers"`
}

func Search(query string) ([]SearchResult, error) {
	cfg := config.Get()
	if cfg.ProwlarrApiKey == "" || cfg.ProwlarrApiKey == "error" {
		return nil, fmt.Errorf("prowlarr api key not set")
	}

	searchURL := fmt.Sprintf("%s/api/v1/search?query=%s&type=search",
		cfg.ProwlarrUrl, url.QueryEscape(query),
	)

	req, err := http.NewRequest("GET", searchURL, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Add("X-Api-Key", cfg.ProwlarrApiKey)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("prowlarr returned status: %s", resp.Status)
	}

	var results []SearchResult
	if err := json.NewDecoder(resp.Body).Decode(&results); err != nil {
		return nil, err
	}

	return results, nil
}
