-- name: CreateDetailed :one
INSERT INTO detaileds (
    title,
    aid,
    library_id,
    alternate_titles,
    description,
    release_date,
    genres,
    poster_url,
    total_episodes,
    total_seasons,
    created_at,
    updated_at
)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, NOW(), NOW())
RETURNING id, created_at, updated_at, deleted_at, title, aid, library_id, alternate_titles, description, release_date, genres, poster_url, total_episodes, total_seasons;

-- name: GetDetailedByLibraryID :one
SELECT id, created_at, updated_at, deleted_at, title, aid, library_id, alternate_titles, description, release_date, genres, poster_url, total_episodes, total_seasons
FROM detaileds
WHERE library_id = $1
  AND deleted_at IS NULL
LIMIT 1;

-- name: SoftDeleteDetailedByLibraryID :exec
UPDATE detaileds
SET deleted_at = NOW(), updated_at = NOW()
WHERE library_id = $1
  AND deleted_at IS NULL;

-- name: CreateEpisode :one
INSERT INTO episodes (
    detailed_id,
    anidb_id,
    type,
    ep_no,
    title,
    air_date,
    created_at,
    updated_at
)
VALUES ($1, $2, $3, $4, $5, $6, NOW(), NOW())
RETURNING id, created_at, updated_at, deleted_at, detailed_id, anidb_id, type, ep_no, title, air_date;

-- name: ListEpisodesByDetailedID :many
SELECT id, created_at, updated_at, deleted_at, detailed_id, anidb_id, type, ep_no, title, air_date
FROM episodes
WHERE detailed_id = $1
  AND deleted_at IS NULL
ORDER BY id ASC;
