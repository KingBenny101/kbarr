package models

import "github.com/uptrace/bun"

// Episode represents a single episode from any metadata source.
type Episode struct {
	bun.BaseModel `bun:"table:episodes,alias:e"`
	Model
	DetailedID uint   `bun:"detailed_id" json:"detailed_id"`
	Source     string `bun:"source" json:"source"`
	ExternalID string `bun:"external_id" json:"external_id"`
	Type       int    `bun:"type" json:"type"`
	EpNo       string `bun:"ep_no" json:"ep_no"`
	Title      string `bun:"title" json:"title"`
	AirDate    string `bun:"air_date" json:"air_date"`
}

// Detailed represents extended metadata for an anime from any metadata source.
type Detailed struct {
	bun.BaseModel   `bun:"table:detaileds,alias:d"`
	Model
	Title           string    `bun:"title" json:"title"`
	Source          string    `bun:"source" json:"source"`
	SourceID        string    `bun:"source_id" json:"source_id"`
	LibraryID       uint      `bun:"library_id" json:"library_id"`
	AlternateTitles string    `bun:"alternate_titles" json:"alternate_titles"`
	Description     string    `bun:"description" json:"description"`
	ReleaseDate     string    `bun:"release_date" json:"release_date"`
	EndDate         string    `bun:"end_date,notnull,default:''" json:"end_date"`
	Genres          string    `bun:"genres" json:"genres"`
	PosterURL       string    `bun:"poster_url" json:"poster_url"`
	TotalEpisodes   int       `bun:"total_episodes" json:"total_episodes"`
	TotalSeasons    int       `bun:"total_seasons" json:"total_seasons"`
	TVDBID          string    `bun:"tvdb_id" json:"tvdb_id"`
	AniListID       string    `bun:"anilist_id" json:"anilist_id"`
	IMDBID          string    `bun:"imdb_id" json:"imdb_id"`
	TMDBID          string    `bun:"tmdb_id" json:"tmdb_id"`
	MALID           string    `bun:"mal_id" json:"mal_id"`
	KitsuID         string    `bun:"kitsu_id" json:"kitsu_id"`
	AnimePlanetID   string    `bun:"animeplanet_id" json:"animeplanet_id"`
	AniSearchID     string    `bun:"anisearch_id" json:"anisearch_id"`
	Episodes        []Episode `bun:"rel:has-many,join:id=detailed_id" json:"episodes"`
	IsNSFW          bool      `json:"is_nsfw" bun:"-"` // set from media, not stored in detaileds table
}
