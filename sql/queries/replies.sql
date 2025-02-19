-- name: GetReplies :many
SELECT * FROM replies;

-- name: StoreReply :many
INSERT INTO replies(id, likes, quote_id, video_id, created_at)
VALUES (
	?,
	?,
	?,
	?,
        datetime('now')
)
RETURNING *;
