package models

// XML types for the AniDB titles dump (anime-titles.xml.gz)

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

// XML types for the AniDB anime detail API response

type AnimeTitle struct {
	Lang  string `xml:"lang,attr"`
	Type  string `xml:"type,attr"`
	Value string `xml:",chardata"`
}

type RelatedAnime struct {
	ID    string `xml:"id,attr"`
	Type  string `xml:"type,attr"`
	Title string `xml:",chardata"`
}

type SimilarAnime struct {
	ID       string `xml:"id,attr"`
	Approval string `xml:"approval,attr"`
	Total    string `xml:"total,attr"`
	Title    string `xml:",chardata"`
}

type Recommendation struct {
	Type string `xml:"type,attr"`
	UID  string `xml:"uid,attr"`
	Text string `xml:",chardata"`
}

type Creator struct {
	ID   string `xml:"id,attr"`
	Type string `xml:"type,attr"`
	Name string `xml:",chardata"`
}

type VoteRating struct {
	Votes string `xml:"votes,attr"`
	Value string `xml:",chardata"`
}

type CountRating struct {
	Count string `xml:"count,attr"`
	Value string `xml:",chardata"`
}

type Ratings struct {
	Permanent CountRating `xml:"permanent"`
	Temporary CountRating `xml:"temporary"`
	Review    CountRating `xml:"review"`
}

type ResourceExternalEntity struct {
	Identifier []string `xml:"identifier"`
	URL        string   `xml:"url"`
}

type Resource struct {
	Type           string                 `xml:"type,attr"`
	ExternalEntity ResourceExternalEntity `xml:"externalentity"`
}

type Tag struct {
	ID            string `xml:"id,attr"`
	ParentID      string `xml:"parentid,attr"`
	Weight        string `xml:"weight,attr"`
	LocalSpoiler  string `xml:"localspoiler,attr"`
	GlobalSpoiler string `xml:"globalspoiler,attr"`
	Verified      string `xml:"verified,attr"`
	Update        string `xml:"update,attr"`
	Infobox       string `xml:"infobox,attr"`
	Name          string `xml:"name"`
	Description   string `xml:"description"`
	PicURL        string `xml:"picurl"`
}

type Character struct {
	ID            string        `xml:"id,attr"`
	Type          string        `xml:"type,attr"`
	Update        string        `xml:"update,attr"`
	Rating        VoteRating    `xml:"rating"`
	Name          string        `xml:"name"`
	Gender        string        `xml:"gender"`
	CharacterType CharacterType `xml:"charactertype"`
	Description   string        `xml:"description"`
	Picture       string        `xml:"picture"`
	Seiyuu        []Seiyuu      `xml:"seiyuu"`
}

type CharacterType struct {
	ID   string `xml:"id,attr"`
	Type string `xml:",chardata"`
}

type Seiyuu struct {
	ID      string `xml:"id,attr"`
	Picture string `xml:"picture,attr"`
	Name    string `xml:",chardata"`
}

type EpisodeNumber struct {
	Type  string `xml:"type,attr"`
	Value string `xml:",chardata"`
}

type AniDBEpisode struct {
	ID      string        `xml:"id,attr"`
	Update  string        `xml:"update,attr"`
	EpNo    EpisodeNumber `xml:"epno"`
	Length  string        `xml:"length"`
	AirDate string        `xml:"airdate"`
	Rating  VoteRating    `xml:"rating"`
	Titles  []AnimeTitle  `xml:"title"`
}

type AnimeDetails struct {
	AID             uint             `xml:"id,attr"`
	Restricted      string           `xml:"restricted,attr"`
	Type            string           `xml:"type"`
	EpisodeCount    int              `xml:"episodecount"`
	StartDate       string           `xml:"startdate"`
	EndDate         string           `xml:"enddate"`
	Titles          []AnimeTitle     `xml:"titles>title"`
	RelatedAnime    []RelatedAnime   `xml:"relatedanime>anime"`
	SimilarAnime    []SimilarAnime   `xml:"similaranime>anime"`
	Recommendations []Recommendation `xml:"recommendations>recommendation"`
	URL             string           `xml:"url"`
	Creators        []Creator        `xml:"creators>name"`
	Description     string           `xml:"description"`
	Ratings         Ratings          `xml:"ratings"`
	Picture         string           `xml:"picture"`
	Resources       []Resource       `xml:"resources>resource"`
	Tags            []Tag            `xml:"tags>tag"`
	Episodes        []AniDBEpisode   `xml:"episodes>episode"`
	Characters      []Character      `xml:"characters>character"`
}
