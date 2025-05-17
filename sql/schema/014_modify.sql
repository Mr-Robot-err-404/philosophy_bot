-- +goose Up
CREATE TABLE IF NOT EXISTS replies_modified(
	id VARCHAR(500) NOT NULL,
	likes INTEGER NOT NULL,
	quote_id INTEGER NOT NULL REFERENCES cornucopia,
	created_at TIMESTAMP NOT NULL,
	video_id VARCHAR(100) NOT NULL REFERENCES videos ON DELETE CASCADE,
	PRIMARY KEY(id)
);

CREATE TABLE IF NOT EXISTS comments_modified(
	id VARCHAR(500) NOT NULL,
	likes INTEGER NOT NULL,
	quote_id INTEGER NOT NULL REFERENCES cornucopia,
	created_at TIMESTAMP NOT NULL,
	PRIMARY KEY(id)
);

INSERT INTO replies_modified
SELECT * FROM replies;

INSERT INTO comments_modified
SELECT * FROM comments;

-- +goose Down
DROP TABLE replies_modified;
DROP TABLE comments_modified;


