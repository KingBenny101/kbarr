-- name: CountMonitorExactMatch :one
SELECT COUNT(*)
FROM monitors
WHERE library_id = $1
  AND season = $2
  AND episode_number = $3
  AND is_episode = $4
  AND is_season = $5
  AND deleted_at IS NULL;

-- name: CreateMonitor :one
INSERT INTO monitors (
    library_id,
    title,
    episode_title,
    season,
    episode_number,
    is_episode,
    is_season,
    anidb_id,
    status,
    created_at,
    updated_at
)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, COALESCE(NULLIF($9::text, ''), 'monitored'), NOW(), NOW())
RETURNING id, created_at, updated_at, deleted_at, library_id, title, episode_title, season, episode_number, is_episode, is_season, anidb_id, status;

-- name: SoftDeleteMonitorByID :exec
UPDATE monitors
SET deleted_at = NOW(), updated_at = NOW()
WHERE id = $1
  AND deleted_at IS NULL;

-- name: SoftDeleteMonitorsByLibraryID :exec
UPDATE monitors
SET deleted_at = NOW(), updated_at = NOW()
WHERE library_id = $1
  AND deleted_at IS NULL;

-- name: SoftDeleteMonitorsByLibraryIDAndSeason :exec
UPDATE monitors
SET deleted_at = NOW(), updated_at = NOW()
WHERE library_id = $1
  AND season = $2
  AND deleted_at IS NULL;

-- name: ListMonitors :many
SELECT id, created_at, updated_at, deleted_at, library_id, title, episode_title, season, episode_number, is_episode, is_season, anidb_id, status
FROM monitors
WHERE deleted_at IS NULL
ORDER BY id ASC;

-- name: ListMonitorsByLibraryID :many
SELECT id, created_at, updated_at, deleted_at, library_id, title, episode_title, season, episode_number, is_episode, is_season, anidb_id, status
FROM monitors
WHERE library_id = $1
  AND deleted_at IS NULL
ORDER BY id ASC;

-- name: SoftDeleteMonitorsByLibraryIDAndAniDBID :exec
UPDATE monitors
SET deleted_at = NOW(), updated_at = NOW()
WHERE library_id = $1
  AND anidb_id = $2
  AND deleted_at IS NULL;
