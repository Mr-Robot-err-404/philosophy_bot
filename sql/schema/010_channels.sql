-- +goose Up
CREATE TABLE IF NOT EXISTS channels(
	id VARCHAR(100) NOT NULL,
	handle VARCHAR(100) NOT NULL,
	title VARCHAR(100) NOT NULL,
	created_at TIMESTAMP,
	frequency INTEGER NOT NULL DEFAULT 1,
	videos_since_post INTEGER NOT NULL DEFAULT 0,
	PRIMARY KEY(id)
);

-- +goose Down
DROP TABLE channels;
