-- V024__concert_event_artist_relevancy_mv.sql
-- Purpose: Pre-compute per-event relevancy (max artist.musicbrainz_score)
--   to eliminate repeated lateral subquery execution at read time.
--   This is optional; enable when read load justifies added maintenance cost.
--
-- Design:
--   1. Create a materialized view concert_event_relevancy_mv that stores:
--        event_id, relevancy (MAX score across matched artists)
--      using the same substring matching rule currently embedded inline.
--   2. Index event_id for fast join.
--   3. Provide a REFRESH helper function with CONCURRENTLY option gate.
--
-- Usage Changes (NOT applied automatically by this migration):
--   After deploying you can refactor queries to:
--     LEFT JOIN concert_event_relevancy_mv cer ON cer.event_id = ce.id
--     and select COALESCE(cer.relevancy,0) AS relevancy
--   eliminating the LATERAL subquery in each query.
--
-- Refresh Strategy:
--   For now, full refresh on demand. If write frequency to concert_event_artists
--   grows, consider an incremental maintenance trigger into a real table.
--
-- Rollback (manual):
--   DROP MATERIALIZED VIEW IF EXISTS concert_event_relevancy_mv;
--   DROP FUNCTION IF EXISTS refresh_concert_event_relevancy_mv(boolean);

CREATE MATERIALIZED VIEW IF NOT EXISTS concert_event_relevancy_mv AS
SELECT
  cea.event_id,
  MAX(ar.musicbrainz_score) AS relevancy
FROM concert_event_artists cea
JOIN concert_artists ca ON ca.id = cea.artist_id
LEFT JOIN artists ar ON lower(ar.name) LIKE ('%' || lower(ca.name) || '%')
GROUP BY cea.event_id;

CREATE UNIQUE INDEX IF NOT EXISTS idx_concert_event_relevancy_mv_event_id
  ON concert_event_relevancy_mv (event_id);

-- Helper function to refresh; pass true for concurrent if large
CREATE OR REPLACE FUNCTION refresh_concert_event_relevancy_mv(p_concurrent boolean DEFAULT false)
RETURNS void LANGUAGE plpgsql AS $$
BEGIN
  IF p_concurrent THEN
    EXECUTE 'REFRESH MATERIALIZED VIEW CONCURRENTLY concert_event_relevancy_mv';
  ELSE
    EXECUTE 'REFRESH MATERIALIZED VIEW concert_event_relevancy_mv';
  END IF;
END;$$;

-- Initial refresh (non-concurrent)
REFRESH MATERIALIZED VIEW concert_event_relevancy_mv;
