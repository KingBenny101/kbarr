# kbarr

A self-hosted anime management application in the spirit of Sonarr and Radarr. Search for anime, add them to your watchlist, and manage your collection — all from a clean web interface.

Built as a hobby project to learn Go.

## Quick Start with Docker

The easiest way to run kbarr is using Docker:

```bash
# Clone the repository
git clone https://github.com/KingBenny101/kbarr.git
cd kbarr

# Start with docker-compose
docker-compose up -d
```

The application will be available at `http://localhost:8282`.

### Configuration

You can customize the following environment variables:

- `KBARR_PORT`: Port to run the server on (default: 8282)
- `KBARR_DB_PATH`: Path to SQLite database file (default: /app/data/kbarr.db)
- `KBARR_CACHE_PATH`: Path for cache files (default: /app/data)
- `KBARR_LOG_LEVEL`: Logging level (default: info)
- `KBARR_ANIDB_CLIENT`: AniDB client name (default: kbarr)
- `KBARR_ANIDB_VERSION`: AniDB client version (default: 1)

### Using Pre-built Images

kbarr images are automatically published to GitHub Container Registry:

```bash
# Pull the latest image
docker pull ghcr.io/kingbenny101/kbarr:latest

# Or use a specific version
docker pull ghcr.io/kingbenny101/kbarr:v1.0.0
```

## Development

### Prerequisites

- Go 1.25+
- Node.js 22+
- SQLite3 development libraries

### Building

```bash
# Build the Go binary
go build -o kbarr .

# Run the application
./kbarr
```

### Frontend Development

```bash
cd frontend
npm install
npm run dev
```

---

## TODO

- [ ] Clean up AniDB description links (e.g. `http://anidb.net/ch2069 [Name]` → plain text)
- [ ] Populate `TitleJP` from AniDB Japanese official title
- [ ] Populate `CoverImage` from AniDB picture field
- [ ] Remove duplicate alternate titles
- [ ] Duplicate detection when adding anime already in the list
- [ ] Delete anime from list
- [ ] Build a proper UI (poster grid, status badges, detail pages)
- [ ] Async task queue for background jobs (Asynq + Redis)
- [ ] Prowlarr/Torznab integration for torrent search
- [ ] Torrent client integration (qBittorrent etc.)
- [ ] Migrate database to PostgreSQL for production use