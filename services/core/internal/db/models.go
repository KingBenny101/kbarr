package db

import (
	"time"

	"github.com/uptrace/bun"
)

type Medium struct {
	bun.BaseModel `bun:"table:media,alias:m"`
	ID            int64      `bun:"id,pk,autoincrement" json:"id"`
	CreatedAt     time.Time  `bun:"created_at,notnull,default:now()" json:"created_at"`
	UpdatedAt     time.Time  `bun:"updated_at,notnull,default:now()" json:"updated_at"`
	DeletedAt     *time.Time `bun:"deleted_at,soft_delete,nullzero" json:"deleted_at"`
	Title         *string    `bun:"title" json:"title"`
	Source        *string    `bun:"source" json:"source"`
	SourceID      *string    `bun:"source_id" json:"source_id"`
	PosterUrl     *string    `bun:"poster_url" json:"poster_url"`
}

type Detailed struct {
	bun.BaseModel   `bun:"table:detaileds,alias:d"`
	ID              int64      `bun:"id,pk,autoincrement" json:"id"`
	CreatedAt       time.Time  `bun:"created_at,notnull,default:now()" json:"created_at"`
	UpdatedAt       time.Time  `bun:"updated_at,notnull,default:now()" json:"updated_at"`
	DeletedAt       *time.Time `bun:"deleted_at,soft_delete,nullzero" json:"deleted_at"`
	Title           *string    `bun:"title" json:"title"`
	Source          *string    `bun:"source" json:"source"`
	SourceID        *string    `bun:"source_id" json:"source_id"`
	LibraryID       *int64     `bun:"library_id" json:"library_id"`
	AlternateTitles *string    `bun:"alternate_titles" json:"alternate_titles"`
	Description     *string    `bun:"description" json:"description"`
	ReleaseDate     *string    `bun:"release_date" json:"release_date"`
	Genres          *string    `bun:"genres" json:"genres"`
	PosterUrl       *string    `bun:"poster_url" json:"poster_url"`
	TotalEpisodes   *int64     `bun:"total_episodes" json:"total_episodes"`
	TotalSeasons    *int64     `bun:"total_seasons" json:"total_seasons"`
	Episodes        []Episode  `bun:"rel:has-many,join:id=detailed_id" json:"episodes,omitempty"`
}

type Episode struct {
	bun.BaseModel `bun:"table:episodes,alias:e"`
	ID            int64      `bun:"id,pk,autoincrement" json:"id"`
	CreatedAt     time.Time  `bun:"created_at,notnull,default:now()" json:"created_at"`
	UpdatedAt     time.Time  `bun:"updated_at,notnull,default:now()" json:"updated_at"`
	DeletedAt     *time.Time `bun:"deleted_at,soft_delete,nullzero" json:"deleted_at"`
	DetailedID    *int64     `bun:"detailed_id" json:"detailed_id"`
	Source        *string    `bun:"source" json:"source"`
	ExternalID    *string    `bun:"external_id" json:"external_id"`
	Type          *int64     `bun:"type" json:"type"`
	EpNo          *string    `bun:"ep_no" json:"ep_no"`
	Title         *string    `bun:"title" json:"title"`
	AirDate       *string    `bun:"air_date" json:"air_date"`
}

type Monitor struct {
	bun.BaseModel `bun:"table:monitors,alias:mon"`
	ID            int64      `bun:"id,pk,autoincrement" json:"id"`
	CreatedAt     time.Time  `bun:"created_at,notnull,default:now()" json:"created_at"`
	UpdatedAt     time.Time  `bun:"updated_at,notnull,default:now()" json:"updated_at"`
	DeletedAt     *time.Time `bun:"deleted_at,soft_delete,nullzero" json:"deleted_at"`
	LibraryID     *int64     `bun:"library_id" json:"library_id"`
	Title         *string    `bun:"title" json:"title"`
	EpisodeTitle  *string    `bun:"episode_title" json:"episode_title"`
	Season        *int64     `bun:"season" json:"season"`
	EpisodeNumber *int64     `bun:"episode_number" json:"episode_number"`
	IsEpisode     *bool      `bun:"is_episode" json:"is_episode"`
	IsSeason      *bool      `bun:"is_season" json:"is_season"`
	Source        *string    `bun:"source" json:"source"`
	ExternalID    *string    `bun:"external_id" json:"external_id"`
	Status        *string    `bun:"status" json:"status"`
}

type SearchQueue struct {
	bun.BaseModel `bun:"table:search_queues,alias:sq"`
	ID            int64      `bun:"id,pk,autoincrement" json:"id"`
	CreatedAt     time.Time  `bun:"created_at,notnull,default:now()" json:"created_at"`
	UpdatedAt     time.Time  `bun:"updated_at,notnull,default:now()" json:"updated_at"`
	DeletedAt     *time.Time `bun:"deleted_at,soft_delete,nullzero" json:"deleted_at"`
	LibraryID     *int64     `bun:"library_id" json:"library_id"`
	Title         *string    `bun:"title" json:"title"`
	EpisodeTitle  *string    `bun:"episode_title" json:"episode_title"`
	Season        *int64     `bun:"season" json:"season"`
	EpisodeNumber *int64     `bun:"episode_number" json:"episode_number"`
	IsEpisode     *bool      `bun:"is_episode" json:"is_episode"`
	IsSeason      *bool      `bun:"is_season" json:"is_season"`
	Status        *string    `bun:"status" json:"status"`
}

type DownloadQueue struct {
	bun.BaseModel `bun:"table:download_queue"`
	ID            int64      `bun:"id,pk,autoincrement" json:"id"`
	CreatedAt     time.Time  `bun:"created_at,notnull,default:now()" json:"created_at"`
	UpdatedAt     time.Time  `bun:"updated_at,notnull,default:now()" json:"updated_at"`
	DeletedAt     *time.Time `bun:"deleted_at,soft_delete,nullzero" json:"deleted_at"`
	MonitorID     *int64     `bun:"monitor_id" json:"monitor_id"`
	Title         *string    `bun:"title" json:"title"`
	TorrentURL    *string    `bun:"torrent_url" json:"torrent_url"`
	SavePath      *string    `bun:"save_path" json:"save_path"`
	TorrentHash   *string    `bun:"torrent_hash" json:"torrent_hash"`
	Status        *string    `bun:"status" json:"status"`
}

type Setting struct {
	bun.BaseModel `bun:"table:settings,alias:s"`
	ID            int64      `bun:"id,pk,autoincrement" json:"id"`
	CreatedAt     time.Time  `bun:"created_at,notnull,default:now()" json:"created_at"`
	UpdatedAt     time.Time  `bun:"updated_at,notnull,default:now()" json:"updated_at"`
	DeletedAt     *time.Time `bun:"deleted_at,soft_delete,nullzero" json:"deleted_at"`
	Key           string     `bun:"key,notnull,unique" json:"key"`
	Value         *string    `bun:"value" json:"value"`
}
