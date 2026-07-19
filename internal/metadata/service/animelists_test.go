package service

import (
	"encoding/json"
	"testing"

	mdmodels "github.com/kingbenny101/kbarr/internal/metadata/models"
)

func TestExternalIDsFromEntry(t *testing.T) {
	sample := `[
	  {"type":"TV","anidb_id":1,"anilist_id":290,"imdb_id":["tt0286390"],"tvdb_id":72025,"themoviedb_id":{"tv":26209}},
	  {"type":"MOVIE","anidb_id":7,"imdb_id":[],"themoviedb_id":{"movie":[128]}},
	  {"type":"TV","anidb_id":205,"imdb_id":["tt1","tt2"],"themoviedb_id":null},
	  {"type":"OVA","anidb_id":999}
	]`

	var entries []mdmodels.AnimeListEntry
	if err := json.Unmarshal([]byte(sample), &entries); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	got := map[uint]ExternalIDs{}
	for _, e := range entries {
		got[e.AniDBID] = externalIDsFromEntry(e)
	}

	want := map[uint]ExternalIDs{
		1:   {TVDB: "72025", AniList: "290", IMDB: "tt0286390", TMDB: "26209"},
		7:   {TMDB: "128"}, // movie sub-key, empty imdb
		205: {IMDB: "tt1"}, // first in array, null tmdb
		999: {},            // only anidb id present
	}

	for aid, w := range want {
		if got[aid] != w {
			t.Errorf("aid %d: got %+v want %+v", aid, got[aid], w)
		}
	}
}
