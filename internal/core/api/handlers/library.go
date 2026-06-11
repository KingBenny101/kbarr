package handlers

import (
	"context"
	"fmt"
	"log/slog"
	"strconv"
	"strings"
	"time"

	"github.com/danielgtaylor/huma/v2"
	"github.com/kingbenny101/kbarr/internal/core/clients"
	"github.com/kingbenny101/kbarr/internal/core/db"
	"github.com/kingbenny101/kbarr/internal/models"
	"github.com/kingbenny101/kbarr/internal/naming"
)

func AddMedia(mc *clients.MetadataClient) func(context.Context, *AddMediaInput) (*AddMediaOutput, error) {
	return func(ctx context.Context, input *AddMediaInput) (*AddMediaOutput, error) {
		media := models.Media{
			Title:    input.Body.Title,
			Source:   input.Body.Source,
			SourceID: input.Body.SourceID,
		}
		slog.Info("AddMedia called", "title", media.Title, "source", media.Source, "source_id", media.SourceID)

		exists, err := db.CheckMediaExists(media.Source, media.SourceID)
		if err != nil {
			slog.Error("Failed to check media existence", "error", err)
			return nil, huma.Error500InternalServerError("failed to check media existence", err)
		}
		if exists {
			slog.Info("Media already exists", "title", media.Title)
			return &AddMediaOutput{Body: MessageResponse{Message: "Media already exists in library!"}}, nil
		}

		slog.Info("Preparing detailed info", "title", media.Title, "source", media.Source, "source_id", media.SourceID)
		ctx, cancel := context.WithTimeout(ctx, 45*time.Second)
		defer cancel()

		prepared, err := mc.Prepare(ctx, media.Source, media.SourceID, media.Title, 0)
		if err != nil {
			slog.Error("Failed to get anime details from metadata service", "error", err)
			return nil, huma.Error502BadGateway("failed to fetch anime details — check your metadata source settings", err)
		}

		if prepared.PosterURL != "" {
			media.PosterURL = prepared.PosterURL
		}
		media.IsNSFW = prepared.IsNSFW
		media.MediaFolder = resolveMediaFolder(media)

		detailed := toDetailedModel(prepared, 0)

		var monitors []models.Monitor
		for _, ep := range prepared.Episodes {
			epNum, err := strconv.Atoi(ep.Number)
			if err != nil {
				continue
			}
			monitors = append(monitors, models.Monitor{
				Title:         prepared.Title,
				EpisodeTitle:  ep.Title,
				Season:        1,
				EpisodeNumber: epNum,
				IsEpisode:     true,
				IsSeason:      false,
				Source:        ep.Source,
				ExternalID:    ep.ExternalID,
				Status:        "unmonitored",
			})
		}

		// Insert media + detailed + monitors atomically so a failure can't leave a
		// half-added show in the library.
		id, err := db.AddMediaWithMonitors(media, detailed, monitors)
		if err != nil {
			slog.Error("Failed to add media", "title", media.Title, "error", err)
			return nil, huma.Error500InternalServerError("failed to save media", err)
		}
		slog.Info("Media added", "id", id, "title", media.Title, "monitors", len(monitors))

		return &AddMediaOutput{Body: MessageResponse{Message: "Media added successfully!"}}, nil
	}
}

func GetMediaList() func(context.Context, *struct{}) (*MediaListOutput, error) {
	return func(ctx context.Context, _ *struct{}) (*MediaListOutput, error) {
		list, err := db.GetAllMedia()
		if err != nil {
			slog.Error("Failed to fetch media list", "error", err)
			return nil, huma.Error500InternalServerError("failed to fetch media list", err)
		}
		return &MediaListOutput{Body: list}, nil
	}
}

func DeleteMedia() func(context.Context, *LibraryIDInput) (*struct{}, error) {
	return func(ctx context.Context, input *LibraryIDInput) (*struct{}, error) {
		id := strconv.FormatUint(uint64(input.ID), 10)
		slog.Info("Delete media request", "id", id)
		if err := db.DeleteMedia(id); err != nil {
			slog.Error("Failed to delete media", "id", id, "error", err)
			return nil, huma.Error500InternalServerError("failed to delete media", err)
		}
		slog.Info("Media deleted", "id", id)
		return nil, nil
	}
}

func UpdateMonitorStatus() func(context.Context, *UpdateMonitorStatusInput) (*struct{}, error) {
	return func(ctx context.Context, input *UpdateMonitorStatusInput) (*struct{}, error) {
		id := strconv.FormatUint(uint64(input.ID), 10)
		slog.Info("Update monitor status", "id", id, "monitored", input.Body.Monitored)
		if err := db.UpdateMediaMonitorStatus(id, input.Body.Monitored); err != nil {
			slog.Error("Failed to update monitor status", "error", err)
			return nil, huma.Error500InternalServerError("failed to update monitor status", err)
		}
		return nil, nil
	}
}

