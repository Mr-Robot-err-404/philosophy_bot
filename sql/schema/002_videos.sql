-- +goose Up
CREATE TABLE IF NOT EXISTS videos(
	id VARCHAR(500) NOT NULL,
	PRIMARY KEY(id)
);

-- +goose Down
DROP TABLE videos;
