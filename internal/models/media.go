package models

import "github.com/uptrace/bun"

// Media represents an anime item in the library.
type Media struct {
	bun.BaseModel `bun:"table:media,alias:m"`
	Model
	Title       string `bun:"title" json:"title"`
	Source      string `bun:"source" json:"source"`
	SourceID    string `bun:"source_id" json:"source_id"`
	PosterURL   string `bun:"poster_url" json:"poster_url"`
	MediaFolder string `bun:"media_folder,notnull,default:''" json:"media_folder"`
	IsNSFW      bool   `bun:"is_nsfw,notnull,default:false" json:"is_nsfw"`
}
