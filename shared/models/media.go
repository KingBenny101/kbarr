package models

// Media represents an anime item from AniDB
type Media struct {
	Model
	Title     string `json:"title"`
	AID       uint   `json:"aid"`
	PosterURL string `json:"poster_url"`
}
