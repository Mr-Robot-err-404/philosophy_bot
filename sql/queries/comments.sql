-- name: GetComments :many
SELECT * FROM comments;

-- name: GetPopularComments :many
SELECT comments.*, cornucopia.quote, cornucopia.author
FROM comments JOIN cornucopia ON comments.quote_id = cornucopia.id
ORDER BY comments.likes DESC;

-- name: CreateComment :one
INSERT INTO comments (id, likes, quote_id, created_at) 
VALUES (
	?,
	0,
	?,
	datetime('now')
)
RETURNING *;

-- name: UpdateCommentLikes :one
UPDATE comments 
SET likes = ?
WHERE id = ?
RETURNING *;
