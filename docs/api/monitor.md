# Monitor

---

## `GET /api/monitor`

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

## `POST /api/monitor`

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

## `POST /api/monitor/bulk`

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

## `DELETE /api/monitor/{id}`

Remove a monitor entry.

**Response (204):** No content.

---

## `POST /api/unmonitor`

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

## `POST /api/unmonitor/season`

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
