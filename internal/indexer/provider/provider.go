package provider

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/kingbenny101/kbarr/internal/config"
	"github.com/kingbenny101/kbarr/internal/models"
	"github.com/uptrace/bun"
)

// SearchRequest carries all metadata the indexer service knows about a pending monitor.
type SearchRequest struct {
	LibraryID       int64
	Title           string
	AlternateTitles []string
	IsSeason        bool
	Season          int64
	EpisodeNumber   int64
	EpisodeCount    int
	Threshold       float64
}

// AllTitles returns the main title followed by all alternate titles.
func (r SearchRequest) AllTitles() []string {
	return append([]string{r.Title}, r.AlternateTitles...)
}

// Provider is implemented by every indexer backend.
type Provider interface {
	Name() string
	IsEnabled(db *bun.DB) bool
	Search(ctx context.Context, db *bun.DB, req SearchRequest) ([]models.TorrentResult, error)
	// Prematched reports whether the provider resolves matches server-side (e.g.
	// kbdex by AniDB id) so its results are already known to be the right anime.
	// The service skips the title-similarity gate for such providers, since their
	// release titles often differ from the monitor title (romaji / alt names).
	Prematched() bool
}

// IndexerEnabled reports whether the named provider should be searched. It reads
// the per-provider toggle (<name>Enabled). For installs created before per-provider
// toggles existed, it falls back to the legacy single-select indexerProvider value
// when the toggle has never been written.
func IndexerEnabled(db *bun.DB, name string) bool {
	if v := config.Get(db, name+"Enabled", ""); v != "" {
		return v == "true"
	}
	return config.Get(db, "indexerProvider", "kbdex") == name
}

var registry []func() Provider

// Register adds a provider factory to the global registry. Call from init().
func Register(factory func() Provider) {
	registry = append(registry, factory)
}

// GetEnabled returns all registered providers that report themselves as enabled.
func GetEnabled(db *bun.DB) []Provider {
	var active []Provider
	for _, f := range registry {
		p := f()
		if p.IsEnabled(db) {
			active = append(active, p)
		}
	}
	return active
}

// DataRootDir returns the root data directory.
func DataRootDir() string {
	if d := strings.TrimSpace(os.Getenv("KBARR_DATA_DIR")); d != "" {
		return d
	}
	return "data"
}

// ── shared cache utilities ─────────────────────────────────────────────────────

var nonAlnum = regexp.MustCompile(`[^a-z0-9]+`)

// CacheKey converts an arbitrary query string into a stable filename.
func CacheKey(query string) string {
	slug := nonAlnum.ReplaceAllString(strings.ToLower(query), "_")
	slug = strings.Trim(slug, "_")
	if len(slug) > 60 {
		slug = slug[:60]
	}
	h := sha256.Sum256([]byte(query))
	return fmt.Sprintf("%s_%x.json", slug, h[:4])
}

// CacheTTL returns the configured cache lifetime.
func CacheTTL(db *bun.DB) time.Duration {
	raw, err := strconv.Atoi(config.Get(db, "prowlarrCacheAge", "3600"))
	if err != nil || raw <= 0 {
		return time.Hour
	}
	return time.Duration(raw) * time.Second
}

// emptyCacheTTL bounds how long an empty (0-result) search is served from cache.
// A transient empty response (indexer momentarily down, release not yet seeded)
// must not suppress retries for the full TTL, or a release appearing minutes later
// goes unnoticed for up to an hour.
const emptyCacheTTL = 5 * time.Minute

// EnforceFileLimit deletes the oldest files in dir until fewer than limit remain.
func EnforceFileLimit(dir string, limit int) {
	if limit <= 0 {
		return
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return
	}
	type fi struct {
		name    string
		modTime time.Time
	}
	var files []fi
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		info, err := e.Info()
		if err != nil {
			continue
		}
		files = append(files, fi{name: e.Name(), modTime: info.ModTime()})
	}
	sort.Slice(files, func(i, j int) bool {
		return files[i].modTime.Before(files[j].modTime)
	})
	for len(files) >= limit {
		os.Remove(filepath.Join(dir, files[0].name))
		files = files[1:]
	}
}

// CacheLoad reads and validates a cached result list.
func CacheLoad(db *bun.DB, dir, query string) ([]models.TorrentResult, bool) {
	path := filepath.Join(dir, CacheKey(query))
	info, err := os.Stat(path)
	if err != nil {
		return nil, false
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, false
	}
	var results []models.TorrentResult
	if err := json.Unmarshal(data, &results); err != nil {
		return nil, false
	}
	// Empty result sets expire much faster than populated ones so transient
	// emptiness doesn't suppress retries for the whole TTL.
	ttl := CacheTTL(db)
	if len(results) == 0 && emptyCacheTTL < ttl {
		ttl = emptyCacheTTL
	}
	if time.Since(info.ModTime()) > ttl {
		return nil, false
	}
	slog.Debug("Cache hit", "dir", filepath.Base(dir), "query", query)
	return results, true
}

// CacheSave writes results to the cache directory.
func CacheSave(dir, query string, results []models.TorrentResult, limit int) {
	if err := os.MkdirAll(dir, 0755); err != nil {
		slog.Warn("Failed to create cache dir", "dir", dir, "error", err)
		return
	}
	data, err := json.Marshal(results)
	if err != nil {
		return
	}
	EnforceFileLimit(dir, limit)
	path := filepath.Join(dir, CacheKey(query))
	if err := os.WriteFile(path, data, 0644); err != nil {
		slog.Warn("Failed to write cache", "dir", filepath.Base(dir), "query", query, "error", err)
	}
}
