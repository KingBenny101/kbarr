---
sidebar_label: Acknowledgements
sidebar_position: 3
---

# Acknowledgements

Credits the third-party work kbarr depends on: public APIs, shared data sets, service integrations, and open-source libraries.

---

## Data sources & APIs

- **[AniDB](https://anidb.net)** — anime metadata, the [anime-titles dump](https://anidb.net/api/anime-titles.xml.gz), and cover images served from `cdn.anidb.net`. AniDB title data is distributed under [CC BY-NC-SA](https://creativecommons.org/licenses/by-nc-sa/3.0/) and is used here under those terms.
- **[AniList](https://anilist.co)** — the [public GraphQL API](https://graphql.anilist.co) that powers Browse & Discover.
- **[Fribb/anime-lists](https://github.com/Fribb/anime-lists)** — cross-source ID mapping used to translate between AniDB, AniList, and other databases.

## Integrations

- **[qBittorrent](https://www.qbittorrent.org/)** — download client.
- **[Prowlarr](https://prowlarr.com/)** — indexer aggregation.
- **[Jellyfin](https://jellyfin.org/)** — media server library refresh.
- **[kbdex](https://github.com/KingBenny101/kbdex)** — torrent search.
- **[FFmpeg](https://ffmpeg.org/)** — `ffprobe` is used by the availability tracker to read media file details.

## Backend libraries (Go)

- **[anitogo](https://github.com/nssteinbrenner/anitogo)** — release-name parsing.
- **[huma](https://github.com/danielgtaylor/huma)** — API framework and OpenAPI generation.
- **[go-chi/chi](https://github.com/go-chi/chi)** — HTTP router.
- **[uptrace/bun](https://github.com/uptrace/bun)** — SQL ORM.
- **[adrg/strutil](https://github.com/adrg/strutil)** — string similarity scoring.
- **[zeebo/bencode](https://github.com/zeebo/bencode)** — torrent file parsing.
- **[golang.org/x/crypto](https://pkg.go.dev/golang.org/x/crypto)** and **[golang.org/x/text](https://pkg.go.dev/golang.org/x/text)** — cryptography and text normalisation.

## Frontend libraries (JS/TS)

- **[Mantine](https://mantine.dev/)** — UI component library.
- **[React](https://react.dev/)** and **[React Router](https://reactrouter.com/)** — application framework and routing.
- **[graphql-request](https://github.com/jasonkuhrt/graphql-request)** — AniList GraphQL client.
- **[TanStack Virtual](https://tanstack.com/virtual)** — list virtualisation.
- **[Tabler Icons](https://tabler.io/icons)** — icon set.
- **[Tiptap](https://tiptap.dev/)** — rich-text editing.
- **[Recharts](https://recharts.org/)** — charts.
- **[Embla Carousel](https://www.embla-carousel.com/)** — carousels.
- **[Day.js](https://day.js.org/)** — date handling.

## Documentation

- **[Docusaurus](https://docusaurus.io/)** — this documentation site.
