# System

---

## `GET /api/health`

Health check. No auth required.

**Response (200):** `OK`

---

## `GET /api/version`

Get the running version of kbarr.

**Response (200):**

```json
{
  "version": "0.1.0"
}
```

---

## `GET /api/workers`

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

## `GET /api/workers/{name}/logs`

Get logs for a specific service. `name` is one of `metadata`, `indexer`, `downloader`.

**Response (200):** JSON log entries.

---
