-- +goose Up
CREATE TABLE IF NOT EXISTS quota(
	id VARCHAR(100) NOT NULL,
	quota INTEGER NOT NULL,
	created_at TIMESTAMP NOT NULL,
	updated_at TIMESTAMP NOT NULL,
	PRIMARY KEY(id)
);

-- +goose Down
DROP TABLE quota;
