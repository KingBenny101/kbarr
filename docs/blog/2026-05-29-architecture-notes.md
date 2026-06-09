---
title: Architecture notes
date: 2026-05-29
authors: [kingbenny101]
tags: []
---

This post covers the current architecture of kbarr.

<!-- truncate -->

## Current structure

- `services/core` handles the main API and frontend serving
- `services/downloader`, `services/indexer`, and `services/metadata` are separate services for handling downloads, indexing, and metadata fetching respectively
- All are docker containers managed by docker-compose

