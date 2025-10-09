-- Add composite index for track similarity performance (range + spotify_track_id)
CREATE INDEX IF NOT EXISTS idx_spotify_top_track_snapshots_range_track
  ON spotify_top_track_snapshots (range, spotify_track_id);
