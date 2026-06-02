package models

type TitleEntry struct {
	AID   uint   `xml:"aid,attr"`
	Type  string `xml:"type,attr"`
	Lang  string `xml:"lang,attr"`
	Title string `xml:",chardata"`
}

type AnimeTitlesEntry struct {
	AID    uint         `xml:"aid,attr"`
	Titles []TitleEntry `xml:"title"`
}

type AnimeTitlesDump struct {
	Anime []AnimeTitlesEntry `xml:"anime"`
}

// SearchResult is a generic search hit returned by any metadata source.
type SearchResult struct {
	Source   string `json:"source"`
	SourceID string `json:"source_id"`
	Title    string `json:"title"`
	Added    bool   `json:"added"`
}

// EpisodeMetadata is a generic episode returned by any metadata source.
type EpisodeMetadata struct {
	ExternalID string `json:"external_id"`
	Source     string `json:"source"`
	Type       int    `json:"type"`
	Number     string `json:"number"`
	Title      string `json:"title"`
	AirDate    string `json:"air_date"`
}

// AnimeMetadata is the generic metadata payload returned by any metadata source.
type AnimeMetadata struct {
	Source          string            `json:"source"`
	SourceID        string            `json:"source_id"`
	LibraryID       uint              `json:"library_id"`
	Title           string            `json:"title"`
	AlternateTitles string            `json:"alternate_titles"`
	Description     string            `json:"description"`
	ReleaseDate     string            `json:"release_date"`
	Genres          string            `json:"genres"`
	PosterURL       string            `json:"poster_url"`
	TotalEpisodes   int               `json:"total_episodes"`
	TotalSeasons    int               `json:"total_seasons"`
	Episodes        []EpisodeMetadata `json:"episodes"`
}
