---
layout: home

hero:
  name: kbarr
  text: Take lite
  tagline: A self-hosted anime management application in the spirit of Sonarr and Radarr.
  actions:
    - theme: brand
      text: Get Started
      link: /installation
    - theme: alt
      text: GitHub
      link: https://github.com/KingBenny101/kbarr

features:
  - title: AniDB
    details: Uses AniDB as the source for anime metadata.
  - title: Docker-first
    details: Runs entirely via Docker Compose. No local dependencies required.
  - title: Built in Go
    details: Hobby project to learn Go. Vibe coded while listening to songs.
---

<div class="home-content">

## What is kbarr?

kbarr is a self-hosted application for managing and automatically downloading anime. You add a show, it monitors for new episodes and handles the rest — searching, downloading, and organising.

It was built as a hobby project to learn Go, using AniDB as the metadata source.

## How it works

kbarr runs as a set of independent services under Docker Compose:

| Service | Role |
|---|---|
| **core** | Main API and frontend |
| **metadata** | Fetches and caches anime metadata from AniDB |
| **indexer** | Matches episodes to torrent files |
| **downloader** | Manages the download queue |

Each service runs independently and communicates over the internal Docker network.

## Workflow

1. **Search** for an anime via the UI
2. **Add** it to your library
3. **Monitor** the seasons and episodes you want
4. kbarr searches for matching torrents and **downloads** them automatically

## Tech stack

- **Backend** — Go
- **Frontend** — React + Mantine
- **Database** — PostgreSQL
- **Metadata** — AniDB
- **Torrent search** — Nyaa.si

## License

kbarr is released under the [MIT License](https://github.com/KingBenny101/kbarr/blob/main/LICENSE).

</div>

<style>
.home-content {
  max-width: 960px;
  margin: 0 auto;
  padding: 48px 24px;
}
</style>
