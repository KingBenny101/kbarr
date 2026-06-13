package models

import "github.com/uptrace/bun"

// Monitor represents a tracked episode or season.
type Monitor struct {
	bun.BaseModel `bun:"table:monitors,alias:mon"`
	Model
	LibraryID     uint   `bun:"library_id" json:"library_id"`
	Title         string `bun:"title" json:"title"`
	EpisodeTitle  string `bun:"episode_title" json:"episode_title"`
	Season        int    `bun:"season" json:"season"`
	EpisodeNumber int    `bun:"episode_number" json:"episode_number"`
	IsEpisode     bool   `bun:"is_episode" json:"is_episode"`
	IsSeason      bool   `bun:"is_season" json:"is_season"`
	Source        string `bun:"source" json:"source"`
	ExternalID    string `bun:"external_id" json:"external_id"`
	Status        string `bun:"status" json:"status"`
	Available     bool   `bun:"available,notnull,default:false" json:"available"`
	Monitored     bool   `bun:"monitored,notnull,default:false" json:"monitored"`
	Quality       string `bun:"quality,notnull,default:''" json:"quality"`
	Subtitles     string `bun:"subtitles,notnull,default:''" json:"subtitles"`
}
