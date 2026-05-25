# Search

---

## `GET /api/search?q=<query>`

**Description:** Search for media.

**Request:** Query parameter `q` with the search term.

**Response (200):**

```json
{
  [
    {
      "title": "Attack on Titan",
      "type": "tv",
      "external_id": "1232",
      "source": "tmdb"
    },
    {
      "title": "Attack on Titan",
      "type": "anime",
      "external_id": "1232",
      "source": "anidb"
    }
  ]
}
```

---
