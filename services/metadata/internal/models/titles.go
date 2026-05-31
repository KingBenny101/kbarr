package models

type TitleEntry struct {
	AID   uint   `xml:"aid,attr"`
	Type  string `xml:"type,attr"`
	Lang  string `xml:"lang,attr"`
	Title string `xml:",chardata"`
}

type AnimeTitlesEntry struct {
	AID    uint         `xml:"aid,attr"`
	Titles []TitleEntry `xml:"title"`
}

type AnimeTitlesDump struct {
	Anime []AnimeTitlesEntry `xml:"anime"`
}

type SearchResult struct {
	AID   uint   `json:"aid"`
	Title string `json:"title"`
	Added bool   `json:"added"`
}

type Media struct {
	ID    uint   `json:"id"`
	Title string `json:"title"`
	AID   uint   `json:"aid"`
}

type Episode struct {
	AniDBID string `json:"anidb_id"`
	Type    int    `json:"type"`
	EpNo    string `json:"ep_no"`
	Title   string `json:"title"`
	AirDate string `json:"air_date"`
}

type Detailed struct {
	AID             uint      `json:"aid"`
	LibraryID       uint      `json:"library_id"`
	Title           string    `json:"title"`
	AlternateTitles string    `json:"alternate_titles"`
	Description     string    `json:"description"`
	ReleaseDate     string    `json:"release_date"`
	Genres          string    `json:"genres"`
	PosterURL       string    `json:"poster_url"`
	TotalEpisodes   int       `json:"total_episodes"`
	TotalSeasons    int       `json:"total_seasons"`
	Episodes        []Episode `json:"episodes"`
}
