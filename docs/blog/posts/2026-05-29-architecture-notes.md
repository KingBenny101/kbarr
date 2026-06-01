---
date: 2026-05-29
title: Architecture notes
---

This post covers the current architecture of kbarr.

## Current structure

- `services/core` handles the main API and frontend serving
- `services/downloader`, `services/indexer`, and `services/metadata` are separate services for handling downloads, indexing, and metadata fetching respectively
- All are docker containers managed by docker-compose
