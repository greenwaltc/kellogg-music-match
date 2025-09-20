ALTER TABLE user_artists
ADD COLUMN rank SMALLINT NOT NULL DEFAULT 1;

ALTER TABLE user_artists
ADD CONSTRAINT user_rank_unique UNIQUE (user_id, rank);

