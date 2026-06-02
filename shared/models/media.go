package models

// Media represents an anime item in the library from any metadata source.
type Media struct {
	Model
	Title     string `json:"title"`
	Source    string `json:"source"`
	SourceID  string `json:"source_id"`
	PosterURL string `json:"poster_url"`
}
