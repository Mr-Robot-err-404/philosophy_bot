-- name: GetQuota :one
SELECT * FROM quota;

-- name: SetupQuota :one
INSERT INTO quota(id, created_at, updated_at, quota)
VALUES(
	?,
	datetime('now'),
        datetime('now'),
	10000
)
RETURNING *;

-- name: RefreshQuota :one
UPDATE quota 
SET updated_at = datetime('now'), quota = 10000
WHERE id = 'not_all_who_wander_are_lost'
RETURNING *;

-- name: UpdateQuota :one
UPDATE quota 
SET updated_at = datetime('now'), quota = ?
WHERE id = 'not_all_who_wander_are_lost'
RETURNING *;

