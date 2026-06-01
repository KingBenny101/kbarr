# Library

---

## `GET /api/library`

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

## `POST /api/library`

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

## `GET /api/library/{id}`

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

## `DELETE /api/library/{id}`

Remove media from the library.

**Response (204):** No content.

---

## `PUT /api/library/{id}/monitor`

Set the monitored status of a library item.

**Request:**

```json
{
  "monitored": true
}
```

**Response (204):** No content.

---

## `GET /api/library/{id}/monitored`

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
