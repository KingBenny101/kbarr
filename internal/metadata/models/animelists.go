package models

// AnimeListEntry is a single entry from Fribb/anime-lists anime-list-full.json,
// mapping an AniDB ID to other database IDs. All non-AniDB fields are optional.
type AnimeListEntry struct {
	AniDBID       uint     `json:"anidb_id"`
	AniListID     *int     `json:"anilist_id"`
	TVDBID        *int     `json:"tvdb_id"`
	IMDBID        []string `json:"imdb_id"`
	TMDBID        *TMDBRef `json:"themoviedb_id"`
	MALID         *int     `json:"mal_id"`
	KitsuID       *int     `json:"kitsu_id"`
	AnimePlanetID string   `json:"anime-planet_id"`
	AniSearchID   *int     `json:"anisearch_id"`
}

// TMDBRef holds the themoviedb_id object, which carries either a tv or a movie ID.
// The movie field can be a single integer or an array of integers in the source JSON.
type TMDBRef struct {
	TV    *int  `json:"tv"`
	Movie []int `json:"movie"`
}
