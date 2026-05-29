# Installation

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
