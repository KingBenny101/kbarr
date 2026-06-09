package handlers

import (
	"encoding/json"

	"github.com/kingbenny101/kbarr/internal/core/clients"
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

type TestDownloadRequest struct {
	TorrentURL string `json:"torrent_url"`
	Title      string `json:"title"`
}

type MessageResponse struct {
	Message string `json:"message"`
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

type MonitorIDInput struct {
	ID uint `path:"id" doc:"Monitor ID"`
}

type DownloadIDInput struct {
	ID int64 `path:"id" doc:"Download queue entry ID"`
}

type SymlinkInput struct {
	ID int64 `path:"id" doc:"Download queue entry ID"`
}

type ServiceNameInput struct {
	Name string `path:"name" enum:"core,metadata,indexer,downloader" doc:"Service name"`
}

type SearchInput struct {
	Q string `query:"q" doc:"Search query" required:"true"`
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

type AddMonitorInput struct {
	Body models.Monitor
}

type BulkAddMonitorInput struct {
	Body []models.Monitor
}

type UnmonitorInput struct {
	Body UnmonitorRequest
}

type UnmonitorSeasonInput struct {
	Body UnmonitorSeasonRequest
}

type AddTestDownloadInput struct {
	Body TestDownloadRequest
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
	Body []db.DownloadQueue
}

type SearchOutput struct {
	Body []clients.SearchResult
}
