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
	Source        string `json:"source"`
	ExternalID    string `json:"external_id"`
	Status        string `json:"status"` // monitored, searching, downloading, downloaded, missing, unmonitored
	Available     bool   `json:"available"`
}