func GetDetailedByMediaID() func(context.Context, *LibraryIDInput) (*DetailedOutput, error) {
	return func(ctx context.Context, input *LibraryIDInput) (*DetailedOutput, error) {
		slog.Info("Fetch detailed info", "id", input.ID)
		detailed, err := db.GetDetailedByLibraryID(input.ID)
		if err != nil {
			slog.Error("Failed to fetch detailed info", "id", input.ID, "error", err)
			return nil, huma.Error404NotFound("media not found", err)
		}
		media, err := db.GetMediaByID(strconv.FormatUint(uint64(input.ID), 10))
		if err == nil {
			detailed.IsNSFW = media.IsNSFW
		}
		return &DetailedOutput{Body: detailed}, nil
	}
}

func UpdateNSFW() func(context.Context, *UpdateNSFWInput) (*struct{}, error) {
	return func(ctx context.Context, input *UpdateNSFWInput) (*struct{}, error) {
		id := strconv.FormatUint(uint64(input.ID), 10)
		slog.Info("Update NSFW status", "id", id, "nsfw", input.Body.NSFW)
		if err := db.UpdateMediaNSFW(id, input.Body.NSFW); err != nil {
			slog.Error("Failed to update NSFW status", "error", err)
			return nil, huma.Error500InternalServerError("failed to update NSFW status", err)
		}
		return nil, nil
	}
}

func GetEpisodes() func(context.Context, *GetEpisodesInput) (*EpisodesOutput, error) {
	return func(ctx context.Context, input *GetEpisodesInput) (*EpisodesOutput, error) {
		var types []int
		if input.Types != "" {
			for _, part := range strings.Split(input.Types, ",") {
				if t, err := strconv.Atoi(strings.TrimSpace(part)); err == nil {
					types = append(types, t)
				}
			}
		}

		result, err := db.QueryEpisodesByLibraryID(input.ID, db.EpisodeQueryParams{
			Types:     types,
			SortField: input.Sort,
			SortOrder: input.Order,
			Page:      input.Page,
			Limit:     input.Limit,
		})
		if err != nil {
			slog.Error("Failed to query episodes", "id", input.ID, "error", err)
			return nil, huma.Error500InternalServerError("failed to fetch episodes", err)
		}
		return &EpisodesOutput{Body: result}, nil
	}
}

// resolveMediaFolder derives a filesystem-safe folder name for a media row,
// disambiguating when two different shows sanitize to the same name. Since hardlinks
// for a show all land in this folder, a collision would mix two shows' episodes and
// make the availability checker skip the ambiguous folder. The source+source_id pair
// is unique (an identical pair is rejected earlier by CheckMediaExists), so it makes
// a guaranteed-unique suffix.
func resolveMediaFolder(media models.Media) string {
	base := naming.SanitizeFilename(media.Title)
	if base == "" {
		base = media.SourceID
	}
	if taken, err := db.MediaFolderTaken(base); err != nil {
		slog.Warn("media_folder uniqueness check failed; using base name", "folder", base, "error", err)
	} else if taken {
		disambiguated := fmt.Sprintf("%s (%s-%s)", base, media.Source, media.SourceID)
		slog.Info("media_folder collision — disambiguating", "base", base, "resolved", disambiguated)
		return disambiguated
	}
	return base
}

func toDetailedModel(d *models.AnimeMetadata, libraryID uint) models.Detailed {
	if d == nil {
		return models.Detailed{}
	}
	result := models.Detailed{
		Source:          d.Source,
		SourceID:        d.SourceID,
		LibraryID:       libraryID,
		Title:           d.Title,
		AlternateTitles: d.AlternateTitles,
		Description:     d.Description,
		ReleaseDate:     d.ReleaseDate,
		Genres:          d.Genres,
		PosterURL:       d.PosterURL,
		TotalEpisodes:   d.TotalEpisodes,
		TotalSeasons:    d.TotalSeasons,
		TVDBID:          d.TVDBID,
		AniListID:       d.AniListID,
		IMDBID:          d.IMDBID,
		TMDBID:          d.TMDBID,
	}
	if len(d.Episodes) > 0 {
		result.Episodes = make([]models.Episode, 0, len(d.Episodes))
		for _, ep := range d.Episodes {
			result.Episodes = append(result.Episodes, models.Episode{
				Source:     ep.Source,
				ExternalID: ep.ExternalID,
				Type:       ep.Type,
				EpNo:       ep.Number,
				Title:      ep.Title,
				AirDate:    ep.AirDate,
			})
		}
	}
	return result
}
