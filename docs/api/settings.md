# Settings

---

## `GET /api/settings`

**Description:** Get the settings.

**Request:** None

**Response (200):**

```json
{
	"anidbClient":       "client",
	"anidbVersion":      "1",
	"anidbSyncInterval": "1440",
	"tmdbApiKey":        "api_key",
	"prowlarrUrl":       "http://localhost:9696",
	"prowlarrApiKey":    "api_key",
	"prowlarrInterval":  "60",
	"autoMonitorOnAdd":  "false",
}
```

---

## `POST /api/settings`

**Description:** Update the settings.

**Request:**

```json
{
	"anidbClient":       "client",
	"anidbVersion":      "1",
	"anidbSyncInterval": "1440",
	"tmdbApiKey":        "api_key",
	"prowlarrUrl":       "http://localhost:9696",
	"prowlarrApiKey":    "api_key",
	"prowlarrInterval":  "60",
	"autoMonitorOnAdd":  "false",
}
```

**Response (200):**

```json
{
	"success":       "true",
	"message":      "Settings updated successfully",
}
```

---