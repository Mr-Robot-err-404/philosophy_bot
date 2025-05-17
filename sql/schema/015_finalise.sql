-- +goose Up
DROP TABLE replies;
DROP TABLE comments;

ALTER TABLE replies_modified 
RENAME TO replies;

ALTER TABLE comments_modified 
RENAME TO comments;
