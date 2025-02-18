-- GetReplies: many
SELECT * FROM replies;

-- StoreReply: one 
INSERT INTO replies(id, likes, quote_id, video_id, created_at)
VALUES (
	?,
	?,
	?,
	?,
        datetime('now')
)
RETURNING *;
