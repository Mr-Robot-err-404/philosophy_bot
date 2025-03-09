-- name: GetComments :many
SELECT * FROM comments;

-- name: CreateComment :one
INSERT INTO comments (id, likes, quote_id, created_at) 
VALUES (
	?,
	0,
	?,
	datetime('now')
)
RETURNING *;
