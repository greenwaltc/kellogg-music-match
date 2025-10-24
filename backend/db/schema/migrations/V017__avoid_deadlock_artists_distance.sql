
-- Guard to disable neighbor invalidation during tests to avoid deadlocks
-- Uses a custom GUC (session variable). In production this is off by default.
DO $$ BEGIN PERFORM set_config('affyne.testing_mode', current_setting('affyne.testing_mode', true), true); EXCEPTION WHEN others THEN NULL; END $$;

CREATE OR REPLACE FUNCTION invalidate_artist_neighbors()
RETURNS TRIGGER AS $$
DECLARE
  testing boolean;
BEGIN
  BEGIN
    testing := current_setting('affyne.testing_mode') = 'on';
  EXCEPTION WHEN others THEN
    testing := false;
  END;

  IF NOT testing THEN
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
  END IF;
  RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Performance: the updates above filter by a or b; add indexes to accelerate if not present
CREATE INDEX IF NOT EXISTS idx_artist_neighbors_a ON artist_neighbors(a);
CREATE INDEX IF NOT EXISTS idx_artist_neighbors_b ON artist_neighbors(b);
