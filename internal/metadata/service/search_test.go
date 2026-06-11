package service

import "testing"

import mdmodels "github.com/kingbenny101/kbarr/internal/metadata/models"

func testIndex() []searchEntry {
	dump := &mdmodels.AnimeTitlesDump{
		Anime: []mdmodels.AnimeTitlesEntry{
			{
				AID: 1,
				Titles: []mdmodels.TitleEntry{
					{Type: "main", Lang: "x-jat", Title: "Shingeki no Kyojin"},
					{Type: "official", Lang: "en", Title: "Attack on Titan"},
					{Type: "official", Lang: "ja", Title: "進撃の巨人"},
					{Type: "syn", Lang: "ru", Title: "Атака титанов"},
				},
			},
			{
				AID: 2,
				Titles: []mdmodels.TitleEntry{
					{Type: "main", Lang: "x-jat", Title: "Mahou Shoujo Madoka Magica"},
					{Type: "official", Lang: "en", Title: "Puella Magi Madoka Magica"},
				},
			},
			{
				AID: 3,
				Titles: []mdmodels.TitleEntry{
					{Type: "official", Lang: "en", Title: "Naruto"},
				},
			},
		},
	}
	return buildSearchIndex(dump)
}

func search(t *testing.T, idx []searchEntry, query string) []searchEntry {
	t.Helper()
	s := &AniDBService{searchIndex: idx}
	results, err := s.SearchTitles(query)
	if err != nil {
		t.Fatalf("SearchTitles(%q): %v", query, err)
	}
	// map results back to entries by AID for assertions
	byAID := map[string]searchEntry{}
	for i := range idx {
		byAID[itoa(idx[i].AID)] = idx[i]
	}
	out := make([]searchEntry, 0, len(results))
	for _, r := range results {
		out = append(out, byAID[r.SourceID])
	}
	// stash the display titles on the returned entries for convenience
	for i := range out {
		for _, r := range results {
			if r.SourceID == itoa(out[i].AID) {
				out[i].DisplayTitle = r.Title
			}
		}
	}
	return out
}

func itoa(u uint) string {
	if u == 0 {
		return "0"
	}
	var b []byte
	for u > 0 {
		b = append([]byte{byte('0' + u%10)}, b...)
		u /= 10
	}
	return string(b)
}

func TestSearchTypoRanksTop(t *testing.T) {
	idx := testIndex()
	res := search(t, idx, "atack on titn")
	if len(res) == 0 {
		t.Fatal("expected results for typo query")
	}
	if res[0].AID != 1 {
		t.Errorf("typo query top result AID = %d, want 1 (Attack on Titan)", res[0].AID)
	}
	if res[0].DisplayTitle != "Attack on Titan" {
		t.Errorf("display title = %q, want canonical English 'Attack on Titan'", res[0].DisplayTitle)
	}
}

func TestSearchSubstring(t *testing.T) {
	idx := testIndex()
	res := search(t, idx, "madoka")
	if len(res) == 0 || res[0].AID != 2 {
		t.Fatalf("madoka query top result = %+v, want AID 2", res)
	}
}

func TestSearchForeignVariantReturnsEnglishTitle(t *testing.T) {
	idx := testIndex()
	// query matches the Japanese variant, but the card must show the English title
	res := search(t, idx, "進撃の巨人")
	if len(res) == 0 || res[0].AID != 1 {
		t.Fatalf("japanese query top result = %+v, want AID 1", res)
	}
	if res[0].DisplayTitle != "Attack on Titan" {
		t.Errorf("display title = %q, want 'Attack on Titan'", res[0].DisplayTitle)
	}
}

func TestSearchMinChars(t *testing.T) {
	idx := testIndex()
	s := &AniDBService{searchIndex: idx}
	res, _ := s.SearchTitles("a")
	if len(res) != 0 {
		t.Errorf("single-char query returned %d results, want 0", len(res))
	}
}

func TestSearchDedupAndDisplay(t *testing.T) {
	idx := testIndex()
	res := search(t, idx, "titan")
	// Attack on Titan has multiple variants but must appear once.
	count := 0
	for _, r := range res {
		if r.AID == 1 {
			count++
		}
	}
	if count != 1 {
		t.Errorf("AID 1 appeared %d times, want 1 (deduped)", count)
	}
}
