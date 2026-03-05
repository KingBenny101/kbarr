# API Routes

### `GET /api/media`

**Description:** Get all media in the library.

**Request:** None

**Response (200):**

```json
{
  "data": [
    {
      "id": 1,
      "title": "Attack on Titan",
      "monitored": true
    }
  ]
}
```

---

### `POST /api/media`

**Description:** Add media to the library.

**Request:**

```json
{
  "title": "Attack on Titan",
  "monitored": true
}
```

**Response (201):**

```json
{
  "id": 1
}
```

---

### `DELETE /api/media/:id`

**Description:** Remove media from the library.

**Response (204):** No content


---

## Endpoint Template

### `METHOD /path`

**Description:** Brief description of what this endpoint does.

**Request:**

```json
{
  "field": "value"
}
```

**Response (200):**

```json
{
  "field": "value"
}
```

**Error Codes:**

- `400` — Bad Request
- `404` — Not Found
- `500` — Server Error
