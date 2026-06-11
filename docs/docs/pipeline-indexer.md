---
sidebar_label: Indexer service
sidebar_position: 3
---

# Indexer service pipeline

The indexer service watches the monitor table and searches torrent indexers for pending episodes and seasons. When it finds a match it inserts a download queue entry for the downloader service to pick up.

## Startup

1. Database connection initialised (same PostgreSQL as core).
2. Provider packages self-register via `init()` (kbdex and Prowlarr are blank-imported in `cmd/indexer/main.go`).
3. `PollAndQueue` starts in a background goroutine.
4. Health/test HTTP server starts on port `8082`.

## Poll loop

Runs every `prowlarrInterval` seconds (default 10 s).

```
Timer fires
      │
      ▼
Query DB for one pending monitor
  WHERE monitored = true
    AND available = false
    AND (status = 'pending'
         OR status = 'searching' for > 10 min)
  ORDER BY is_season DESC  ← season packs searched first
      │
      ▼
Mark monitor status = 'searching'
      │
      ▼
Build SearchRequest {title, alternate_titles, season, episode, library_id, ...}
      │
      ▼
GetEnabled(db) → active providers (each enabled via its kbdexEnabled / prowlarrEnabled toggle; results from all are merged)
      │
      ▼
Provider.Search(ctx, db, req)   [see provider detail below]
      │
      ▼
buildMatchLog  →  filter by title similarity + season + episode
      │
      ▼
pickBest  →  quality rank + seeders sort, blacklist check
      │
      ├── match found → insert download_queue row, set monitor status = 'queued'
      └── no match    → set monitor status = 'pending' (retry next cycle)
```

## Provider: kbdex

kbdex is an AniDB-aware search service. It handles title resolution and episode filtering server-side.

```
SearchRequest
      │
      ▼
lookupAnidbID: SELECT source_id FROM media WHERE id = library_id
      │
      ▼
GET {kbdexUrl}/search?anidb_id=N[&season=S][&episode=E]
      │  Results cached on disk keyed by "kbdex:{id}:s{S}:e{E}"
      │  Cache TTL = prowlarrCacheAge
      ▼
[]TorrentResult  (already filtered by kbdex server-side via anitopy)
```

## Provider: Prowlarr

Prowlarr is a generic indexer aggregator. kbarr queries it with multiple title variants and merges results.

```
SearchRequest
      │
      ▼
Build query variants from title + alternate titles
  ├── Season searches: "Title S01", "Title Season 1", "Title Batch", ...
  └── Episode searches: "Title S01E05", "Title - 05", ...
      │
      ▼
For each variant: GET {prowlarrUrl}/api/v1/search?query=...&apikey=...
      │  Each variant result set cached on disk (prowlarrCacheAge TTL)
      ▼
Merge + deduplicate by DownloadURL
      │
      ▼
[]TorrentResult
```

## Client-side filtering (shared by both providers)

After the provider returns results, `buildMatchLog` and `pickBest` apply a second pass:

**buildMatchLog** — runs each torrent filename through `anitogo` (Go torrent filename parser), then checks:
1. Title similarity ≥ `matchThreshold` (default 80%) using Sørensen–Dice bigrams
2. Parsed season matches the monitor's season
3. For season packs: rejects individual episode releases
4. For episodes: parsed episode matches exactly

**pickBest** — from passing results:
1. Removes blacklisted torrents
2. Drops results above the `preferredQuality` cap (e.g. drops 4K if preference is 1080p)
3. Sorts by quality rank descending, then seeder count descending
4. Returns the top entry

## Key packages

| Package | Role |
|---|---|
| `internal/indexer/provider/` | Provider interface, registry, shared cache utilities |
| `internal/indexer/provider/kbdex/` | kbdex provider implementation |
| `internal/indexer/provider/prowlarr/` | Prowlarr provider implementation |
| `internal/indexer/service/` | Poll loop, match scoring, download queue insertion |
| `internal/parser/` | anitogo wrapper with memoisation and Japanese episode fallbacks |
