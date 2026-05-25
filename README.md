# kbarr

A self-hosted anime management application in the spirit of Sonarr and Radarr.

Built as a hobby project to learn Go.

Uses AniDB as the source for anime metadata.

## Quick Start

Requirements: Docker and Docker Compose.

You also need qBittorrent running separately and reachable at the default URL
(`http://host.docker.internal:8080` in Docker, `http://localhost:8080` for local
`air` runs unless you override `QBITTORRENT_URL`).

```bash
curl -O https://raw.githubusercontent.com/kingbenny101/kbarr/main/docker-compose.yml
curl -O https://raw.githubusercontent.com/kingbenny101/kbarr/main/.env.example
cp .env.example .env
```

Edit .env and set PROWLARR_API_KEY, then:

```bash
docker compose up -d
```

kbarr is now running at http://localhost:8282.

To update to the latest version:

```bash
docker compose pull
docker compose up -d
```

## AI Usage

Vibe coded while listening to songs.

## License

kbarr is licensed under the MIT License. See [LICENSE](LICENSE) for more details.
