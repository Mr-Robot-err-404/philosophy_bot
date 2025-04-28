-- name: GetReplies :many
SELECT * FROM replies;

-- name: GetPopularReplies :many
SELECT replies.*, cornucopia.quote, cornucopia.author
FROM replies JOIN cornucopia ON replies.quote_id = cornucopia.id
ORDER BY replies.likes DESC;

-- name: StoreReply :one
INSERT INTO replies(id, likes, quote_id, video_id, created_at)
VALUES (
	?,
	?,
	?,
	?,
        datetime('now')
)
RETURNING *;

-- name: LinkVideo :one
UPDATE replies
SET video_id = ?
WHERE id = ?
RETURNING *;

-- name: UpdateLikes :one
UPDATE replies
SET likes = ?
WHERE id = ?
RETURNING *;
