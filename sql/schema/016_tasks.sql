-- +goose Up
CREATE TABLE tasks(
	id VARCHAR(100) NOT NULL,
	video_id VARCHAR(100) NOT NULL,
	channel_id VARCHAR(100) NOT NULL,
	quote_id INTEGER NOT NULL REFERENCES cornucopia,
	status VARCHAR(20) NOT NULL DEFAULT 'pending',
	attempts INTEGER NOT NULL DEFAULT 0,
	quota_cost INTEGER NOT NULL DEFAULT 0,
	comment_id VARCHAR(100),
	error TEXT,
	created_at TIMESTAMP NOT NULL,
	active_at TIMESTAMP NOT NULL,
	claimed_at TIMESTAMP,
	completed_at TIMESTAMP,
	PRIMARY KEY(id)
);

CREATE INDEX idx_tasks_due ON tasks(status, active_at);
CREATE UNIQUE INDEX idx_tasks_video ON tasks(video_id);

-- +goose Down
DROP INDEX idx_tasks_video;
DROP INDEX idx_tasks_due;
DROP TABLE tasks;
