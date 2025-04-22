-- +goose Up
ALTER TABLE comments
DROP COLUMN channel_id;
