-- GetVideos: many
SELECT * FROM videos;

-- SaveVideo: one
INSERT INTO videos (id)
VALUES (?)
RETURNING *;
