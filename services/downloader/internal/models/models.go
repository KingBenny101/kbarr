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
}
