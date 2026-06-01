# Downloads

---

## `GET /api/downloads`

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

## `DELETE /api/downloads/{id}`

Remove an entry from the download queue.

**Response (204):** No content.

---
