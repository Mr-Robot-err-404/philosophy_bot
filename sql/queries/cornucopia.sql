-- name: GetQuotes :many
SELECT * FROM cornucopia;

-- name: CreateQuote :one
INSERT INTO cornucopia (id, quote, author, categories, created_at)
VALUES (
	?,
	?, 
	?, 
	?,
        datetime('now')
)
RETURNING *;
