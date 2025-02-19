// Code generated by sqlc. DO NOT EDIT.
// versions:
//   sqlc v1.27.0
// source: login.sql

package database

import (
	"context"
)

const createLoginDetails = `-- name: CreateLoginDetails :one
INSERT INTO login(id, created_at, last_login)
VALUES(
	?,
	datetime('now'),
        datetime('now')
)
RETURNING id, created_at, last_login
`

func (q *Queries) CreateLoginDetails(ctx context.Context, id string) (Login, error) {
	row := q.db.QueryRowContext(ctx, createLoginDetails, id)
	var i Login
	err := row.Scan(&i.ID, &i.CreatedAt, &i.LastLogin)
	return i, err
}

const getLoginDetails = `-- name: GetLoginDetails :one
SELECT id, created_at, last_login FROM login
`

func (q *Queries) GetLoginDetails(ctx context.Context) (Login, error) {
	row := q.db.QueryRowContext(ctx, getLoginDetails)
	var i Login
	err := row.Scan(&i.ID, &i.CreatedAt, &i.LastLogin)
	return i, err
}

const updateLogin = `-- name: UpdateLogin :one
UPDATE login
SET last_login = datetime('now')
WHERE id = ?
RETURNING id, created_at, last_login
`

func (q *Queries) UpdateLogin(ctx context.Context, id string) (Login, error) {
	row := q.db.QueryRowContext(ctx, updateLogin, id)
	var i Login
	err := row.Scan(&i.ID, &i.CreatedAt, &i.LastLogin)
	return i, err
}
