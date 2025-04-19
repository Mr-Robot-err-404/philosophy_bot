-- +goose Up
CREATE TABLE channel_quote_usage(
	channel_id VARCHAR(100) NOT NULL,
	quote_id INTEGER NOT NULL,
	PRIMARY KEY (channel_id, quote_id),
	FOREIGN KEY (channel_id) REFERENCES channels(id),
	FOREIGN KEY (quote_id) REFERENCES cornucopia(id)
);

-- +goose Down
DROP TABLE channel_quote_usage;
