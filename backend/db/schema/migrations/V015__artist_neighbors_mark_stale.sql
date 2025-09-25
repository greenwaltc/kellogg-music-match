-- Migration: Mark artist_neighbors entries stale instead of deleting
-- Version: V015
-- Description: Update invalidation to set updated_at='epoch' so TTL-based refresh will recompute lazily.

-- Replace the invalidate function created in V014 to mark rows stale.
CREATE OR REPLACE FUNCTION invalidate_artist_neighbors()
RETURNS trigger
LANGUAGE plpgsql
AS $$
BEGIN
  -- For INSERT/DELETE/UPDATE of artist_id, mark corresponding cache rows stale
  IF TG_OP IN ('INSERT','UPDATE') THEN
    UPDATE artist_neighbors
      SET updated_at = 'epoch'
      WHERE a = NEW.artist_id OR b = NEW.artist_id;
  END IF;
  IF TG_OP IN ('DELETE','UPDATE') THEN
    UPDATE artist_neighbors
      SET updated_at = 'epoch'
      WHERE a = OLD.artist_id OR b = OLD.artist_id;
  END IF;
  RETURN NULL; -- statement-level invalidation; nothing to modify in the base row
END;
$$;

-- Performance: the updates above filter by a or b; add indexes to accelerate if not present
CREATE INDEX IF NOT EXISTS idx_artist_neighbors_a ON artist_neighbors(a);
CREATE INDEX IF NOT EXISTS idx_artist_neighbors_b ON artist_neighbors(b);
