package models

import (
	"time"

	"github.com/uptrace/bun"
)

type Monitor struct {
	bun.BaseModel `bun:"table:monitors,alias:mon"`
	ID            int64      `bun:"id,pk"`
	CreatedAt     time.Time  `bun:"created_at"`
	UpdatedAt     time.Time  `bun:"updated_at"`
	DeletedAt     *time.Time `bun:"deleted_at,soft_delete,nullzero"`
	LibraryID     *int64     `bun:"library_id"`
	Title         *string    `bun:"title"`
	EpisodeTitle  *string    `bun:"episode_title"`
	Season        *int64     `bun:"season"`
	EpisodeNumber *int64     `bun:"episode_number"`
	IsEpisode     *bool      `bun:"is_episode"`
	IsSeason      *bool      `bun:"is_season"`
	AnidbID       *string    `bun:"anidb_id"`
	Status        *string    `bun:"status"`
}

type DownloadQueue struct {
	bun.BaseModel `bun:"table:download_queue"`
	ID            int64      `bun:"id,pk,autoincrement"`
	CreatedAt     time.Time  `bun:"created_at,notnull,default:now()"`
	UpdatedAt     time.Time  `bun:"updated_at,notnull,default:now()"`
	DeletedAt     *time.Time `bun:"deleted_at,soft_delete,nullzero"`
	MonitorID     *int64     `bun:"monitor_id"`
	Title         *string    `bun:"title"`
	TorrentURL    *string    `bun:"torrent_url"`
	SavePath      *string    `bun:"save_path"`
	Status        *string    `bun:"status"`
}

type SearchResult struct {
	Title       string
	DownloadURL string
	Size        int64
	Indexer     string
	Seeds       int
	Peers       int
}
