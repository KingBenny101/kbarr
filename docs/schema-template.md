# Database Schema

## Tables

### `media`

**Purpose:** Stores all anime/TV/movie metadata and library status.

| Column             | Type     | Constraints                 | Notes                                 |
| ------------------ | -------- | --------------------------- | ------------------------------------- |
| `id`               | INTEGER  | PRIMARY KEY, AUTO INCREMENT | Unique identifier                     |
| `title`            | TEXT     | NOT NULL                    | Display title                         |
| `title_original`   | TEXT     |                             | Original language title               |
| `alternate_titles` | JSON     |                             | Array of alternative titles           |
| `description`      | TEXT     |                             | Long description                      |
| `status`           | TEXT     | DEFAULT 'watching'          | watching, completed, on_hold, dropped |
| `type`             | TEXT     |                             | anime, tv, movie                      |
| `episodes`         | INTEGER  | DEFAULT 0                   | Total episodes                        |
| `seasons`          | INTEGER  | DEFAULT 0                   | Total seasons                         |
| `year`             | INTEGER  |                             | Release year                          |
| `cover_image`      | TEXT     |                             | URL to poster/cover                   |
| `banner_image`     | TEXT     |                             | URL to banner                         |
| `external_id`      | TEXT     |                             | ID from external service              |
| `source`           | TEXT     |                             | anidb, tmdb, etc.                     |
| `monitored`        | BOOLEAN  | DEFAULT false               | Whether to track new releases         |
| `added_at`         | DATETIME | AUTO TIMESTAMP              | When added to library                 |
| `updated_at`       | DATETIME | AUTO TIMESTAMP              | Last modified                         |

---

### `settings`

**Purpose:** App-wide configuration key-value pairs.

| Column  | Type    | Constraints      | Notes                         |
| ------- | ------- | ---------------- | ----------------------------- |
| `id`    | INTEGER | PRIMARY KEY      | Unique identifier             |
| `key`   | TEXT    | UNIQUE, NOT NULL | Setting name (e.g., "apiKey") |
| `value` | TEXT    |                  | Setting value                 |

**Indexes:**

- UNIQUE on `key` for fast lookups


---

### `table_name`

**Purpose:** Brief description of what this table stores.

| Column       | Type     | Constraints                 | Notes              |
| ------------ | -------- | --------------------------- | ------------------ |
| `id`         | INTEGER  | PRIMARY KEY, AUTO INCREMENT | Unique identifier  |
| `field_name` | TEXT     | NOT NULL                    | Description        |
| `created_at` | DATETIME | DEFAULT CURRENT_TIMESTAMP   | Auto-set on insert |
| `updated_at` | DATETIME | DEFAULT CURRENT_TIMESTAMP   | Auto-updated       |

**Indexes:**

- PRIMARY KEY on `id`
- UNIQUE on `field_name` (if applicable)

---

