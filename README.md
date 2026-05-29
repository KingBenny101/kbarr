# kbarr

A self-hosted anime management application in the spirit of Sonarr and Radarr.

Built as a hobby project to learn Go.

Uses AniDB as the source for anime metadata.

## Quick Start

Requirements: Docker and Docker Compose.

Download the latest release files:

```bash
curl -LO https://github.com/kingbenny101/kbarr/releases/latest/download/docker-compose.yml
curl -LO https://github.com/kingbenny101/kbarr/releases/latest/download/example.env
cp example.env .env
```

Edit `.env` and set `PROWLARR_API_KEY`, then:

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