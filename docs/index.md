# kbarr

A self-hosted media management application in the spirit of Sonarr and Radarr. 

Built as a hobby project to learn Go.

Frontend is vibe coded.

## Quick Start with Docker

The easiest way to run kbarr is using Docker, kbarr images are automatically published to GitHub Container Registry:

```bash
# Pull the latest image
docker pull ghcr.io/kingbenny101/kbarr:latest

# Run the container (equivalent to docker-compose)
docker run -d \
  --name kbarr \
  -p 8282:8282 \
  -v kbarr-data:/app/data \
  --restart unless-stopped \
  ghcr.io/kingbenny101/kbarr
```

The application will be available at `http://localhost:8282`.

## License

kbarr is licensed under the MIT License. See [LICENSE](LICENSE) for more details.
