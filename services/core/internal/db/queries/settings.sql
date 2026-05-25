-- name: ListSettings :many
SELECT id, created_at, updated_at, deleted_at, key, value
FROM settings
WHERE deleted_at IS NULL
ORDER BY key ASC;

-- name: UpsertSetting :one
INSERT INTO settings (
    key,
    value,
    created_at,
    updated_at
)
VALUES ($1, $2, NOW(), NOW())
ON CONFLICT (key) DO UPDATE
SET value = EXCLUDED.value,
    updated_at = NOW(),
    deleted_at = NULL
RETURNING id, created_at, updated_at, deleted_at, key, value;

-- name: InsertSettingIfMissing :exec
INSERT INTO settings (
    key,
    value,
    created_at,
    updated_at
)
VALUES ($1, $2, NOW(), NOW())
ON CONFLICT (key) DO NOTHING;
