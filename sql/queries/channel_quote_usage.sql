-- name: IsQuoteUsed :one
SELECT EXISTS(SELECT 1 FROM channel_quote_usage WHERE channel_id = ? AND quote_id = ?);

-- name: SelectUnusedQuotes :many
SELECT *
FROM cornucopia q
WHERE q.id NOT IN (
    SELECT quote_id 
    FROM channel_quote_usage 
    WHERE channel_id = ?
);

-- name: SaveUsage :one
INSERT INTO channel_quote_usage(channel_id, quote_id)
VALUES (?, ?)
RETURNING *;
