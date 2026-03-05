package models

import (
	"time"

	"gorm.io/datatypes"
)

// Media represents a generic media item (Anime from AniDB or TV/Movie from TMDB)
type Media struct {
	ID              int                         `json:"id"               gorm:"primaryKey;autoIncrement"`
	Title           string                      `json:"title"`
	TitleOriginal   string                      `json:"title_original"`
	AlternateTitles datatypes.JSONSlice[string] `json:"alternate_titles"`
	Description     string                      `json:"description"`
	Status          string                      `json:"status"           gorm:"default:watching"`
	Type            string                      `json:"type"`
	Episodes        int                         `json:"episodes"         gorm:"default:0"`
	Seasons         int                         `json:"seasons"          gorm:"default:0"`
	Year            int                         `json:"year"`
	CoverImage      string                      `json:"cover_image"`
	BannerImage     string                      `json:"banner_image"`
	ExternalID      string                      `json:"external_id"`
	Source          string                      `json:"source"`
	Monitored       bool                        `json:"monitored"        gorm:"default:false"`
	AddedAt         time.Time                   `json:"added_at"         gorm:"autoCreateTime"`
	UpdatedAt       time.Time                   `json:"updated_at"       gorm:"autoUpdateTime"`
}
