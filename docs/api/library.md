# Library

---

## `GET /api/libary`

**Description:** Get all media in the library.

**Request:** None

**Response (200):**

```json
{
  [
    {
      "title": "Attack on Titan",
      "type": "anime",
      "external_id": "1",
      "source": "anidb"
    },
    {
      "title": "The Witcher",
      "type": "tv",
      "external_id": "2",
      "source": "tmdb"
    }
  ]
}
```

---

## `POST /api/library`

**Description:** Add media to the library.

**Request:**

```json
{
  "title": "Attack on Titan",
  "external_id": "1",
  "type": "anime",
  "source": "anidb"
}
```
**Response (201):**

```json
{
  "message": "Media added successfully!"
}
```

**Response (200):**

```json
{
  "message": "Media already exists in library!"
}
```

---

## `GET /api/library/:id`

**Description:** Get a specific media item from the library.

**Request:** Path parameter `id` with the media ID.

**Response (200):**

```json
{
  "id": 1
}
```

---

## `DELETE /api/library/:id`

**Description:** Remove media from the library.

**Response (204):** No content


---