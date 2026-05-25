-- name: CreateMedia :one
INSERT INTO media (
    title,
    aid,
    poster_url,
    created_at,
    updated_at
)
VALUES ($1, $2, $3, NOW(), NOW())
RETURNING id, created_at, updated_at, deleted_at, title, aid, poster_url;

-- name: SoftDeleteMediaByID :exec
UPDATE media
SET deleted_at = NOW(), updated_at = NOW()
WHERE id = $1
  AND deleted_at IS NULL;

-- name: CountMediaByAID :one
SELECT COUNT(*)
FROM media
WHERE aid = $1
  AND deleted_at IS NULL;

-- name: ListMedia :many
SELECT id, created_at, updated_at, deleted_at, title, aid, poster_url
FROM media
WHERE deleted_at IS NULL
ORDER BY id ASC;

-- name: GetMediaByID :one
SELECT id, created_at, updated_at, deleted_at, title, aid, poster_url
FROM media
WHERE id = $1
  AND deleted_at IS NULL
LIMIT 1;

-- name: TouchMediaMonitorStatus :exec
UPDATE media
SET updated_at = NOW()
WHERE id = $1
  AND deleted_at IS NULL;
