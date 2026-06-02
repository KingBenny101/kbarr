package models

import (
	"time"

	"github.com/uptrace/bun"
)

type DownloadQueue struct {
	bun.BaseModel `bun:"table:download_queue"`
	ID            int64      `bun:"id,pk"`
	CreatedAt     time.Time  `bun:"created_at"`
	UpdatedAt     time.Time  `bun:"updated_at"`
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
	Progress          *float64   `bun:"progress"`
	ProgressUpdatedAt *time.Time `bun:"progress_updated_at"`
}

type TorrentBlacklist struct {
	bun.BaseModel `bun:"table:torrent_blacklist"`
	ID          int64     `bun:"id,pk,autoincrement"`
	CreatedAt   time.Time `bun:"created_at,notnull,default:now()"`
	TorrentHash *string   `bun:"torrent_hash"`
	TorrentName *string   `bun:"torrent_name"`
	Title       *string   `bun:"title"`
}

type Monitor struct {
	bun.BaseModel `bun:"table:monitors"`
	ID            int64      `bun:"id,pk"`
	DeletedAt     *time.Time `bun:"deleted_at,soft_delete,nullzero"`
	LibraryID     *int64     `bun:"library_id"`
	IsSeason      *bool      `bun:"is_season"`
	IsEpisode     *bool      `bun:"is_episode"`
}
