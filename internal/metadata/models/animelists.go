package models

// AnimeListEntry is a single entry from Fribb/anime-lists anime-list-full.json,
// mapping an AniDB ID to other database IDs. All non-AniDB fields are optional.
type AnimeListEntry struct {
	AniDBID   uint     `json:"anidb_id"`
	AniListID *int     `json:"anilist_id"`
	TVDBID    *int     `json:"tvdb_id"`
	IMDBID    string   `json:"imdb_id"`
	TMDBID    *TMDBRef `json:"themoviedb_id"`
}

// TMDBRef holds the themoviedb_id object, which carries either a tv or a movie ID.
type TMDBRef struct {
	TV    *int `json:"tv"`
	Movie *int `json:"movie"`
}
