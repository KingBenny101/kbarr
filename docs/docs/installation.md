---
sidebar_label: Installation
sidebar_position: 1
---

# Installation

Requirements: Docker and Docker Compose.

Download the latest release files:

```bash
curl -LO https://github.com/kingbenny101/kbarr/releases/latest/download/docker-compose.yml
curl -LO https://github.com/kingbenny101/kbarr/releases/latest/download/example.env
cp example.env .env
```

Edit `.env` with your values:

| Variable | Description |
|---|---|
| `DATA_DIR_HOST` | Host path where images and AniDB cache are stored |
| `LIBRARY_DIR_HOST` | Host path for the shared media library, mounted at `/library`. Holds `downloads/` (where qBittorrent saves) and `media/` (where kbarr writes hardlinks) |
| `POSTGRES_USER` | Postgres username |
| `POSTGRES_PASSWORD` | Postgres password |
| `POSTGRES_DB` | Postgres database name |
| `PORT` | Host port kbarr is exposed on (default: `8282`) |

### Connect qBittorrent

kbarr does not run qBittorrent — point it at your existing instance under
**Settings → Downloader**, and make sure its **save path** lands inside the shared
library so kbarr can hardlink finished torrents:

- qBittorrent must save downloads to the host folder `LIBRARY_DIR_HOST/downloads`
  (this maps to `/library/downloads`, kbarr's default **Download path**).
- kbarr writes organised hardlinks to `/library/media` (default **Media path**).
- Both live under the single `/library` mount **on purpose**: hardlinks only work
  when source and destination are on the same filesystem. If you change these
  paths, keep them under one shared mount or kbarr will fail to link completed
  downloads.

Then start:

```bash
docker compose up -d
```

kbarr is now running at http://localhost:8282.

To update to the latest version:

```bash
docker compose pull
docker compose up -d
```

## Local Development

Requirements: Docker, Docker Compose, Go, Node.js, [air](https://github.com/air-verse/air), tmux.

Clone the repo:

```bash
git clone https://github.com/kingbenny101/kbarr.git
cd kbarr
```

`dev.env` is already included with sensible defaults for local development. No changes needed to get started.

Start all services in a tmux session:

```bash
make dev
```

This opens a tmux session (`kbarr`) with each service in its own window — `db` (postgres + pgadmin), `indexer`, `metadata`, `downloader`, `core`, and `frontend`. Services hot-reload on file changes via `air`.

To stop everything:

```bash
make dev-down
```

### Individual services

Start only the database (postgres + pgadmin):

```bash
make db        # start
make db-down   # stop
```

Run a specific service:

```bash
make run-metadata
make run-indexer
make run-downloader
make run-core
make run-frontend
```

pgadmin is available at http://localhost:5050 (email: `admin@local.dev`, password: `admin`).
