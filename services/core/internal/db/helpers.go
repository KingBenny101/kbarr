package db

import (
	"fmt"
	"time"

	"github.com/kingbenny101/kbarr/shared/models"
)

func ensureDB() error {
	if DB == nil {
		return fmt.Errorf("database not initialized")
	}
	return nil
}

func stringPtr(value string) *string {
	return &value
}

func int64Ptr(value int64) *int64 {
	return &value
}

func boolPtr(value bool) *bool {
	return &value
}

func timePtr(value time.Time) *time.Time {
	return &value
}

func derefString(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

func derefInt64(value *int64) int64 {
	if value == nil {
		return 0
	}
	return *value
}

func derefBool(value *bool) bool {
	if value == nil {
		return false
	}
	return *value
}

func mediumFromModel(m models.Media) Medium {
	return Medium{
		ID:        int64(m.ID),
		CreatedAt: m.CreatedAt,
		UpdatedAt: m.UpdatedAt,
		Title:     stringPtr(m.Title),
		Source:    stringPtr(m.Source),
		SourceID:  stringPtr(m.SourceID),
		PosterUrl: stringPtr(m.PosterURL),
	}
}

func mediumToModel(m Medium) models.Media {
	return models.Media{
		Model: models.Model{
			ID:        uint(m.ID),
			CreatedAt: m.CreatedAt,
			UpdatedAt: m.UpdatedAt,
			DeletedAt: m.DeletedAt,
		},
		Title:     derefString(m.Title),
		Source:    derefString(m.Source),
		SourceID:  derefString(m.SourceID),
		PosterURL: derefString(m.PosterUrl),
	}
}

func episodeFromModel(e models.Episode, detailedID *int64) Episode {
	return Episode{
		DetailedID: detailedID,
		Source:     stringPtr(e.Source),
		ExternalID: stringPtr(e.ExternalID),
		Type:       int64Ptr(int64(e.Type)),
		EpNo:       stringPtr(e.EpNo),
		Title:      stringPtr(e.Title),
		AirDate:    stringPtr(e.AirDate),
	}
}

func episodeToModel(e Episode) models.Episode {
	return models.Episode{
		Model: models.Model{
			ID:        uint(e.ID),
			CreatedAt: e.CreatedAt,
			UpdatedAt: e.UpdatedAt,
			DeletedAt: e.DeletedAt,
		},
		DetailedID: uint(derefInt64(e.DetailedID)),
		Source:     derefString(e.Source),
		ExternalID: derefString(e.ExternalID),
		Type:       int(derefInt64(e.Type)),
		EpNo:       derefString(e.EpNo),
		Title:      derefString(e.Title),
		AirDate:    derefString(e.AirDate),
	}
}

func detailedFromModel(d models.Detailed) Detailed {
	return Detailed{
		ID:              int64(d.ID),
		CreatedAt:       d.CreatedAt,
		UpdatedAt:       d.UpdatedAt,
		Title:           stringPtr(d.Title),
		Source:          stringPtr(d.Source),
		SourceID:        stringPtr(d.SourceID),
		LibraryID:       int64Ptr(int64(d.LibraryID)),
		AlternateTitles: stringPtr(d.AlternateTitles),
		Description:     stringPtr(d.Description),
		ReleaseDate:     stringPtr(d.ReleaseDate),
		Genres:          stringPtr(d.Genres),
		PosterUrl:       stringPtr(d.PosterURL),
		TotalEpisodes:   int64Ptr(int64(d.TotalEpisodes)),
		TotalSeasons:    int64Ptr(int64(d.TotalSeasons)),
	}
}

func detailedToModel(d Detailed, episodes []Episode) models.Detailed {
	mappedEpisodes := make([]models.Episode, 0, len(episodes))
	for _, episode := range episodes {
		mappedEpisodes = append(mappedEpisodes, episodeToModel(episode))
	}

	return models.Detailed{
		Model: models.Model{
			ID:        uint(d.ID),
			CreatedAt: d.CreatedAt,
			UpdatedAt: d.UpdatedAt,
			DeletedAt: d.DeletedAt,
		},
		Title:           derefString(d.Title),
		Source:          derefString(d.Source),
		SourceID:        derefString(d.SourceID),
		LibraryID:       uint(derefInt64(d.LibraryID)),
		AlternateTitles: derefString(d.AlternateTitles),
		Description:     derefString(d.Description),
		ReleaseDate:     derefString(d.ReleaseDate),
		Genres:          derefString(d.Genres),
		PosterURL:       derefString(d.PosterUrl),
		TotalEpisodes:   int(derefInt64(d.TotalEpisodes)),
		TotalSeasons:    int(derefInt64(d.TotalSeasons)),
		Episodes:        mappedEpisodes,
	}
}

func monitorFromModel(m models.Monitor) Monitor {
	return Monitor{
		ID:            int64(m.ID),
		CreatedAt:     m.CreatedAt,
		UpdatedAt:     m.UpdatedAt,
		LibraryID:     int64Ptr(int64(m.LibraryID)),
		Title:         stringPtr(m.Title),
		EpisodeTitle:  stringPtr(m.EpisodeTitle),
		Season:        int64Ptr(int64(m.Season)),
		EpisodeNumber: int64Ptr(int64(m.EpisodeNumber)),
		IsEpisode:     boolPtr(m.IsEpisode),
		IsSeason:      boolPtr(m.IsSeason),
		Source:        stringPtr(m.Source),
		ExternalID:    stringPtr(m.ExternalID),
		Status:        stringPtr(m.Status),
	}
}

func monitorToModel(m Monitor) models.Monitor {
	return models.Monitor{
		Model: models.Model{
			ID:        uint(m.ID),
			CreatedAt: m.CreatedAt,
			UpdatedAt: m.UpdatedAt,
			DeletedAt: m.DeletedAt,
		},
		LibraryID:     uint(derefInt64(m.LibraryID)),
		Title:         derefString(m.Title),
		EpisodeTitle:  derefString(m.EpisodeTitle),
		Season:        int(derefInt64(m.Season)),
		EpisodeNumber: int(derefInt64(m.EpisodeNumber)),
		IsEpisode:     derefBool(m.IsEpisode),
		IsSeason:      derefBool(m.IsSeason),
		Source:        derefString(m.Source),
		ExternalID:    derefString(m.ExternalID),
		Status:        derefString(m.Status),
	}
}

