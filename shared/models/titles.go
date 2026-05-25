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
