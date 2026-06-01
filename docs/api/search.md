# Search

---

## `GET /api/search?q={query}`

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

## `GET /api/search-queue`

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

## `DELETE /api/search-queue/{id}`

Remove an entry from the search queue.

**Response (204):** No content.

---
