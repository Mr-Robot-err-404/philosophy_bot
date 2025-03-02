-- name: GetChannels :many
SELECT * FROM channels;

-- name: CreateChannel :one
INSERT INTO channels(id, title, handle, created_at)
VALUES (
	?,
	?,
	?,
        datetime('now')
)
RETURNING *;

-- name: DeleteChannel :one
DELETE FROM channels
WHERE id = ?
RETURNING *;
