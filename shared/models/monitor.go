package models

// Monitor represents a tracked episode or season
type Monitor struct {
	Model
	LibraryID     uint   `json:"library_id"` // Link to Media.ID
	Title         string `json:"title"`
	EpisodeTitle  string `json:"episode_title"`
	Season        int    `json:"season"`
	EpisodeNumber int    `json:"episode_number"`
	IsEpisode     bool   `json:"is_episode"`
	IsSeason      bool   `json:"is_season"`
	AniDBID       string `json:"anidb_id"`
	Status        string `json:"status"` // monitored, searching, downloading, downloaded, failed
}

// SearchQueue represents an item waiting to be searched in Prowlarr
type SearchQueue struct {
	Model
	LibraryID     uint   `json:"library_id"`
	Title         string `json:"title"`
	EpisodeTitle  string `json:"episode_title"`
	Season        int    `json:"season"`
	EpisodeNumber int    `json:"episode_number"`
	IsEpisode     bool   `json:"is_episode"`
	IsSeason      bool   `json:"is_season"`
	Status        string `json:"status"` // pending, searching, failed
}
