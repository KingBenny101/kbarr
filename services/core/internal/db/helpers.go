package db

import (
	"database/sql"
	"fmt"

	dbgen "github.com/kingbenny101/kbarr/services/core/internal/db/generated"
	"github.com/kingbenny101/kbarr/shared/models"
)

func ensureQueries() error {
	if Queries == nil {
		return fmt.Errorf("database queries not initialized")
	}
	return nil
}

func toNullString(value string) sql.NullString {
	return sql.NullString{String: value, Valid: true}
}

func toNullInt64(value int64) sql.NullInt64 {
	return sql.NullInt64{Int64: value, Valid: true}
}

func toNullInt64FromUint(value uint) sql.NullInt64 {
	return sql.NullInt64{Int64: int64(value), Valid: true}
}

func toNullBool(value bool) sql.NullBool {
	return sql.NullBool{Bool: value, Valid: true}
}

func valueString(value sql.NullString) string {
	if value.Valid {
		return value.String
	}
	return ""
}

func valueInt64(value sql.NullInt64) int64 {
	if value.Valid {
		return value.Int64
	}
	return 0
}

func valueBool(value sql.NullBool) bool {
	if value.Valid {
		return value.Bool
	}
	return false
}

func toMediaModel(row dbgen.Medium) models.Media {
	m := models.Media{
		Title:     valueString(row.Title),
		AID:       uint(valueInt64(row.Aid)),
		PosterURL: valueString(row.PosterUrl),
	}
	m.ID = uint(row.ID)
	m.CreatedAt = row.CreatedAt
	m.UpdatedAt = row.UpdatedAt
	return m
}

func toEpisodeModel(row dbgen.Episode) models.Episode {
	ep := models.Episode{
		DetailedID: uint(valueInt64(row.DetailedID)),
		AniDBID:    valueString(row.AnidbID),
		Type:       int(valueInt64(row.Type)),
		EpNo:       valueString(row.EpNo),
		Title:      valueString(row.Title),
		AirDate:    valueString(row.AirDate),
	}
	ep.ID = uint(row.ID)
	ep.CreatedAt = row.CreatedAt
	ep.UpdatedAt = row.UpdatedAt
	return ep
}

func toDetailedModel(row dbgen.Detailed, episodes []dbgen.Episode) models.Detailed {
	mappedEpisodes := make([]models.Episode, 0, len(episodes))
	for _, episode := range episodes {
		mappedEpisodes = append(mappedEpisodes, toEpisodeModel(episode))
	}

	d := models.Detailed{
		Title:           valueString(row.Title),
		AID:             uint(valueInt64(row.Aid)),
		LibraryID:       uint(valueInt64(row.LibraryID)),
		AlternateTitles: valueString(row.AlternateTitles),
		Description:     valueString(row.Description),
		ReleaseDate:     valueString(row.ReleaseDate),
		Genres:          valueString(row.Genres),
		PosterURL:       valueString(row.PosterUrl),
		TotalEpisodes:   int(valueInt64(row.TotalEpisodes)),
		TotalSeasons:    int(valueInt64(row.TotalSeasons)),
		Episodes:        mappedEpisodes,
	}
	d.ID = uint(row.ID)
	d.CreatedAt = row.CreatedAt
	d.UpdatedAt = row.UpdatedAt
	return d
}

func toMonitorModel(row dbgen.Monitor) models.Monitor {
	m := models.Monitor{
		LibraryID:     uint(valueInt64(row.LibraryID)),
		Title:         valueString(row.Title),
		EpisodeTitle:  valueString(row.EpisodeTitle),
		Season:        int(valueInt64(row.Season)),
		EpisodeNumber: int(valueInt64(row.EpisodeNumber)),
		IsEpisode:     valueBool(row.IsEpisode),
		IsSeason:      valueBool(row.IsSeason),
		AniDBID:       valueString(row.AnidbID),
		Status:        valueString(row.Status),
	}
	m.ID = uint(row.ID)
	m.CreatedAt = row.CreatedAt
	m.UpdatedAt = row.UpdatedAt
	return m
}

func toSearchQueueModel(row dbgen.SearchQueue) models.SearchQueue {
	q := models.SearchQueue{
		LibraryID:     uint(valueInt64(row.LibraryID)),
		Title:         valueString(row.Title),
		EpisodeTitle:  valueString(row.EpisodeTitle),
		Season:        int(valueInt64(row.Season)),
		EpisodeNumber: int(valueInt64(row.EpisodeNumber)),
		IsEpisode:     valueBool(row.IsEpisode),
		IsSeason:      valueBool(row.IsSeason),
		Status:        valueString(row.Status),
	}
	q.ID = uint(row.ID)
	q.CreatedAt = row.CreatedAt
	q.UpdatedAt = row.UpdatedAt
	return q
}
