-- name: CreateTask :one
INSERT INTO tasks (id, video_id, channel_id, quote_id, quota_cost, status, created_at, active_at)
VALUES (
	?,
	?,
	?,
	?,
	?,
	'pending',
	datetime('now'),
	?
)
RETURNING *;

-- name: GetDueTasks :many
SELECT * FROM tasks
WHERE status = 'pending' AND active_at <= datetime('now')
ORDER BY active_at
LIMIT ?;

-- name: GetPendingTasks :many
SELECT * FROM tasks
WHERE status = 'pending'
ORDER BY active_at;

-- name: GetRecentTasks :many
SELECT tasks.*, cornucopia.quote, cornucopia.author
FROM tasks LEFT JOIN cornucopia ON tasks.quote_id = cornucopia.id
ORDER BY tasks.created_at DESC
LIMIT ?;

-- name: FindTaskByVideo :one
SELECT * FROM tasks
WHERE video_id = ?;

-- name: ClaimTask :one
UPDATE tasks
SET status = 'running', attempts = attempts + 1
WHERE id = ? AND status = 'pending'
RETURNING *;

-- name: CompleteTask :one
UPDATE tasks
SET status = 'done', comment_id = ?, completed_at = datetime('now'), error = NULL
WHERE id = ?
RETURNING *;

-- name: FailTask :one
UPDATE tasks
SET status = 'failed', error = ?, completed_at = datetime('now')
WHERE id = ?
RETURNING *;

-- name: RetryTask :one
UPDATE tasks
SET status = 'pending', error = ?, active_at = ?
WHERE id = ?
RETURNING *;

-- name: ReleaseStaleTasks :many
UPDATE tasks
SET status = 'pending'
WHERE status = 'running' AND active_at <= ?
RETURNING *;

-- name: CountTasksByStatus :many
SELECT status, COUNT(*) AS total
FROM tasks
GROUP BY status;

-- name: SumPendingQuota :one
SELECT COALESCE(SUM(quota_cost), 0) AS reserved
FROM tasks
WHERE status IN ('pending', 'running');

-- name: DeleteOldTasks :many
DELETE FROM tasks
WHERE status IN ('done', 'failed') AND completed_at < ?
RETURNING *;
