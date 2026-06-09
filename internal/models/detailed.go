package models

// Episode represents a single episode from any metadata source.
type Episode struct {
	Model
	DetailedID uint   `json:"detailed_id"`
	Source     string `json:"source"`
	ExternalID string `json:"external_id"`
	Type       int    `json:"type"`
	EpNo       string `json:"ep_no"`
	Title      string `json:"title"`
	AirDate    string `json:"air_date"`
}

// Detailed represents extended metadata for an anime from any metadata source.
type Detailed struct {
	Model
	Title           string    `json:"title"`
	Source          string    `json:"source"`
	SourceID        string    `json:"source_id"`
	LibraryID       uint      `json:"library_id"`
	AlternateTitles string    `json:"alternate_titles"`
	Description     string    `json:"description"`
	ReleaseDate     string    `json:"release_date"`
	Genres          string    `json:"genres"`
	PosterURL       string    `json:"poster_url"`
	TotalEpisodes   int       `json:"total_episodes"`
	TotalSeasons    int       `json:"total_seasons"`
	Episodes        []Episode `json:"episodes"`
}
