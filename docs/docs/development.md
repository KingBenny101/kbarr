---
sidebar_label: Development
sidebar_position: 2
---

# Development Setup

This page is for contributors and anyone who wants to run kbarr from source.

---

## Prerequisites

- [Docker](https://docs.docker.com/engine/install/) and Docker Compose
- [Go](https://go.dev/dl/) (see `go.mod` for the required version)
- [Node.js](https://nodejs.org/) (LTS)
- [air](https://github.com/air-verse/air) — live reload for Go services
- [overmind](https://github.com/DarthSim/overmind) — process manager for local dev

---

## Setup

Clone the repository:

```bash
git clone https://github.com/kingbenny101/kbarr.git
cd kbarr
```

`dev.env` is included with sensible defaults. No changes are needed to get started.

Start all services via overmind:

```bash
make dev
```

This launches all services (from the Procfile) with each in its own process:

| Window | What runs |
|---|---|
| `db` | PostgreSQL + pgAdmin |
| `core` | Main API + frontend |
| `metadata` | AniDB metadata service |
| `indexer` | Torrent search service |
| `downloader` | Download queue service |

All Go services hot-reload on file changes via `air`. The frontend uses Vite's dev server.

To stop everything:

```bash
make dev-down
```

---

## Running individual services

Start only the database:

```bash
make db        # start
make db-down   # stop
```

Run a specific service manually:

```bash
make run-core
make run-metadata
make run-indexer
make run-downloader
make run-frontend
```

pgAdmin is available at **http://localhost:5050** — email: `admin@local.dev`, password: `admin`.

---

## Tech stack

| Layer | Technology |
|---|---|
| Backend services | Go |
| Frontend | React, TypeScript, Mantine |
| Database | PostgreSQL (via [bun ORM](https://bun.uptrace.dev/)) |
| API | [huma v2](https://huma.rocks/) (OpenAPI 3.0) |
| Metadata | AniDB UDP/HTTP API |
| Torrent search | kbdex and Prowlarr (additional indexers can be wired in) |
| Download client | qBittorrent (additional clients can be wired in) |

---

## Generating the API spec

The OpenAPI spec is generated from the running server:

```bash
go run ./cmd/genspec
```

To regenerate the docs API pages after updating the spec:

```bash
npm run gen-api-docs   # from the docs/ directory
```
