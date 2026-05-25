package models

type Episode struct {
	Model
	DetailedID uint   `json:"detailed_id"` // Foreign key to Detailed.ID
	AniDBID    string `json:"anidb_id"`
	Type       int    `json:"type"`
	EpNo       string `json:"ep_no"`
	Title      string `json:"title"`
	AirDate    string `json:"air_date"`
}

// Detailed represents an anime item from AniDB with extended metadata.
type Detailed struct {
	Model
	Title           string    `json:"title"`
	AID             uint      `json:"aid"`
	LibraryID       uint      `json:"library_id"` // Foreign key to Media.ID
	AlternateTitles string    `json:"alternate_titles"`
	Description     string    `json:"description"`
	ReleaseDate     string    `json:"release_date"`
	Genres          string    `json:"genres"`
	PosterURL       string    `json:"poster_url"`
	TotalEpisodes   int       `json:"total_episodes"`
	TotalSeasons    int       `json:"total_seasons"`
	Episodes        []Episode `json:"episodes"`
}
