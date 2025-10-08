-- Ensure efficient lookups and upserts on spotify_top_artist_snapshots by (user_id, range, item_rank)
-- (There may already be a unique constraint; this creates a supporting index if not present.)
CREATE INDEX IF NOT EXISTS idx_spotify_top_artist_snapshots_user_range_rank
    ON spotify_top_artist_snapshots(user_id, range, item_rank);
