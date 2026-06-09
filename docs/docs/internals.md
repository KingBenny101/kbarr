---
sidebar_label: Internal Services
sidebar_position: 2
---

# Internal Services

These services run inside the Docker network and are not directly accessible to users. They are called by the core service.

## Metadata (port 8081)

Fetches and caches anime metadata from external sources (AniDB).

| Method | Path | Description |
|--------|------|-------------|
| `GET` | `/health` | Liveness check |
| `GET` | `/logs` | Service logs |
| `GET` | `/search?q=` | Search anime titles |
| `GET` | `/anime/{aid}` | Get full details by AniDB ID |
| `POST` | `/prepare` | Fetch and cache metadata for a source/ID |

## Indexer (port 8082)

Searches torrent indexers (kbdex or Prowlarr) for monitored episodes.

| Method | Path | Description |
|--------|------|-------------|
| `GET` | `/health` | Liveness check |
| `GET` | `/logs` | Service logs |
| `POST` | `/test` | Test the active indexer (kbdex or Prowlarr) |
| `POST` | `/test/kbdex` | Test kbdex specifically |

## Downloader (port 8083)

Manages torrent downloads via qBittorrent.

| Method | Path | Description |
|--------|------|-------------|
| `GET` | `/health` | Liveness check |
| `GET` | `/logs` | Service logs |
| `POST` | `/torrent/delete` | Delete a torrent from qBittorrent by hash |
| `POST` | `/symlinks/create` | Create symlinks for a completed download entry |
| `POST` | `/files/delete` | Remove symlink directory for a media entry |
| `POST` | `/trigger` | Manually trigger the download processing loop |
| `POST` | `/test` | Test qBittorrent connection |
