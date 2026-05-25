-- name: CountSearchQueueExactMatch :one
SELECT COUNT(*)
FROM search_queues
WHERE library_id = $1
  AND season = $2
  AND episode_number = $3
  AND is_episode = $4
  AND is_season = $5
  AND deleted_at IS NULL;

-- name: CreateSearchQueueEntry :one
INSERT INTO search_queues (
    library_id,
    title,
    episode_title,
    season,
    episode_number,
    is_episode,
    is_season,
    status,
    created_at,
    updated_at
)
VALUES ($1, $2, $3, $4, $5, $6, $7, COALESCE(NULLIF($8::text, ''), 'pending'), NOW(), NOW())
RETURNING id, created_at, updated_at, deleted_at, library_id, title, episode_title, season, episode_number, is_episode, is_season, status;

-- name: ListSearchQueue :many
SELECT id, created_at, updated_at, deleted_at, library_id, title, episode_title, season, episode_number, is_episode, is_season, status
FROM search_queues
WHERE deleted_at IS NULL
ORDER BY created_at ASC;

-- name: SoftDeleteSearchQueueEntryByID :exec
UPDATE search_queues
SET deleted_at = NOW(), updated_at = NOW()
WHERE id = $1
  AND deleted_at IS NULL;
