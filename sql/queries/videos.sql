-- name: GetVideos :many
SELECT * FROM videos;

-- name: SaveVideo :one
INSERT INTO videos (id)
VALUES (?)
RETURNING *;
