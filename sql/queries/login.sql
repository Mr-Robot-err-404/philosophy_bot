-- name: CreateLoginDetails :one
INSERT INTO login(id, created_at, last_login)
VALUES(
	?,
	datetime('now'),
        datetime('now')
)
RETURNING *;

-- name: GetLoginDetails :one
SELECT * FROM login
WHERE id = ?;

-- name: UpdateLogin :one
UPDATE login
SET last_login = datetime('now')
WHERE id = ?
RETURNING *;
