# kbarr

A self-hosted anime management application in the spirit of Sonarr and Radarr. 

Built as a hobby project to learn Go.

Uses AniDB as the source for anime metadata.

## Quick Start

Requirements: Docker and Docker Compose.

```bash
curl -O https://raw.githubusercontent.com/KingBenny101/kbarr/main/docker-compose.yml
curl -O https://raw.githubusercontent.com/KingBenny101/kbarr/main/.env.example
cp .env.example .env
```

Edit .env and set PROWLARR_API_KEY and DOWNLOAD_DIR_HOST, then:

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

AI is used to implement small features and in debugging, but all the code is manually reviewed
 

## License

kbarr is licensed under the MIT License. See [LICENSE](LICENSE) for more details.
