package service

import (
	"sort"
	"strconv"
	"strings"

	mdmodels "github.com/kingbenny101/kbarr/internal/metadata/models"
	imodels "github.com/kingbenny101/kbarr/internal/models"
	"github.com/kingbenny101/kbarr/internal/textmatch"
)

const (
	searchMinChars  = 2
	searchLimit     = 40
	searchThreshold = 55.0 // minimum score (0-100) to include a result
)

// searchEntry is one anime in the precomputed search index: its canonical
// display title plus the normalised forms of all its title variants.
type searchEntry struct {
	AID          uint
	DisplayTitle string
	NormTitles   []string
}

// buildSearchIndex precomputes, for every anime, a canonical display title and
// the de-duplicated normalised forms of all its title variants. Built once when
// the titles dump loads so per-query work is just scoring.
func buildSearchIndex(dump *mdmodels.AnimeTitlesDump) []searchEntry {
	if dump == nil {
		return nil
	}
	index := make([]searchEntry, 0, len(dump.Anime))
	for _, anime := range dump.Anime {
		norms := make([]string, 0, len(anime.Titles))
		seen := map[string]struct{}{}
		for _, t := range anime.Titles {
			n := textmatch.Normalize(t.Title)
			if n == "" {
				continue
			}
			if _, dup := seen[n]; dup {
				continue
			}
			seen[n] = struct{}{}
			norms = append(norms, n)
		}
		if len(norms) == 0 {
			continue
		}
		index = append(index, searchEntry{
			AID:          anime.AID,
			DisplayTitle: canonicalTitle(anime.Titles),
			NormTitles:   norms,
		})
	}
	return index
}

// canonicalTitle picks an English-preferred display title for an anime:
// official English → main (AniDB romaji) → x-jat romaji → first available.
func canonicalTitle(titles []mdmodels.TitleEntry) string {
	var official, main, xjat, first string
	for _, t := range titles {
		if first == "" {
			first = t.Title
		}
		switch {
		case t.Type == "official" && t.Lang == "en" && official == "":
			official = t.Title
		case t.Type == "main" && main == "":
			main = t.Title
		case t.Lang == "x-jat" && xjat == "":
			xjat = t.Title
		}
	}
	switch {
	case official != "":
		return official
	case main != "":
		return main
	case xjat != "":
		return xjat
	default:
		return first
	}
}

// SearchTitles returns library-addable anime ranked by how well their titles
// match the query. It scores every anime by the best of its title variants
// (exact > prefix > substring > fuzzy), dedupes by AniDB ID, and caps the result
// count.
func (s *AniDBService) SearchTitles(query string) ([]imodels.SearchResult, error) {
	if err := s.ensureIndexLoaded(); err != nil {
		return nil, err
	}

	s.mu.RLock()
	index := s.searchIndex
	s.mu.RUnlock()

	q := textmatch.Normalize(query)
	if len([]rune(q)) < searchMinChars {
		return []imodels.SearchResult{}, nil
	}

	type scored struct {
		entry *searchEntry
		score float64
	}
	scoredEntries := make([]scored, 0, 64)

	for i := range index {
		e := &index[i]
		best := 0.0
		for _, nt := range e.NormTitles {
			sc := cheapScore(q, nt)
			if sc > best {
				best = sc
			}
		}
		// Fuzzy fallback only when the cheap pass found no strong (substring)
		// match — keeps common typeahead queries fast, pays the bigram cost only
		// for typo/word-order queries.
		if best < 80 {
			for _, nt := range e.NormTitles {
				if !lengthComparable(q, nt) {
					continue
				}
				if sc := textmatch.SimilarityNorm(q, nt); sc > best {
					best = sc
				}
			}
		}
		if best >= searchThreshold {
			scoredEntries = append(scoredEntries, scored{entry: e, score: best})
		}
	}

	sort.Slice(scoredEntries, func(i, j int) bool {
		if scoredEntries[i].score != scoredEntries[j].score {
			return scoredEntries[i].score > scoredEntries[j].score
		}
		ti, tj := scoredEntries[i].entry.DisplayTitle, scoredEntries[j].entry.DisplayTitle
		if len(ti) != len(tj) {
			return len(ti) < len(tj)
		}
		return ti < tj
	})

	if len(scoredEntries) > searchLimit {
		scoredEntries = scoredEntries[:searchLimit]
	}

	results := make([]imodels.SearchResult, 0, len(scoredEntries))
	for _, se := range scoredEntries {
		results = append(results, imodels.SearchResult{
			Source:   "anidb",
			SourceID: strconv.FormatUint(uint64(se.entry.AID), 10),
			Title:    se.entry.DisplayTitle,
		})
	}
	return results, nil
}

// BestTitleMatch scores the given titles (e.g. an external source's english,
// romaji and native titles) against every anime's title variants and returns
// the single highest-scoring AniDB ID. It applies no threshold — the caller
// decides what score is good enough. ok is false when nothing scored above zero.
func (s *AniDBService) BestTitleMatch(titles []string) (aid uint, score float64, ok bool) {
	if err := s.ensureIndexLoaded(); err != nil {
		return 0, 0, false
	}

	s.mu.RLock()
	index := s.searchIndex
	s.mu.RUnlock()

	norms := make([]string, 0, len(titles))
	for _, t := range titles {
		n := textmatch.Normalize(t)
		if len([]rune(n)) < searchMinChars {
			continue
		}
		norms = append(norms, n)
	}
	if len(norms) == 0 {
		return 0, 0, false
	}

	best := 0.0
	var bestAID uint
	for i := range index {
		e := &index[i]
		for _, q := range norms {
			entryBest := 0.0
			for _, nt := range e.NormTitles {
				if sc := cheapScore(q, nt); sc > entryBest {
					entryBest = sc
				}
			}
			if entryBest < 80 {
				for _, nt := range e.NormTitles {
					if !lengthComparable(q, nt) {
						continue
					}
					if sc := textmatch.SimilarityNorm(q, nt); sc > entryBest {
						entryBest = sc
					}
				}
			}
			if entryBest > best {
				best = entryBest
				bestAID = e.AID
			}
		}
	}

	if best <= 0 {
		return 0, 0, false
	}
	return bestAID, best, true
}

// cheapScore scores a normalised query against a normalised title using only
// cheap string operations: exact, prefix, then substring, weighted by how much
// of the title the query covers.
func cheapScore(q, title string) float64 {
	if q == title {
		return 100
	}
	coverage := float64(len(q)) / float64(len(title))
	if coverage > 1 {
		coverage = 1
	}
	switch {
	case strings.HasPrefix(title, q):
		return 90 + 10*coverage
	case strings.Contains(title, q):
		return 80 + 10*coverage
	}
	return 0
}

// lengthComparable gates the fuzzy pass: skip titles whose length is wildly off
// the query length, since they cannot be a good typo/word-order match anyway.
func lengthComparable(q, title string) bool {
	lq, lt := len(q), len(title)
	if lq == 0 || lt == 0 {
		return false
	}
	longer, shorter := lq, lt
	if shorter > longer {
		longer, shorter = shorter, longer
	}
	return float64(shorter)/float64(longer) >= 0.4
}
