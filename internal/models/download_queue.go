package models

import (
	"time"

	"github.com/uptrace/bun"
)

// DownloadQueue tracks active and historical torrent downloads.
type DownloadQueue struct {
	bun.BaseModel     `bun:"table:download_queue"`
	ID                int64      `bun:"id,pk,autoincrement" json:"id"`
	CreatedAt         time.Time  `bun:"created_at,notnull,default:now()" json:"created_at"`
	UpdatedAt         time.Time  `bun:"updated_at,notnull,default:now()" json:"updated_at"`
	DeletedAt         *time.Time `bun:"deleted_at,soft_delete,nullzero" json:"-"`
	MonitorID         *int64     `bun:"monitor_id" json:"monitor_id"`
	Title             *string    `bun:"title" json:"title"`
	TorrentName       *string    `bun:"torrent_name" json:"torrent_name"`
	TorrentURL        *string    `bun:"torrent_url" json:"torrent_url"`
	SavePath          *string    `bun:"save_path" json:"save_path"`
	TorrentHash       *string    `bun:"torrent_hash" json:"torrent_hash"`
	Indexer           *string    `bun:"indexer" json:"indexer"`
	Size              *int64     `bun:"size" json:"size"`
	Seeders           *int       `bun:"seeders" json:"seeders"`
	Status            *string    `bun:"status" json:"status"`
	Progress          *float64   `bun:"progress" json:"progress"`
	ProgressUpdatedAt *time.Time `bun:"progress_updated_at" json:"progress_updated_at"`
}

// TorrentBlacklist records torrents that should not be re-downloaded.
type TorrentBlacklist struct {
	bun.BaseModel `bun:"table:torrent_blacklist"`
	ID            int64     `bun:"id,pk,autoincrement" json:"id"`
	CreatedAt     time.Time `bun:"created_at,notnull,default:now()" json:"created_at"`
	TorrentHash   *string   `bun:"torrent_hash" json:"torrent_hash"`
	TorrentName   *string   `bun:"torrent_name" json:"torrent_name"`
	Title         *string   `bun:"title" json:"title"`
}
