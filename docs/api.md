# API Reference

All protected endpoints require an `Authorization: Bearer <token>` header. Tokens are obtained via login.

---

## System

### `GET /api/health`

Health check. No auth required.

**Response (200):** `OK`

---

### `GET /api/version`

Get the running version of kbarr.

**Response (200):**

```json
{
  "version": "0.1.0"
}
```

---

### `GET /api/workers`

Get the status of all background services.

**Response (200):**

```json
[
  {
    "name": "metadata",
    "display_name": "Metadata",
    "running": true,
    "error": ""
  }
]
```

---

### `GET /api/workers/{name}/logs`

Get logs for a specific service. `name` is one of `metadata`, `indexer`, `downloader`.

**Response (200):** JSON log entries.

---

## Auth

### `POST /api/auth/login`

No auth required.

**Request:**

```json
{
  "username": "admin",
  "password": "password"
}
```

**Response (200):**

```json
{
  "token": "eyJ..."
}
```

---

### `GET /api/auth/me`

Get the currently authenticated user.

**Response (200):**

```json
{
  "username": "admin"
}
```

---

### `POST /api/auth/logout`

Revoke the current token.

**Response (200):** No content.

---

### `POST /api/auth/credentials`

Change username or password.

**Request:**

```json
{
  "currentPassword": "old",
  "newUsername": "admin",
  "newPassword": "new"
}
```

**Response (200):** No content.

---

## Library

### `GET /api/library`

Get all media in the library.

**Response (200):**

```json
[
  {
    "id": 1,
    "title": "Attack on Titan",
    "type": "anime",
    "external_id": "1",
    "source": "anidb"
  }
]
```

---

### `POST /api/library`

Add media to the library.

**Request:**

```json
{
  "title": "Attack on Titan",
  "external_id": "1",
  "type": "anime",
  "source": "anidb"
}
```

**Response (201):** Added successfully.

**Response (200):** Already exists in library.

---

### `GET /api/library/{id}`

Get a specific library item.

**Response (200):**

```json
{
  "id": 1,
  "title": "Attack on Titan",
  "type": "anime",
  "external_id": "1",
  "source": "anidb"
}
```

---

### `DELETE /api/library/{id}`

Remove media from the library.

**Response (204):** No content.

---

### `PUT /api/library/{id}/monitor`

Set the monitored status of a library item.

**Request:**

```json
{
  "monitored": true
}
```

**Response (204):** No content.

---

### `GET /api/library/{id}/monitored`

Get all monitors for a library item.

**Response (200):**

```json
[
  {
    "id": 1,
    "library_id": 1,
    "title": "Attack on Titan",
    "season": 1,
    "episode": 1
  }
]
```

---

## Monitor

### `GET /api/monitor`

Get all monitored items.

**Response (200):**

```json
[
  {
    "id": 1,
    "library_id": 1,
    "title": "Attack on Titan",
    "season": 1,
    "episode": 1
  }
]
```

---

### `POST /api/monitor`

Add a single monitor entry.

**Request:**

```json
{
  "library_id": 1,
  "title": "Attack on Titan",
  "season": 1,
  "episode": 1
}
```

**Response (201):** No content.

---

### `POST /api/monitor/bulk`

Add multiple monitor entries at once.

**Request:**

```json
[
  {
    "library_id": 1,
    "title": "Attack on Titan",
    "season": 1,
    "episode": 1
  }
]
```

**Response (201):** No content.

---

### `DELETE /api/monitor/{id}`

Remove a monitor entry.

**Response (204):** No content.

---

### `POST /api/unmonitor`

Unmonitor a specific episode.

**Request:**

```json
{
  "library_id": 1,
  "anidb_id": "1"
}
```

**Response (204):** No content.

---

### `POST /api/unmonitor/season`

Unmonitor an entire season.

**Request:**

```json
{
  "library_id": 1,
  "season": 1
}
```

**Response (204):** No content.

---

## Search

### `GET /api/search?q={query}`

Search for media across all sources.

**Response (200):**

```json
[
  {
    "title": "Attack on Titan",
    "type": "anime",
    "external_id": "1",
    "source": "anidb",
    "added": false
  }
]
```

---

### `GET /api/search-queue`

Get all pending search queue entries.

**Response (200):**

```json
[
  {
    "id": 1,
    "title": "Attack on Titan",
    "season": 1,
    "episode": 1
  }
]
```

---

### `DELETE /api/search-queue/{id}`

Remove an entry from the search queue.

**Response (204):** No content.

---

## Downloads

### `GET /api/downloads`

Get all entries in the download queue.

**Response (200):**

```json
[
  {
    "id": 1,
    "title": "Attack on Titan",
    "season": 1,
    "episode": 1,
    "status": "downloading"
  }
]
```

---

### `DELETE /api/downloads/{id}`

Remove an entry from the download queue.

**Response (204):** No content.

---

## Settings

### `GET /api/settings`

Get all settings.

**Response (200):**

```json
{
  "anidbClient": "client",
  "anidbVersion": "1",
  "anidbSyncInterval": "1440",
  "prowlarrUrl": "http://localhost:9696",
  "prowlarrApiKey": "api_key",
  "prowlarrInterval": "60",
  "autoMonitorOnAdd": "false"
}
```

---

### `POST /api/settings`

Update settings. Only include keys you want to change.

**Request:**

```json
{
  "prowlarrUrl": "http://localhost:9696",
  "prowlarrApiKey": "api_key"
}
```

**Response (200):**

```json
{
  "success": "true",
  "message": "Settings updated successfully"
}
```

---
