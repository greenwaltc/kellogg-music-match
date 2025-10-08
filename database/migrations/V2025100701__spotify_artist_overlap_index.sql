-- Adds composite index to accelerate similarity queries on spotify_top_artist_snapshots
-- Safe to run multiple times due to IF NOT EXISTS
-- Created 2025-10-07

CREATE INDEX IF NOT EXISTS idx_spotify_top_artist_snapshots_range_artist
    ON spotify_top_artist_snapshots(range, spotify_artist_id);
