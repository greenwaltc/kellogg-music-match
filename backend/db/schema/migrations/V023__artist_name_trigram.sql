-- V023__artist_name_trigram.sql
-- Purpose: Improve performance of relevancy lateral subqueries that perform
--   pattern scans:  lower(ar.name) LIKE ('%' || lower(ca2.name) || '%')
-- Strategy:
--   1. Ensure pg_trgm extension is available (needed for efficient GIN index
--      usage on wildcard / similarity searches with leading %).
--   2. Create a GIN trigram index on lower(artists.name) to accelerate
--      substring searches when deriving MAX(musicbrainz_score) per event.
--
-- Expected Impact:
--   The lateral subquery in queries:
--     GetUpcomingConcertEventsInCity
--     GetChicagoEventsWithArtistSearch
--     GetConcertEventsInDateRangeWithInterest
--   will switch from sequential scans over artists to index-assisted bitmap
--   scans using pg_trgm, substantially reducing latency when the artists
--   table is large (47k+ MusicBrainz records) and event fan-out grows.
--
-- Safety:
--   IF NOT EXISTS guards allow re-running without error. Extension creation is
--   idempotent. Index build is non-concurrent (Flyway wraps in a txn by
--   default) — acceptable given modest table size; switch to CONCURRENTLY if
--   future downtime constraints arise (requires Flyway config to disable txn).
--
-- Rollback (manual):
--   DROP INDEX IF EXISTS idx_artists_lower_name_trgm;
--   (Extension usually left in place; dropping requires ensuring no other deps.)

CREATE EXTENSION IF NOT EXISTS pg_trgm WITH SCHEMA public;

CREATE INDEX IF NOT EXISTS idx_artists_lower_name_trgm
  ON artists USING gin (lower(name) gin_trgm_ops);

-- Optionally, if queries begin filtering by concert_artists.name similarly,
-- you may add:
-- CREATE INDEX IF NOT EXISTS idx_concert_artists_lower_name_trgm
--   ON concert_artists USING gin (lower(name) gin_trgm_ops);

-- Verification suggestion (post-deploy):
-- EXPLAIN (ANALYZE, BUFFERS)
-- SELECT MAX(ar.musicbrainz_score)
-- FROM concert_event_artists cea2
-- JOIN concert_artists ca2 ON cea2.artist_id = ca2.id
-- LEFT JOIN artists ar ON lower(ar.name) LIKE ('%' || lower(ca2.name) || '%')
-- WHERE cea2.event_id = '<some-event-uuid>';
-- Ensure the plan shows Bitmap Index Scans using idx_artists_lower_name_trgm
