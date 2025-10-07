-- V026: Spotify top artists and tracks (medium_term preferences)
-- Stores snapshots of a user's Spotify top artists and tracks for matching and recommendations.
-- A snapshot approach lets us re-sync and compare changes over time. For simplicity we keep only latest per user for now.

CREATE TABLE IF NOT EXISTS spotify_top_artist_snapshots (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    fetched_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    range TEXT NOT NULL DEFAULT 'medium_term',
    item_rank INT NOT NULL, -- 1-based rank
    spotify_artist_id TEXT NOT NULL,
    name TEXT NOT NULL,
    genres TEXT[] DEFAULT '{}',
    popularity INT,
    image_url TEXT,
    UNIQUE(user_id, range, item_rank)
);

CREATE INDEX IF NOT EXISTS idx_spotify_top_artist_user_range ON spotify_top_artist_snapshots(user_id, range);
CREATE INDEX IF NOT EXISTS idx_spotify_top_artist_spotify_id ON spotify_top_artist_snapshots(spotify_artist_id);

CREATE TABLE IF NOT EXISTS spotify_top_track_snapshots (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    fetched_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    range TEXT NOT NULL DEFAULT 'medium_term',
    item_rank INT NOT NULL,
    spotify_track_id TEXT NOT NULL,
    name TEXT NOT NULL,
    artist_names TEXT[] DEFAULT '{}',
    artist_ids TEXT[] DEFAULT '{}',
    album_name TEXT,
    album_id TEXT,
    popularity INT,
    preview_url TEXT,
    duration_ms INT,
    image_url TEXT,
    UNIQUE(user_id, range, item_rank)
);

CREATE INDEX IF NOT EXISTS idx_spotify_top_track_user_range ON spotify_top_track_snapshots(user_id, range);
CREATE INDEX IF NOT EXISTS idx_spotify_top_track_spotify_id ON spotify_top_track_snapshots(spotify_track_id);

-- Optional convenience view for current medium_term top artists/tracks (latest fetched_at per user)
CREATE OR REPLACE VIEW v_current_spotify_top_artists AS
SELECT st.* FROM spotify_top_artist_snapshots st
JOIN (
  SELECT user_id, range, MAX(fetched_at) AS max_fetched
  FROM spotify_top_artist_snapshots
  GROUP BY user_id, range
) latest ON latest.user_id = st.user_id AND latest.range = st.range AND latest.max_fetched = st.fetched_at;

CREATE OR REPLACE VIEW v_current_spotify_top_tracks AS
SELECT st.* FROM spotify_top_track_snapshots st
JOIN (
  SELECT user_id, range, MAX(fetched_at) AS max_fetched
  FROM spotify_top_track_snapshots
  GROUP BY user_id, range
) latest ON latest.user_id = st.user_id AND latest.range = st.range AND latest.max_fetched = st.fetched_at;

COMMENT ON TABLE spotify_top_artist_snapshots IS 'Snapshot of a user''s Spotify top artists at a given sync time.';
COMMENT ON TABLE spotify_top_track_snapshots IS 'Snapshot of a user''s Spotify top tracks at a given sync time.';
COMMENT ON COLUMN spotify_top_artist_snapshots.range IS 'Spotify time range: short_term, medium_term, long_term';
COMMENT ON COLUMN spotify_top_track_snapshots.range IS 'Spotify time range: short_term, medium_term, long_term';
