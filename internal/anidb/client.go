package anidb

import (
	"encoding/xml"
	"fmt"
	"net/http"

	"github.com/kingbenny101/kbarr/internal/config"
)

const anidbHTTPAPI = "http://api.anidb.net:9001/httpapi"

type AnimeTitle struct {
	Lang  string `xml:"lang,attr"`
	Type  string `xml:"type,attr"`
	Value string `xml:",chardata"`
}

type RelatedAnime struct {
	ID    string `xml:"id,attr"`
	Type  string `xml:"type,attr"`
	Title string `xml:",chardata"`
}

type AnimeDetails struct {
	AID          string         `xml:"id,attr"`
	Type         string         `xml:"type"`
	EpisodeCount int            `xml:"episodecount"`
	StartDate    string         `xml:"startdate"`
	EndDate      string         `xml:"enddate"`
	Titles       []AnimeTitle   `xml:"titles>title"`
	RelatedAnime []RelatedAnime `xml:"relatedanime>anime"`
	Description  string         `xml:"description"`
	Picture      string         `xml:"picture"`
}

func (a *AnimeDetails) MainTitle() string {
	for _, t := range a.Titles {
		if t.Type == "main" {
			return t.Value
		}
	}
	return ""
}

func (a *AnimeDetails) OfficialTitles() []string {
	var titles []string
	for _, t := range a.Titles {
		if t.Type == "official" {
			titles = append(titles, t.Value)
		}
	}
	return titles
}

func GetAnimeDetails(aid string, cfg *config.Config) (*AnimeDetails, error) {
	url := fmt.Sprintf("%s?request=anime&client=%s&clientver=%s&protover=1&aid=%s",
		anidbHTTPAPI, cfg.AniDBClient, cfg.AniDBVersion, aid,
	)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("User-Agent", fmt.Sprintf("%s/%s", cfg.AniDBClient, cfg.AniDBVersion))

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to call anidb: %w", err)
	}
	defer resp.Body.Close()

	var details AnimeDetails
	err = xml.NewDecoder(resp.Body).Decode(&details)
	if err != nil {
		return nil, fmt.Errorf("failed to decode anidb response: %w", err)
	}

	return &details, nil
}