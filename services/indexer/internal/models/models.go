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
	Source        *string    `bun:"source"`
	ExternalID    *string    `bun:"external_id"`
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
	TorrentName   *string    `bun:"torrent_name"`
	TorrentURL    *string    `bun:"torrent_url"`
	SavePath      *string    `bun:"save_path"`
	TorrentHash   *string    `bun:"torrent_hash"`
	Indexer       *string    `bun:"indexer"`
	Size          *int64     `bun:"size"`
	Seeders       *int       `bun:"seeders"`
	Status        *string    `bun:"status"`
}

type TorrentBlacklist struct {
	bun.BaseModel `bun:"table:torrent_blacklist"`
	ID          int64   `bun:"id,pk"`
	TorrentHash *string `bun:"torrent_hash"`
	TorrentName *string `bun:"torrent_name"`
}

type SearchResult struct {
	Title       string
	FileName    string
	DownloadURL string
	Size        int64
	Indexer     string
	Seeds       int
	Peers       int
}

type Detailed struct {
	bun.BaseModel   `bun:"table:detaileds"`
	LibraryID       *int64  `bun:"library_id"`
	AlternateTitles *string `bun:"alternate_titles"`
	Title           *string `bun:"title"`
}
