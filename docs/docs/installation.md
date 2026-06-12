---
sidebar_label: Installation
sidebar_position: 1
---

# Installation

This guide walks through setting up kbarr from scratch. No prior Docker experience is assumed.

---

## 1. Install Docker

kbarr runs entirely via Docker Compose. If you don't have Docker installed:

- **Linux:** follow the [official Docker Engine install guide](https://docs.docker.com/engine/install/) for your distro, then install the [Docker Compose plugin](https://docs.docker.com/compose/install/linux/)
- **Mac / Windows:** install [Docker Desktop](https://www.docker.com/products/docker-desktop/) — it includes Compose

Verify the install:

```bash
docker --version
docker compose version
```

---

## 2. Download kbarr

Create a folder for kbarr and download the two files it needs:

```bash
mkdir kbarr && cd kbarr
curl -LO https://github.com/kingbenny101/kbarr/releases/latest/download/docker-compose.yml
curl -LO https://github.com/kingbenny101/kbarr/releases/latest/download/example.env
cp example.env .env
```

---

## 3. Configure the environment

Open `.env` in a text editor. The only value you **must** set is `LIBRARY_DIR_HOST`:

```env
# Path on your machine where kbarr stores all media.
# qBittorrent saves downloads into <LIBRARY_DIR_HOST>/downloads
# kbarr writes organised files into <LIBRARY_DIR_HOST>/media
LIBRARY_DIR_HOST=/path/to/your/library
```

Everything else has sensible defaults and can be left alone to get started.

### Library path explained

kbarr uses **hardlinks** to organise finished downloads without duplicating files. Hardlinks only work when the source and destination are on the same filesystem — this is why everything lives under one shared folder:

```
/your/library/
├── downloads/    ← qBittorrent saves here
└── media/        ← kbarr writes organised files here
```

Both paths must be on the **same disk**. If they are on different disks, kbarr will fall back to copying files instead.

:::tip
If you are running on a NAS or mounting an external drive, make sure `LIBRARY_DIR_HOST` points to a path on that drive. Don't split `downloads/` and `media/` across different mounts.
:::

---

## 4. Start kbarr

```bash
docker compose up -d
```

kbarr is now running at **http://localhost:8282** (or the `PORT` you set in `.env`).

The default login is `admin` / `admin`. Change it immediately under **Settings → General → Account**.

---

## 5. Set up AniDB credentials

kbarr uses AniDB to fetch anime metadata, episode lists, and artwork. Access requires registering an API client.

1. Register an API client at [anidb.net/client](https://anidb.net/software/add)
2. In kbarr, go to **Settings → Metadata**
3. Enter your **Client ID** and **Client Version**

:::note
AniDB rate-limits API access strictly. kbarr handles this automatically with pacing and backoff — do not run multiple instances using the same client.
:::

---

## 6. Set up a torrent indexer

kbarr needs a source to search for torrents. You can use **kbdex**, **Prowlarr**, or both.

### Option A — kbdex (recommended)

[kbdex](https://github.com/KingBenny101/kbdex) is an AniDB-aware torrent search engine built alongside kbarr. It matches releases using AniDB series IDs rather than title text, which means it finds the right torrent even when release names use different romanisations or languages. It requires no API key — just point kbarr at your running kbdex instance.

1. Deploy kbdex — see the [kbdex README](https://github.com/KingBenny101/kbdex) for setup instructions
2. In kbarr, go to **Settings → Indexer → kbdex**
3. Enter the **URL** kbdex is accessible at (e.g. `http://localhost:8000`)

### Option B — Prowlarr

[Prowlarr](https://github.com/Prowlarr/Prowlarr) is an indexer manager that aggregates many torrent sites (Nyaa, AniDex, etc.) into a single search API. It requires its own installation and configuration.

1. Install Prowlarr — see the [Prowlarr installation docs](https://wiki.servarr.com/prowlarr/installation)
2. Add your indexers inside Prowlarr (e.g. Nyaa, AniDex)
3. In kbarr, go to **Settings → Indexer → Prowlarr** and enter:
   - **URL** — the address Prowlarr is accessible at (e.g. `http://localhost:9696`)
   - **API key** — found in Prowlarr under Settings → General → Security

You can enable both at the same time — kbarr merges results from all active providers.

---

## 7. Set up qBittorrent

kbarr does not run qBittorrent — it connects to your existing instance.

1. Make sure qBittorrent is installed and accessible from your host machine
   - If running natively, it's usually at `http://localhost:8080`
   - If running in Docker, ensure it's on the same network or exposed on a port
2. In qBittorrent, set the **Default save path** to `<LIBRARY_DIR_HOST>/downloads`
   - In the qBittorrent Web UI: Tools → Options → Downloads → Default Save Path
3. In kbarr, go to **Settings → Downloader → qBittorrent** and enter:
   - **URL** — e.g. `http://localhost:8080`
   - **Username** and **Password** — your qBittorrent Web UI credentials

:::info Download path inside Docker
If qBittorrent runs inside Docker, its save path is expressed in its own filesystem namespace. Make sure the path it saves to maps to `<LIBRARY_DIR_HOST>/downloads` on the host so kbarr can read the finished files.
:::

Use **Test connection** in the settings to verify everything is wired up correctly.

---

## Updating

To update kbarr to the latest version:

```bash
docker compose pull
docker compose up -d
```

---

## Uninstalling

```bash
docker compose down -v
```

This stops all containers and removes the internal Postgres volume. Your `LIBRARY_DIR_HOST` folder and `.env` file are not touched.
