-- name: GetChannels :many
SELECT * FROM channels;

-- name: CreateChannel :one
INSERT INTO channels(id, title, handle, created_at, frequency, videos_since_post)
VALUES (
	?,
	?,
	?,
        datetime('now'),
	?,
	?
)
RETURNING *;

-- name: DeleteChannel :one
DELETE FROM channels
WHERE id = ?
RETURNING *;

-- name: FindChannel :one
SELECT * FROM channels
WHERE id = ?;

-- name: UpdateChannelFreq :one
UPDATE channels
SET frequency = ?
WHERE id = ?
RETURNING *;

-- name: UpdateVideosSincePost :one
UPDATE channels
SET videos_since_post = ?
WHERE id = ?
RETURNING *;
