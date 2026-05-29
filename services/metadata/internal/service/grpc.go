package service

import (
	"context"
	"time"

	"github.com/kingbenny101/kbarr/shared/models"
	metadatapb "github.com/kingbenny101/kbarr/shared/proto/metadata"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type GRPCServer struct {
	metadatapb.UnimplementedMetadataServiceServer
	svc *AniDBService
}

func NewGRPCServer(svc *AniDBService) *GRPCServer {
	return &GRPCServer{svc: svc}
}

func (s *GRPCServer) SearchTitles(ctx context.Context, req *metadatapb.AniDBSearchTitlesRequest) (*metadatapb.AniDBSearchTitlesResponse, error) {
	results, err := s.svc.SearchTitles(req.GetQuery())
	if err != nil {
		return nil, err
	}

	out := &metadatapb.AniDBSearchTitlesResponse{Results: make([]*metadatapb.AniDBSearchResult, 0, len(results))}
	for _, result := range results {
		out.Results = append(out.Results, &metadatapb.AniDBSearchResult{
			Aid:   uint64(result.AID),
			Title: result.Title,
			Added: result.Added,
		})
	}

	return out, nil
}

func (s *GRPCServer) GetAnimeDetails(ctx context.Context, req *metadatapb.AniDBGetAnimeDetailsRequest) (*metadatapb.AniDBAnimeDetails, error) {
	details, err := s.svc.GetAnimeDetails(uint(req.GetAid()))
	if err != nil {
		return nil, err
	}

	return toAnimeDetailsProto(details), nil
}

func (s *GRPCServer) PrepareDetailedFromMedia(ctx context.Context, req *metadatapb.AniDBMedia) (*metadatapb.AniDBDetailed, error) {
	media := toMediaModel(req)
	detailed, err := s.svc.PrepareDetailedFromMedia(&media)
	if err != nil {
		return nil, err
	}

	return toDetailedProto(detailed), nil
}

func toMediaModel(media *metadatapb.AniDBMedia) models.Media {
	if media == nil {
		return models.Media{}
	}

	result := models.Media{
		Title:     media.GetTitle(),
		AID:       uint(media.GetAid()),
		PosterURL: media.GetPosterUrl(),
	}
	result.ID = uint(media.GetId())
	result.CreatedAt = fromTimestamp(media.GetCreatedAt())
	result.UpdatedAt = fromTimestamp(media.GetUpdatedAt())
	result.DeletedAt = fromTimestampPtr(media.GetDeletedAt())
	return result
}

func toAnimeDetailsProto(details *models.AnimeDetails) *metadatapb.AniDBAnimeDetails {
	if details == nil {
		return nil
	}

	out := &metadatapb.AniDBAnimeDetails{
		Aid:          uint64(details.AID),
		Restricted:   details.Restricted,
		Type:         details.Type,
		EpisodeCount: int32(details.EpisodeCount),
		StartDate:    details.StartDate,
		EndDate:      details.EndDate,
		Url:          details.URL,
		Description:  details.Description,
		Picture:      details.Picture,
	}

	if len(details.Titles) > 0 {
		out.Titles = make([]*metadatapb.AniDBAnimeTitle, 0, len(details.Titles))
		for _, title := range details.Titles {
			out.Titles = append(out.Titles, &metadatapb.AniDBAnimeTitle{Lang: title.Lang, Type: title.Type, Value: title.Value})
		}
	}
	if len(details.RelatedAnime) > 0 {
		out.RelatedAnime = make([]*metadatapb.AniDBRelatedAnime, 0, len(details.RelatedAnime))
		for _, related := range details.RelatedAnime {
			out.RelatedAnime = append(out.RelatedAnime, &metadatapb.AniDBRelatedAnime{Id: related.ID, Type: related.Type, Title: related.Title})
		}
	}
	if len(details.SimilarAnime) > 0 {
		out.SimilarAnime = make([]*metadatapb.AniDBSimilarAnime, 0, len(details.SimilarAnime))
		for _, similar := range details.SimilarAnime {
			out.SimilarAnime = append(out.SimilarAnime, &metadatapb.AniDBSimilarAnime{Id: similar.ID, Approval: similar.Approval, Total: similar.Total, Title: similar.Title})
		}
	}
	if len(details.Recommendations) > 0 {
		out.Recommendations = make([]*metadatapb.AniDBRecommendation, 0, len(details.Recommendations))
		for _, recommendation := range details.Recommendations {
			out.Recommendations = append(out.Recommendations, &metadatapb.AniDBRecommendation{Type: recommendation.Type, Uid: recommendation.UID, Text: recommendation.Text})
		}
	}
	if len(details.Creators) > 0 {
		out.Creators = make([]*metadatapb.AniDBCreator, 0, len(details.Creators))
		for _, creator := range details.Creators {
			out.Creators = append(out.Creators, &metadatapb.AniDBCreator{Id: creator.ID, Type: creator.Type, Name: creator.Name})
		}
	}
	out.Ratings = &metadatapb.AniDBRatings{
		Permanent: &metadatapb.AniDBCountRating{Count: details.Ratings.Permanent.Count, Value: details.Ratings.Permanent.Value},
		Temporary: &metadatapb.AniDBCountRating{Count: details.Ratings.Temporary.Count, Value: details.Ratings.Temporary.Value},
		Review:    &metadatapb.AniDBCountRating{Count: details.Ratings.Review.Count, Value: details.Ratings.Review.Value},
	}
	if len(details.Resources) > 0 {
		out.Resources = make([]*metadatapb.AniDBResource, 0, len(details.Resources))
		for _, resource := range details.Resources {
			out.Resources = append(out.Resources, &metadatapb.AniDBResource{
				Type: resource.Type,
				ExternalEntity: &metadatapb.AniDBResourceExternalEntity{
					Identifier: append([]string(nil), resource.ExternalEntity.Identifier...),
					Url:        resource.ExternalEntity.URL,
				},
			})
		}
	}
	if len(details.Tags) > 0 {
		out.Tags = make([]*metadatapb.AniDBTag, 0, len(details.Tags))
		for _, tag := range details.Tags {
			out.Tags = append(out.Tags, &metadatapb.AniDBTag{
				Id:            tag.ID,
				ParentId:      tag.ParentID,
				Weight:        tag.Weight,
				LocalSpoiler:  tag.LocalSpoiler,
				GlobalSpoiler: tag.GlobalSpoiler,
				Verified:      tag.Verified,
				Update:        tag.Update,
				Infobox:       tag.Infobox,
				Name:          tag.Name,
				Description:   tag.Description,
				PicUrl:        tag.PicURL,
			})
		}
	}
	if len(details.Episodes) > 0 {
		out.Episodes = make([]*metadatapb.AniDBAnimeEpisode, 0, len(details.Episodes))
		for _, episode := range details.Episodes {
			ep := &metadatapb.AniDBAnimeEpisode{
				Id:      episode.ID,
				Update:  episode.Update,
				Length:  episode.Length,
				AirDate: episode.AirDate,
				Rating:  &metadatapb.AniDBVoteRating{Votes: episode.Rating.Votes, Value: episode.Rating.Value},
			}
			ep.EpNo = &metadatapb.AniDBEpisodeNumber{Type: episode.EpNo.Type, Value: episode.EpNo.Value}
			if len(episode.Titles) > 0 {
				ep.Titles = make([]*metadatapb.AniDBAnimeTitle, 0, len(episode.Titles))
				for _, title := range episode.Titles {
					ep.Titles = append(ep.Titles, &metadatapb.AniDBAnimeTitle{Lang: title.Lang, Type: title.Type, Value: title.Value})
				}
			}
			out.Episodes = append(out.Episodes, ep)
		}
	}
	if len(details.Characters) > 0 {
		out.Characters = make([]*metadatapb.AniDBCharacter, 0, len(details.Characters))
		for _, character := range details.Characters {
			mapped := &metadatapb.AniDBCharacter{
				Id:          character.ID,
				Type:        character.Type,
				Update:      character.Update,
				Rating:      &metadatapb.AniDBVoteRating{Votes: character.Rating.Votes, Value: character.Rating.Value},
				Name:        character.Name,
				Gender:      character.Gender,
				Description: character.Description,
				Picture:     character.Picture,
				CharacterType: &metadatapb.AniDBCharacterType{
					Id:   character.CharacterType.ID,
					Type: character.CharacterType.Type,
				},
			}
			if len(character.Seiyuu) > 0 {
				mapped.Seiyuu = make([]*metadatapb.AniDBSeiyuu, 0, len(character.Seiyuu))
				for _, seiyuu := range character.Seiyuu {
					mapped.Seiyuu = append(mapped.Seiyuu, &metadatapb.AniDBSeiyuu{Id: seiyuu.ID, Picture: seiyuu.Picture, Name: seiyuu.Name})
				}
			}
			out.Characters = append(out.Characters, mapped)
		}
	}

	return out
}

func toDetailedProto(detailed models.Detailed) *metadatapb.AniDBDetailed {
	result := &metadatapb.AniDBDetailed{
		Id:              uint64(detailed.ID),
		CreatedAt:       toTimestamp(detailed.CreatedAt),
		UpdatedAt:       toTimestamp(detailed.UpdatedAt),
		DeletedAt:       toTimestampPtr(detailed.DeletedAt),
		Title:           detailed.Title,
		Aid:             uint64(detailed.AID),
		LibraryId:       uint64(detailed.LibraryID),
		AlternateTitles: detailed.AlternateTitles,
		Description:     detailed.Description,
		ReleaseDate:     detailed.ReleaseDate,
		Genres:          detailed.Genres,
		PosterUrl:       detailed.PosterURL,
		TotalEpisodes:   int32(detailed.TotalEpisodes),
		TotalSeasons:    int32(detailed.TotalSeasons),
	}

	if len(detailed.Episodes) > 0 {
		result.Episodes = make([]*metadatapb.AniDBDetailedEpisode, 0, len(detailed.Episodes))
		for _, episode := range detailed.Episodes {
			result.Episodes = append(result.Episodes, &metadatapb.AniDBDetailedEpisode{
				Id:         uint64(episode.ID),
				CreatedAt:  toTimestamp(episode.CreatedAt),
				UpdatedAt:  toTimestamp(episode.UpdatedAt),
				DeletedAt:  toTimestampPtr(episode.DeletedAt),
				DetailedId: uint64(episode.DetailedID),
				AnidbId:    episode.AniDBID,
				Type:       int32(episode.Type),
				EpNo:       episode.EpNo,
				Title:      episode.Title,
				AirDate:    episode.AirDate,
			})
		}
	}

	return result
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
