package service

import (
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/kingbenny101/kbarr/internal/config"
	mdmodels "github.com/kingbenny101/kbarr/internal/metadata/models"
)

// animeListsURL is the Fribb/anime-lists full mapping of AniDB IDs to other
// database IDs (TVDB, AniList, IMDB, TMDB, ...). Refreshed roughly weekly.
const animeListsURL = "https://raw.githubusercontent.com/Fribb/anime-lists/master/anime-list-full.json"

// ExternalIDs holds the cross-database IDs we surface for an anime. Each field is
// empty when the mapping has no value for it.
type ExternalIDs struct {
	TVDB        string
	AniList     string
	IMDB        string
	TMDB        string
	MAL         string
	Kitsu       string
	AnimePlanet string
	AniSearch   string
}

// LoadAnimeListsMapping downloads (when stale) and parses the Fribb anime-lists
// mapping into an in-memory anidb_id -> ExternalIDs index. It reuses the AniDB
// sync interval to decide cache freshness so it refreshes alongside the titles dump.
func (s *AniDBService) LoadAnimeListsMapping() error {
	ttl := config.GetMinutes(s.db, "anidbSyncInterval", 1440*time.Minute, time.Minute)
	listsFile := filepath.Join(DataRootDir(), "metadata", "anime-list-full.json")

	if s.shouldDownloadTitles(listsFile, ttl) {
		if err := s.downloadAnimeLists(listsFile); err != nil {
			slog.Error("Failed to download anime-lists mapping", "error", err)
			return err
		}
	} else {
		slog.Info("Anime-lists mapping is fresh, loading from cache")
	}

	return s.parseAnimeLists(listsFile)
}

func (s *AniDBService) downloadAnimeLists(listsFile string) error {
	resp, err := s.httpClient.Get(animeListsURL)
	if err != nil {
		return fmt.Errorf("failed to download anime-lists mapping: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("anime-lists request failed with status %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read anime-lists mapping: %w", err)
	}

	if err := os.WriteFile(listsFile, data, 0644); err != nil {
		return fmt.Errorf("failed to cache anime-lists mapping: %w", err)
	}

	slog.Info("Anime-lists mapping cached to", "path", listsFile)
	return nil
}

func (s *AniDBService) parseAnimeLists(listsFile string) error {
	data, err := os.ReadFile(listsFile)
	if err != nil {
		return fmt.Errorf("failed to read cached anime-lists mapping: %w", err)
	}

	var entries []mdmodels.AnimeListEntry
	if err := json.Unmarshal(data, &entries); err != nil {
		return fmt.Errorf("failed to parse anime-lists mapping: %w", err)
	}

	index := make(map[uint]ExternalIDs, len(entries))
	reverse := make(map[int]uint, len(entries))
	for _, e := range entries {
		if e.AniDBID == 0 {
			continue
		}
		index[e.AniDBID] = externalIDsFromEntry(e)
		if e.AniListID != nil {
			reverse[*e.AniListID] = e.AniDBID
		}
	}

	s.alMu.Lock()
	s.animeLists = index
	s.aniListByID = reverse
	s.alMu.Unlock()

	slog.Info("Anime-lists mapping loaded", "entries", len(index))
	return nil
}

// LookupExternalIDs returns the cross-database IDs for an AniDB ID, or a zero
// value if the mapping has no entry for it (or has not loaded yet).
func (s *AniDBService) LookupExternalIDs(aid uint) ExternalIDs {
	s.alMu.RLock()
	defer s.alMu.RUnlock()
	return s.animeLists[aid]
}

// LookupAniDBByAniList resolves an AniList ID back to its AniDB ID using the
// reverse Fribb index. The bool is false when no mapping exists (e.g. AniList
// titles with no AniDB entry, such as Chinese/Korean content) or the mapping
// has not loaded yet.
func (s *AniDBService) LookupAniDBByAniList(anilistID int) (uint, bool) {
	s.alMu.RLock()
	defer s.alMu.RUnlock()
	aid, ok := s.aniListByID[anilistID]
	return aid, ok
}

// ResolveAniList resolves an AniList ID to an AniDB ID. It first tries the
// direct Fribb mapping; on a miss it falls back to fuzzy-matching the supplied
// titles (e.g. AniList's english/romaji/native titles) against the AniDB title
// index, accepting the best match only if it clears the configurable
// anilistTitleMatchThreshold. matched is "mapping", "title", or "" (not found).
func (s *AniDBService) ResolveAniList(anilistID int, titles []string) (aid uint, matched string) {
	if id, ok := s.LookupAniDBByAniList(anilistID); ok {
		return id, "mapping"
	}

	if len(titles) == 0 {
		return 0, ""
	}

	id, score, ok := s.BestTitleMatch(titles)
	if !ok {
		return 0, ""
	}

	threshold := 90.0
	if v, err := strconv.Atoi(config.Get(s.db, "anilistTitleMatchThreshold", "90")); err == nil {
		threshold = float64(v)
	}
	if score < threshold {
		return 0, ""
	}
	return id, "title"
}

func externalIDsFromEntry(e mdmodels.AnimeListEntry) ExternalIDs {
	ids := ExternalIDs{}
	if e.TVDBID != nil {
		ids.TVDB = strconv.Itoa(*e.TVDBID)
	}
	if e.AniListID != nil {
		ids.AniList = strconv.Itoa(*e.AniListID)
	}
	// imdb_id may be empty or a comma-joined list; take the first value.
	if imdb := strings.TrimSpace(e.IMDBID); imdb != "" {
		ids.IMDB = strings.TrimSpace(strings.Split(imdb, ",")[0])
	}
	if e.TMDBID != nil {
		switch {
		case e.TMDBID.TV != nil:
			ids.TMDB = strconv.Itoa(*e.TMDBID.TV)
		case e.TMDBID.Movie != nil:
			ids.TMDB = strconv.Itoa(*e.TMDBID.Movie)
		}
	}
	if e.MALID != nil {
		ids.MAL = strconv.Itoa(*e.MALID)
	}
	if e.KitsuID != nil {
		ids.Kitsu = strconv.Itoa(*e.KitsuID)
	}
	if ap := strings.TrimSpace(e.AnimePlanetID); ap != "" {
		ids.AnimePlanet = ap
	}
	if e.AniSearchID != nil {
		ids.AniSearch = strconv.Itoa(*e.AniSearchID)
	}
	return ids
}
