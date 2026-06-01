# Settings

---

## `GET /api/settings`

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

## `POST /api/settings`

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
