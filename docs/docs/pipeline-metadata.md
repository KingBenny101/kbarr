---
sidebar_label: Metadata service
sidebar_position: 2
---

# Metadata service pipeline

The metadata service is the only component that talks to AniDB. It resolves anime titles, fetches episode lists, and caches locally so the rest of the stack avoids direct AniDB calls.

## Startup

1. Data directories are created under `KBARR_DATA_DIR/metadata/`.
2. SQLite database is initialised (separate from the main PostgreSQL DB).
3. A background goroutine starts the AniDB titles sync loop.
4. The HTTP server starts on port `8081` (configurable via `ANIDB_SERVICE_PORT`).

## AniDB titles sync

AniDB publishes a compressed dump of all anime titles at a public URL. The metadata service downloads and caches this dump on startup and then re-fetches it every `anidbSyncInterval` minutes (default 1440 = 24 h).

```
Startup / timer fires
      │
      ▼
Download anime-titles.xml.gz from AniDB
      │
      ▼
Parse XML → in-memory title index (AniDB ID → []title variants)
      │
      ▼
Persist to SQLite for fast lookup
```

The title index is used by the `/search` endpoint to match free-text queries to AniDB IDs without a network call.

## Add-media flow (`POST /prepare`)

Called by core when a user adds a title to the library.

```
POST /prepare  {source, source_id, title}
      │
      ▼
Look up AniDB ID (source_id is already the AniDB ID for source=anidb)
      │
      ▼
Fetch full anime details from AniDB HTTP API
  ├── Uses anidbClient / anidbVersion settings for rate-limit compliance
  └── Response cached to disk (KBARR_DATA_DIR/metadata/details/)
      │
      ▼
Fetch poster image, cached to disk (metadata/images/)
      │
      ▼
Return PreparedMedia {title, poster_url, episodes[], alternate_titles[]}
      │
      ▼
Core inserts media + detailed + episodes rows into PostgreSQL
```

## Search flow (`GET /search?q=`)

Used by the frontend search bar. Searches the local title index, not AniDB live.

```
GET /search?q=steins gate
      │
      ▼
Normalise query (lowercase, strip punctuation)
      │
      ▼
Bigram similarity scan over in-memory title index
      │
      ▼
Return top matches [{anidb_id, title, type}]
```

## Key packages

| Package | Role |
|---|---|
| `internal/metadata/service/` | AniDB sync, title search, detail fetch |
| `internal/metadata/api/` | HTTP handler wiring |
| `internal/metadata/db/` | SQLite ORM for local title/detail cache |
| `internal/metadata/models/` | AniDB XML parse structs |
