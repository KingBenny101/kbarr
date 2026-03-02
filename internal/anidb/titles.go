package anidb

import (
	"compress/gzip"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/kingbenny101/kbarr/internal/config"
	"github.com/kingbenny101/kbarr/internal/logger"
)

const titlesDumpURL = "https://anidb.net/api/anime-titles.xml.gz"
const titlesCacheMaxAge = 24 * time.Hour

var titlesDump *AnimeTitlesDump

type TitleEntry struct {
	AID   string `xml:"aid,attr"`
	Type  string `xml:"type,attr"`
	Lang  string `xml:"lang,attr"`
	Title string `xml:",chardata"`
}

type AnimeTitlesEntry struct {
	AID    string       `xml:"aid,attr"`
	Titles []TitleEntry `xml:"title"`
}

type AnimeTitlesDump struct {
	Anime []AnimeTitlesEntry `xml:"anime"`
}

type SearchResult struct {
	AID   string
	Title string
}

func LoadTitlesDump(cfg *config.Config) error {
	titlesFile := filepath.Join(cfg.CachePath, "kbarr-titles.xml")

	if shouldDownload(titlesFile) {
		err := downloadTitlesDump(titlesFile, cfg)
		if err != nil {
			return err
		}
	} else {
		logger.Log.Info("[AniDB] Titles dump is fresh, loading from cache...")
	}

	return parseTitlesDump(titlesFile)
}

func shouldDownload(titlesFile string) bool {
	info, err := os.Stat(titlesFile)
	if err != nil {
		return true
	}
	return time.Since(info.ModTime()) > titlesCacheMaxAge
}

func downloadTitlesDump(titlesFile string, cfg *config.Config) error {
	logger.Log.Info("[AniDB] Downloading titles dump...")

	req, err := http.NewRequest("GET", titlesDumpURL, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("User-Agent", fmt.Sprintf("%s/%s", cfg.AniDBClient, cfg.AniDBVersion))

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to download titles dump: %w", err)
	}
	defer resp.Body.Close()

	gz, err := gzip.NewReader(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to decompress titles dump: %w", err)
	}
	defer gz.Close()

	data, err := io.ReadAll(gz)
	if err != nil {
		return fmt.Errorf("failed to read titles dump: %w", err)
	}

	err = os.WriteFile(titlesFile, data, 0644)
	if err != nil {
		return fmt.Errorf("failed to cache titles dump: %w", err)
	}

	logger.Log.Infof("[AniDB] Titles dump cached to %s", titlesFile)
	return nil
}

func parseTitlesDump(titlesFile string) error {
	data, err := os.ReadFile(titlesFile)
	if err != nil {
		return fmt.Errorf("failed to read cached titles dump: %w", err)
	}

	var dump AnimeTitlesDump
	err = xml.Unmarshal(data, &dump)
	if err != nil {
		return fmt.Errorf("failed to parse titles dump: %w", err)
	}

	titlesDump = &dump
	logger.Log.Infof("[AniDB] Titles dump loaded: %d anime entries", len(dump.Anime))
	return nil
}

func SearchTitles(query string) []SearchResult {
	if titlesDump == nil {
		logger.Log.Warn("[AniDB] Titles dump not loaded")
		return nil
	}

	query = strings.ToLower(query)
	var results []SearchResult

	for _, anime := range titlesDump.Anime {
		for _, t := range anime.Titles {
			if strings.Contains(strings.ToLower(t.Title), query) {
				results = append(results, SearchResult{
					AID:   anime.AID,
					Title: t.Title,
				})
				break
			}
		}
	}

	return results
}
