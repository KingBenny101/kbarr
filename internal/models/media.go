package models

import "time"

// Media represents a generic media item (Anime from AniDB or TV/Movie from TMDB)
type Media struct {
	ID              int       `json:"id" db:"id"`
	Title           string    `json:"title" db:"title"`
	TitleOriginal   string    `json:"title_original" db:"title_original"`
	AlternateTitles []string  `json:"alternate_titles" db:"alternate_titles"`
	Description     string    `json:"description" db:"description"`
	Status          string    `json:"status" db:"status"`
	Type            string    `json:"type" db:"type"` // e.g., "anime", "tv", "movie"
	Episodes        int       `json:"episodes" db:"episodes"`
	Seasons         int       `json:"seasons" db:"seasons"`
	Year            int       `json:"year" db:"year"`
	CoverImage      string    `json:"cover_image" db:"cover_image"`
	BannerImage     string    `json:"banner_image" db:"banner_image"`
	ExternalID      string    `json:"external_id" db:"external_id"` // Unified external identifier
	Source          string    `json:"source" db:"source"`           // Source provider, e.g., "anidb", "tmdb"
	AddedAt         time.Time `json:"added_at" db:"added_at"`
	UpdatedAt       time.Time `json:"updated_at" db:"updated_at"`
}
