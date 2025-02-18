-- +goose Up
CREATE TABLE IF NOT EXISTS replies(
	id VARCHAR(500) NOT NULL,
	likes INTEGER NOT NULL,
	quote_id INTEGER NOT NULL REFERENCES cornucopia,
	created_at TIMESTAMP,
	PRIMARY KEY(id)
);

-- +goose Down
DROP TABLE replies;
