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
| `POSTGRES_USER` | Postgres username |
| `POSTGRES_PASSWORD` | Postgres password |
| `POSTGRES_DB` | Postgres database name |
| `PORT` | Host port kbarr is exposed on (default: `8282`) |

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
