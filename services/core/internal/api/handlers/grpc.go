package handlers

import (
	"time"

	"github.com/kingbenny101/kbarr/shared/models"
	indexerpb "github.com/kingbenny101/kbarr/shared/proto/indexer"
	metadatapb "github.com/kingbenny101/kbarr/shared/proto/metadata"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type ProwlarrSearchResult struct {
	Title       string `json:"title"`
	DownloadURL string `json:"downloadUrl"`
	Size        int64  `json:"size"`
	Indexer     string `json:"indexer"`
	Seeds       int    `json:"seeders"`
	Peers       int    `json:"leechers"`
}

func toAniDBMediaProto(media models.Media) *metadatapb.AniDBMedia {
	return &metadatapb.AniDBMedia{
		Id:        uint64(media.ID),
		CreatedAt: toTimestamp(media.CreatedAt),
		UpdatedAt: toTimestamp(media.UpdatedAt),
		DeletedAt: toTimestampPtr(media.DeletedAt),
		Title:     media.Title,
		Aid:       uint64(media.AID),
		PosterUrl: media.PosterURL,
	}
}

func toDetailedModel(detailed *metadatapb.AniDBDetailed) models.Detailed {
	if detailed == nil {
		return models.Detailed{}
	}

	result := models.Detailed{
		Title:           detailed.GetTitle(),
		AID:             uint(detailed.GetAid()),
		LibraryID:       uint(detailed.GetLibraryId()),
		AlternateTitles: detailed.GetAlternateTitles(),
		Description:     detailed.GetDescription(),
		ReleaseDate:     detailed.GetReleaseDate(),
		Genres:          detailed.GetGenres(),
		PosterURL:       detailed.GetPosterUrl(),
		TotalEpisodes:   int(detailed.GetTotalEpisodes()),
		TotalSeasons:    int(detailed.GetTotalSeasons()),
	}
	if detailed.GetId() != 0 {
		result.ID = uint(detailed.GetId())
	}
	result.CreatedAt = fromTimestamp(detailed.GetCreatedAt())
	result.UpdatedAt = fromTimestamp(detailed.GetUpdatedAt())
	result.DeletedAt = fromTimestampPtr(detailed.GetDeletedAt())
	if episodes := detailed.GetEpisodes(); len(episodes) > 0 {
		result.Episodes = make([]models.Episode, 0, len(episodes))
		for _, episode := range episodes {
			if episode == nil {
				continue
			}
			mapped := models.Episode{
				DetailedID: uint(episode.GetDetailedId()),
				AniDBID:    episode.GetAnidbId(),
				Type:       int(episode.GetType()),
				EpNo:       episode.GetEpNo(),
				Title:      episode.GetTitle(),
				AirDate:    episode.GetAirDate(),
			}
			mapped.ID = uint(episode.GetId())
			mapped.CreatedAt = fromTimestamp(episode.GetCreatedAt())
			mapped.UpdatedAt = fromTimestamp(episode.GetUpdatedAt())
			mapped.DeletedAt = fromTimestampPtr(episode.GetDeletedAt())
			result.Episodes = append(result.Episodes, mapped)
		}
	}

	return result
}

func toProwlarrSearchResults(results []*indexerpb.ProwlarrSearchResult) []ProwlarrSearchResult {
	if len(results) == 0 {
		return []ProwlarrSearchResult{}
	}

	out := make([]ProwlarrSearchResult, 0, len(results))
	for _, result := range results {
		if result == nil {
			continue
		}
		out = append(out, ProwlarrSearchResult{
			Title:       result.GetTitle(),
			DownloadURL: result.GetDownloadUrl(),
			Size:        result.GetSize(),
			Indexer:     result.GetIndexer(),
			Seeds:       int(result.GetSeeds()),
			Peers:       int(result.GetPeers()),
		})
	}

	return out
}

func toTimestamp(value time.Time) *timestamppb.Timestamp {
	if value.IsZero() {
		return nil
	}
	return timestamppb.New(value)
}

func toTimestampPtr(value *time.Time) *timestamppb.Timestamp {
	if value == nil || value.IsZero() {
		return nil
	}
	return timestamppb.New(*value)
}

func fromTimestamp(value *timestamppb.Timestamp) time.Time {
	if value == nil {
		return time.Time{}
	}
	return value.AsTime()
}

func fromTimestampPtr(value *timestamppb.Timestamp) *time.Time {
	if value == nil {
		return nil
	}
	timeValue := value.AsTime()
	return &timeValue
}
