---
sidebar_label: Downloader service
sidebar_position: 4
---

# Downloader service pipeline

The downloader service manages the full lifecycle of a torrent download: adding it to qBittorrent, polling progress, organising completed files into the media library, and notifying Jellyfin.

## Startup

1. Database connection initialised (same PostgreSQL as core).
2. qBittorrent provider self-registers via `init()` (blank-imported in `cmd/downloader/main.go`).
3. `PollAndDownload` starts in a background goroutine.
4. Internal HTTP server starts on port `8083`.

## Poll loop

Runs every `downloaderInterval` seconds (default 5 s). Each cycle runs two passes:

### Pass 1 — ProcessPending

Picks up entries in the download queue that haven't been submitted to qBittorrent yet.

```
SELECT from download_queue WHERE status = 'pending'
      │
      ▼
Login to qBittorrent (session cookie reused across cycles)
      │
      ▼
Resolve torrent URL
  ├── Magnet link → pass through directly
  └── HTTP URL    → fetch bytes, detect XML error responses (some indexers
                    redirect to an error page instead of 404ing)
      │
      ▼
POST /api/v2/torrents/add to qBittorrent
      │
      ▼
Poll qBittorrent for the torrent hash (up to 30 s)
      │
      ▼
Update download_queue: status = 'downloading', torrent_hash = '...'
```

### Pass 2 — UpdateDownloading

Checks progress on all active downloads and handles completion or stalls.

```
SELECT from download_queue WHERE status = 'downloading'
      │
      ▼
Login to qBittorrent
      │
      ▼
FetchTorrents by hash
      │
      ├── Not found in qBittorrent → mark entry as error
      │
      ├── progress < 1.0
      │     ├── Check stallTimeout: if no progress for N seconds → blacklist
      │     │   torrent, delete from qBittorrent, reset monitor to 'pending'
      │     └── Update progress in DB
      │
      └── progress = 1.0 → onComplete()
```

## Completion flow (`onComplete`)

```
Download complete
      │
      ▼
Resolve monitor → media_folder (from media table)
      │
      ▼
Walk download save_path recursively
  For each video file (extension in allowedVideoExtensions):
    ├── Parse filename with anitogo → season, episode, title
    ├── Determine destination: {mediaPath}/{show}/{Title} - S01E05.mkv
    └── os.Link(src, dst)  ← hardlink (requires same filesystem)
      │
      ▼
If no video files found → blacklist torrent, delete from qBittorrent,
                           reset monitor to 'pending' (try a different release)
      │
      ▼
Write tvshow.nfo to {mediaPath}/{show}/tvshow.nfo  (Jellyfin AniDB match)
  <?xml version="1.0" encoding="utf-8"?>
  <tvshow>
    <title>Steins;Gate</title>
    <uniqueid type="anidb" default="true">6987</uniqueid>
  </tvshow>
      │
      ▼
POST {jellyfinUrl}/Library/Refresh  (if jellyfinUrl is configured)
      │
      ▼
Update download_queue: status = 'completed'
Update monitor: status = 'downloaded'
```

## Stall detection

For each downloading entry, the downloader records `progress_updated_at` whenever progress changes. If `stallTimeout` seconds pass with no progress change, the torrent is considered stalled:

1. Torrent hash added to `torrent_blacklist`
2. Torrent deleted from qBittorrent
3. Download queue entry soft-deleted
4. Monitor reset to `status = 'pending'` so the indexer searches for a different release

Set `stallTimeout = 0` to disable stall detection entirely.

## Hardlink vs copy

Files are organised with `os.Link` (hardlinks), not copies or symlinks. This means:
- Zero extra disk space used
- The original torrent file in the download directory and the organised file in the media directory are the same inode
- **Source and destination must be on the same filesystem** — if `downloadPath` and `mediaPath` are on different volumes this will fail silently

## Key packages

| Package | Role |
|---|---|
| `internal/downloader/provider/` | TorrentClient interface, provider registry |
| `internal/downloader/provider/qbittorrent/` | qBittorrent HTTP client implementation |
| `internal/downloader/service/` | Poll loop, hardlink creation, NFO writing, Jellyfin trigger |
| `internal/naming/` | `SanitizeFilename` — strips filesystem-unsafe characters while preserving Unicode (Japanese titles etc.) |
