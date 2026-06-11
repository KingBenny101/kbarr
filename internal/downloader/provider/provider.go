package provider

import (
	"context"
	"path/filepath"
	"strings"

	"github.com/uptrace/bun"
)

// TorrentInfo holds the state of a single torrent as reported by the download client.
type TorrentInfo struct {
	Hash        string
	Name        string
	ContentPath string
	State       string
	Size        int64
	Progress    float64
	ETA         int64
	SavePath    string
	Category    string
}

func pathBase(p string) string {
	if i := strings.LastIndexAny(p, "/\\"); i >= 0 {
		return p[i+1:]
	}
	return p
}

// ContentName returns the actual on-disk name of the torrent content.
func (t TorrentInfo) ContentName() string {
	if t.ContentPath != "" {
		return pathBase(t.ContentPath)
	}
	return pathBase(t.Name)
}

// ContentRelPath returns the path of the torrent content relative to the save directory.
func (t TorrentInfo) ContentRelPath() string {
	if t.ContentPath == "" {
		return pathBase(t.Name)
	}
	if t.SavePath != "" {
		cp := strings.ReplaceAll(t.ContentPath, "\\", "/")
		sp := strings.TrimRight(strings.ReplaceAll(t.SavePath, "\\", "/"), "/")
		if strings.HasPrefix(cp, sp+"/") {
			return filepath.FromSlash(cp[len(sp)+1:])
		}
	}
	return pathBase(t.ContentPath)
}

// TorrentClient is the interface implemented by each download client backend.
type TorrentClient interface {
	Login(ctx context.Context) error
	AddTorrent(ctx context.Context, torrentURL string) (hash string, err error)
	FetchTorrents(ctx context.Context, hash, category string) ([]TorrentInfo, error)
	DeleteTorrent(ctx context.Context, hash string, deleteFiles bool) error
	// DefaultSavePath returns the download client's own default save directory,
	// expressed in the client's path namespace.
	DefaultSavePath(ctx context.Context) (string, error)
	// SetLocation moves a torrent's files to the given location (client namespace).
	SetLocation(ctx context.Context, hash, location string) error
}

type factory func(db *bun.DB) TorrentClient

var registry = map[string]factory{}

// Register adds a named client factory. Call from init().
func Register(name string, f factory) {
	registry[name] = f
}

// Get returns the configured download client (currently always qbittorrent).
func Get(db *bun.DB) TorrentClient {
	name := "qbittorrent"
	if f, ok := registry[name]; ok {
		return f(db)
	}
	return nil
}
