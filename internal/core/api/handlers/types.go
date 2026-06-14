package handlers

import (
	"encoding/json"

	"github.com/kingbenny101/kbarr/internal/config"
	"github.com/kingbenny101/kbarr/internal/core/db"
	"github.com/kingbenny101/kbarr/internal/models"
)

// ---- Shared response/request types ----

type LoginRequest struct {
	Username string `json:"username" example:"admin"`
	Password string `json:"password" example:"password"`
}

type TokenResponse struct {
	Token string `json:"token" example:"eyJ..."`
}

type UsernameResponse struct {
	Username string `json:"username" example:"admin"`
	// EnvManaged is true when credentials come from the environment, in which case
	// the UI disables the change-credentials form.
	EnvManaged bool `json:"envManaged"`
}

type CredentialsRequest struct {
	CurrentPassword string `json:"currentPassword"`
	NewUsername     string `json:"newUsername"`
	NewPassword     string `json:"newPassword"`
}

type MonitorStatusRequest struct {
	Monitored bool `json:"monitored"`
}

type UnmonitorRequest struct {
	LibraryID  uint   `json:"library_id"`
	ExternalID string `json:"external_id"`
}

type UnmonitorSeasonRequest struct {
	LibraryID uint `json:"library_id"`
	Season    int  `json:"season"`
}


type MessageResponse struct {
	Message string `json:"message"`
}

type RefreshResponse struct {
	Message     string `json:"message"`
	NewEpisodes int    `json:"newEpisodes"`
}

type RefreshOutput struct {
	Body RefreshResponse
}

type VersionResponse struct {
	Version string `json:"version" example:"0.1.0"`
}

type TestResult struct {
	Ok      string `json:"ok" example:"true"`
	Message string `json:"message"`
}

// ---- Input types ----

type LibraryIDInput struct {
	ID uint `path:"id" doc:"Library ID"`
}

type DeleteMediaInput struct {
	ID          uint `path:"id" doc:"Library ID"`
	DeleteFiles bool `query:"deleteFiles" doc:"Also remove the show's folder and organised files from the media path"`
}

type MonitorIDInput struct {
	ID uint `path:"id" doc:"Monitor ID"`
}

type DownloadIDInput struct {
	ID int64 `path:"id" doc:"Download queue entry ID"`
}

type HardlinkInput struct {
	ID int64 `path:"id" doc:"Download queue entry ID"`
}

type ServiceNameInput struct {
	Name string `path:"name" enum:"core,metadata,indexer,downloader" doc:"Service name"`
}

type SearchInput struct {
	Q string `query:"q" doc:"Search query" required:"true"`
}

type ResolveAniListInput struct {
	ID     int      `path:"id" doc:"AniList ID"`
	Titles []string `query:"title" doc:"Optional title hints used for fuzzy fallback when no direct mapping exists"`
}

type ResolveAniListOutput struct {
	Body struct {
		AID     string `json:"aid" doc:"Resolved AniDB ID (empty when not found)"`
		Found   bool   `json:"found" doc:"Whether an AniDB match was found"`
		Matched string `json:"matched" doc:"How it resolved: \"mapping\", \"title\", or empty when not found"`
	}
}

type GetEpisodesInput struct {
	ID    uint   `path:"id"`
	Types string `query:"types" doc:"Comma-separated episode type filter (e.g. 1,2)"`
	Sort  string `query:"sort"  doc:"Sort field" enum:"ep_no,title" default:"ep_no"`
	Order string `query:"order" doc:"Sort order" enum:"asc,desc"   default:"asc"`
	Page  int    `query:"page"  doc:"Page number"                  default:"1" minimum:"1"`
	Limit int    `query:"limit" doc:"Page size"                    default:"10" minimum:"1" maximum:"10000"`
}

type DeleteDownloadInput struct {
	ID   int64              `path:"id" doc:"Download queue entry ID"`
	Body *db.DeleteOptions
}

type AddMediaRequest struct {
	Title    string `json:"title"`
	Source   string `json:"source"`
	SourceID string `json:"source_id"`
}

type AddMediaInput struct {
	Body AddMediaRequest
}

type UpdateMonitorStatusInput struct {
	ID   uint `path:"id" doc:"Library ID"`
	Body MonitorStatusRequest
}

type NSFWRequest struct {
	NSFW bool `json:"nsfw"`
}

type UpdateNSFWInput struct {
	ID   uint `path:"id" doc:"Library ID"`
	Body NSFWRequest
}

// MonitorRequest is the client-supplied payload for creating monitors. It is
// kept separate from models.Monitor so that server-managed fields (ID,
// timestamps, status, available) are not part of the request contract.
type MonitorRequest struct {
	LibraryID     uint   `json:"library_id"`
	Title         string `json:"title"`
	EpisodeTitle  string `json:"episode_title"`
	Season        int    `json:"season"`
	EpisodeNumber int    `json:"episode_number"`
	IsEpisode     bool   `json:"is_episode"`
	IsSeason      bool   `json:"is_season"`
	Source        string `json:"source"`
	ExternalID    string `json:"external_id"`
	Monitored     bool   `json:"monitored,omitempty"`
}

// ToModel maps the request DTO onto the persistence model. Status and the
// availability/timestamp fields are left for the DB layer to populate.
func (r MonitorRequest) ToModel() models.Monitor {
	return models.Monitor{
		LibraryID:     r.LibraryID,
		Title:         r.Title,
		EpisodeTitle:  r.EpisodeTitle,
		Season:        r.Season,
		EpisodeNumber: r.EpisodeNumber,
		IsEpisode:     r.IsEpisode,
		IsSeason:      r.IsSeason,
		Source:        r.Source,
		ExternalID:    r.ExternalID,
		Monitored:     r.Monitored,
	}
}

type AddMonitorInput struct {
	Body MonitorRequest
}

type BulkAddMonitorInput struct {
	Body []MonitorRequest
}

type UnmonitorInput struct {
	Body UnmonitorRequest
}

type UnmonitorSeasonInput struct {
	Body UnmonitorSeasonRequest
}

type UpdateSettingsInput struct {
	Body map[string]string
}

type LoginInput struct {
	Body LoginRequest
}

type ChangeCredentialsInput struct {
	Body CredentialsRequest
}

type LogoutInput struct {
	Authorization string `header:"Authorization" doc:"Bearer token"`
}

type MeInput struct {
	Authorization string `header:"Authorization" doc:"Bearer token"`
}

// ---- Output wrapper types ----

type HealthOutput struct {
	Body MessageResponse
}

type TokenOutput struct {
	Body TokenResponse
}

type UsernameOutput struct {
	Body UsernameResponse
}

type VersionOutput struct {
	Body VersionResponse
}

type TestResultOutput struct {
	Body TestResult
}

type SettingsOutput struct {
	Body map[string]string `doc:"Key-value settings map"`
}

type SettingsSchemaOutput struct {
	Body []config.SettingDef `doc:"Array of setting definitions"`
}

type WorkersOutput struct {
	Body []ServiceHealth
}

type ServiceLogsOutput struct {
	Body json.RawMessage `doc:"Array of log entries"`
}

type MediaListOutput struct {
	Body []models.Media
}

type AddMediaOutput struct {
	Body MessageResponse
}

type DetailedOutput struct {
	Body models.Detailed
}

type EpisodesOutput struct {
	Body db.EpisodeQueryResult
}

type MonitorListOutput struct {
	Body []models.Monitor
}

type DownloadListOutput struct {
	Body []models.DownloadQueue
}

type SearchOutput struct {
	Body []models.SearchResult
}
