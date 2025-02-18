-- +goose Up
CREATE TABLE IF NOT EXISTS cornucopia (
        id INTEGER NOT NULL,
	quote VARCHAR(500) NOT NULL,
	author VARCHAR(500) NOT NULL,
	categories VARCHAR(500) NOT NULL,
	created_at TIMESTAMP,
	PRIMARY KEY (id)
);

-- +goose Down
DROP TABLE cornucopia;
